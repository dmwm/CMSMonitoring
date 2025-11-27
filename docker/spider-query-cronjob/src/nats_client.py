"""
NATS JetStream connection and publishing utilities.
"""

import os
import json
import logging
import asyncio

from nats.aio.client import Client as NATS

# Global NATS JetStream connection
_nats_connection = None
_nats_jetstream = None


def get_nats_connection(args):
    """
    Get or create a NATS JetStream connection.
    """
    global _nats_connection, _nats_jetstream

    if _nats_connection is None or _nats_connection.is_closed:
        nats_servers = getattr(args, "nats_servers", None) or os.getenv(
            "NATS_SERVER", "nats://nats.cluster.local:4222"
        )
        nats_servers = [s.strip() for s in nats_servers.split(",")]

        try:
            # Get or create event loop
            try:
                loop = asyncio.get_event_loop()
                if loop.is_closed():
                    raise RuntimeError("Event loop is closed")
            except RuntimeError:
                loop = asyncio.new_event_loop()
                asyncio.set_event_loop(loop)

            _nats_connection = NATS()
            loop.run_until_complete(_nats_connection.connect(servers=nats_servers))
            _nats_jetstream = _nats_connection.jetstream()

            # Ensure the stream exists
            stream_name = getattr(args, "nats_stream_name", None) or os.getenv(
                "NATS_QUEUE_NAME", "CMS_HTCONDOR_QUEUE"
            )
            subject = getattr(args, "nats_subject", None) or os.getenv(
                "NATS_SUBJECT", "cms.htcondor.queue.job"
            )

            try:
                loop.run_until_complete(_nats_jetstream.stream_info(stream_name))
            except Exception:
                # Stream doesn't exist, create it
                logging.info("Creating NATS JetStream stream: %s", stream_name)
                loop.run_until_complete(
                    _nats_jetstream.add_stream(name=stream_name, subjects=[subject])
                )

            logging.info("Connected to NATS JetStream at %s", nats_servers)
        except Exception as e:
            logging.error("Failed to connect to NATS JetStream: %s", str(e))
            raise

    return _nats_connection, _nats_jetstream


def publish_job_to_nats(jetstream, subject, job_id, job_doc, args):
    """
    Publish a single job to NATS JetStream.
    """
    try:
        message_data = {"id": job_id, "doc": job_doc}
        message_json = json.dumps(message_data).encode("utf-8")

        # Get or create event loop
        try:
            loop = asyncio.get_event_loop()
            if loop.is_closed():
                raise RuntimeError("Event loop is closed")
        except RuntimeError:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)

        loop.run_until_complete(jetstream.publish(subject, message_json))
        return True
    except Exception as e:
        logging.error("Failed to publish job %s to NATS: %s", job_id, str(e))
        return False
