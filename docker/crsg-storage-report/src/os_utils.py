import hashlib
import json
import logging
import time
from datetime import datetime, timezone
from typing import Any, Iterable, Optional
from opensearchpy import OpenSearch, helpers, RequestsHttpConnection
from opensearchpy.exceptions import ConnectionTimeout, TransportError
from requests_gssapi import HTTPSPNEGOAuth, OPTIONAL
import src.constants as const
from src.logging import logger

RUCIO_T2_CH_CERN_SITE = "T2_CH_CERN"
RUCIO_T2_TIER = "T2"
RUCIO_T0_TIER = "T0"


def get_opensearch_client(os_host: str, ca_cert_path: str) -> OpenSearch:
    return OpenSearch(
        [os_host],
        use_ssl=True,
        verify_certs=True,
        ca_certs=ca_cert_path,
        connection_class=RequestsHttpConnection,
        http_auth=HTTPSPNEGOAuth(mutual_authentication=OPTIONAL),
    )


def _deterministic_doc_id(doc: dict[str, Any]) -> str:
    """Stable id per monthly snapshot row (idempotent re-runs overwrite same month)."""
    key = "|".join(
        [
            str(doc.get("snapshot_month", "")),
            str(doc.get("tier", "")),
            str(doc.get("rse_type", "")),
        ]
    )
    return hashlib.sha256(key.encode("utf-8")).hexdigest()


def _wrap_records_for_bulk(
    index: str, doc_iterable: Iterable[dict[str, Any]]
) -> Iterable[dict[str, Any]]:
    for doc in doc_iterable:
        yield {
            "_op_type": "index",
            "_index": index,
            "_id": _deterministic_doc_id(doc),
            "_source": doc,
        }


def os_upload_docs_in_bulk(
    os_host: str, ca_cert_path: str, index: str, doc_iterable: Iterable[dict[str, Any]]
):
    os_client = get_opensearch_client(os_host=os_host, ca_cert_path=ca_cert_path)
    bulk_actions = _wrap_records_for_bulk(index=index, doc_iterable=doc_iterable)

    success, failures = helpers.bulk(os_client, bulk_actions, max_retries=3)

    return success, failures


def _search_with_retries(
    os_client: OpenSearch,
    index: str,
    query: dict[str, Any],
    request_timeout_seconds: int = 120,
    max_attempts: int = 3,
) -> dict[str, Any]:
    for attempt in range(1, max_attempts + 1):
        try:
            return os_client.search(
                index=index,
                body=query,
                request_timeout=request_timeout_seconds,
            )
        except (ConnectionTimeout, TimeoutError):
            if attempt == max_attempts:
                raise
            time.sleep(attempt)
        except TransportError as exc:
            # Retry timeout-like transport errors, fail fast otherwise.
            status_code = getattr(exc, "status_code", None)
            if status_code in (408, 429, 502, 503, 504) and attempt < max_attempts:
                time.sleep(attempt)
                continue
            raise


def _safe_float(value: Any) -> float:
    try:
        return float(value)
    except (TypeError, ValueError):
        return 0.0


def _safe_epoch_seconds(value: Any) -> Optional[int]:
    if isinstance(value, (int, float)):
        numeric_ts = int(float(value))
        if numeric_ts > 1_000_000_000_000:
            return int(numeric_ts / 1000)
        return numeric_ts

    if isinstance(value, str):
        raw_value = value.strip()
        if not raw_value:
            return None
        if raw_value.endswith("Z"):
            raw_value = raw_value[:-1] + "+00:00"
        try:
            parsed_dt = datetime.fromisoformat(raw_value)
            if parsed_dt.tzinfo is None:
                parsed_dt = parsed_dt.replace(tzinfo=timezone.utc)
            return int(parsed_dt.timestamp())
        except ValueError:
            return None
    return None


def _get_nested_field(doc: dict[str, Any], dotted_field: str) -> Any:
    value: Any = doc
    for field in dotted_field.split("."):
        if not isinstance(value, dict):
            return None
        value = value.get(field)
    return value


