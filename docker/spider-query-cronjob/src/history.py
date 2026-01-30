"""
Methods for processing the history in a schedd queue.
"""

import datetime
import time
import traceback
import threading
from multiprocessing import Manager
from queue import Empty
from concurrent.futures import ProcessPoolExecutor, as_completed
from queue import Queue
from typing import Tuple

import classad
import htcondor

from opentelemetry import context as otel_context
from opentelemetry import trace

from utils import send_email_alert, global_logger
from otel_setup import trace_span
from nats_client import (
    get_nats_connection,
    get_jetstream_context,
    publish_jobs_to_nats,
    set_checkpoint,
    get_all_checkpoints,
    close_nats_connection,
)
import constants as const

# TODO: Improve exceptions
@trace_span("query_schedd_history")
def query_schedd_history(starttime: float, last_completion: float, schedd_ad: classad.ClassAd, job_queue = None):
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
    from opentelemetry import trace
    current_span = trace.get_current_span()
    if current_span:
        schedd_name = schedd_ad["Name"]
        current_span.set_attribute("schedd.name", schedd_name)
        current_span.set_attribute("last_completion", last_completion)
        current_span.set_attribute("use_queue", job_queue is not None)

    schedd = htcondor.Schedd(schedd_ad)
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

    # Query history and fetch ALL data into a list FIRST
    # This is critical: the schedd.history() iterator blocks for 5+ seconds per item
    # during iteration. By fetching all data first (before creating NATS connection),
    # the blocking happens when NATS isn't involved, so it doesn't interfere with
    # NATS network I/O processing.
    history_jobs = []
    try:
        if not const.DRY_RUN:
            global_logger.info("Fetching all history data for %s (this may take a while)...", schedd_ad["Name"])
            fetch_start = time.time()
            history_iter = schedd.history(history_query, [], match=-1)
            # Fetch all jobs into a list - blocking happens here, before NATS is involved
            history_jobs = list(history_iter)
            fetch_duration = time.time() - fetch_start
            global_logger.info(
                "Fetched %d history jobs for %s in %.2f seconds",
                len(history_jobs),
                schedd_ad["Name"],
                fetch_duration
            )
        else:
            history_jobs = []
    except RuntimeError:
        global_logger.error(
            "Failed to query schedd %s for job history: %s",
            schedd_ad["Name"],
            traceback.format_exc(),
        )
        current_span.set_attribute("error", True)
        current_span.set_attribute("error.message", traceback.format_exc())
        error = True
        history_jobs = []

    # Process each job in history
    # If using queue: enqueue jobs for main process to publish
    # Otherwise: publish directly (backward compatibility)
    if not error:
        # Initialize batch for direct publishing (backward compatibility mode)
        job_batch = []
        
        try:
            global_logger.debug("Enqueuing %d jobs from %s to publish queue", len(history_jobs), schedd_ad["Name"])
            
            for idx, job_ad in enumerate(history_jobs):
                counts["count"] += 1

                # Update latest completion time based on job's completion date
                job_completion = (
                    job_ad.get("CompletionDate")
                    or job_ad.get("EnteredCurrentStatus")
                    or job_ad.get("RecordTime")
                )
                if job_completion and job_completion > latest_completion:
                    latest_completion = job_completion

                # Either enqueue for main process to publish, or publish directly
                if not const.DRY_RUN:
                    try:
                        job_queue.put((schedd_ad["Name"], job_ad), timeout=30)
                        counts["published_count"] += 1  # Count as "published" since it's queued
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

            # Publish any remaining jobs in the final batch
            if job_batch:
                batch_published = publish_jobs_to_nats(
                    jetstream, const.NATS_SUBJECT, job_batch
                )
                counts["published_count"] += batch_published
                
                global_logger.info(
                    "Finished publishing all %d jobs for %s. Published: %d, Failed: %d",
                    len(history_jobs),
                    schedd_ad["Name"],
                    counts["published_count"],
                    len(history_jobs) - counts["published_count"]
                )

        except Exception:
            global_logger.error(
                "Failure when processing schedd history query on %s: %s",
                schedd_ad["Name"],
                traceback.format_exc()
            )
            error = True

    # Checkpoint update happens in the main process after all publishing is done

    total_time = (time.time() - starttime) / 60.0
    last_formatted = datetime.datetime.fromtimestamp(last_completion).strftime(
        "%Y-%m-%d %H:%M:%S"
    )
    global_logger.info(
        "Schedd %-25s history: queried %5d jobs, enqueued %5d jobs; last completion %s; "
        "query time %.2f min",
        schedd_ad["Name"],
        counts["count"],
        counts["published_count"],
        last_formatted,
        total_time,
    )

    return (latest_completion, counts)



