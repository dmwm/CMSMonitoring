"""
Methods for processing the history in a schedd queue.
"""

import datetime
import json
import logging
import multiprocessing
import os
import time
import traceback

import classad
import htcondor

from utils import send_email_alert, time_remaining, TIMEOUT_MINS
from convert_to_json import convert_to_json, unique_doc_id
from nats import get_nats_connection, publish_job_to_nats

# Main query time, should be same with cron schedule.
QUERY_TIME_PERIOD = 720  # 12 minutes

# Even in checkpoint.json last query time is older than this, older than "now()-12h" results will be ignored.
CRAB_MAX_QUERY_TIME_SPAN = 12 * 3600  # 12 hours

# If last query time in checkpoint.json is too old, but not from crab, results older
# than "now()-RETENTION_POLICY" will be ignored.
RETENTION_POLICY = 39 * 24 * 3600  # 39 days

_WORKDIR = os.getenv("SPIDER_WORKDIR", "/opt/spider")
_CHECKPOINT_JSON = os.path.join(_WORKDIR, "checkpoint.json")


def process_schedd(
    starttime, last_completion, checkpoint_queue, schedd_ad, args, metadata=None
):
    """
    Given a schedd, process its entire set of history since last checkpoint.
    """
    my_start = time.time()
    pool_name = schedd_ad.get("CMS_Pool", "Unknown")
    if time_remaining(starttime) < 10:
        message = (
            "No time remaining to process %s history; exiting." % schedd_ad["Name"]
        )
        logging.error(message)
        send_email_alert(
            args.email_alerts, "spider_cms history timeout warning", message
        )
        return last_completion

    metadata = metadata or {}
    schedd = htcondor.Schedd(schedd_ad)
    _q = """
        (JobUniverse == 5) && (CMS_Type != "DONOTMONIT")
        &&
        (
            EnteredCurrentStatus >= %(last_completion)d
            || CRAB_PostJobLastUpdate >= %(last_completion)d
        )
        """
    history_query = classad.ExprTree(_q % {"last_completion": last_completion - QUERY_TIME_PERIOD})
    logging.info(
        "Querying %s for history: %s.  " "%.1f minutes of ads",
        schedd_ad["Name"],
        history_query,
        (time.time() - last_completion) / 60.0,
    )
    count = 0
    published_count = 0
    sent_warnings = False
    timed_out = False
    error = False
    latest_completion = last_completion
    
    # Get NATS JetStream connection
    try:
        _, jetstream = get_nats_connection(args)
        # Use a different subject for history if specified
        history_subject = getattr(args, 'nats_history_subject', None) or os.getenv('NATS_HISTORY_SUBJECT', 'cms.htcondor.history.job')
    except Exception as e:
        logging.error("Failed to get NATS connection: %s", str(e))
        error = True
        jetstream = None
        history_subject = None
    
    try:
        if not args.dry_run:
            history_iter = schedd.history(history_query, [], match=-1)
        else:
            history_iter = []
    except RuntimeError:
        message = "Failed to query schedd for job history: %s" % schedd_ad["Name"]
        exc = traceback.format_exc()
        message += "\n{}".format(exc)
        logging.error(message)
        error = True
        history_iter = []
    
    # Process each job in history and publish to NATS
    if not error and jetstream is not None:
        try:
            for job_ad in history_iter:
                if time_remaining(starttime) < 10:
                    message = (
                        "History crawler on %s has been running for "
                        "more than %d minutes; exiting"
                        % (schedd_ad["Name"], TIMEOUT_MINS)
                    )
                    logging.error(message)
                    send_email_alert(
                        args.email_alerts, "spider_cms history timeout warning", message
                    )
                    timed_out = True
                    break

                dict_ad = None
                try:
                    dict_ad = convert_to_json(
                        job_ad,
                        return_dict=True,
                        reduce_data=not args.keep_full_queue_data,
                        pool_name=pool_name,
                    )
                except Exception as e:
                    message = "Failure when converting document on %s history: %s" % (
                        schedd_ad["Name"],
                        str(e),
                    )
                    logging.warning(message)
                    if not sent_warnings:
                        send_email_alert(
                            args.email_alerts,
                            "spider_cms history document conversion error",
                            message,
                        )
                        sent_warnings = True
                    continue

                if not dict_ad:
                    continue

                job_id = unique_doc_id(dict_ad)
                count += 1

                # Update latest completion time based on job's completion date
                job_completion = dict_ad.get("CompletionDate") or dict_ad.get("EnteredCurrentStatus") or dict_ad.get("RecordTime")
                if job_completion and job_completion > latest_completion:
                    latest_completion = job_completion

                # Publish each job to NATS JetStream
                if not args.dry_run:
                    if publish_job_to_nats(jetstream, history_subject, job_id, dict_ad, args):
                        published_count += 1
                else:
                    logging.debug("DRY RUN: Would publish history job %s", job_id)

        except Exception as e:
            message = "Failure when processing schedd history query on %s: %s" % (
                schedd_ad["Name"],
                str(e),
            )
            exc = traceback.format_exc()
            message += "\n{}".format(exc)
            logging.error(message)
            send_email_alert(
                args.email_alerts, "spider_cms schedd history query error", message
            )
            error = True
    
    # If we got to this point without a timeout and processed jobs, update the checkpoint
    if not timed_out and not error and count > 0:
        checkpoint_queue.put((schedd_ad["Name"], latest_completion))
    
    total_time = (time.time() - my_start) / 60.0
    last_formatted = datetime.datetime.fromtimestamp(last_completion).strftime(
        "%Y-%m-%d %H:%M:%S"
    )
    logging.warning(
        "Schedd %-25s history: queried %5d jobs, published %5d jobs; last completion %s; "
        "query time %.2f min",
        schedd_ad["Name"],
        count,
        published_count,
        last_formatted,
        total_time,
    )
    
    return latest_completion


