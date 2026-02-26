"""
NATS JetStream connection and publishing utilities.
"""

import json
import logging
import asyncio
import time
import sys
from nats.aio.client import Client as NATS
from nats.js import JetStreamContext

import classad
from otel_setup import trace_span

# Global NATS JetStream connection
_nats_connection = None
_nats_jetstream = None
_nats_loop = None
_checkpoint_kv = None

# Usually these spans are not needed and increase noise. Use for debugging if needed.
# @trace_span("get_nats_connection")
def get_nats_connection(nats_servers: str, stream_name: str, subject: str):
    """
    Get or create a NATS JetStream connection.
    Returns (connection, jetstream) tuple.
    The event loop is stored globally and reused for all operations.
    """
    from opentelemetry import trace

    current_span = trace.get_current_span()
    if current_span:
        current_span.set_attribute("nats.stream_name", stream_name)
        current_span.set_attribute("nats.subject", subject)

    global _nats_connection, _nats_jetstream, _nats_loop

    if _nats_connection is None or _nats_connection.is_closed:
        nats_servers = [s.strip() for s in nats_servers.split(",")]

        try:
            # Clean up any existing connection state before creating new ones
            # This is critical in multi-process contexts to avoid state conflicts
            if _nats_connection is not None:
                # Connection exists but is closed - clean it up
                try:
                    if (
                        _nats_loop is not None
                        and not _nats_loop.is_closed()
                        and not _nats_connection.is_closed
                    ):
                        _nats_loop.run_until_complete(_nats_connection.close())
                except Exception as e:
                    logging.debug(
                        "Error closing existing NATS connection (may already be closed): %s",
                        str(e),
                    )
                _nats_connection = None
                _nats_jetstream = None

            # In multi-process contexts (ProcessPoolExecutor), each worker process
            # needs its own isolated event loop. Always create a fresh event loop.
            # Close any existing loop if it exists to avoid conflicts
            try:
                existing_loop = asyncio.get_event_loop()
                if not existing_loop.is_closed():
                    existing_loop.close()
            except (RuntimeError, AttributeError):
                pass
            # Create a completely fresh event loop for this process
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            # Store the loop for reuse
            _nats_loop = loop
            _nats_connection = NATS()
            # TODO: Why do we need to create new streams?
            logging.warning("Connecting to NATS JetStream at %s", nats_servers)
            loop.run_until_complete(_nats_connection.connect(servers=nats_servers))
            _nats_jetstream = _nats_connection.jetstream()

            # Ensure the stream exists
            try:
                loop.run_until_complete(_nats_jetstream.stream_info(stream_name))
            except Exception:
                # Stream doesn't exist, create it
                logging.info("Creating NATS JetStream stream: %s", stream_name)
                try:
                    loop.run_until_complete(
                        _nats_jetstream.add_stream(name=stream_name, subjects=[subject])
                    )
                except Exception as e:
                    # TODO: Improve this
                    # If stream already exists with this subject (error 10065), that's fine
                    if "10065" in str(e) or "subjects overlap" in str(e).lower():
                        logging.info("Stream with subject %s already exists", subject)
                    else:
                        raise

            logging.info("Connected to NATS JetStream at %s", nats_servers)
        except Exception as e:
            logging.error("Failed to connect to NATS JetStream: %s", str(e))
            raise

    return _nats_connection, _nats_jetstream


@trace_span("get_jetstream_context")
def get_jetstream_context(nats_servers: str):
    """
    Get or create a NATS JetStream context without creating a stream.
    This is useful for KeyValue operations that don't need a stream.
    The connection is shared with get_nats_connection if already established.

    Args:
        nats_servers: Comma-separated list of NATS server URLs

    Returns:
        JetStreamContext: The JetStream context for operations
    """
    global _nats_connection, _nats_jetstream, _nats_loop

    # Reuse existing connection if available
    if _nats_connection is not None and not _nats_connection.is_closed:
        return _nats_jetstream

    # Create new connection if needed
    nats_servers_list = [s.strip() for s in nats_servers.split(",")]

    try:
        # Clean up any existing connection state before creating new ones
        # This is critical in multi-process contexts to avoid state conflicts
        if _nats_connection is not None:
            # Connection exists but is closed - clean it up
            try:
                if (
                    _nats_loop is not None
                    and not _nats_loop.is_closed()
                    and not _nats_connection.is_closed
                ):
                    _nats_loop.run_until_complete(_nats_connection.close())
            except Exception as e:
                logging.debug(
                    "Error closing existing NATS connection (may already be closed): %s",
                    str(e),
                )
            _nats_connection = None
            _nats_jetstream = None

        # In multi-process contexts, each worker process needs its own isolated
        # event loop. Always create a fresh event loop.
        # Close any existing loop if it exists to avoid conflicts
        try:
            existing_loop = asyncio.get_event_loop()
            if not existing_loop.is_closed():
                existing_loop.close()
        except (RuntimeError, AttributeError):
            pass
        # Create a completely fresh event loop for this process
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)

        # Store the loop for reuse (important for publish operations)
        _nats_loop = loop

        _nats_connection = NATS()
        logging.debug(
            "Connecting to NATS JetStream at %s (for KeyValue operations)",
            nats_servers_list,
        )
        loop.run_until_complete(_nats_connection.connect(servers=nats_servers_list))
        _nats_jetstream = _nats_connection.jetstream()

        logging.info("Connected to NATS JetStream at %s", nats_servers_list)
    except Exception as e:
        logging.error("Failed to connect to NATS JetStream: %s", str(e))
        raise

    return _nats_jetstream

