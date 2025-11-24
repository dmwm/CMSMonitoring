"""
Query the jobs in queue for given set of schedds and publish to NATS JetStream.
"""

import os
import time
import logging
import resource
import traceback

import classad
import htcondor

from utils import send_email_alert, time_remaining, TIMEOUT_MINS
from convert_to_json import convert_to_json, unique_doc_id
from nats import get_nats_connection, publish_job_to_nats


def query_schedd_queue(starttime, schedd_ad, args):
    """
    Query a schedd for jobs and publish each job to NATS JetStream.
    """
    my_start = time.time()
    pool_name = schedd_ad.get("CMS_Pool", "Unknown")
    logging.info("Querying %s queue for jobs.", schedd_ad["Name"])
    
    if time_remaining(starttime) < 10:
        message = (
            "No time remaining to run queue crawler on %s; "
            "exiting." % schedd_ad["Name"]
        )
        logging.error(message)
        send_email_alert(args.email_alerts, "spider_cms queue timeout warning", message)
        return 0

    # Get NATS JetStream connection
    try:
        nats_connection, jetstream = get_nats_connection(args)
        subject = getattr(args, 'nats_subject', None) or os.getenv('NATS_SUBJECT', 'cms.htcondor.queue.job')
    except Exception as e:
        logging.error("Failed to get NATS connection: %s", str(e))
        return 0

    count_since_last_report = 0
    count = 0
    published_count = 0
    cpu_usage = resource.getrusage(resource.RUSAGE_SELF).ru_utime
    sent_warnings = False

    schedd = htcondor.Schedd(schedd_ad)
    # Query for a snapshot of the jobs running/idle/held,
    # but only the completed that had changed in the last period of time.
    _completed_since = starttime - (TIMEOUT_MINS + 1) * 60
    query = """
         (JobUniverse == 5) && (CMS_Type != "DONOTMONIT")
         &&
         (
             JobStatus < 3 || JobStatus > 4
             || EnteredCurrentStatus >= %(completed_since)d
             || CRAB_PostJobLastUpdate >= %(completed_since)d
         )
         """ % {
        "completed_since": _completed_since
    }
    
    try:
        query_iter = schedd.xquery(constraint=query) if not args.dry_run else []
        for job_ad in query_iter:
            if time_remaining(starttime) < 10:
                message = (
                    "Queue crawler on %s has been running for "
                    "more than %d minutes; exiting"
                    % (schedd_ad["Name"], TIMEOUT_MINS)
                )
                logging.error(message)
                send_email_alert(
                    args.email_alerts, "spider_cms queue timeout warning", message
                )
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
                message = "Failure when converting document on %s queue: %s" % (
                    schedd_ad["Name"],
                    str(e),
                )
                logging.warning(message)
                if not sent_warnings:
                    send_email_alert(
                        args.email_alerts,
                        "spider_cms queue document conversion error",
                        message,
                    )
                    sent_warnings = True
                continue

            if not dict_ad:
                continue

            job_id = unique_doc_id(dict_ad)
            count += 1
            count_since_last_report += 1

            # Publish each job to NATS JetStream
            if not args.dry_run:
                if publish_job_to_nats(jetstream, subject, job_id, dict_ad, args):
                    published_count += 1
            else:
                logging.debug("DRY RUN: Would publish job %s", job_id)

            # Report progress periodically
            if count_since_last_report >= 1000:
                cpu_usage_now = resource.getrusage(resource.RUSAGE_SELF).ru_utime
                cpu_usage = cpu_usage_now - cpu_usage
                processing_rate = count_since_last_report / cpu_usage if cpu_usage > 0 else 0
                cpu_usage = cpu_usage_now
                logging.info(
                    "Query for %s: processed %d jobs, published %d jobs "
                    "(%.1f jobs per CPU-second)",
                    schedd_ad["Name"],
                    count,
                    published_count,
                    processing_rate,
                )
                count_since_last_report = 0

            if args.max_documents_to_process and count > args.max_documents_to_process:
                logging.warning(
                    "Aborting after %d documents (--max_documents_to_process option)"
                    % args.max_documents_to_process
                )
                break

    except RuntimeError as e:
        logging.error(
            "Failed to query schedd %s for jobs: %s", schedd_ad["Name"], str(e)
        )
    except Exception as e:
        message = "Failure when querying schedd queue on %s: %s" % (
            schedd_ad["Name"],
            str(e),
        )
        logging.error(message)
        send_email_alert(
            args.email_alerts, "spider_cms schedd queue query error", message
        )
        traceback.print_exc()

    total_time = (time.time() - my_start) / 60.0
    logging.warning(
        "Schedd %-25s queue: queried %5d jobs, published %5d jobs; query time %.2f min",
        schedd_ad["Name"],
        count,
        published_count,
        total_time,
    )

    return count
