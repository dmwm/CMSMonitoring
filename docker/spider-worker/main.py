from src.queues import process_nats_queue
import src.constants as const
from src.otel_setup import global_logger


def main():
    global_logger.info("Starting spider worker")
    process_nats_queue(
        nats_batch_size=const.NATS_BATCH_SIZE,
        nats_fetch_timeout=const.NATS_FETCH_TIMEOUT,
        nats_idle_sleep=const.NATS_IDLE_SLEEP,
        metadata=None,
        email_alerts=None
    )

if __name__ == "__main__":
    main()
