import signal
import time

import src.amq as amq
import src.os_utils as os_utils
from src.nats_consumer import NATSQueueConsumer
from src.utils import send_email_alert, convert_dates_to_millisecs
import src.constants as const
from src.convert_to_json import process_ad

from otel_setup import global_logger, update_execution_id


def process_nats_queue(
    nats_batch_size=10,
    nats_fetch_timeout=0.1,
    nats_idle_sleep=1,
    feed_es_for_queues=False,
    feed_amq=True,
    metadata=None,
    email_alerts=None,
):
    """
    Continuously pull jobs from the NATS JetStream queue and forward them to ES and AMQ.
    Runs until interrupted by SIGTERM/SIGINT (graceful shutdown).
    
    Note: Jobs are published to NATS in batches of 100. The fetch batch size should
    ideally be a multiple of 100 (e.g., 100, 200, 500, 1000) to efficiently process
    complete publish batches.
    
    Args:
        nats_batch_size: The number of jobs to fetch from the NATS JetStream queue at a time.
                        Should be a multiple of 100 (the publish batch size) for optimal efficiency.
                        Default: 1000 (10 publish batches).
        nats_fetch_timeout: The timeout for fetching jobs from the NATS JetStream queue.
                            A very short timeout (0.1s) returns immediately with available
                            messages to avoid throttling and process messages as fast as possible.
                            Default: 0.1 seconds.
        nats_idle_sleep: The time to sleep between batches when no jobs are available.
        feed_es_for_queues: Whether to feed the jobs to the ES index.
        feed_amq: Whether to feed the jobs to the AMQ queue.
        metadata: The metadata to add to the jobs.
        email_alerts: The email alerts to send.

    Returns:
        None

    Raises:
        Exception: If there is an error fetching jobs from the NATS JetStream queue.
        Exception: If there is an error posting jobs to the ES index.
        Exception: If there is an error posting jobs to the AMQ queue.
    """
    shutdown_requested = False

    def signal_handler(signum, frame):
        """Handle shutdown signals gracefully."""
        nonlocal shutdown_requested
        global_logger.warning("Received signal %d, initiating graceful shutdown...", signum)
        shutdown_requested = True

    # Register signal handlers for graceful shutdown
    signal.signal(signal.SIGTERM, signal_handler)
    signal.signal(signal.SIGINT, signal_handler)

    consumer = NATSQueueConsumer()

    # Publish batch size - jobs are published in batches of 100
    PUBLISH_BATCH_SIZE = 100
    
    counts = {
        "total_jobs": 0,
        "total_batches": 0,
        "total_publish_batches": 0,  # Track complete publish batches (100 jobs each)
        "total_sent_amq": 0,
        "total_upload_time": 0.0,
        "total_sent_os": 0,
        "total_upload_time_os": 0.0,
    }
    sleeping = False
    execution_id_updated = False  # Track if we've updated execution_id from query cronjob
    
    # Warn if fetch batch size is not a multiple of publish batch size
    if nats_batch_size % PUBLISH_BATCH_SIZE != 0:
        global_logger.warning(
            "Fetch batch size (%d) is not a multiple of publish batch size (%d). "
            "Consider using a multiple for optimal efficiency.",
            nats_batch_size, PUBLISH_BATCH_SIZE
        )
    
    try:
        while not shutdown_requested:
            batch = consumer.fetch_jobs(
                batch_size=nats_batch_size, timeout=nats_fetch_timeout
            )
            
            # Update execution_id from query cronjob if present (only once, from first batch)
            if not execution_id_updated and batch.query_execution_id:
                update_execution_id(batch.query_execution_id)
                execution_id_updated = True
                global_logger.info(
                    "Linked worker logs to query cronjob execution_id: %s", 
                    batch.query_execution_id
                )
            
            if not batch.jobs:
                if shutdown_requested:
                    break
                if not sleeping:
                    global_logger.info("No jobs in NATS, sleeping until next batch")
                time.sleep(nats_idle_sleep)
                sleeping = True
                continue
            
            sleeping = False
            bunch = batch.jobs
            job_count = len(bunch)
            global_logger.debug(
                "Jobs fetched from NATS: %d jobs (~%.1f publish batches)",
                job_count, job_count / PUBLISH_BATCH_SIZE
            )
            counts["total_jobs"] += job_count

            success = True
            
            try:
                amq_bunch = []
                job_sizes = []
                for job_id, classad_obj in bunch:
                    try:
                        # Convert ClassAd to dictionary
                        dict_ad = process_ad(
                            classad_obj,
                            return_dict=True,
                            reduce_data=const.REDUCE_DATA,
                            pool_name="Unknown",  # TODO: Get pool_name from metadata or job
                        )
                        if dict_ad:
                            amq_bunch.append((job_id, convert_dates_to_millisecs(dict_ad)))
                            job_sizes.append(len(dict_ad.keys()))
                    except Exception as e:
                        global_logger.warning("Failed to convert ClassAd to dict for job %s: %s", job_id, e)
                        continue
                global_logger.debug("Job sizes: Average %s, Max %s, Min %s", sum(job_sizes) / len(job_sizes), max(job_sizes), min(job_sizes))
                try:
                    if const.DRY_RUN:
                        global_logger.info("Dry run, would have uploaded %d jobs to OpenSearch", len(amq_bunch))
                    else:
                        os_upload_start_time = time.time()
                        success, _ = os_utils.os_upload_docs_in_bulk(list(doc[1] for doc in amq_bunch), index_prefix=const.OS_INDEX_TEMPLATE, timestamp=time.time())
                        global_logger.debug("Uploaded %d/%d jobs to OpenSearch in %.1f seconds", success, len(amq_bunch), time.time() - os_upload_start_time)
                        counts["total_sent_os"] += success
                        counts["total_upload_time_os"] += time.time() - os_upload_start_time
                except Exception as e:
                    global_logger.error("Error uploading to OpenSearch: %s", e)
                    # success = False
                try:
                    if const.DRY_RUN:
                        global_logger.info("Dry run, would have uploaded %d jobs to AMQ", len(amq_bunch))
                        batch.finalize(success)
                        continue
                    sent, received, elapsed = amq.post_ads(amq_bunch, metadata=metadata)
                    global_logger.debug(
                        "Uploaded %d/%d docs to StompAMQ in %.1f seconds",
                        sent,
                        received,
                        elapsed,
                    )
                    counts["total_sent_amq"] += sent
                    counts["total_upload_time"] += elapsed
                except Exception as exc:
                    success = False
                    global_logger.error("Error posting to AMQ: %s", exc)
                    send_email_alert(
                        email_alerts,
                        "spider_cms NATS queue AMQ failure",
                        str(exc),
                    )
            except Exception as exc:  # pylint: disable=broad-except
                success = False
                global_logger.error(
                    "Failed to process NATS batch with %d jobs: %s",
                    len(bunch),
                    exc,
                )

            batch.finalize(success)
            counts["total_batches"] += 1
            if success:
                global_logger.debug(
                    "NATS batch processed successfully: %d jobs (~%.1f publish batches)",
                    job_count, job_count / PUBLISH_BATCH_SIZE
                )
            else:
                global_logger.error(
                    "Failed to process NATS batch (%d jobs, ~%.1f publish batches), sleeping for %s seconds",
                    job_count, job_count / PUBLISH_BATCH_SIZE, nats_idle_sleep
                )
                if not shutdown_requested:
                    time.sleep(nats_idle_sleep)

    except KeyboardInterrupt:
        global_logger.warning("Received KeyboardInterrupt, initiating graceful shutdown...")
    finally:
        consumer.close()

    global_logger.info(
        "Processing of NATS queue finished. Summary: "
        "total_jobs=%d, total_fetch_batches=%d, total_publish_batches=~%d, "
        "total_sent_amq=%d, total_sent_os=%d",
        counts["total_jobs"],
        counts["total_batches"],
        counts["total_publish_batches"],
        counts["total_sent_amq"],
        counts["total_sent_os"],
    )
