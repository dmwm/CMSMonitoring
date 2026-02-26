"""
Query the jobs in queue for given set of schedds and publish to NATS JetStream.
"""

import time
import traceback
from concurrent.futures import ProcessPoolExecutor, as_completed
import htcondor
import classad

from otel_setup import global_meter, trace_span
from utils import send_email_alert, time_remaining, global_logger
from nats_client import publish_jobs_to_nats, get_nats_connection, close_nats_connection
import constants as const
from opentelemetry import trace
from opentelemetry.trace import Status, StatusCode

# Create metrics for per-schedd operations
# schedd_duration_histogram = global_meter.create_histogram(
#     name="spider_cms.schedd.queue.duration",
#     description="Duration of queue query per schedd in seconds",
#     unit="s",
# )
jobs_queried_counter = global_meter.create_counter(
    name="spider_cms.schedd.queue.jobs",
    description="Number of jobs queried per schedd",
    unit="1",
)
jobs_published_counter = global_meter.create_counter(
    name="spider_cms.schedd.queue.jobs_published",
    description="Number of jobs published per schedd",
    unit="1",
)


@trace_span("query_single_schedd")
def query_single_schedd(
    starttime: float,
    schedd_ad: classad.ClassAd,
) -> dict:
    """
    Query a single schedd for jobs and publish each job to NATS JetStream.
    Intended to be safe to use from multiprocessing executors.
    Each worker process creates its own NATS connection to avoid pickling issues.
    """
    my_start = time.time()
    counts = {
        "count": 0,
        "published_count": 0,
    }
    schedd_name = schedd_ad["Name"]
    pool_name = schedd_ad.get("CMS_Pool", "Unknown")
    
    # Set span attributes for tracing
    current_span = trace.get_current_span()
    if current_span:
        current_span.set_attribute("schedd.name", schedd_name)
        current_span.set_attribute("schedd.pool", pool_name)

    # Create NATS connection in this worker process
    # This avoids pickling issues with asyncio objects
    nats_connection = None
    jetstream = None
    try:
        nats_connection, jetstream = get_nats_connection(
            const.NATS_SERVER, const.NATS_STREAM_NAME, const.NATS_SUBJECT
        )
    except Exception as e:
        global_logger.error(
            "Failed to get NATS connection for schedd %s: %s", schedd_name, str(e)
        )
        return counts

    # Use try-finally to ensure NATS connection is always closed, allowing process reuse
    try:
        global_logger.info("Querying %s queue for jobs.", schedd_name)

        if time_remaining(starttime, timeout=const.TIMEOUT_MINS * 60) < 10:
            message = (
                "No time remaining to run queue crawler on %s; exiting." % schedd_ad["Name"]
            )
            global_logger.error(message)
            send_email_alert(
                const.EMAIL_ALERTS, "spider_cms queue timeout warning", message
            )
            return counts

        schedd = htcondor.Schedd(schedd_ad)
        # Query for a snapshot of the jobs running/idle/held,
        # but only the completed that had changed in the last period of time.
        _completed_since = starttime - (const.TIMEOUT_MINS + 1) * 60
        query = """
                (JobUniverse == 5) && (CMS_Type != "DONOTMONIT")
                &&
                (
                    JobStatus < 3 || JobStatus > 4
                    || EnteredCurrentStatus >= %(completed_since)d
                    || CRAB_PostJobLastUpdate >= %(completed_since)d
                )
                """ % {"completed_since": _completed_since}

        job_batch = []  # Initialize outside try block so it's accessible in finally
        try:
            # TODO: Move the NATS logic to a separate function
            query_iter = schedd.xquery(constraint=query) if not const.DRY_RUN else []
            if not const.DRY_RUN:
                # Collect jobs in batches for efficient publishing
                for job_ad in query_iter:
                    counts["count"] += 1
                    job_batch.append(job_ad)
                    
                    if len(job_batch) >= const.NATS_BATCH_SIZE:
                        batch_published = publish_jobs_to_nats(
                            jetstream, const.NATS_SUBJECT, job_batch, batch_size=const.NATS_BATCH_SIZE
                        )
                        counts["published_count"] += batch_published
                        job_batch = []  # Clear batch after publishing
            
            else:
                for _ in range(len(query_iter)):
                    counts["count"] += 1
                global_logger.info(
                    "DRY RUN: Would publish jobs from %s: %d",
                    schedd_name,
                    counts["count"],
                )

        except RuntimeError as e:
            global_logger.error(
                "Failed to query schedd %s for jobs: %s", schedd_name, str(e)
            )
        except Exception as e:
            message = "Failure when querying schedd queue on %s: %s" % (
                schedd_name,
                str(e),
            )
            global_logger.error(message)
            send_email_alert(
                const.EMAIL_ALERTS, "spider_cms schedd queue query error", message
            )
            traceback.print_exc()
        finally:
            # Always publish any remaining jobs, even if an exception occurred
            if job_batch and not const.DRY_RUN:
                global_logger.debug(
                    "Publishing %d remaining jobs in batch for %s",
                    len(job_batch),
                    schedd_name,
                )
                try:
                    batch_published = publish_jobs_to_nats(
                        jetstream, const.NATS_SUBJECT, job_batch, batch_size=len(job_batch)
                    )
                    counts["published_count"] += batch_published
                    if batch_published < len(job_batch):
                        global_logger.warning(
                            "Only %d/%d remaining jobs published for %s",
                            batch_published,
                            len(job_batch),
                            schedd_name,
                        )
                except Exception as publish_error:
                    global_logger.error(
                        "Failed to publish remaining %d jobs for %s: %s",
                        len(job_batch),
                        schedd_name,
                        str(publish_error),
                    )

        total_time = (time.time() - my_start) / 60.0

        # Record metrics for this schedd
        schedd_attributes = {
            "schedd": schedd_name,
            "pool": pool_name,
        }
        jobs_queried_counter.add(
            counts["count"],
            attributes=schedd_attributes,
        )
        jobs_published_counter.add(
            counts["published_count"],
            attributes=schedd_attributes,
        )

        global_logger.info(
            "Completed querying %-25s: queried %5d jobs, published %5d jobs; query time %.2f min",
            schedd_name,
            counts["count"],
            counts["published_count"],
            total_time,
        )

        # Set trace attributes with final totals (after all publishing is complete)
        current_span = trace.get_current_span()
        if current_span:
            current_span.set_attribute("job.count", counts["count"])
            current_span.set_attribute("batch_size", const.NATS_BATCH_SIZE)
            current_span.set_attribute("jobs.published", counts["published_count"])
            current_span.set_attribute("jobs.failed", counts["count"] - counts["published_count"])

        return counts
    finally:
        # Close NATS connection to allow process to be immediately reused for next task
        # This prevents delays when ProcessPoolExecutor reuses the worker process
        # Without this, the next task has to wait for cleanup of the old connection
        if nats_connection is not None:
            try:
                close_nats_connection(connection=nats_connection, timeout=5)
            except Exception as close_error:
                global_logger.warning(
                    "Error closing NATS connection for %s (non-fatal): %s",
                    schedd_name,
                    str(close_error),
                )


