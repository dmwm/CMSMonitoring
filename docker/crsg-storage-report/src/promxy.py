import datetime
from bisect import bisect_right
from typing import Any, Iterable, Iterator, Optional, TypedDict

import src.constants as const
from src.logging import logger


class UsageRecord(TypedDict):
    timestamp: str
    snapshot_month: str
    rse_type: str
    tier: str
    disk_pledge: float
    disk_usable: float
    disk_experiment_use: float
    disk_local_use: float
    disk_reserved_free: float
    rucio_dynamic: float
    rucio_locked: float
    rucio_locally_managed: float
    rucio_available: float
    non_rucio_buffers: float


CMSSST_DISK_FIELDS = (
    "disk_pledge",
    "disk_usable",
    "disk_experiment_use",
    "disk_local_use",
    "disk_reserved_free",
)
CMSSST_TB_TO_BYTES = 1000 * 1000 * 1000 * 1000
MAX_WARNING_SAMPLES_PER_KIND = 5


def scale_rucio_sum_from_source(raw: float) -> float:
    """Convert Rucio index sum field to bytes (same unit as CMSSST tier aggregates)."""
    if const.RUCIO_USAGE_SUM_UNIT == "tb":
        return float(raw) * CMSSST_TB_TO_BYTES
    return float(raw)


def _safe_float(value: Any) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return 0.0


def aggregate_cmssst_disk_by_tier(
    cmssst_docs: Iterable[dict[str, Any]],
) -> dict[str, dict[str, float]]:
    tier_sums: dict[str, dict[str, float]] = {}

    for doc in cmssst_docs:
        raw_data = doc.get("data", {})
        if not isinstance(raw_data, dict):
            continue

        name = raw_data.get("name")
        if not isinstance(name, str) or len(name) < 2:
            continue
        tier = name[:2]

        if tier not in tier_sums:
            tier_sums[tier] = {field: 0.0 for field in CMSSST_DISK_FIELDS}

        
        if name == "T2_CH_CERN":
            t2_ch_cern_disk_fields = list(
                set(CMSSST_DISK_FIELDS) - {"disk_usable", "disk_experiment_use", "disk_local_use", "disk_reserved_free"}
            )
            if "T0" not in tier_sums:
                tier_sums["T0"] = {field: 0.0 for field in CMSSST_DISK_FIELDS}
            tier_sums["T0"]["disk_experiment_use"] += (
                _safe_float(raw_data.get("disk_experiment_use")) * CMSSST_TB_TO_BYTES
            )
            tier_sums["T0"]["disk_local_use"] += (
                _safe_float(raw_data.get("disk_local_use")) * CMSSST_TB_TO_BYTES
            )
            tier_sums["T0"]["disk_reserved_free"] += (
                _safe_float(raw_data.get("disk_reserved_free")) * CMSSST_TB_TO_BYTES
            )

            for field in t2_ch_cern_disk_fields:
                tier_sums[tier][field] += _safe_float(raw_data.get(field)) * CMSSST_TB_TO_BYTES

        else:
            for field in CMSSST_DISK_FIELDS:
                if name == "T0_CH_CERN":
                    if field == "disk_reserved_free":
                        continue
                tier_sums[tier][field] += _safe_float(raw_data.get(field)) * CMSSST_TB_TO_BYTES

    return tier_sums


def _safe_int_timestamp(value: Any) -> Optional[int]:
    try:
        return int(float(value))
    except (TypeError, ValueError):
        return None


def _safe_epoch_seconds(value: Any) -> Optional[int]:
    if isinstance(value, str):
        raw_value = value.strip()
        if not raw_value:
            return None
        if raw_value.endswith("Z"):
            raw_value = raw_value[:-1] + "+00:00"
        try:
            parsed_dt = datetime.datetime.fromisoformat(raw_value)
            if parsed_dt.tzinfo is None:
                parsed_dt = parsed_dt.replace(tzinfo=datetime.timezone.utc)
            return int(parsed_dt.timestamp())
        except ValueError:
            pass

    numeric_ts = _safe_int_timestamp(value)
    if numeric_ts is None:
        return None

    if numeric_ts > 1_000_000_000_000:
        return int(numeric_ts / 1000)
    return numeric_ts


def _get_nested_field(doc: dict[str, Any], dotted_field: str) -> Any:
    value: Any = doc
    for field in dotted_field.split("."):
        if not isinstance(value, dict):
            return None
        value = value.get(field)
    return value


def _timestamp_to_iso_utc(ts: int) -> str:
    return datetime.datetime.fromtimestamp(ts, datetime.timezone.utc).isoformat() + "Z"


def epoch_month_key(ts: int) -> tuple[int, int]:
    dt = datetime.datetime.fromtimestamp(ts, datetime.timezone.utc)
    return (dt.year, dt.month)


def snapshot_month_label(ts: int) -> str:
    y, m = epoch_month_key(ts)
    return f"{y}-{m:02d}"