@trace_span("nats_publisher_thread")
def nats_publisher_thread(job_queue: Queue[Tuple[str, classad.ClassAd]], stop_event: threading.Event, stats: dict, thread_id: int = 0):
    """
    Publisher thread that consumes jobs from the queue and publishes them to NATS.
    Runs in the main process to avoid NATS connection contention.
    Uses the global NATS connection since there's only one publisher thread.
    
    Args:
        job_queue: Queue containing (schedd_name, job_ad) tuples
        stop_event: Event to signal when to stop (when all workers are done)
        stats: Dictionary to track publishing statistics (shared across threads)
        thread_id: Thread identifier for logging
    """
    current_span = trace.get_current_span()
    if current_span:
        current_span.set_attribute("thread_id", thread_id)
    
    global_logger.info("Starting NATS publisher thread %d", thread_id)
    
    nats_connection = None
    jetstream = None
    try:
        nats_connection, jetstream = get_nats_connection(
            const.NATS_SERVER, const.NATS_STREAM_NAME, const.NATS_SUBJECT
        )
        global_logger.info("NATS publisher thread %d connected to NATS", thread_id)
    except Exception as e:
        global_logger.error("Failed to get NATS connection in publisher thread %d: %s", thread_id, str(e))
        stats["error"] = True
        return
    
    published_count = 0
    failed_count = 0
    BATCH_SIZE = 100  
    job_batch = []
    
    while True:
        try:
            try:
                schedd_name, job_ad = job_queue.get(timeout=1.0)
                job_batch.append(job_ad)
            except Empty:
                if stop_event.is_set():
                    if job_batch and not const.DRY_RUN:
                        batch_published = publish_jobs_to_nats(
                            jetstream, const.NATS_SUBJECT, job_batch, batch_size=BATCH_SIZE
                        )
                        published_count += batch_published
                        failed_count += len(job_batch) - batch_published
                        job_batch = []
                    break
                # If we have a batch ready, publish it even if queue is temporarily empty
                if len(job_batch) >= BATCH_SIZE:
                    # Fall through to publish the batch
                    pass
                else:
                    continue
            
            
            # Publish batch when it reaches BATCH_SIZE
            if len(job_batch) >= BATCH_SIZE:
                if not const.DRY_RUN:
                    batch_published = publish_jobs_to_nats(
                        jetstream, const.NATS_SUBJECT, job_batch, batch_size=BATCH_SIZE
                    )
                    published_count += batch_published
                    failed_count += len(job_batch) - batch_published
                    if failed_count % 10 == 0 and failed_count > 0:
                        global_logger.warning("Publisher thread: %d failed publishes so far", failed_count)
                else:
                    published_count += len(job_batch)
                    if published_count % 1000 == 0 and published_count > 0:
                        global_logger.info("Publisher thread: %d published so far", published_count)

                job_batch = []
            
            try:
                job_queue.task_done()
            except ValueError:
                # task_done() called more times than items retrieved - ignore
                pass
            
        except Exception as e:
            global_logger.error("Error in publisher thread: %s", str(e))
            failed_count += len(job_batch)
            # Clear the batch on error to avoid reprocessing
            job_batch = []
            import traceback
            global_logger.error("Traceback: %s", traceback.format_exc())

    global_logger.info(
        "Publisher thread finished: published %d jobs, failed %d jobs",
        published_count,
        failed_count
    )
    
    current_span.set_attribute("jobs.published", published_count)
    current_span.set_attribute("jobs.failed", failed_count)
    
    # Note: We don't close the global NATS connection here since it may be used elsewhere
    # (e.g., for checkpoint updates). The connection will be closed at the end of query_job_history()