def fetch_latest_docs_per_group(
    os_host: str,
    ca_cert_path: str,
    index: str,
    filter_field: str,
    filter_value: str,
    group_field: str,
    sort_field: str,
    exclude_field: str = const.CMSSST_OS_EXCLUDE_FIELD,
    exclude_rse_regex: str = const.EXCLUDE_RSE_REGEX,
    batch_size: int = 1000,
) -> list[dict[str, Any]]:
    os_client = get_opensearch_client(os_host=os_host, ca_cert_path=ca_cert_path)
    latest_docs: list[dict[str, Any]] = []
    after_key = None

    while True:
        by_group_agg: dict[str, Any] = {
            "composite": {
                "size": batch_size,
                "sources": [{"group_key": {"terms": {"field": group_field}}}],
            },
            "aggs": {
                "latest_doc": {
                    "top_hits": {
                        "size": 1,
                        "sort": [{sort_field: {"order": "desc", "unmapped_type": "date"}}],
                    }
                }
            },
        }
        if after_key:
            by_group_agg["composite"]["after"] = after_key

        bool_query: dict[str, Any] = {
            "filter": [{"query_string": {"query": f'{filter_field}:"{filter_value}"'}}]
        }
        if exclude_rse_regex:
            bool_query["must_not"] = [
                {"query_string": {"query": f"{exclude_field}:/{exclude_rse_regex}/"}}
            ]

        query = {
            "size": 0,
            "query": {"bool": bool_query},
            "aggs": {"by_group": by_group_agg},
        }

        resp = _search_with_retries(os_client=os_client, index=index, query=query)
        buckets = resp["aggregations"]["by_group"]["buckets"]

        for bucket in buckets:
            top_hit = bucket["latest_doc"]["hits"]["hits"]
            if top_hit:
                latest_docs.append(top_hit[0].get("_source", {}))

        after_key = resp["aggregations"]["by_group"].get("after_key")
        if not buckets or not after_key:
            break

    return latest_docs