def build_cmssst_history_by_group(
    cmssst_docs: Iterable[dict[str, Any]],
    *,
    group_field: str = "data.name",
    timestamp_field: str = "metadata.timestamp",
) -> dict[str, list[tuple[int, dict[str, Any]]]]:
    history_by_group: dict[str, list[tuple[int, dict[str, Any]]]] = {}
    invalid_timestamp_count = 0
    missing_group_count = 0

    for doc in cmssst_docs:
        group_value = _get_nested_field(doc=doc, dotted_field=group_field)
        if not isinstance(group_value, str) or not group_value.strip():
            missing_group_count += 1
            if missing_group_count <= MAX_WARNING_SAMPLES_PER_KIND:
                logger.warning(
                    "Skipping CMSSST doc with missing/invalid group field `%s`", group_field
                )
            continue

        timestamp_value = _get_nested_field(doc=doc, dotted_field=timestamp_field)
        parsed_ts = _safe_epoch_seconds(timestamp_value)
        if parsed_ts is None:
            invalid_timestamp_count += 1
            if invalid_timestamp_count <= MAX_WARNING_SAMPLES_PER_KIND:
                logger.warning(
                    "Skipping CMSSST doc with invalid timestamp in `%s`: %s",
                    timestamp_field,
                    timestamp_value,
                )
            continue

        history_by_group.setdefault(group_value, []).append((parsed_ts, doc))

    for group_timeline in history_by_group.values():
        group_timeline.sort(key=lambda entry: entry[0])

    if invalid_timestamp_count > MAX_WARNING_SAMPLES_PER_KIND:
        logger.warning(
            "Suppressed %s additional invalid CMSSST timestamp warnings.",
            invalid_timestamp_count - MAX_WARNING_SAMPLES_PER_KIND,
        )
    if missing_group_count > MAX_WARNING_SAMPLES_PER_KIND:
        logger.warning(
            "Suppressed %s additional missing CMSSST group warnings.",
            missing_group_count - MAX_WARNING_SAMPLES_PER_KIND,
        )

    return history_by_group


def resolve_cmssst_tier_sums_at_timestamp(
    cmssst_history: dict[str, list[tuple[int, dict[str, Any]]]],
    snapshot_ts_utc: int,
) -> dict[str, dict[str, float]]:
    tier_sums: dict[str, dict[str, float]] = {}

    for group_timeline in cmssst_history.values():
        if not group_timeline:
            continue

        timestamps = [entry[0] for entry in group_timeline]
        idx = bisect_right(timestamps, snapshot_ts_utc) - 1
        if idx < 0:
            continue

        selected_doc = group_timeline[idx][1]
        raw_data = selected_doc.get("data", {})
        if not isinstance(raw_data, dict):
            continue

        name = raw_data.get("name")
        if not isinstance(name, str) or len(name) < 2:
            continue
        tier = name[:2]

        if tier not in tier_sums:
            tier_sums[tier] = {field: 0.0 for field in CMSSST_DISK_FIELDS}

        for field in CMSSST_DISK_FIELDS:
            tier_sums[tier][field] += _safe_float(raw_data.get(field)) * CMSSST_TB_TO_BYTES

    return tier_sums


def compute_usage_records(
    rucio_totals_by_ts: dict[int, dict[tuple[str, str], dict[str, float]]],
    cmssst_tier_sums_by_ts: dict[int, dict[str, dict[str, float]]],
) -> Iterator[UsageRecord]:
    """
    Merge OpenSearch Rucio locked/dynamic totals (per snapshot epoch, tier, rse_type) with
    CMSSST tier disk sums. CMSSST values are bytes; Rucio totals are scaled via scale_rucio_sum_from_source.
    """
    for snapshot_ts in sorted(rucio_totals_by_ts.keys()):
        snap_month = snapshot_month_label(snapshot_ts)
        timestamp = _timestamp_to_iso_utc(snapshot_ts)
        cmssst_by_tier = cmssst_tier_sums_by_ts.get(snapshot_ts, {})

        for (tier, rse_type), totals in sorted(rucio_totals_by_ts[snapshot_ts].items()):
            if not tier or not rse_type:
                logger.warning(
                    "Skipping Rucio row with empty tier/rse_type at snapshot %s: tier=%r rse_type=%r",
                    timestamp,
                    tier,
                    rse_type,
                )
                continue
            locked_raw = totals.get("locked", 0.0)
            dynamic_raw = totals.get("dynamic", 0.0)
            rucio_locked = scale_rucio_sum_from_source(locked_raw)
            rucio_dynamic = scale_rucio_sum_from_source(dynamic_raw)

            cmssst_sums = cmssst_by_tier.get(tier, {})
            disk_pledge = _safe_float(cmssst_sums.get("disk_pledge"))
            disk_experiment_use = _safe_float(cmssst_sums.get("disk_experiment_use"))
            disk_reserved_free = _safe_float(cmssst_sums.get("disk_reserved_free"))
            disk_local_use = _safe_float(cmssst_sums.get("disk_local_use"))
            disk_usable = _safe_float(cmssst_sums.get("disk_usable"))

            rucio_available = (
                disk_experiment_use - disk_reserved_free - rucio_locked - rucio_dynamic
            )
            non_rucio_buffers = disk_usable - disk_experiment_use + disk_reserved_free
            if rucio_available < 0:
                non_rucio_buffers += rucio_available

            yield {
                "timestamp": timestamp,
                "snapshot_month": snap_month,
                "rse_type": rse_type,
                "tier": tier,
                "disk_pledge": disk_pledge,
                "disk_usable": disk_usable,
                "disk_experiment_use": disk_experiment_use,
                "disk_local_use": disk_local_use,
                "disk_reserved_free": disk_reserved_free,
                "rucio_dynamic": rucio_dynamic,
                "rucio_locked": rucio_locked,
                "rucio_locally_managed": disk_local_use,
                "rucio_available": rucio_available,
                "non_rucio_buffers": non_rucio_buffers,
            }