def update_checkpoint(
    jetstream, name, completion_date, kv_bucket_name=const.CHECKPOINT_KV_BUCKET
):
    """
    Update checkpoint for a schedd in NATS KeyValue store.

    Args:
        jetstream: NATS JetStream context
        name: Schedd name
        completion_date: Completion timestamp
        kv_bucket_name: KeyValue bucket name
    """
    if jetstream is None:
        global_logger.error("Cannot update checkpoint: NATS JetStream connection is None")
        return

    success = set_checkpoint(jetstream, name, completion_date, kv_bucket_name)
    if not success:
        global_logger.warning(
            "Failed to update checkpoint for %s in KeyValue store. Completion date: %s",
            name,
            completion_date,
        )


def load_checkpoints_and_prepare_tasks(schedd_ads, metadata=None):
    """
    Load checkpoints from NATS KeyValue store and prepare tasks for processing.
    Each worker process will directly update checkpoints in the KV store.
    """
    metadata = metadata or {}
    metadata["spider_source"] = "condor_history"

    jetstream = None
    try:
        # Use get_jetstream_context instead of get_nats_connection since
        # KeyValue operations don't require a stream to be created
        jetstream = get_jetstream_context(const.NATS_SERVER)
    except Exception as e:
        global_logger.warning(
            "Failed to get NATS connection for checkpoints. "
            "Will use default time windows. Error: %s",
            str(e),
        )

    # Load checkpoints from KeyValue store
    if jetstream is not None:
        try:
            checkpoint = get_all_checkpoints(jetstream, const.CHECKPOINT_KV_BUCKET)
            global_logger.info(
                "Loaded %d checkpoints from KeyValue store", len(checkpoint)
            )
        except Exception as e:
            global_logger.warning(
                "Failed to load checkpoints from KeyValue store. "
                "Empty dict will be used. Error: %s",
                str(e),
            )
            checkpoint = {}
    else:
        global_logger.warning(
            "NATS connection not available. Using empty checkpoint dict. "
            "All schedds will use default time windows."
        )
        checkpoint = {}

    # Prepare tasks with their checkpoints
    tasks = []
    for schedd_ad in schedd_ads:
        name = schedd_ad["Name"]

        # Check for last completion time
        last_completion = checkpoint.get(
            name, time.time() - const.HISTORY_QUERY_MAX_N_MINUTES * 60
        )
        last_completion = max(last_completion, time.time() - const.RETENTION_POLICY)

        # For CRAB, only ever get a maximum of 12 h
        if (
            name.startswith("crab")
            and last_completion < time.time() - const.CRAB_MAX_QUERY_TIME_SPAN
        ):
            last_completion = time.time() - const.HISTORY_QUERY_MAX_N_MINUTES * 60

        tasks.append((schedd_ad, last_completion))

    return tasks


