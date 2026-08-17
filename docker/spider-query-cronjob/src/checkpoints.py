import time
import constants as const
from nats_client import (
    JetStreamContext,
    set_checkpoint,
    get_all_checkpoints,
    close_nats_connection,
    get_nats_connection,
)
from utils import global_logger


def update_checkpoint(
    jetstream: JetStreamContext,
    name: str,
    completion_date: float,
    kv_bucket_name: str = const.CHECKPOINT_KV_BUCKET,
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
        global_logger.error(
            "Cannot update checkpoint: NATS JetStream connection is None"
        )
        return

    success = set_checkpoint(jetstream, name, completion_date, kv_bucket_name)
    if not success:
        global_logger.warning(
            "Failed to update checkpoint for %s in KeyValue store. Completion date: %s",
            name,
            completion_date,
        )


def load_checkpoints(schedd_ads):
    """
    Load checkpoints from NATS KeyValue store and prepare tasks for processing.
    Each worker process will directly update checkpoints in the KV store.
    """

    nats_connection = None
    jetstream = None
    try:
        # Use get_jetstream_context instead of get_nats_connection since
        # KeyValue operations don't require a stream to be created
        nats_connection, jetstream = get_nats_connection(
            const.NATS_SERVER, const.NATS_STREAM_NAME, const.NATS_SUBJECT
        )
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
        finally:
            if jetstream is not None:
                close_nats_connection(connection=nats_connection, timeout=5)
    else:
        global_logger.warning(
            "NATS connection not available. Using empty checkpoint dict. "
            "All schedds will use default time windows."
        )
        checkpoint = {}

    # Check for last completion time
    last_completions = {}
    for schedd_ad in schedd_ads:
        name = schedd_ad["Name"]
        last_completion = checkpoint.get(
            name, time.time() - const.HISTORY_QUERY_MAX_N_MINUTES * 60
        )
        last_completions[name] = (
            schedd_ad,
            max(last_completion, time.time() - const.RETENTION_POLICY),
        )

    return last_completions


def prepare_tasks(last_completions: dict):
    tasks = []
    for schedd_ad, last_completion in last_completions.values():
        schedd_name = schedd_ad["Name"]
        # For CRAB, only ever get a maximum of 12 h
        if (
            schedd_name.startswith("crab")
            and last_completion < time.time() - const.CRAB_MAX_QUERY_TIME_SPAN
        ):
            last_completion = time.time() - const.HISTORY_QUERY_MAX_N_MINUTES * 60
        tasks.append((schedd_ad, last_completion))
    return tasks
