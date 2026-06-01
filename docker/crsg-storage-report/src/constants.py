import os
import logging
import calendar
from datetime import datetime, timezone, timedelta
from typing import Optional


DEBUG_MODE = os.getenv("DEBUG_MODE", "false").lower() == "true"
LOG_LEVEL = logging.DEBUG if DEBUG_MODE else logging.INFO

DRY_RUN = os.getenv("DRY_RUN", "false").lower() == "true"

INDEX = os.environ.get("OPENSEARCH_INDEX", "rucio-used-space-1h")

OS_HOST = os.environ.get("OPENSEARCH_HOST", "https://os-cms.cern.ch:443/os")
CERT_PATH = os.environ.get("CERT_PATH", "/etc/pki/tls/certs/ca-bundle.trust.crt")

CMSSST_OS_HOST = os.environ.get(
    "CMSSST_OS_HOST", "https://monit-opensearch-lt.cern.ch:443/os"
)
CMSSST_OS_INDEX = os.environ.get("CMSSST_OS_INDEX", "monit_prod_cmssst*")
CMSSST_OS_FILTER_FIELD = os.environ.get(
    "CMSSST_OS_FILTER_FIELD", "metadata.monit_hdfs_path"
)
CMSSST_OS_FILTER_VALUE = os.environ.get("CMSSST_OS_FILTER_VALUE", "scap15min")
CMSSST_OS_GROUP_FIELD = os.environ.get("CMSSST_OS_GROUP_FIELD", "data.name")
CMSSST_OS_SORT_FIELD = os.environ.get("CMSSST_OS_SORT_FIELD", "metadata.timestamp")
EXCLUDE_RSE_REGEX = os.environ.get(
    "EXCLUDE_RSE_REGEX", ".*(Test|test|_Temp|_temp)"
)
CMSSST_OS_EXCLUDE_FIELD = os.environ.get("CMSSST_OS_EXCLUDE_FIELD", "data.name")

QUERY_DATE = os.environ.get("QUERY_DATE", None)
QUERY_DATE_FORMAT = "%Y-%m-%d"

BACKFILL_ENABLED = os.getenv("BACKFILL_ENABLED", "false").lower() == "true"
BACKFILL_START_DATE = os.environ.get("BACKFILL_START_DATE", None)
BACKFILL_END_DATE = os.environ.get("BACKFILL_END_DATE", None)

def _resolve_query_time(query_date: Optional[str]) -> datetime:
    if not query_date:
        return datetime.now(timezone.utc)

    # For snapshot queries, QUERY_DATE is interpreted as UTC start-of-day.
    return datetime.strptime(query_date, QUERY_DATE_FORMAT).replace(tzinfo=timezone.utc)


QUERY_TIME = _resolve_query_time(QUERY_DATE)


def _resolve_utc_date(date_str: Optional[str], env_name: str) -> Optional[datetime]:
    if not date_str:
        return None
    try:
        return datetime.strptime(date_str, QUERY_DATE_FORMAT).replace(tzinfo=timezone.utc)
    except ValueError as exc:
        raise ValueError(
            f"{env_name} must use format {QUERY_DATE_FORMAT}, got '{date_str}'"
        ) from exc


def _resolve_positive_int_env(env_name: str, default: int) -> int:
    raw_value = os.environ.get(env_name, str(default))
    try:
        parsed_value = int(raw_value)
    except ValueError as exc:
        raise ValueError(f"{env_name} must be a positive integer, got '{raw_value}'") from exc

    if parsed_value < 0:
        raise ValueError(f"{env_name} must be >= 0, got '{raw_value}'")
    return parsed_value