@trace_span("query_job_history")
def query_job_history(schedd_ads: list[classad.ClassAd], starttime: float, metadata: dict = None):
    """
    Queries the job history for each schedd and publishes the jobs to NATS JetStream.
    Uses a queue-based approach: worker processes enqueue jobs, main process publishes them.
    This eliminates NATS connection contention between worker processes.
    
    Params:
    - schedd_ads: A list of schedd ads to query.
    - starttime: The start time of the query in seconds since the epoch.
    - metadata: A dictionary of metadata to be passed to the worker processes.
    """
    from opentelemetry import trace
    current_span = trace.get_current_span()
    if current_span:
        current_span.set_attribute("schedd.count", len(schedd_ads))
        current_span.set_attribute("max_history_processes", const.MAX_HISTORY_PROCESSES)
    
    # Load checkpoints and prepare tasks
    tasks = load_checkpoints_and_prepare_tasks(schedd_ads, metadata)

    # Create a Manager to create shareable queue that can be passed to worker processes
    manager = Manager()
    job_queue = manager.Queue()
    # Shared counters for thread-safe statistics
    published_counter = manager.Value('i', 0)
    failed_counter = manager.Value('i', 0)
    stop_event = threading.Event()
    publisher_stats = {"published": published_counter, "failed": failed_counter, "error": False}

    # Capture current trace context so the publisher thread inherits it (same trace).
    # Contextvars do not propagate to new threads by default.
    parent_ctx = otel_context.get_current()

    def run_publisher_with_context():
        token = otel_context.attach(parent_ctx)
        try:
            nats_publisher_thread(job_queue, stop_event, publisher_stats, 0)
        finally:
            otel_context.detach(token)

    # Start single publisher thread
    publisher_thread = threading.Thread(
        target=run_publisher_with_context,
        daemon=False,
    )
    publisher_thread.start()
    global_logger.info("Started NATS publisher thread with queue monitoring")

    # Track checkpoints per schedd
    schedd_checkpoints = {}
    
    # Use ProcessPoolExecutor with as_completed, same pattern as spider_cms.py
    # Workers now enqueue jobs instead of publishing directly
    with ProcessPoolExecutor(max_workers=const.MAX_HISTORY_PROCESSES) as executor:
        futures = {
            executor.submit(
                query_schedd_history,
                starttime,
                last_completion,
                schedd_ad,
                job_queue,  # Pass queue to workers
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
                
                # Store checkpoint for this schedd
                if latest_completion:
                    schedd_checkpoints[schedd_name] = latest_completion
                
                global_logger.info(
                    "Completed querying job history for schedd %s; latest completion: %s, jobs: %d, published: %d",
                    schedd_name,
                    datetime.datetime.fromtimestamp(latest_completion).strftime(
                        "%Y-%m-%d %H:%M:%S"
                    )
                    if latest_completion
                    else "N/A",
                    counts["count"],
                    counts["published_count"],
                )
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

    # All workers are done - signal publisher thread to finish processing remaining jobs
    global_logger.info("All worker processes completed, waiting for publisher thread to finish...")
    stop_event.set()
    
    # Wait for publisher thread to finish
    publisher_thread.join(timeout=300)  # 5 minute timeout
    if publisher_thread.is_alive():
        global_logger.error("Publisher thread did not finish within timeout")
    else:
        total_published = publisher_stats["published"].value if "published" in publisher_stats else 0
        total_failed = publisher_stats["failed"].value if "failed" in publisher_stats else 0
        global_logger.info(
            "Publisher thread completed: published %d jobs, failed %d jobs",
            total_published,
            total_failed
        )
    
    # Update checkpoints for all schedds
    if schedd_checkpoints:
        try:
            jetstream = get_jetstream_context(const.NATS_SERVER)
            for schedd_name, latest_completion in schedd_checkpoints.items():
                update_checkpoint(jetstream, schedd_name, latest_completion)
                global_logger.info(
                    "Updated checkpoint for %s: %s",
                    schedd_name,
                    datetime.datetime.fromtimestamp(latest_completion).strftime(
                        "%Y-%m-%d %H:%M:%S"
                    )
                )
        except Exception as e:
            global_logger.error("Failed to update checkpoints: %s", str(e))
        finally:
            # Clean up the global NATS connection used for checkpoints
            try:
                close_nats_connection(timeout=5.0)
                global_logger.debug("Closed global NATS connection after checkpoint updates")
            except Exception as e:
                global_logger.warning("Error closing global NATS connection: %s", str(e))
    
    # Shutdown the manager to clean up resources
    manager.shutdown()
    global_logger.info("Manager shutdown complete")

    global_logger.warning(
        "Processing time for history: %.2f mins", ((time.time() - starttime) / 60.0)
    )
