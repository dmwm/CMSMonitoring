"""
Methods for processing the history in a schedd queue.
"""

import datetime
from re import S
import time
import traceback
from concurrent.futures import ProcessPoolExecutor, as_completed

import classad
import htcondor

from opentelemetry import trace

from utils import send_email_alert, global_logger
from otel_setup import global_meter, trace_span
from nats_client import (
    get_nats_connection,
    publish_jobs_to_nats,
    close_nats_connection,
)
from checkpoints import load_checkpoints, prepare_tasks, update_checkpoint
import constants as const

# Metrics for history job
history_jobs_queried_counter = global_meter.create_counter(
    name="spider_cms.history.jobs",
    description="Number of jobs queried per schedd in history",
    unit="1",
)
history_jobs_not_published_counter = global_meter.create_counter(
    name="spider_cms.history.jobs_not_published",
    description="Number of jobs not published per schedd in history",
    unit="1",
)
history_query_duration_histogram = global_meter.create_histogram(
    name="spider_cms.history.query_duration",
    description="Duration of history query per schedd in seconds",
    unit="s",
)
total_history_query_duration_histogram = global_meter.create_histogram(
    name="spider_cms.history.total_query_duration",
    description="Total duration of history query for all schedds in seconds",
    unit="s",
)
time_since_last_checkpoint_histogram = global_meter.create_histogram(
    name="spider_cms.history.time_since_last_checkpoint",
    description="Time since last checkpoint in seconds",
    unit="s",
)


# TODO: Improve exceptions
@trace_span("query_schedd_history")
def query_schedd_history(starttime: float, last_completion: float, schedd_ad: classad.ClassAd):
    """
    Given a schedd, process its entire set of history since last checkpoint.
    If job_queue is provided, jobs are queued for publishing by the main process.
    Otherwise, jobs are published directly (backward compatibility).
    
    Params:
    - starttime: The start time of the query in seconds since the epoch.
    - last_completion: The last completion time of the query in seconds since the epoch.
    - schedd_ad: The schedd ad to query.
    - job_queue: Optional multiprocessing Queue to enqueue jobs for publishing.

    Returns:
    - Tuple of (latest_completion_time, job_count) if job_queue is provided
    - latest_completion_time (float) if job_queue is None (backward compatibility)
    """
    schedd_start_time = time.time()
    time_since_last_checkpoint_histogram.record(schedd_start_time - last_completion)
    from opentelemetry import trace
    current_span = trace.get_current_span()
    if current_span:
        schedd_name = schedd_ad["Name"]
        current_span.set_attribute("schedd.name", schedd_name)
        current_span.set_attribute("last_completion", last_completion)

    schedd = htcondor.Schedd(schedd_ad)
    # TODO: Aren't we missing jobs because of using CRAB_PostJobLastUpdate in the constraint?
    _q = """
        (JobUniverse == 5) && (CMS_Type != "DONOTMONIT")
        &&
        (
            EnteredCurrentStatus >= %(last_completion)d
            || CRAB_PostJobLastUpdate >= %(last_completion)d
        )
        """
    history_query = classad.ExprTree(
        _q % {"last_completion": last_completion - const.QUERY_TIME_PERIOD}
    )
    global_logger.info(
        "Querying %s for history: %s.  %.1f minutes of ads",
        schedd_ad["Name"],
        history_query,
        (time.time() - last_completion) / 60.0,
    )
    counts = {
        "count": 0,
        "published_count": 0,
    }   
    error = False
    latest_completion = last_completion
    jetstream = None

    history_jobs = []
    try:
        if not const.DRY_RUN:
            try:
                nats_connection, jetstream = get_nats_connection(
                    const.NATS_SERVER, const.NATS_STREAM_NAME, const.NATS_SUBJECT
                )
            except Exception as e:
                global_logger.error("Failed to get NATS connection for schedd %s: %s", schedd_ad["Name"], str(e))
                error = True
                return (latest_completion, counts)

            global_logger.info("Fetching history data for %s...", schedd_ad["Name"])
            history_iter = iter(schedd.history(history_query, [], match=-1))
            # Fetch all jobs into a list - blocking happens here, before NATS is involved
            global_logger.info(
                "Fetched history jobs for %s",
                schedd_ad["Name"],
            )
        else:
            history_iter = iter([])
    except Exception:
        global_logger.error(
            "Failed to query schedd %s for job history: %s",
            schedd_ad["Name"],
            traceback.format_exc(),
        )
        if current_span:
            current_span.set_attribute("error", True)
            current_span.set_attribute("error.message", traceback.format_exc())
        error = True
        history_iter = iter([])

    # Process each job in history
    if not error:
        job_batch = []
        
        try:
            global_logger.debug("Enqueuing %d jobs from %s to publish queue", len(history_jobs), schedd_ad["Name"])
            
            for idx, job_ad in enumerate(history_iter):
                counts["count"] += 1
                job_batch.append(job_ad)
                # Update latest completion time based on job's completion date
                job_completion = (
                    job_ad.get("CompletionDate")
                    or job_ad.get("EnteredCurrentStatus")
                    or job_ad.get("RecordTime")
                )
                if job_completion and job_completion > latest_completion:
                    latest_completion = job_completion

                if not const.DRY_RUN:
                    try:
                        if len(job_batch) >= const.NATS_BATCH_SIZE:
                            counts["published_count"] += publish_jobs_to_nats(
                                jetstream, const.NATS_SUBJECT, job_batch, batch_size=const.NATS_BATCH_SIZE
                            )
                            job_batch = []
                    except Exception as e:
                        global_logger.warning(
                            "Failed to enqueue job %d/%d for %s: %s",
                            idx + 1,
                            len(history_jobs),
                            schedd_ad["Name"],
                            str(e)
                        )
                else:
                    global_logger.debug("DRY RUN: Would publish history job %s", job_ad)

            global_logger.debug(
                "Finished enqueuing all %d jobs for %s",
                len(history_jobs),
                schedd_ad["Name"]
            )
        except Exception:
            global_logger.error(
                "Failure when processing schedd history query on %s: %s",
                schedd_ad["Name"],
                traceback.format_exc()
            )
            error = True
        finally:
            # Publish any remaining jobs in the final batch
            if job_batch:
                counts["published_count"] += publish_jobs_to_nats(
                    jetstream, const.NATS_SUBJECT, job_batch
                )                
                global_logger.info(
                    "Finished publishing all %d jobs for %s. Published: %d, Failed: %d",
                    counts["count"],
                    schedd_ad["Name"],
                    counts["published_count"],
                    counts["count"] - counts["published_count"]
                )
            # if jetstream is not None:
            #     close_nats_connection(connection=nats_connection, timeout=5)


    total_time = (time.time() - starttime) / 60.0
    last_formatted = datetime.datetime.fromtimestamp(last_completion).strftime(
        "%Y-%m-%d %H:%M:%S"
    )
    query_time = time.time() - schedd_start_time

    # Record metrics for this schedd
    schedd_attributes = {"schedd": schedd_ad["Name"]}
    history_jobs_queried_counter.add(counts["count"], attributes=schedd_attributes)
    history_jobs_not_published_counter.add(counts["count"] - counts["published_count"], attributes=schedd_attributes)
    history_query_duration_histogram.record(query_time, attributes=schedd_attributes)

    global_logger.info(
        "Finished querying %-25s history: queried %5d jobs, enqueued %5d jobs; last completion %s; "
        "query time %.2f seconds (total time %.2f min)",
        schedd_ad["Name"],
        counts["count"], counts["published_count"], last_formatted, query_time, total_time
    )

    return (latest_completion, counts)