@trace_span("query_schedds")
def query_schedds(
    starttime: float,
    schedd_ads: list[classad.ClassAd],
) -> dict:
    """
    Sequential wrapper kept for compatibility; queries all schedds one by one.
    """
    total_counts = {
        "count": 0,
        "published_count": 0,
    }
    for schedd_ad in schedd_ads:
        counts = query_single_schedd(
            starttime, schedd_ad
        )
        total_counts["count"] += counts.get("count", 0)
        total_counts["published_count"] += counts.get("published_count", 0)
    
    # Check if published count differs from total count
    if total_counts["count"] != total_counts["published_count"]:
        error_message = (
            "@@@ MISMATCH: Total jobs queried: %d, Total jobs published: %d. "
            "Some jobs were not successfully published to NATS."
        ) % (total_counts["count"], total_counts["published_count"])
        global_logger.error(error_message)
        trace.get_current_span().set_status(Status(StatusCode.ERROR, error_message))
        trace.get_current_span().set_attribute("error", True)
    
    return total_counts

@trace_span("query_running_jobs")
def query_running_jobs(
    starttime: float,
    schedd_ads: list[classad.ClassAd],
) -> dict[str, int]:
    """
    Parallel wrapper for querying all schedds for running jobs in parallel.
    The jobs are then published to NATS JetStream.
    Params:
    - starttime: The start time of the query in seconds since the epoch.
    - schedd_ads: A list of schedd ads to query.
    """
    current_span = trace.get_current_span()
    if current_span:
        current_span.set_attribute("schedd.count", len(schedd_ads))
        current_span.set_attribute("max_processes", const.MAX_PROCESSES)
    
    total_counts = {
            "count": 0,
            "published_count": 0,
        }
    with ProcessPoolExecutor(max_workers=const.MAX_PROCESSES) as executor:
        futures = {
            executor.submit(
                query_single_schedd,
                starttime,
                schedd_ad,
            ): schedd_ad.get("Name")
            for schedd_ad in schedd_ads
        }
        for future in as_completed(futures):
            schedd_name = futures[future]
            try:
                counts = future.result()
                total_counts["count"] += counts.get("count", 0)
                total_counts["published_count"] += counts.get("published_count", 0)
            except Exception as exc:  # pylint: disable=broad-except
                global_logger.error(
                    "Schedd %s generated an exception: %s", schedd_name, str(exc)
                )

    total_duration = time.time() - starttime
    global_logger.info("@@@ Total querying time: %.2f mins, queried %d jobs, published %d jobs", (total_duration / 60.0), total_counts["count"], total_counts["published_count"])
    
    return total_counts
