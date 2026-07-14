from logging import DEBUG
import itertools
import src.constants as const
from src.promxy import query_promxy, generate_os_docs
from src.helpers import generate_date_ranges
from src.os_utils import os_upload_docs_in_bulk
from src.logging import logger


logger.debug(
    f"""
Environment Configuration:
  PROMXY_URL       = {const.PROMXY_URL}
  PROM_QUERY       = {const.PROM_QUERY}
  OPENSEARCH_INDEX = {const.INDEX}
  OPENSEARCH_HOST  = {const.OS_HOST}
  CERT_PATH        = {const.CERT_PATH}
  START_DATE       = {const.START_DATE}
  END_DATE         = {const.END_DATE}
  STEP             = {const.STEP}
"""
)

# PromQL gives inconsistent responses for query ranges spanning longer
# than 30 days, so we split our date range in smaller chunks if necessary.
date_ranges = generate_date_ranges(const.START, const.END)
logger.debug(f"Date ranges: {date_ranges}")
responses = []
for range_start, range_end in date_ranges:
    logger.info(f"Current date range: {range_start.date()} - {range_end.date()}")
    promxy_request_params = {
        "query": const.PROM_QUERY,
        "start": int(range_start.timestamp()),
        "end": int(range_end.timestamp()),
        "step": const.STEP,
    }
    responses.append(
        query_promxy(promxy_url=const.PROMXY_URL, params=promxy_request_params)
    )

if const.DRY_RUN:
    n_samples = 2
    generator = generate_os_docs(responses=responses, os_index=const.INDEX)
    doc_samples = list(itertools.islice(generator, n_samples))
    total_documents = n_samples + sum(1 for _ in generator)

    logger.info(f"Document samples: {doc_samples}")
    logger.info(f"[DRY RUN] {total_documents} docs generated.")
else:
    logger.info(f"Uploading to OpenSearch: {const.OS_HOST},\n Index: {const.INDEX}")
    successes, failures = os_upload_docs_in_bulk(
        os_host=const.OS_HOST,
        ca_cert_path=const.CERT_PATH,
        doc_iterable=generate_os_docs(responses=responses, os_index=const.INDEX),
    )
    logger.info(f"Uploaded {successes} docs.")
    if failures:
        if logger.getEffectiveLevel() == DEBUG:
            for failure in failures:
                logger.debug(f"Failure detail: {failure}")
        logger.error(f"{len(failures)} OpenSearch index failures occurred.")