def fetch_rucio_locked_dynamic_totals_composite(
    os_host: str,
    ca_cert_path: str,
    index: str,
    *,
    time_field: str,
    sum_field: str,
    tier_field: str,
    rse_type_field: str,
    lock_field: str,
    lock_value_locked: str,
    lock_value_dynamic: str,
    rse_identity_field: str,
    gte_epoch_sec: int,
    lte_epoch_sec: int,
    batch_size: int,
) -> dict[tuple[str, str], dict[str, float]]:
    """
    Server-side aggregation: group by (tier, rse_type, rse_identity, lock_state), sum all docs per
    key, then aggregate usage into (tier, rse_type) locked/dynamic totals.
    """
    os_client = get_opensearch_client(os_host=os_host, ca_cert_path=ca_cert_path)
    results: dict[tuple[str, str], dict[str, float]] = {}
    total_buckets = 0
    total_docs_summed = 0
    missing_identity_buckets = 0
    remapped_t2_ch_cern_buckets = 0

    def _search_total(query: dict[str, Any]) -> str:
        """
        Return a compact hits.total representation for diagnostics.
        """
        try:
            resp = _search_with_retries(os_client=os_client, index=index, query=query)
        except Exception as exc:  # pragma: no cover - diagnostics only
            return f"error:{type(exc).__name__}:{exc}"

        total = (resp.get("hits") or {}).get("total")
        if isinstance(total, dict):
            value = total.get("value")
            relation = total.get("relation")
            return f"{value} ({relation})"
        return str(total)


    def _accumulate_lock_state(lock_value: str, result_key: str) -> None:
        nonlocal total_buckets, total_docs_summed, missing_identity_buckets, remapped_t2_ch_cern_buckets
        after_key: Optional[dict[str, Any]] = None
        page = 0

        while True:
            composite_agg: dict[str, Any] = {
                "size": batch_size,
                "sources": [
                    {"tier_key": {"terms": {"field": tier_field}}},
                    {"rtype_key": {"terms": {"field": rse_type_field}}},
                    {
                        "rse_key": {
                            "terms": {
                                "field": rse_identity_field,
                                "missing_bucket": True,
                            }
                        }
                    },
                ],
            }
            if after_key:
                composite_agg["after"] = after_key

            query: dict[str, Any] = {
                "size": 0,
                "query": {
                    "bool": {
                        "filter": [
                            {
                                "range": {
                                    time_field: {
                                        "gte": gte_epoch_sec,
                                        "lte": lte_epoch_sec,
                                        "format": "epoch_second",
                                    }
                                }
                            },
                            {"term": {lock_field: lock_value}},
                            # Mirror Grafana filter semantics using query_string syntax.
                            # Note: "all" values in the compared dashboard are:
                            # - access/create: _exists_:data.dataset
                            # - other dimensions: *
                            {
                                "query_string": {
                                    "query": (
                                        "_exists_:data.dataset "
                                        "AND data.is_dataset_valid:1 "
                                        "AND data.dbs_has_ds_name:True"
                                    )
                                }
                            },
                        ]
                    }
                },
                "aggs": {
                    "by_entity": {
                        "composite": composite_agg,
                        "aggs": {"sum_usage": {"sum": {"field": sum_field}}},
                    }
                },
            }

            logger.info(
                (
                    "Rucio query request [%s]: index=%s time_field=%s "
                    "window_epoch=[%s,%s] lock_field=%s lock_value=%s "
                    "sum_field=%s tier_field=%s rse_type_field=%s rse_identity_field=%s "
                    "page=%s after_key=%s"
                ),
                result_key,
                index,
                time_field,
                gte_epoch_sec,
                lte_epoch_sec,
                lock_field,
                lock_value,
                sum_field,
                tier_field,
                rse_type_field,
                rse_identity_field,
                page + 1,
                after_key,
            )
            if logger.isEnabledFor(logging.DEBUG):
                logger.debug(
                    "Rucio query body [%s]: %s",
                    result_key,
                    json.dumps(query, sort_keys=True),
                )

            resp = _search_with_retries(os_client=os_client, index=index, query=query)
            agg = resp.get("aggregations", {}).get("by_entity", {})
            buckets = agg.get("buckets", [])
            logger.info(
                "Rucio query response [%s]: buckets_in_page=%s has_after_key=%s",
                result_key,
                len(buckets),
                bool(agg.get("after_key")),
            )

            page += 1
            total_buckets += len(buckets)
            logger.info(
                "Rucio dedup progress [%s]: page=%s buckets_in_page=%s total_buckets=%s",
                result_key,
                page,
                len(buckets),
                total_buckets,
            )

            for bucket in buckets:
                key = bucket.get("key") or {}
                tier = str(key.get("tier_key", ""))
                rse_type = str(key.get("rtype_key", ""))
                rse_identity = key.get("rse_key")

                if rse_identity in (None, ""):
                    missing_identity_buckets += 1

                usage_value = _safe_float((bucket.get("sum_usage") or {}).get("value"))
                if (tier, rse_type) not in results:
                    results[(tier, rse_type)] = {"locked": 0.0, "dynamic": 0.0}
                results[(tier, rse_type)][result_key] += usage_value

                if (
                    tier == RUCIO_T2_TIER
                    and rse_identity == RUCIO_T2_CH_CERN_SITE
                ):
                    if (RUCIO_T0_TIER, rse_type) not in results:
                        results[(RUCIO_T0_TIER, rse_type)] = {"locked": 0.0, "dynamic": 0.0}
                    results[(tier, rse_type)][result_key] -= usage_value
                    results[(RUCIO_T0_TIER, rse_type)][result_key] += usage_value
                    remapped_t2_ch_cern_buckets += 1

                total_docs_summed += int(bucket.get("doc_count", 0))

            after_key = agg.get("after_key")
            if not after_key:
                break

    _accumulate_lock_state(lock_value_locked, "locked")
    _accumulate_lock_state(lock_value_dynamic, "dynamic")

    if missing_identity_buckets > 0:
        logger.warning(
            "Rucio docs missing `%s`: %s aggregation buckets used missing identity",
            rse_identity_field,
            missing_identity_buckets,
        )
    logger.info(
        "Rucio aggregation diagnostics: unique_entities=%s docs_summed=%s grouped_rows=%s",
        total_buckets,
        total_docs_summed,
        len(results),
    )
    if remapped_t2_ch_cern_buckets > 0:
        logger.info(
            (
                "Rucio T2_CH_CERN remap applied: moved %s bucket(s) from tier `%s` "
                "to tier `%s` for locked/dynamic totals"
            ),
            remapped_t2_ch_cern_buckets,
            RUCIO_T2_TIER,
            RUCIO_T0_TIER,
        )

    return results