# Daily Rucio storage usage (OpenSearch on RUCIO_USAGE_OS_HOST). Required for the job to run.
RUCIO_USAGE_OS_HOST = os.environ.get(
    "RUCIO_USAGE_OS_HOST", "https://monit-opensearch.cern.ch:443/os"
)
RUCIO_USAGE_OS_INDEX = os.environ.get("RUCIO_USAGE_OS_INDEX", "monit_prod_cms_rucio_raw_daily_stats*")
RUCIO_USAGE_SUM_FIELD = os.environ.get("RUCIO_USAGE_SUM_FIELD", "data.dbs_size")
RUCIO_USAGE_TIME_FIELD = os.environ.get("RUCIO_USAGE_TIME_FIELD", "metadata.timestamp")
RUCIO_USAGE_RSE_TIER_FIELD = os.environ.get("RUCIO_USAGE_RSE_TIER_FIELD", "data.rse_tier")
RUCIO_USAGE_RSE_TYPE_FIELD = os.environ.get("RUCIO_USAGE_RSE_TYPE_FIELD", "data.rse_type")
RUCIO_USAGE_RSE_IDENTITY_FIELD = os.environ.get("RUCIO_USAGE_RSE_IDENTITY_FIELD", "data.rse")
RUCIO_USAGE_LOCK_FIELD = os.environ.get("RUCIO_USAGE_LOCK_FIELD", "data.dbs_is_d_locked")
RUCIO_USAGE_LOCK_VALUE_LOCKED = os.environ.get("RUCIO_USAGE_LOCK_VALUE_LOCKED", "locked")
RUCIO_USAGE_LOCK_VALUE_DYNAMIC = os.environ.get("RUCIO_USAGE_LOCK_VALUE_DYNAMIC", "dynamic")
RUCIO_UPLOAD_DAY_OF_MONTH = _resolve_positive_int_env("RUCIO_UPLOAD_DAY_OF_MONTH", 3)
if not 1 <= RUCIO_UPLOAD_DAY_OF_MONTH <= 31:
    raise ValueError(
        f"RUCIO_UPLOAD_DAY_OF_MONTH must be between 1 and 31, got '{RUCIO_UPLOAD_DAY_OF_MONTH}'"
    )
RUCIO_REPORTING_MONTH_OFFSET = _resolve_positive_int_env("RUCIO_REPORTING_MONTH_OFFSET", 1)
RUCIO_USAGE_COMPOSITE_BATCH_SIZE = _resolve_positive_int_env(
    "RUCIO_USAGE_COMPOSITE_BATCH_SIZE", 500
)
_raw_sum_unit = os.environ.get("RUCIO_USAGE_SUM_UNIT", "bytes").lower()
if _raw_sum_unit not in ("bytes", "tb"):
    raise ValueError(
        "RUCIO_USAGE_SUM_UNIT must be 'bytes' or 'tb', "
        f"got '{os.environ.get('RUCIO_USAGE_SUM_UNIT')}'"
    )
RUCIO_USAGE_SUM_UNIT = _raw_sum_unit


def require_rucio_usage_config() -> None:
    if not RUCIO_USAGE_OS_INDEX:
        raise ValueError(
            "RUCIO_USAGE_OS_INDEX must be set to the OpenSearch index pattern for "
            "daily Rucio storage usage (queried on RUCIO_USAGE_OS_HOST)."
        )


def subtract_months(year: int, month: int, months_to_subtract: int) -> tuple[int, int]:
    """Return calendar (year, month) after subtracting months_to_subtract from (year, month)."""
    m = month - months_to_subtract
    y = year
    while m <= 0:
        m += 12
        y -= 1
    return (y, m)


def utc_next_month_start(year: int, month: int) -> datetime:
    if month == 12:
        return datetime(year + 1, 1, 1, tzinfo=timezone.utc)
    return datetime(year, month + 1, 1, tzinfo=timezone.utc)


def month_epoch_bounds(year: int, month: int) -> tuple[int, int]:
    """
    Inclusive epoch-second window for a UTC calendar month (year, month).
    """
    month_start = datetime(year, month, 1, tzinfo=timezone.utc)
    month_end = utc_next_month_start(year, month) - timedelta(seconds=1)
    return (int(month_start.timestamp()), int(month_end.timestamp()))