# This creates a lot of spans and can increase load in the trace storage. Use only for debugging.
# @trace_span("publish_jobs_to_nats")
def publish_jobs_to_nats(
    jetstream: JetStreamContext,
    subject: str,
    job_docs: list[classad.ClassAd],
    batch_size: int = 100,
    timeout: int = None,
    loop=None,
):
    """
    Publish jobs to NATS JetStream in batches.
    This is the primary method for publishing jobs. Jobs are published concurrently
    within each batch for better performance.

    Args:
        jetstream: NATS JetStream context
        subject: NATS subject to publish to
        job_docs: List of job documents (ClassAd) - can be a single-item list
        batch_size: Number of jobs to publish per batch (default: 100)
        timeout: Timeout in seconds for each publish operation (default from constants)
        loop: Optional event loop to use (for thread-local connections). If None, uses global _nats_loop

    Returns:
        int: Number of successfully published jobs
    """
    from opentelemetry import trace

    current_span = trace.get_current_span()
    if current_span:
        current_span.set_attribute("nats.subject", subject)
        # Note: job.count and batch_size are set at the query level, not per-publish call
        # to avoid overwriting with leftover batch values

    global _nats_loop, _nats_connection

    # Import here to avoid circular imports
    import constants as const

    if timeout is None:
        timeout = const.NATS_PUBLISH_TIMEOUT

    if not job_docs:
        return 0

    try:
        function_start = time.time()
        # Check if connection is still alive before attempting to publish
        if _nats_connection is not None and _nats_connection.is_closed:
            logging.warning("NATS connection is closed, cannot publish")
            return 0

        # Use provided loop (for thread-local connections) or fall back to global
        if loop is None:
            if _nats_loop is None or _nats_loop.is_closed():
                logging.info("_nats_loop is None or closed, creating new loop")
                try:
                    loop = asyncio.get_event_loop()
                    if loop.is_closed():
                        raise RuntimeError("Event loop is closed")
                except RuntimeError:
                    logging.info("Creating new event loop")
                    loop = asyncio.new_event_loop()
                    asyncio.set_event_loop(loop)
            else:
                loop = _nats_loop

        # Check if loop is already running - run_until_complete() can't be called if it is
        if loop.is_running():
            logging.error(
                "Event loop is already running, cannot use run_until_complete()"
            )
            return 0

        # Import here to avoid circular imports
        from otel_setup import EXECUTION_ID

        # Prepare all messages with headers containing execution_id for correlation
        # NATS Python library expects headers as dict[str, str] (not dict[str, list])
        messages = []
        headers = {"X-Query-Execution-Id": str(EXECUTION_ID)}
        for job_doc in job_docs:
            job_str = str(job_doc)
            message_bytes = json.dumps(job_str).encode("utf-8")
            messages.append((message_bytes, headers))

        async def _publish_batch(batch_messages):
            """Publish a batch of messages concurrently."""

            async def _publish_single(msg_data):
                try:
                    msg_bytes, msg_headers = msg_data
                    await jetstream.publish(
                        subject, msg_bytes, headers=msg_headers, timeout=timeout
                    )
                    return True
                except Exception as e:
                    logging.error("Failed to publish single job in batch: %s", str(e))
                    logging.error("Job size: %s kB", sys.getsizeof(msg_bytes) / 1024)
                    return False

            # Publish all messages in the batch concurrently
            results = await asyncio.gather(
                *[_publish_single(msg) for msg in batch_messages],
                return_exceptions=True,
            )
            # Count successful publishes (True values, excluding exceptions)
            return sum(1 for r in results if r is True)

        # Process jobs in batches
        total_published = 0

        for batch_idx in range(0, len(job_docs), batch_size):
            batch = messages[batch_idx : batch_idx + batch_size]

            # Calculate batch timeout - since publishes are concurrent, they should complete within roughly the timeout time
            # Use a multiplier to account for network overhead with many concurrent requests
            batch_timeout = timeout * min(
                3, len(batch)
            )  # Cap at 3x timeout for large batches

            try:
                # Publish the batch with timeout
                publish_coro = asyncio.wait_for(
                    _publish_batch(batch), timeout=batch_timeout
                )
                batch_published = loop.run_until_complete(publish_coro)
                total_published += batch_published

            except asyncio.TimeoutError:
                logging.error(
                    "Timeout publishing batch to NATS",
                )
                # Continue with next batch
                continue
            except Exception as e:
                logging.error(
                    "Exception publishing batch: %s, type=%s",
                    str(e),
                    type(e).__name__,
                )
                # Continue with next batch
                continue

        total_time = time.time() - function_start
        if total_published > 0:
            logging.debug(
                "Published %d/%d jobs in %.3fs (%.2f jobs/sec)",
                total_published,
                len(job_docs),
                total_time,
                total_published / total_time if total_time > 0 else 0,
            )

        # Update span attributes with results
        # Note: jobs.published and jobs.failed are set at the query/history level with totals
        # to avoid overwriting with per-batch values
        if current_span:
            current_span.set_attribute("publish.duration_seconds", total_time)

        return total_published

    except Exception as e:
        total_time = time.time() - function_start
        logging.error(
            "Failed to publish jobs batch to NATS (elapsed=%.3fs): %s, type=%s",
            total_time,
            str(e),
            type(e).__name__,
        )
        import traceback

        logging.error("Traceback: %s", traceback.format_exc())
        return 0