def fetch_docs_in_time_range(
    os_host: str,
    ca_cert_path: str,
    index: str,
    filter_field: str,
    filter_value: str,
    sort_field: str,
    start_ts_utc: int,
    end_ts_utc: int,
    exclude_field: str = const.CMSSST_OS_EXCLUDE_FIELD,
    exclude_rse_regex: str = const.EXCLUDE_RSE_REGEX,
    batch_size: int = 1000,
) -> list[dict[str, Any]]:
    os_client = get_opensearch_client(os_host=os_host, ca_cert_path=ca_cert_path)
    docs: list[dict[str, Any]] = []
    search_after: Optional[list[Any]] = None

    while True:
        bool_query: dict[str, Any] = {
            "filter": [
                {"query_string": {"query": f'{filter_field}:"{filter_value}"'}},
                {
                    "range": {
                        sort_field: {
                            "gte": start_ts_utc,
                            "lte": end_ts_utc,
                            "format": "epoch_second",
                        }
                    }
                },
            ]
        }
        if exclude_rse_regex:
            bool_query["must_not"] = [
                {"query_string": {"query": f"{exclude_field}:/{exclude_rse_regex}/"}}
            ]

        query: dict[str, Any] = {
            "size": batch_size,
            "_source": [
                "data.name",
                "data.disk_pledge",
                "data.disk_usable",
                "data.disk_experiment_use",
                "data.disk_local_use",
                "data.disk_reserved_free",
                sort_field,
            ],
            "query": {"bool": bool_query},
            "sort": [
                {sort_field: {"order": "asc", "unmapped_type": "date"}},
                {"_id": {"order": "asc"}},
            ],
        }
        if search_after:
            query["search_after"] = search_after

        resp = _search_with_retries(os_client=os_client, index=index, query=query)
        hits = resp.get("hits", {}).get("hits", [])
        if not hits:
            break

        for hit in hits:
            source = hit.get("_source", {})
            if isinstance(source, dict):
                docs.append(source)

        search_after = hits[-1].get("sort")
        if not search_after:
            break

    return docs


def fetch_latest_docs_per_group_before_timestamp(
    os_host: str,
    ca_cert_path: str,
    index: str,
    filter_field: str,
    filter_value: str,
    group_field: str,
    sort_field: str,
    snapshot_ts_utc: int,
    exclude_field: str = const.CMSSST_OS_EXCLUDE_FIELD,
    exclude_rse_regex: str = const.EXCLUDE_RSE_REGEX,
    batch_size: int = 1000,
) -> list[dict[str, Any]]:
    os_client = get_opensearch_client(os_host=os_host, ca_cert_path=ca_cert_path)
    latest_docs: list[dict[str, Any]] = []
    after_key = None

    while True:
        by_group_agg: dict[str, Any] = {
            "composite": {
                "size": batch_size,
                "sources": [{"group_key": {"terms": {"field": group_field}}}],
            },
            "aggs": {
                "latest_doc": {
                    "top_hits": {
                        "size": 1,
                        "sort": [{sort_field: {"order": "desc", "unmapped_type": "date"}}],
                        "_source": [
                            "data.name",
                            "data.disk_pledge",
                            "data.disk_usable",
                            "data.disk_experiment_use",
                            "data.disk_local_use",
                            "data.disk_reserved_free",
                            sort_field,
                        ],
                    }
                }
            },
        }
        if after_key:
            by_group_agg["composite"]["after"] = after_key

        bool_query: dict[str, Any] = {
            "filter": [
                {"query_string": {"query": f'{filter_field}:"{filter_value}"'}},
                {
                    "range": {
                        sort_field: {
                            "lte": snapshot_ts_utc,
                            "format": "epoch_second",
                        }
                    }
                },
            ]
        }
        if exclude_rse_regex:
            bool_query["must_not"] = [
                {"query_string": {"query": f"{exclude_field}:/{exclude_rse_regex}/"}}
            ]

        query = {
            "size": 0,
            "query": {"bool": bool_query},
            "aggs": {"by_group": by_group_agg},
        }

        resp = _search_with_retries(os_client=os_client, index=index, query=query)
        buckets = resp["aggregations"]["by_group"]["buckets"]

        for bucket in buckets:
            top_hit = bucket["latest_doc"]["hits"]["hits"]
            if top_hit:
                latest_docs.append(top_hit[0].get("_source", {}))

        after_key = resp["aggregations"]["by_group"].get("after_key")
        if not buckets or not after_key:
            break

    return latest_docs