def utc_calendar_day_epoch_bounds(year: int, month: int, day: int) -> tuple[int, int]:
    """
    Inclusive epoch-second window for one UTC calendar day (year, month, day).
    """
    last_dom = calendar.monthrange(year, month)[1]
    if not 1 <= day <= last_dom:
        raise ValueError(
            f"day must be 1..{last_dom} for {year}-{month:02d}, got {day}"
        )
    day_start = datetime(year, month, day, tzinfo=timezone.utc)
    day_end = day_start + timedelta(days=1) - timedelta(seconds=1)
    return (int(day_start.timestamp()), int(day_end.timestamp()))


def last_utc_day_epoch_bounds(year: int, month: int) -> tuple[int, int]:
    """
    Inclusive epoch-second window for the last UTC calendar day of (year, month).
    """
    last_day = calendar.monthrange(year, month)[1]
    return utc_calendar_day_epoch_bounds(year, month, last_day)


def query_time_for_month_end(year: int, month: int) -> datetime:
    """Last second of UTC calendar month (snapshot instant aligned with CMSSST queries)."""
    return utc_next_month_start(year, month) - timedelta(seconds=1)


def iter_months_inclusive(start: tuple[int, int], end: tuple[int, int]):
    """Yield (year, month) from start through end inclusive; both must be valid calendar months."""
    y, m = start
    end_y, end_m = end
    while (y, m) <= (end_y, end_m):
        yield (y, m)
        m += 1
        if m > 12:
            m = 1
            y += 1


def _month_tuple_from_datetime(dt: datetime) -> tuple[int, int]:
    return (dt.year, dt.month)


def resolve_reporting_month_for_run(ref: datetime) -> tuple[int, int]:
    """
    Resolve reporting month from run timestamp:
    - data uploaded on RUCIO_UPLOAD_DAY_OF_MONTH in month M
    - output reporting month defaults to M-1 (offset configurable)
    - runs before upload day use the previous anchor month
    """
    anchor_year, anchor_month = ref.year, ref.month
    if ref.day < RUCIO_UPLOAD_DAY_OF_MONTH:
        anchor_year, anchor_month = subtract_months(anchor_year, anchor_month, 1)
    return subtract_months(
        anchor_year,
        anchor_month,
        RUCIO_REPORTING_MONTH_OFFSET,
    )


def _backfill_month_range() -> list[tuple[int, int]]:
    now_utc = datetime.now(timezone.utc)
    start_dt = _resolve_utc_date(BACKFILL_START_DATE, "BACKFILL_START_DATE")
    end_dt = _resolve_utc_date(BACKFILL_END_DATE, "BACKFILL_END_DATE")

    if start_dt is None:
        start_dt = (now_utc - timedelta(days=365)).replace(
            hour=0, minute=0, second=0, microsecond=0
        )
    if end_dt is None:
        end_dt = now_utc

    if start_dt > end_dt:
        raise ValueError(
            f"BACKFILL_START_DATE ({start_dt.date()}) cannot be later than "
            f"BACKFILL_END_DATE ({end_dt.date()})"
        )

    start_month = _month_tuple_from_datetime(start_dt)
    end_month = _month_tuple_from_datetime(end_dt)
    return list(iter_months_inclusive(start_month, end_month))


def build_monthly_query_times() -> list[datetime]:
    """
    One Promxy instant query time per UTC month: last second of each month.
    Regular run: one deterministic reporting month from run date and upload schedule.
    Backfill: all months from BACKFILL_START_DATE through BACKFILL_END_DATE (inclusive).
    """
    if BACKFILL_ENABLED:
        month_tuples = _backfill_month_range()
    else:
        month_tuples = [resolve_reporting_month_for_run(QUERY_TIME)]

    if not month_tuples:
        raise ValueError("No months to process.")

    return [query_time_for_month_end(y, m) for y, m in month_tuples]


# Backwards-compatible name for callers that still reference build_backfill_query_times.
build_backfill_query_times = build_monthly_query_times