def encode_key(key: str) -> str:
    """
    Encode a schedd name for use as a NATS JetStream KeyValue key.
    (NATS KeyValue keys cannot contain certain characters like periods (.) and @ symbols.)

    Args:
        key: Original key (e.g., "crab3@vocms0195.cern.ch")

    Returns:
        str: Encoded key safe for use in NATS KeyValue store
    """
    return key.replace("@", "__at__").replace(".", "__dot__")


def decode_key(encoded_key: str) -> str:
    """
    Decode a KeyValue key back to the original string.

    Args:
        encoded_key: Encoded key from NATS KeyValue store

    Returns:
        str: Original schedd name
    """
    return encoded_key.replace("__at__", "@").replace("__dot__", ".")


def get_checkpoint_kv(
    jetstream: JetStreamContext, kv_bucket_name: str = "spider_checkpoints"
):
    """
    Get or create a KeyValue store for checkpoints.

    Args:
        jetstream: NATS JetStream context
        kv_bucket_name: Name of the KeyValue bucket to use for checkpoints

    Returns:
        KeyValue store instance
    """
    global _checkpoint_kv

    if _checkpoint_kv is None:
        try:
            loop = asyncio.get_event_loop()
            if loop.is_closed():
                raise RuntimeError("Event loop is closed")
        except RuntimeError:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)

        try:
            # Try to get existing KeyValue store
            _checkpoint_kv = loop.run_until_complete(
                jetstream.key_value(kv_bucket_name)
            )
            logging.debug("Connected to existing KeyValue store: %s", kv_bucket_name)
        except Exception:
            logging.error("Failed to get KeyValue store: %s", kv_bucket_name)

    return _checkpoint_kv


def get_checkpoint(
    jetstream: JetStreamContext,
    schedd_name: str,
    kv_bucket_name: str = "spider_checkpoints",
):
    """
    Get checkpoint (last completion time) for a schedd from KeyValue store.

    Args:
        jetstream: NATS JetStream context
        schedd_name: Name of the schedd
        kv_bucket_name: Name of the KeyValue bucket

    Returns:
        float: Last completion timestamp, or None if not found
    """
    try:
        kv = get_checkpoint_kv(jetstream, kv_bucket_name)

        try:
            loop = asyncio.get_event_loop()
            if loop.is_closed():
                raise RuntimeError("Event loop is closed")
        except RuntimeError:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)

        # Encode the schedd name to handle special characters like . and @
        encoded_key = encode_key(schedd_name)
        entry = loop.run_until_complete(kv.get(encoded_key))
        if entry:
            completion_time = float(entry.value.decode("utf-8"))
            logging.debug(
                "Retrieved checkpoint for %s: %s", schedd_name, completion_time
            )
            return completion_time
    except Exception as e:
        # Key not found or other error - this is expected for first run
        if "10037" not in str(e) and "not found" not in str(e).lower():
            logging.warning(
                "Failed to get checkpoint for %s from KeyValue store: %s",
                schedd_name,
                str(e),
            )

    return None