def update_checkpoint(name, completion_date):
    try:
        with open(_CHECKPOINT_JSON, "r") as fd:
            checkpoint = json.load(fd)
    except Exception as e:
        logging.warning("!!! checkpoint.json is not found or not readable. "
                        "It will be created and fresh results will be written. " + str(e))
        checkpoint = {}

    checkpoint[name] = completion_date

    with open(_CHECKPOINT_JSON, "w") as fd:
        json.dump(checkpoint, fd)


def process_histories(schedd_ads, starttime, pool, args, metadata=None):
    """
    Process history files for each schedd listed in a given
    multiprocessing pool
    """
    try:
        checkpoint = json.load(open(_CHECKPOINT_JSON))
    except Exception as e:
        # Exception should be general
        logging.warning("!!! checkpoint.json is not found or not readable. Empty dict will be used. " + str(e))
        checkpoint = {}

    futures = []
    metadata = metadata or {}
    metadata["spider_source"] = "condor_history"

    manager = multiprocessing.Manager()
    checkpoint_queue = manager.Queue()

    for schedd_ad in schedd_ads:
        name = schedd_ad["Name"]

        # Check for last completion time
        # If there was no previous completion, get last 12 h
        history_query_max_n_minutes = args.history_query_max_n_minutes  # Default 12 * 60
        last_completion = checkpoint.get(name, time.time() - history_query_max_n_minutes * 60)
        last_completion = max(last_completion, time.time() - RETENTION_POLICY)

        # For CRAB, only ever get a maximum of 12 h
        if name.startswith("crab") and last_completion < time.time() - CRAB_MAX_QUERY_TIME_SPAN:
            last_completion = time.time() - history_query_max_n_minutes * 60

        future = pool.apply_async(
            process_schedd,
            (starttime, last_completion, checkpoint_queue, schedd_ad, args, metadata),
        )
        futures.append((name, future))

    def _chkp_updater():
        while True:
            try:
                job = checkpoint_queue.get()
                if job is None:  # Swallow poison pill
                    break
            except EOFError as error:
                logging.warning(
                    "EOFError - Nothing to consume left in the queue %s", error
                )
                break
            update_checkpoint(*job)

    chkp_updater = multiprocessing.Process(target=_chkp_updater)
    chkp_updater.start()

    # Check whether one of the processes timed out and reset their last
    # completion checkpoint in case
    timed_out = False
    for name, future in futures:
        if time_remaining(starttime, positive=False) > -20:
            try:
                future.get(time_remaining(starttime) + 10)
            except multiprocessing.TimeoutError:
                # This implies that the checkpoint hasn't been updated
                message = "Schedd %s history timed out; ignoring progress." % name
                exc = traceback.format_exc()
                message += "\n{}".format(exc)
                logging.error(message)
                send_email_alert(
                    args.email_alerts, "spider_cms history timeout warning", message
                )
            except Exception as e:
                # Catch any other exceptions from process_schedd
                message = (
                    "Error while processing history data of %s: %s"
                    % (name, str(e))
                )
                exc = traceback.format_exc()
                message += "\n{}".format(exc)
                logging.error(message)
                send_email_alert(
                    args.email_alerts,
                    "spider_cms history processing error warning",
                    message,
                )
        else:
            timed_out = True
            break
    if timed_out:
        pool.terminate()

    checkpoint_queue.put(None)  # Send a poison pill
    chkp_updater.join()

    logging.warning(
        "Processing time for history: %.2f mins", ((time.time() - starttime) / 60.0)
    )
