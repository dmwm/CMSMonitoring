import signal
import time
import logging

import opentelemetry
import src.amq as amq
import src.os_utils as os_utils
from src.nats_consumer import NATSQueueConsumer
from src.utils import send_email_alert, convert_dates_to_millisecs
import src.constants as const
from src.convert_to_json import process_ad

from src.otel_setup import (
    global_logger,
    global_meter,
    initialize_worker_trace,
    finalize_worker_trace,
)


WORKER_LIFETIME_SECONDS_HISTOGRAM = global_meter.create_histogram(
    name="spider_worker_lifetime_seconds",
    description="How long a spider worker process stayed alive before shutdown",
    unit="s",
)
WORKER_IDLE_SECONDS_HISTOGRAM = global_meter.create_histogram(
    name="spider_worker_idle_seconds",
    description="How long a spider worker process was idle before processing a batch",
    unit="s",
)
WORKER_JOBS_PROCESSED_PER_LIFETIME_HISTOGRAM = global_meter.create_histogram(
    name="spider_worker_jobs_processed_per_lifetime",
    description="Number of jobs processed by a worker during its lifetime",
    unit="1",
)
WORKER_JOBS_PUBLISHED_AMQ_PER_LIFETIME_HISTOGRAM = global_meter.create_histogram(
    name="spider_worker_jobs_published_amq_per_lifetime",
    description="Number of jobs published to AMQ by a worker during its lifetime",
    unit="1",
)
WORKER_JOBS_PUBLISHED_OS_PER_LIFETIME_HISTOGRAM = global_meter.create_histogram(
    name="spider_worker_jobs_published_os_per_lifetime",
    description="Number of jobs published to OpenSearch by a worker during its lifetime",
    unit="1",
)
NATS_FETCH_BATCH_SIZE_HISTOGRAM = global_meter.create_histogram(
    name="spider_worker_nats_fetch_batch_size",
    description="Number of jobs in each non-empty batch fetched from NATS",
    unit="1",
)
NATS_BATCH_PROCESS_SECONDS_HISTOGRAM = global_meter.create_histogram(
    name="spider_worker_nats_batch_process_seconds",
    description="Elapsed processing time for each non-empty fetched NATS batch",
    unit="s",
)
NATS_REDELIVERED_MESSAGES_COUNTER = global_meter.create_counter(
    name="spider_worker_nats_redelivered_messages_total",
    description="Number of redelivered NATS messages observed while fetching batches",
    unit="1",
)
WORKER_SHUTDOWNS_COUNTER = global_meter.create_counter(
    name="spider_worker_shutdowns_total",
    description="Number of spider worker processes that have shut down",
    unit="1",
)


