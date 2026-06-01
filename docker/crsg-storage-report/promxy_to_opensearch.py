from logging import DEBUG
import calendar
import itertools
from datetime import datetime
from typing import Optional
import src.constants as const
from src.promxy import (
    compute_usage_records,
    aggregate_cmssst_disk_by_tier,
)
from src.os_utils import (
    os_upload_docs_in_bulk,
    fetch_latest_docs_per_group_before_timestamp,
    fetch_rucio_locked_dynamic_totals_composite,
)
from src.logging import logger


def resolve_query_times() -> list[datetime]:
    """Always monthly: one query time per UTC month (last second of month)."""
    query_times = const.build_monthly_query_times()
    mode = "backfill" if const.BACKFILL_ENABLED else "regular"
    if const.BACKFILL_ENABLED:
        logger.info(
            "%s: %s monthly snapshot(s) from UTC %s to %s",
            mode,
            len(query_times),
            query_times[0].isoformat(),
            query_times[-1].isoformat(),
        )
    else:
        logger.info(
            "regular: resolved reporting snapshot month %s (upload_day=%s offset=%s)",
            query_times[0].strftime("%Y-%m"),
            const.RUCIO_UPLOAD_DAY_OF_MONTH,
            const.RUCIO_REPORTING_MONTH_OFFSET,
        )
    return query_times


const.require_rucio_usage_config()

logger.debug(
    f"""
Environment Configuration:
  OPENSEARCH_INDEX = {const.INDEX}
  OPENSEARCH_HOST  = {const.OS_HOST}
  CERT_PATH        = {const.CERT_PATH}
  CMSSST_OS_HOST   = {const.CMSSST_OS_HOST}
  CMSSST_OS_INDEX  = {const.CMSSST_OS_INDEX}
  CMSSST_FILTER    = {const.CMSSST_OS_FILTER_FIELD}:{const.CMSSST_OS_FILTER_VALUE}
  EXCLUDE_RSE_REGEX = {const.EXCLUDE_RSE_REGEX}
  CMSSST_GROUP_BY  = {const.CMSSST_OS_GROUP_FIELD}
  CMSSST_EXCLUDE_BY = {const.CMSSST_OS_EXCLUDE_FIELD}
  CMSSST_SORT_BY   = {const.CMSSST_OS_SORT_FIELD}
  RUCIO_USAGE_OS_HOST = {const.RUCIO_USAGE_OS_HOST}
  RUCIO_USAGE_OS_INDEX = {const.RUCIO_USAGE_OS_INDEX}
  RUCIO_USAGE_SUM_FIELD = {const.RUCIO_USAGE_SUM_FIELD}
  RUCIO_USAGE_TIME_FIELD = {const.RUCIO_USAGE_TIME_FIELD}
  RUCIO_USAGE_RSE_TIER_FIELD = {const.RUCIO_USAGE_RSE_TIER_FIELD}
  RUCIO_USAGE_RSE_TYPE_FIELD = {const.RUCIO_USAGE_RSE_TYPE_FIELD}
  RUCIO_USAGE_RSE_IDENTITY_FIELD = {const.RUCIO_USAGE_RSE_IDENTITY_FIELD}
  RUCIO_USAGE_LOCK_FIELD = {const.RUCIO_USAGE_LOCK_FIELD}
QUERY_DATE       = {const.QUERY_DATE}
QUERY_TIME_UTC   = {const.QUERY_TIME.isoformat()}
BACKFILL_ENABLED = {const.BACKFILL_ENABLED}
BACKFILL_START   = {const.BACKFILL_START_DATE}
BACKFILL_END     = {const.BACKFILL_END_DATE}
RUCIO_UPLOAD_DAY_OF_MONTH = {const.RUCIO_UPLOAD_DAY_OF_MONTH}
RUCIO_REPORTING_MONTH_OFFSET = {const.RUCIO_REPORTING_MONTH_OFFSET}
"""
)

query_times = resolve_query_times()

rucio_totals_by_ts: dict[int, dict[tuple[str, str], dict[str, float]]] = {}

for query_time_dt in query_times:
    query_ts = int(query_time_dt.timestamp())
    y_m = (query_time_dt.year, query_time_dt.month)
    last_dom = calendar.monthrange(y_m[0], y_m[1])[1]
    first_candidate_day = max(1, last_dom - 9)
    rucio_totals: dict[tuple[str, str], dict[str, float]] = {}
    chosen_day: Optional[int] = None

    for day in range(last_dom, first_candidate_day - 1, -1):
        gte_sec, lte_sec = const.utc_calendar_day_epoch_bounds(y_m[0], y_m[1], day)
        rucio_totals = fetch_rucio_locked_dynamic_totals_composite(
            os_host=const.RUCIO_USAGE_OS_HOST,
            ca_cert_path=const.CERT_PATH,
            index=const.RUCIO_USAGE_OS_INDEX,
            time_field=const.RUCIO_USAGE_TIME_FIELD,
            sum_field=const.RUCIO_USAGE_SUM_FIELD,
            tier_field=const.RUCIO_USAGE_RSE_TIER_FIELD,
            rse_type_field=const.RUCIO_USAGE_RSE_TYPE_FIELD,
            rse_identity_field=const.RUCIO_USAGE_RSE_IDENTITY_FIELD,
            lock_field=const.RUCIO_USAGE_LOCK_FIELD,
            lock_value_locked=const.RUCIO_USAGE_LOCK_VALUE_LOCKED,
            lock_value_dynamic=const.RUCIO_USAGE_LOCK_VALUE_DYNAMIC,
            gte_epoch_sec=gte_sec,
            lte_epoch_sec=lte_sec,
            batch_size=const.RUCIO_USAGE_COMPOSITE_BATCH_SIZE,
        )
        if rucio_totals:
            chosen_day = day
            break

    if chosen_day is None:
        logger.warning(
            "Rucio usage OpenSearch agg for reporting month %04d-%02d: no data in last 10 UTC days "
            "(tried days %02d..%02d)",
            y_m[0],
            y_m[1],
            first_candidate_day,
            last_dom,
        )
    elif chosen_day == last_dom:
        logger.info(
            "Rucio usage OpenSearch agg for reporting month %04d-%02d: using last UTC day %04d-%02d-%02d "
            "epoch window [%s, %s]",
            y_m[0],
            y_m[1],
            y_m[0],
            y_m[1],
            last_dom,
            gte_sec,
            lte_sec,
        )
    else:
        logger.info(
            "Rucio usage OpenSearch agg for reporting month %04d-%02d: no data on last UTC day; "
            "fallback to UTC day %04d-%02d-%02d epoch window [%s, %s]",
            y_m[0],
            y_m[1],
            y_m[0],
            y_m[1],
            chosen_day,
            gte_sec,
            lte_sec,
        )

    rucio_totals_by_ts[query_ts] = rucio_totals

