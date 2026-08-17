"""
Helper utilities to consume jobs from the NATS JetStream queue.
"""

from __future__ import annotations

import asyncio
import json
from dataclasses import dataclass
from typing import List, Optional, Sequence, Tuple
import classad
from nats.aio.client import Client as NATS
from nats.js.api import ConsumerConfig, AckPolicy, DeliverPolicy

import src.constants as const
from src.convert_to_json import unique_doc_id
from src.otel_setup import global_logger


def _normalize_header_value(value):
    """Normalize NATS header value to a single string."""
    if isinstance(value, list):
        return value[0] if value else None
    if isinstance(value, str):
        return value
    return None


def _get_event_loop() -> asyncio.AbstractEventLoop:
    """
    Get the current asyncio loop or create a new one when running
    in a synchronous context.
    """
    try:
        loop = asyncio.get_event_loop()
        if loop.is_closed():
            raise RuntimeError("Existing asyncio loop is closed")
        return loop
    except RuntimeError:
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        return loop


def _str_to_classad(s: str) -> classad.ClassAd:
    """
    Convert a string to a ClassAd object.
    """
    return classad.ClassAd(s)

@dataclass
class NATSBatch:
    """
    Container holding the raw JetStream messages together with the parsed jobs.
    """

    jobs: List[Tuple[str, classad.ClassAd]]
    _messages: Sequence
    _loop: asyncio.AbstractEventLoop
    traceparent: str = None  # First valid traceparent seen in the batch
    tracestate: str = None  # tracestate associated with the first traceparent
    latest_traceparent: str = None  # Latest traceparent seen (for mismatch diagnostics)
    latest_tracestate: str = None  # Latest tracestate seen (for mismatch diagnostics)
    has_traceparent_mismatch: bool = False
    redelivered_messages: int = 0

    def ack(self) -> None:
        """Acknowledge all messages in the batch using parallel async operations."""
        if not self._messages:
            return
        try:
            # Batch all ack operations and run them in parallel for better throughput
            ack_tasks = [msg.ack() for msg in self._messages]
            self._loop.run_until_complete(asyncio.gather(*ack_tasks, return_exceptions=True))
        except Exception as exc:  # pragma: no cover - network failure
            global_logger.error("Failed to batch ack NATS messages: %s", exc)
            # Fallback to individual acks if batch fails
            for msg in self._messages:
                try:
                    self._loop.run_until_complete(msg.ack())
                except Exception as msg_exc:
                    global_logger.error("Failed to ack individual NATS message: %s", msg_exc)

    def nak(self) -> None:
        """Request re-delivery for all messages in the batch using parallel async operations."""
        if not self._messages:
            return
        try:
            # Batch all nak operations and run them in parallel for better throughput
            nak_tasks = [msg.nak() for msg in self._messages]
            self._loop.run_until_complete(asyncio.gather(*nak_tasks, return_exceptions=True))
        except Exception as exc:  # pragma: no cover - network failure
            global_logger.error("Failed to batch NAK NATS messages: %s", exc)
            # Fallback to individual naks if batch fails
            for msg in self._messages:
                try:
                    self._loop.run_until_complete(msg.nak())
                except Exception as msg_exc:
                    global_logger.error("Failed to NAK individual NATS message: %s", msg_exc)

    def finalize(self, success: bool) -> None:
        if success:
            self.ack()
        else:
            self.nak()

    def chunked(self, chunk_size: int) -> List["NATSBatch"]:
        """Split this batch into smaller sub-batches for earlier finalize/ack."""
        if chunk_size <= 0 or len(self.jobs) <= chunk_size:
            return [self]

        result: List[NATSBatch] = []
        for start in range(0, len(self.jobs), chunk_size):
            end = start + chunk_size
            result.append(
                NATSBatch(
                    jobs=self.jobs[start:end],
                    _messages=self._messages[start:end],
                    _loop=self._loop,
                    traceparent=self.traceparent,
                    tracestate=self.tracestate,
                    latest_traceparent=self.latest_traceparent,
                    latest_tracestate=self.latest_tracestate,
                    has_traceparent_mismatch=self.has_traceparent_mismatch,
                    redelivered_messages=self.redelivered_messages,
                )
            )
        return result