@trace_span("query_job_history")
def query_job_history(schedd_ads: list[classad.ClassAd], starttime: float) -> dict[str, int]:
    """
    Queries the job history for each schedd and publishes the jobs to NATS JetStream.
    Uses a queue-based approach: worker processes enqueue jobs, main process publishes them.
    This eliminates NATS connection contention between worker processes.
    
    Params:
    - schedd_ads: A list of schedd ads to query.
    - starttime: The start time of the query in seconds since the epoch.
    """
    current_span = trace.get_current_span()
    if current_span:
        current_span.set_attribute("schedd.count", len(schedd_ads))
        current_span.set_attribute("max_history_processes", const.MAX_HISTORY_PROCESSES)
    
    # Load checkpoints and prepare tasks
    last_completions = load_checkpoints(schedd_ads)
    tasks = prepare_tasks(last_completions)

    total_counts = {
        "count": 0,
        "published_count": 0,
    }

    with ProcessPoolExecutor(max_workers=const.MAX_HISTORY_PROCESSES) as executor:
        futures = {
            executor.submit(
                query_schedd_history,
                starttime,
                last_completion,
                schedd_ad,
            ): schedd_ad.get("Name")
            for schedd_ad, last_completion in tasks
        }

        for future in as_completed(futures):
            schedd_name = futures[future]
            try:
                result = future.result()
                # Result is either a tuple (latest_completion, count) or just latest_completion
                if isinstance(result, tuple):
                    latest_completion, counts = result
                else:
                    latest_completion = result
                    counts = {
                        "count": 0,
                        "published_count": 0,
                    }
                
                total_counts["count"] += counts.get("count", 0)
                total_counts["published_count"] += counts.get("published_count", 0)

                # Update checkpoint for this schedd
                if latest_completion:
                    try:
                        nats_connection, jetstream = get_nats_connection(const.NATS_SERVER, const.NATS_STREAM_NAME, const.NATS_SUBJECT)
                        update_checkpoint(jetstream, schedd_name, latest_completion)
                    finally:
                        # Clean up the global NATS connection used for checkpoints
                        try:
                            close_nats_connection(nats_connection, timeout=5.0)
                            global_logger.debug("Closed global NATS connection after checkpoint updates")
                        except Exception as e:
                            global_logger.warning("Error closing global NATS connection: %s", str(e))
                
            except Exception as exc:  # pylint: disable=broad-except
                message = "Schedd %s history generated an exception: %s" % (
                    schedd_name,
                    str(exc),
                )
                exc_trace = traceback.format_exc()
                message += "\n{}".format(exc_trace)
                global_logger.error(message)
                send_email_alert(
                    const.EMAIL_ALERTS,
                    "spider_cms history processing error warning",
                    message,
                )
    
    total_query_time = time.time() - starttime
    total_history_query_duration_histogram.record(total_query_time)
    global_logger.warning(
        "Processing time for history: %.2f mins, total counts: %d, published counts: %d", ((time.time() - starttime) / 60.0), total_counts["count"], total_counts["published_count"]
    )
    return total_counts
