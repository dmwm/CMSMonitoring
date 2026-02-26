from src.queues import process_nats_queue
from src.otel_setup import global_logger


def main():
    global_logger.info("Starting spider worker")
    process_nats_queue(
        nats_batch_size=1000,
        nats_fetch_timeout=0.1,
        nats_idle_sleep=1,
        feed_es_for_queues=False,
        feed_amq=True,
        metadata=None,
        email_alerts=None
    )

if __name__ == "__main__":
    main()