cmssst_source_docs: list[dict] = []
cmssst_tier_sums_by_ts: dict[int, dict[str, dict[str, float]]] = {}

for idx, query_time_dt in enumerate(query_times):
    query_ts = int(query_time_dt.timestamp())
    cmssst_docs_at_ts = fetch_latest_docs_per_group_before_timestamp(
        os_host=const.CMSSST_OS_HOST,
        ca_cert_path=const.CERT_PATH,
        index=const.CMSSST_OS_INDEX,
        filter_field=const.CMSSST_OS_FILTER_FIELD,
        filter_value=const.CMSSST_OS_FILTER_VALUE,
        exclude_field=const.CMSSST_OS_EXCLUDE_FIELD,
        exclude_rse_regex=const.EXCLUDE_RSE_REGEX,
        group_field=const.CMSSST_OS_GROUP_FIELD,
        sort_field=const.CMSSST_OS_SORT_FIELD,
        snapshot_ts_utc=query_ts,
    )
    if idx == 0:
        cmssst_source_docs = cmssst_docs_at_ts
    cmssst_tier_sums_by_ts[query_ts] = aggregate_cmssst_disk_by_tier(cmssst_docs_at_ts)

logger.info(
    "Prepared CMSSST tier sums for %s monthly snapshots (per-timestamp OpenSearch queries)",
    len(cmssst_tier_sums_by_ts),
)

if const.DRY_RUN:
    n_samples = 2
    generator = compute_usage_records(
        rucio_totals_by_ts=rucio_totals_by_ts,
        cmssst_tier_sums_by_ts=cmssst_tier_sums_by_ts,
    )
    doc_samples = list(itertools.islice(generator, n_samples))
    total_documents = n_samples + sum(1 for _ in generator)

    cmssst_samples = cmssst_source_docs[:n_samples]
    first_snapshot_ts = min(cmssst_tier_sums_by_ts.keys())
    cmssst_tier_sums_sample = dict(
        itertools.islice(cmssst_tier_sums_by_ts[first_snapshot_ts].items(), n_samples)
    )
    first_rucio_ts = min(rucio_totals_by_ts.keys())
    rucio_sample = dict(
        itertools.islice(rucio_totals_by_ts[first_rucio_ts].items(), n_samples)
    )
    logger.info(f"Document samples: {doc_samples}")
    logger.info(f"CMSSST source samples (not uploaded yet): {cmssst_samples}")
    logger.info(f"CMSSST tier disk sums sample: {cmssst_tier_sums_sample}")
    logger.info(f"Rucio locked/dynamic totals sample: {rucio_sample}")
    logger.info(f"CMSSST source docs fetched: {len(cmssst_source_docs)}")
    logger.info(
        "CMSSST tiers aggregated for first snapshot: %s",
        len(cmssst_tier_sums_by_ts[min(cmssst_tier_sums_by_ts.keys())]),
    )
    logger.info(
        "[DRY RUN CHECKLIST] Validation steps: "
        "1) compare one historical reporting month old-vs-new by (tier,rse_type), "
        "2) compare new totals against Grafana with matched filters and last-day UTC window, "
        "3) confirm dedup diagnostics report non-negative duplicate count, "
        "4) verify final grouped rows align with expected tier/rse_type cardinality."
    )
    logger.info(f"[DRY RUN] {total_documents} docs generated.")
else:
    logger.info(f"Uploading to OpenSearch: {const.OS_HOST},\n Index: {const.INDEX}")
    merged_doc_iterable = compute_usage_records(
        rucio_totals_by_ts=rucio_totals_by_ts,
        cmssst_tier_sums_by_ts=cmssst_tier_sums_by_ts,
    )
    successes, failures = os_upload_docs_in_bulk(
        os_host=const.OS_HOST,
        ca_cert_path=const.CERT_PATH,
        index=const.INDEX,
        doc_iterable=merged_doc_iterable,
    )
    logger.info(f"Uploaded {successes} docs.")
    if failures:
        if logger.getEffectiveLevel() == DEBUG:
            for failure in failures:
                logger.debug(f"Failure detail: {failure}")
        logger.error(f"{len(failures)} OpenSearch index failures occurred.")