def process_nats_queue(
    nats_batch_size=const.NATS_BATCH_SIZE,
    nats_fetch_timeout=const.NATS_FETCH_TIMEOUT,
    nats_idle_sleep=const.NATS_IDLE_SLEEP,
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
                        Default: value of NATS_BATCH_SIZE env var (1000 if unset).
        nats_fetch_timeout: The timeout for fetching jobs from the NATS JetStream queue.
                            A very short timeout (0.1s) returns immediately with available
                            messages to avoid throttling and process messages as fast as possible.
                            Default: value of NATS_FETCH_TIMEOUT env var (0.2s if unset).
        nats_idle_sleep: The time to sleep between batches when no jobs are available.
                        Default: value of NATS_IDLE_SLEEP env var (0.2s if unset).
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
    worker_start_time = time.time()

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
        "total_idle_seconds": 0.0,
    }
    sleeping = False
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

            initialize_worker_trace(
                traceparent=batch.traceparent,
                tracestate=batch.tracestate,
            )
            if batch.has_traceparent_mismatch:
                # TODO: Rotate span per execution when a batch contains mixed traceparents.
                global_logger.warning(
                    "Batch contained mixed root traceparent values; using first traceparent for this worker. "
                    "first=%s latest=%s",
                    batch.traceparent,
                    batch.latest_traceparent,
                )
            
            if not batch.jobs:
                if shutdown_requested:
                    break
                if not sleeping:
                    global_logger.info("No jobs in NATS, sleeping until next batch")
                counts["total_idle_seconds"] += nats_idle_sleep
                time.sleep(nats_idle_sleep)
                sleeping = True
                continue
            
            sleeping = False
            job_count = len(batch.jobs)
            NATS_FETCH_BATCH_SIZE_HISTOGRAM.record(job_count)
            if batch.redelivered_messages:
                NATS_REDELIVERED_MESSAGES_COUNTER.add(batch.redelivered_messages)
            global_logger.debug(
                "Jobs fetched from NATS: %d jobs (~%.1f publish batches)",
                job_count, job_count / PUBLISH_BATCH_SIZE
            )
            counts["total_jobs"] += job_count

            batch_processing_start = time.time()
            all_chunks_success = True
            for chunk in batch.chunked(const.NATS_CHUNK_SIZE):
                success = True
                bunch = chunk.jobs
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
                    if global_logger.isEnabledFor(logging.DEBUG) and job_sizes:
                        global_logger.debug(
                            "Job sizes: Average %s, Max %s, Min %s",
                            sum(job_sizes) / len(job_sizes),
                            max(job_sizes),
                            min(job_sizes),
                        )
                    try:
                        if const.DRY_RUN:
                            global_logger.info("Dry run, would have uploaded %d jobs to OpenSearch", len(amq_bunch))
                        elif const.SKIP_OS_UPLOAD:
                            global_logger.info("Upload to OpenSearch is disabled, skipping upload")
                            success = True
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
                            chunk.finalize(success)
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

                chunk.finalize(success)
                if not success:
                    all_chunks_success = False
            NATS_BATCH_PROCESS_SECONDS_HISTOGRAM.record(
                max(time.time() - batch_processing_start, 0.0)
            )

            counts["total_batches"] += 1
            counts["total_publish_batches"] += job_count / PUBLISH_BATCH_SIZE
            if all_chunks_success:
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
        worker_lifetime_seconds = max(time.time() - worker_start_time, 0.0)

        WORKER_LIFETIME_SECONDS_HISTOGRAM.record(worker_lifetime_seconds)
        WORKER_JOBS_PROCESSED_PER_LIFETIME_HISTOGRAM.record(counts["total_jobs"])
        WORKER_JOBS_PUBLISHED_AMQ_PER_LIFETIME_HISTOGRAM.record(
            counts["total_sent_amq"]
        )
        WORKER_JOBS_PUBLISHED_OS_PER_LIFETIME_HISTOGRAM.record(
            counts["total_sent_os"]
        )
        WORKER_IDLE_SECONDS_HISTOGRAM.record(counts["total_idle_seconds"])
        WORKER_SHUTDOWNS_COUNTER.add(1)
        
        finalize_worker_trace(
            worker_seconds=worker_lifetime_seconds,
            total_jobs=counts["total_jobs"],
            total_sent_amq=counts["total_sent_amq"],
            total_sent_os=counts["total_sent_os"],
        )

        # Best-effort flush so shutdown metrics are exported before process exit.
        try:
            opentelemetry.metrics.get_meter_provider().force_flush()
        except Exception as exc:  # pylint: disable=broad-except
            global_logger.warning("Failed to force flush metrics during shutdown: %s", exc)
        try:
            opentelemetry.trace.get_tracer_provider().force_flush()
        except Exception as exc:  # pylint: disable=broad-except
            global_logger.warning("Failed to force flush traces during shutdown: %s", exc)

    global_logger.info(
        "Processing of NATS queue finished. Summary: "
        "total_jobs=%d, total_fetch_batches=%d, total_publish_batches=~%.1f, "
        "total_sent_amq=%d, total_sent_os=%d",
        counts["total_jobs"],
        counts["total_batches"],
        counts["total_publish_batches"],
        counts["total_sent_amq"],
        counts["total_sent_os"],
    )