class NATSQueueConsumer:
    """
    Thin synchronous wrapper around the asyncio-based NATS JetStream client.
    """

    def __init__(self):
        self.loop = _get_event_loop()
        self.nc: Optional[NATS] = None
        self.jetstream = None
        self.subscription = None

        self.stream_name = const.NATS_STREAM_NAME
        self.subject = const.NATS_SUBJECT
        self.consumer_name = const.NATS_CONSUMER_NAME
        self._connect()

    def _connect(self) -> None:
        servers = const.NATS_SERVER
        server_list = [s.strip() for s in str(servers).split(",") if s.strip()]

        self.nc = NATS()
        self.loop.run_until_complete(self.nc.connect(servers=server_list))
        self.jetstream = self.nc.jetstream()

        # Ensure the stream exists - the publisher should have created it.
        try:
            self.loop.run_until_complete(self.jetstream.stream_info(self.stream_name))
        except Exception as exc:
            global_logger.error(
                "NATS stream %s is not available: %s", self.stream_name, exc
            )
            raise

        # Ensure there is a durable consumer bound to the queue subject.
        # Note: If the consumer already exists, max_ack_pending cannot be changed.
        # To update it, delete the existing consumer first (e.g., via NATS CLI).
        try:
            self.loop.run_until_complete(
                self.jetstream.consumer_info(self.stream_name, self.consumer_name)
            )
            global_logger.info(
                "Using existing consumer %s for stream %s",
                self.consumer_name,
                self.stream_name,
            )
        except Exception:
            global_logger.info(
                "Creating durable consumer %s for stream %s with optimized settings for throughput",
                self.consumer_name,
                self.stream_name,
            )
            config = ConsumerConfig(
                durable_name=self.consumer_name,
                ack_policy=AckPolicy.EXPLICIT,
                filter_subject=self.subject,
                deliver_policy=DeliverPolicy.ALL,
                max_ack_pending=100000,  # High limit to allow many in-flight messages (you already have 100k)
                ack_wait=const.NATS_ACK_WAIT_SECONDS,
                # Note: Other throughput optimizations:
                # - Increase max_waiting_pulls if you see "Waiting Pulls" hitting limits
                # - Consider rate_limit/rate_limit_burst if server-side throttling is an issue
                # - Existing consumers keep prior ack_wait unless recreated
            )
            self.loop.run_until_complete(
                self.jetstream.add_consumer(
                    stream=self.stream_name,
                    config=config,
                )
            )

        self.subscription = self.loop.run_until_complete(
            self.jetstream.pull_subscribe(
                subject=self.subject,
                durable=self.consumer_name,
                stream=self.stream_name,
            )
        )

    def fetch_jobs(self, batch_size: int, timeout: float) -> NATSBatch:
        """
        Fetch a batch of jobs from JetStream. Returns immediately with available
        messages, even if fewer than batch_size. Returns an empty batch when no
        messages are available before the timeout.
        """
        if not self.subscription:
            return NATSBatch([], [], self.loop)

        try:
            messages = self.loop.run_until_complete(
                self.subscription.fetch(batch_size, timeout=timeout)
            )
        except asyncio.TimeoutError:
            return NATSBatch([], [], self.loop)

        jobs: List[Tuple[str, classad.ClassAd]] = []
        valid_messages = []
        first_traceparent = None
        first_tracestate = None
        latest_traceparent = None
        latest_tracestate = None
        has_traceparent_mismatch = False
        redelivered_messages = 0
        
        for msg in messages:
            try:
                metadata = getattr(msg, "metadata", None)
                if metadata and getattr(metadata, "num_delivered", 1) > 1:
                    redelivered_messages += 1

                if msg.headers:
                    message_traceparent = _normalize_header_value(
                        msg.headers.get("traceparent")
                    )
                    message_tracestate = _normalize_header_value(
                        msg.headers.get("tracestate")
                    )
                    if message_traceparent:
                        latest_traceparent = message_traceparent
                        latest_tracestate = message_tracestate
                        if first_traceparent is None:
                            first_traceparent = message_traceparent
                            first_tracestate = message_tracestate
                        elif message_traceparent != first_traceparent:
                            has_traceparent_mismatch = True
                
                payload = json.loads(msg.data.decode("utf-8"))
                job_doc = _str_to_classad(payload)
                job_id = unique_doc_id(job_doc)
                jobs.append((job_id, job_doc))
                valid_messages.append(msg)
            except Exception as exc:
                global_logger.error("Dropping malformed NATS job payload: %s", exc)
                try:
                    self.loop.run_until_complete(msg.ack())
                except Exception as ack_exc:  # pragma: no cover
                    global_logger.error("Failed to ack malformed message: %s", ack_exc)

        batch = NATSBatch(jobs=jobs, _messages=valid_messages, _loop=self.loop)
        if first_traceparent:
            batch.traceparent = first_traceparent
            batch.tracestate = first_tracestate
            batch.latest_traceparent = latest_traceparent
            batch.latest_tracestate = latest_tracestate
            batch.has_traceparent_mismatch = has_traceparent_mismatch
        batch.redelivered_messages = redelivered_messages
        return batch

    def close(self) -> None:
        if self.nc and not self.nc.is_closed:
            try:
                # Use close() instead of drain() because this is a pull subscriber
                # and all fetched messages have already been acked/naked via batch.finalize().
                self.loop.run_until_complete(self.nc.close())
            except Exception as exc:  # pragma: no cover
                global_logger.warning("Error while closing NATS connection: %s", exc)