def set_checkpoint(
    jetstream: JetStreamContext,
    schedd_name: str,
    completion_date: float,
    kv_bucket_name: str = "spider_checkpoints",
):
    """
    Set checkpoint (last completion time) for a schedd in KeyValue store.

    Args:
        jetstream: NATS JetStream context
        schedd_name: Name of the schedd
        completion_date: Completion timestamp to store
        kv_bucket_name: Name of the KeyValue bucket

    Returns:
        bool: True if successful, False otherwise
    """
    try:
        kv = get_checkpoint_kv(jetstream, kv_bucket_name)
        try:
            loop = asyncio.get_event_loop()
            if loop.is_closed():
                raise RuntimeError("Event loop is closed")
        except RuntimeError:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)

        # Encode the schedd name to handle special characters like . and @
        encoded_key = encode_key(schedd_name)
        # Store completion time as string
        value = str(completion_date).encode("utf-8")
        loop.run_until_complete(kv.put(encoded_key, value))
        logging.info(
            "Updated checkpoint for %s: %s",
            schedd_name,
            time.strftime("%Y-%m-%d %H:%M:%S", time.localtime(completion_date)),
        )
        return True
    except Exception as e:
        logging.error(
            "Failed to set checkpoint for %s in KeyValue store: %s", schedd_name, str(e)
        )
        return False


def get_all_checkpoints(
    jetstream: JetStreamContext, kv_bucket_name: str = "spider_checkpoints"
):
    """
    Get all checkpoints from KeyValue store.

    Args:
        jetstream: NATS JetStream context
        kv_bucket_name: Name of the KeyValue bucket

    Returns:
        dict: Dictionary mapping schedd names to completion timestamps
    """
    checkpoints = {}
    try:
        kv = get_checkpoint_kv(jetstream, kv_bucket_name)

        try:
            loop = asyncio.get_event_loop()
            if loop.is_closed():
                raise RuntimeError("Event loop is closed")
        except RuntimeError:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)

        # Get all keys from the KeyValue store (keys() returns a coroutine that returns a list)
        async def _get_all_keys():
            return await kv.keys()

        keys = loop.run_until_complete(_get_all_keys())
        for encoded_key in keys:
            try:
                entry = loop.run_until_complete(kv.get(encoded_key))
                if entry:
                    completion_time = float(entry.value.decode("utf-8"))
                    # Decode the key back to the original schedd name
                    schedd_name = decode_key(encoded_key)
                    checkpoints[schedd_name] = completion_time
            except Exception as e:
                logging.warning(
                    "Failed to get checkpoint for key %s: %s", encoded_key, str(e)
                )
                continue

        logging.debug("Retrieved %d checkpoints from KeyValue store", len(checkpoints))
    except Exception as e:
        logging.warning("Failed to get all checkpoints from KeyValue store: %s", str(e))

    return checkpoints


def close_nats_connection(connection=None, loop=None, timeout=5.0):
    """
    Properly close a NATS connection and wait for all background tasks to complete.
    This prevents "Task was destroyed but it is pending" errors.

    Args:
        connection: NATS connection to close (if None, closes global connection)
        loop: Event loop to use (if None, uses global _nats_loop or gets current loop)
        timeout: Maximum time to wait for connection to close (seconds)

    Returns:
        bool: True if closed successfully, False otherwise
    """
    global _nats_connection, _nats_jetstream, _nats_loop

    # Use provided connection or global connection
    if connection is None:
        connection = _nats_connection

    if connection is None or connection.is_closed:
        logging.debug("NATS connection is None or already closed")
        return True

    # Determine which loop to use
    if loop is None:
        loop = _nats_loop
        if loop is None or loop.is_closed():
            try:
                loop = asyncio.get_event_loop()
                if loop.is_closed():
                    raise RuntimeError("Event loop is closed")
            except RuntimeError:
                logging.warning(
                    "No valid event loop available for closing NATS connection"
                )
                return False

    if loop.is_closed():
        logging.warning("Event loop is closed, cannot close NATS connection properly")
        return False

    try:

        async def _close():
            """Close the connection and wait for all tasks to complete."""
            if not connection.is_closed:
                # Close the connection - this will stop background tasks
                await connection.close()
                # Give a small delay to allow tasks to finish
                await asyncio.sleep(0.1)

        # Run the close operation with a timeout
        if loop.is_running():
            logging.warning(
                "Event loop is running, cannot use run_until_complete() to close connection"
            )
            return False

        loop.run_until_complete(asyncio.wait_for(_close(), timeout=timeout))

        # Clear global references if this was the global connection
        if connection == _nats_connection:
            _nats_connection = None
            _nats_jetstream = None
            # Don't close the loop here - it might be reused
            # _nats_loop = None

        logging.debug("NATS connection closed successfully")
        return True
    except asyncio.TimeoutError:
        logging.warning("Timeout closing NATS connection after %s seconds", timeout)
        return False
    except Exception as e:
        logging.warning("Error closing NATS connection: %s", str(e))
        return False
