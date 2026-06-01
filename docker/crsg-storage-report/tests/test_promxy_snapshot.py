from __future__ import annotations

import importlib
import pytest
import sys
from pathlib import Path


PROMXY_COMPONENT_ROOT = Path(__file__).resolve().parents[1]
if str(PROMXY_COMPONENT_ROOT) not in sys.path:
    sys.path.insert(0, str(PROMXY_COMPONENT_ROOT))

import src.constants as constants  # noqa: E402
from src.promxy import (  # noqa: E402
    CMSSST_TB_TO_BYTES,
    compute_usage_records,
    build_cmssst_history_by_group,
    resolve_cmssst_tier_sums_at_timestamp,
)
from src.os_utils import (  # noqa: E402
    os_upload_docs_in_bulk,
    fetch_docs_in_time_range,
    fetch_rucio_locked_dynamic_totals_composite,
    _deterministic_doc_id,
)


def test_query_date_maps_to_utc_start_of_day(monkeypatch):
    monkeypatch.setenv("QUERY_DATE", "2026-02-11")
    monkeypatch.setenv("RUCIO_USAGE_OS_INDEX", "test-rucio-*")
    reloaded_constants = importlib.reload(constants)

    assert reloaded_constants.QUERY_TIME.year == 2026
    assert reloaded_constants.QUERY_TIME.month == 2
    assert reloaded_constants.QUERY_TIME.day == 11
    assert reloaded_constants.QUERY_TIME.hour == 0
    assert reloaded_constants.QUERY_TIME.minute == 0
    assert reloaded_constants.QUERY_TIME.second == 0
    assert reloaded_constants.QUERY_TIME.utcoffset().total_seconds() == 0


def test_compute_usage_records_merges_rucio_and_cmssst():
    ts = 1700000000
    rucio_totals_by_ts = {
        ts: {
            ("T1", "DISK"): {"locked": 85.0, "dynamic": 20.0},
        }
    }
    cmssst_tier_sums_by_ts = {
        ts: {
            "T1": {
                "disk_usable": 200.0,
                "disk_experiment_use": 180.0,
                "disk_local_use": 40.0,
                "disk_reserved_free": 10.0,
                "disk_pledge": 0.0,
            }
        }
    }

    docs = list(
        compute_usage_records(
            rucio_totals_by_ts=rucio_totals_by_ts,
            cmssst_tier_sums_by_ts=cmssst_tier_sums_by_ts,
        )
    )

    assert len(docs) == 1
    doc = docs[0]
    assert doc["snapshot_month"] == "2023-11"
    assert doc["tier"] == "T1"
    assert doc["rse_type"] == "DISK"
    assert doc["rucio_locked"] == 85.0
    assert doc["rucio_dynamic"] == 20.0
    assert doc["rucio_locally_managed"] == 40.0
    assert doc["rucio_available"] == 65.0
    assert doc["non_rucio_buffers"] == 95.0


def test_compute_usage_records_separate_rows_per_snapshot_month():
    ts_nov = 1700000000
    ts_jan = 1704067200
    rucio_totals_by_ts = {
        ts_nov: {("T1", "DISK"): {"locked": 70.0, "dynamic": 10.0}},
        ts_jan: {("T1", "DISK"): {"locked": 60.0, "dynamic": 20.0}},
    }
    cmssst_tier_sums_by_ts = {
        ts_nov: {
            "T1": {
                "disk_usable": 160.0,
                "disk_experiment_use": 130.0,
                "disk_local_use": 11.0,
                "disk_reserved_free": 10.0,
                "disk_pledge": 0.0,
            }
        },
        ts_jan: {
            "T1": {
                "disk_usable": 160.0,
                "disk_experiment_use": 130.0,
                "disk_local_use": 11.0,
                "disk_reserved_free": 10.0,
                "disk_pledge": 0.0,
            }
        },
    }

    docs = list(
        compute_usage_records(
            rucio_totals_by_ts=rucio_totals_by_ts,
            cmssst_tier_sums_by_ts=cmssst_tier_sums_by_ts,
        )
    )

    assert len(docs) == 2
    by_month = {doc["snapshot_month"]: doc for doc in docs}
    assert by_month["2023-11"]["rucio_locked"] == 70.0
    assert by_month["2024-01"]["rucio_locked"] == 60.0


def test_os_upload_docs_in_bulk_wraps_plain_records(monkeypatch):
    captured = {}

    class DummyHelpers:
        @staticmethod
        def bulk(client, actions, max_retries):
            captured["client"] = client
            captured["actions"] = list(actions)
            captured["max_retries"] = max_retries
            return 2, []

    import src.os_utils as os_utils_module

    monkeypatch.setattr(os_utils_module, "helpers", DummyHelpers)
    monkeypatch.setattr(
        os_utils_module,
        "get_opensearch_client",
        lambda os_host, ca_cert_path: {"host": os_host, "cert": ca_cert_path},
    )

    docs = [
        {
            "timestamp": "2026-02-11T00:00:00+00:00Z",
            "snapshot_month": "2026-02",
            "tier": "T1",
            "rse_type": "DISK",
            "rucio_locked": 1.0,
            "rucio_dynamic": 0.5,
        },
        {
            "timestamp": "2026-02-12T00:00:00+00:00Z",
            "snapshot_month": "2026-02",
            "tier": "T2",
            "rse_type": "DISK",
            "rucio_locked": 2.0,
            "rucio_dynamic": 1.0,
        },
    ]
    success, failures = os_upload_docs_in_bulk(
        os_host="https://os.example",
        ca_cert_path="/tmp/ca.pem",
        index="test-index",
        doc_iterable=docs,
    )

    assert success == 2
    assert failures == []
    assert captured["client"] == {"host": "https://os.example", "cert": "/tmp/ca.pem"}
    assert captured["max_retries"] == 3
    assert captured["actions"] == [
        {
            "_op_type": "index",
            "_index": "test-index",
            "_id": _deterministic_doc_id(docs[0]),
            "_source": docs[0],
        },
        {
            "_op_type": "index",
            "_index": "test-index",
            "_id": _deterministic_doc_id(docs[1]),
            "_source": docs[1],
        },
    ]


def test_fetch_docs_in_time_range_uses_search_after_pagination(monkeypatch):
    captured_queries = []

    class DummyClient:
        def __init__(self):
            self.call_count = 0

        def search(self, index, body, request_timeout=None):
            captured_queries.append((index, body))
            self.call_count += 1
            if self.call_count == 1:
                return {
                    "hits": {
                        "hits": [
                            {
                                "_source": {
                                    "data": {
                                        "name": "T1_US_Test",
                                        "disk_usable": 10.0,
                                    },
                                    "metadata": {"timestamp": "2026-01-01T00:00:00Z"},
                                },
                                "sort": [1704067200, "doc-1"],
                            }
                        ]
                    }
                }
            return {"hits": {"hits": []}}

    import src.os_utils as os_utils_module

    monkeypatch.setattr(
        os_utils_module,
        "get_opensearch_client",
        lambda os_host, ca_cert_path: DummyClient(),
    )

    docs = fetch_docs_in_time_range(
        os_host="https://os.example",
        ca_cert_path="/tmp/ca.pem",
        index="cmssst-index",
        filter_field="metadata.monit_hdfs_path",
        filter_value="scap15min",
        sort_field="metadata.timestamp",
        start_ts_utc=1704067200,
        end_ts_utc=1704153600,
    )

    assert len(docs) == 1
    assert docs[0]["data"]["name"] == "T1_US_Test"
    assert len(captured_queries) == 2
    assert captured_queries[0][1]["query"]["bool"]["filter"][1]["range"]["metadata.timestamp"] == {
        "gte": 1704067200,
        "lte": 1704153600,
        "format": "epoch_second",
    }
    assert captured_queries[0][1]["query"]["bool"]["must_not"] == [
        {"query_string": {"query": "data.name:/.*(Test|test|_Temp|_temp)/"}}
    ]
    assert captured_queries[1][1]["search_after"] == [1704067200, "doc-1"]


def test_fetch_rucio_locked_dynamic_totals_composite_paginates(monkeypatch):
    captured: list[dict] = []
    pages = [
        {
            "aggregations": {
                "by_entity": {
                    "buckets": [
                        {
                            "key": {
                                "tier_key": "T1",
                                "rtype_key": "DISK",
                                "rse_key": "T1_RSE",
                            },
                            "doc_count": 2,
                            "sum_usage": {"value": 20.0},
                        }
                    ],
                    "after_key": {
                        "tier_key": "T1",
                        "rtype_key": "DISK",
                        "rse_key": "T1_RSE",
                    },
                }
            }
        },
        {
            "aggregations": {
                "by_entity": {
                    "buckets": [
                        {
                            "key": {
                                "tier_key": "T2",
                                "rtype_key": "TAPE",
                                "rse_key": "T2_RSE",
                            },
                            "doc_count": 1,
                            "sum_usage": {"value": 1.0},
                        },
                    ]
                }
            }
        },
        {
            "aggregations": {
                "by_entity": {
                    "buckets": []
                }
            }
        },
    ]
    page_iter = iter(pages)

    import src.os_utils as os_utils_module

    def fake_search(os_client, index, query, request_timeout_seconds=120, max_attempts=3):
        captured.append(query)
        return next(page_iter)

    monkeypatch.setattr(os_utils_module, "_search_with_retries", fake_search)

    out = fetch_rucio_locked_dynamic_totals_composite(
        os_host="https://os.example",
        ca_cert_path="/ca.pem",
        index="rucio-idx",
        time_field="metadata.timestamp",
        sum_field="data.used_bytes",
        tier_field="data.rse_tier",
        rse_type_field="data.rse_type",
        rse_identity_field="data.rse",
        lock_field="data.dbs_is_d_locked",
        lock_value_locked="locked",
        lock_value_dynamic="dynamic",
        gte_epoch_sec=100,
        lte_epoch_sec=200,
        batch_size=50,
    )

    assert out[("T1", "DISK")] == {"locked": 20.0, "dynamic": 0.0}
    assert out[("T2", "TAPE")] == {"locked": 0.0, "dynamic": 1.0}
    assert len(captured) == 3
    assert "after" not in captured[0]["aggs"]["by_entity"]["composite"]
    assert captured[1]["aggs"]["by_entity"]["composite"]["after"] == {
        "tier_key": "T1",
        "rtype_key": "DISK",
        "rse_key": "T1_RSE",
    }
    assert captured[0]["query"]["bool"]["filter"][1] == {
        "term": {"data.dbs_is_d_locked": "locked"}
    }
    assert captured[2]["query"]["bool"]["filter"][1] == {
        "term": {"data.dbs_is_d_locked": "dynamic"}
    }


def test_fetch_rucio_locked_dynamic_totals_logs_aggregation_diagnostics(monkeypatch, caplog):
    pages = [
        {
            "aggregations": {
                "by_entity": {
                    "buckets": [
                        {
                            "key": {
                                "tier_key": "T1",
                                "rtype_key": "DISK",
                                "rse_key": "T1_RSE",
                                "lock_key": "locked",
                            },
                            "doc_count": 2,
                            "sum_usage": {"value": 20.0},
                        }
                    ]
                }
            }
        },
    ]
    page_iter = iter(pages)
    import src.os_utils as os_utils_module

    def fake_search(os_client, index, query, request_timeout_seconds=120, max_attempts=3):
        return next(page_iter)

    monkeypatch.setattr(os_utils_module, "_search_with_retries", fake_search)

    with caplog.at_level("INFO"):
        _ = fetch_rucio_locked_dynamic_totals_composite(
            os_host="https://os.example",
            ca_cert_path="/ca.pem",
            index="rucio-idx",
            time_field="metadata.timestamp",
            sum_field="data.used_bytes",
            tier_field="data.rse_tier",
            rse_type_field="data.rse_type",
            rse_identity_field="data.rse",
            lock_field="data.dbs_is_d_locked",
            lock_value_locked="locked",
            lock_value_dynamic="dynamic",
            gte_epoch_sec=100,
            lte_epoch_sec=200,
            batch_size=50,
        )

    assert "Rucio aggregation diagnostics: unique_entities=1 docs_summed=2 grouped_rows=1" in caplog.text


def test_fetch_rucio_locked_dynamic_totals_remaps_t2_ch_cern_to_t0(monkeypatch):
    pages = [
        {
            "aggregations": {
                "by_entity": {
                    "buckets": [
                        {
                            "key": {
                                "tier_key": "T0",
                                "rtype_key": "DISK",
                                "rse_key": "T0_CH_CERN",
                            },
                            "doc_count": 1,
                            "sum_usage": {"value": 100.0},
                        },
                        {
                            "key": {
                                "tier_key": "T2",
                                "rtype_key": "DISK",
                                "rse_key": "T2_CH_CERN",
                            },
                            "doc_count": 1,
                            "sum_usage": {"value": 10.0},
                        },
                        {
                            "key": {
                                "tier_key": "T2",
                                "rtype_key": "DISK",
                                "rse_key": "T2_US_SITE",
                            },
                            "doc_count": 1,
                            "sum_usage": {"value": 30.0},
                        },
                    ]
                }
            }
        },
        {
            "aggregations": {
                "by_entity": {
                    "buckets": [
                        {
                            "key": {
                                "tier_key": "T0",
                                "rtype_key": "DISK",
                                "rse_key": "T0_CH_CERN",
                            },
                            "doc_count": 1,
                            "sum_usage": {"value": 4.0},
                        },
                        {
                            "key": {
                                "tier_key": "T2",
                                "rtype_key": "DISK",
                                "rse_key": "T2_CH_CERN",
                            },
                            "doc_count": 1,
                            "sum_usage": {"value": 5.0},
                        },
                        {
                            "key": {
                                "tier_key": "T2",
                                "rtype_key": "DISK",
                                "rse_key": "T2_US_SITE",
                            },
                            "doc_count": 1,
                            "sum_usage": {"value": 7.0},
                        },
                    ]
                }
            }
        },
    ]
    page_iter = iter(pages)
    import src.os_utils as os_utils_module

    def fake_search(os_client, index, query, request_timeout_seconds=120, max_attempts=3):
        return next(page_iter)

    monkeypatch.setattr(os_utils_module, "_search_with_retries", fake_search)

    out = fetch_rucio_locked_dynamic_totals_composite(
        os_host="https://os.example",
        ca_cert_path="/ca.pem",
        index="rucio-idx",
        time_field="metadata.timestamp",
        sum_field="data.used_bytes",
        tier_field="data.rse_tier",
        rse_type_field="data.rse_type",
        rse_identity_field="data.rse",
        lock_field="data.dbs_is_d_locked",
        lock_value_locked="locked",
        lock_value_dynamic="dynamic",
        gte_epoch_sec=100,
        lte_epoch_sec=200,
        batch_size=50,
    )

    assert out[("T2", "DISK")] == {"locked": 30.0, "dynamic": 7.0}
    assert out[("T0", "DISK")] == {"locked": 110.0, "dynamic": 9.0}


def test_month_epoch_bounds():
    gte, lte = constants.month_epoch_bounds(2026, 2)
    assert gte < lte
    assert lte - gte == (28 * 86400) - 1


def test_utc_calendar_day_epoch_bounds_mid_month():
    gte, lte = constants.utc_calendar_day_epoch_bounds(2026, 2, 14)
    assert lte - gte == 86399
    assert gte == int(constants.datetime(2026, 2, 14, 0, 0, 0, tzinfo=constants.timezone.utc).timestamp())
    assert lte == int(constants.datetime(2026, 2, 14, 23, 59, 59, tzinfo=constants.timezone.utc).timestamp())


def test_utc_calendar_day_epoch_bounds_rejects_invalid_day():
    with pytest.raises(ValueError, match="day must be"):
        constants.utc_calendar_day_epoch_bounds(2026, 2, 29)


def test_last_utc_day_epoch_bounds_31_day_month():
    gte, lte = constants.last_utc_day_epoch_bounds(2026, 1)
    assert lte - gte == 86399
    assert gte == int(constants.datetime(2026, 1, 31, 0, 0, 0, tzinfo=constants.timezone.utc).timestamp())
    assert lte == int(constants.datetime(2026, 1, 31, 23, 59, 59, tzinfo=constants.timezone.utc).timestamp())


def test_last_utc_day_epoch_bounds_leap_year_february():
    gte, lte = constants.last_utc_day_epoch_bounds(2024, 2)
    assert lte - gte == 86399
    assert gte == int(constants.datetime(2024, 2, 29, 0, 0, 0, tzinfo=constants.timezone.utc).timestamp())
    assert lte == int(constants.datetime(2024, 2, 29, 23, 59, 59, tzinfo=constants.timezone.utc).timestamp())


def test_last_utc_day_epoch_bounds_30_day_month():
    gte, lte = constants.last_utc_day_epoch_bounds(2026, 4)
    assert lte - gte == 86399
    assert gte == int(constants.datetime(2026, 4, 30, 0, 0, 0, tzinfo=constants.timezone.utc).timestamp())
    assert lte == int(constants.datetime(2026, 4, 30, 23, 59, 59, tzinfo=constants.timezone.utc).timestamp())


def test_backfill_query_defaults_to_monthly_window(monkeypatch):
    monkeypatch.setenv("BACKFILL_ENABLED", "true")
    monkeypatch.setenv("RUCIO_USAGE_OS_INDEX", "ix")
    monkeypatch.delenv("BACKFILL_START_DATE", raising=False)
    monkeypatch.delenv("BACKFILL_END_DATE", raising=False)

    reloaded_constants = importlib.reload(constants)
    query_times = reloaded_constants.build_backfill_query_times()

    assert len(query_times) >= 12
    assert query_times[0].day >= 28
    assert query_times[0].hour == 23
    assert query_times[0].minute == 59
    assert query_times[0].second == 59


def test_backfill_query_uses_custom_range_monthly(monkeypatch):
    monkeypatch.setenv("BACKFILL_ENABLED", "true")
    monkeypatch.setenv("RUCIO_USAGE_OS_INDEX", "ix")
    monkeypatch.setenv("BACKFILL_START_DATE", "2026-01-01")
    monkeypatch.setenv("BACKFILL_END_DATE", "2026-01-02")

    reloaded_constants = importlib.reload(constants)
    query_times = reloaded_constants.build_backfill_query_times()

    assert [ts.isoformat() for ts in query_times] == [
        "2026-01-31T23:59:59+00:00",
    ]


def test_regular_run_monthly_uses_previous_month_when_after_upload_day(monkeypatch):
    monkeypatch.setenv("BACKFILL_ENABLED", "false")
    monkeypatch.setenv("QUERY_DATE", "2026-04-10")
    monkeypatch.setenv("RUCIO_USAGE_OS_INDEX", "ix")
    monkeypatch.setenv("RUCIO_UPLOAD_DAY_OF_MONTH", "3")
    monkeypatch.setenv("RUCIO_REPORTING_MONTH_OFFSET", "1")

    reloaded_constants = importlib.reload(constants)
    query_times = reloaded_constants.build_monthly_query_times()

    months = [(dt.year, dt.month) for dt in query_times]
    assert months == [(2026, 3)]


def test_regular_run_monthly_before_upload_day_uses_older_reporting_month(monkeypatch):
    monkeypatch.setenv("BACKFILL_ENABLED", "false")
    monkeypatch.setenv("QUERY_DATE", "2026-04-01")
    monkeypatch.setenv("RUCIO_USAGE_OS_INDEX", "ix")
    monkeypatch.setenv("RUCIO_UPLOAD_DAY_OF_MONTH", "3")
    monkeypatch.setenv("RUCIO_REPORTING_MONTH_OFFSET", "1")

    reloaded_constants = importlib.reload(constants)
    query_times = reloaded_constants.build_monthly_query_times()

    months = [(dt.year, dt.month) for dt in query_times]
    assert months == [(2026, 2)]


def test_query_time_for_month_end_leap_year_february():
    from src.constants import query_time_for_month_end

    dt = query_time_for_month_end(2024, 2)
    assert dt.month == 2
    assert dt.day == 29
    assert dt.hour == 23 and dt.minute == 59 and dt.second == 59


def test_backfill_cmssst_alignment_uses_historical_values_per_timestamp():
    cmssst_docs = [
        {
            "metadata": {"timestamp": "2023-11-14T00:00:00+00:00"},
            "data": {
                "name": "T1_US_A",
                "disk_usable": 120.0,
                "disk_experiment_use": 100.0,
                "disk_local_use": 10.0,
                "disk_reserved_free": 5.0,
            },
        },
        {
            "metadata": {"timestamp": "2023-11-15T00:00:00+00:00"},
            "data": {
                "name": "T1_US_A",
                "disk_usable": 200.0,
                "disk_experiment_use": 150.0,
                "disk_local_use": 20.0,
                "disk_reserved_free": 10.0,
            },
        },
    ]
    cmssst_history = build_cmssst_history_by_group(cmssst_docs)
    ts_a = 1700000000
    ts_b = 1704067200
    cmssst_tier_sums_by_ts = {
        ts_a: resolve_cmssst_tier_sums_at_timestamp(cmssst_history, ts_a),
        ts_b: resolve_cmssst_tier_sums_at_timestamp(cmssst_history, ts_b),
    }
    rucio_totals_by_ts = {
        ts_a: {("T1", "DISK"): {"locked": 70.0, "dynamic": 30.0}},
        ts_b: {("T1", "DISK"): {"locked": 60.0, "dynamic": 20.0}},
    }

    docs = list(
        compute_usage_records(
            rucio_totals_by_ts=rucio_totals_by_ts,
            cmssst_tier_sums_by_ts=cmssst_tier_sums_by_ts,
        )
    )

    assert len(docs) == 2
    by_timestamp = {doc["timestamp"]: doc for doc in docs}
    assert by_timestamp["2023-11-14T22:13:20+00:00Z"]["rucio_locally_managed"] == 10.0 * CMSSST_TB_TO_BYTES
    assert by_timestamp["2024-01-01T00:00:00+00:00Z"]["rucio_locally_managed"] == 20.0 * CMSSST_TB_TO_BYTES


def test_backfill_cmssst_alignment_uses_zero_when_no_history_at_timestamp():
    cmssst_docs = [
        {
            "metadata": {"timestamp": "2023-11-15T00:00:00+00:00"},
            "data": {
                "name": "T1_US_A",
                "disk_usable": 180.0,
                "disk_experiment_use": 140.0,
                "disk_local_use": 15.0,
                "disk_reserved_free": 8.0,
            },
        }
    ]
    cmssst_history = build_cmssst_history_by_group(cmssst_docs)
    early_sums = resolve_cmssst_tier_sums_at_timestamp(cmssst_history, 1699990000)
    late_sums = resolve_cmssst_tier_sums_at_timestamp(cmssst_history, 1700086400)

    assert "T1" not in early_sums
    assert late_sums["T1"]["disk_local_use"] == 15.0 * CMSSST_TB_TO_BYTES


def test_build_cmssst_history_skips_malformed_timestamp():
    cmssst_docs = [
        {
            "metadata": {"timestamp": "bad-ts"},
            "data": {
                "name": "T1_US_A",
                "disk_usable": 10.0,
                "disk_experiment_use": 8.0,
                "disk_local_use": 1.0,
                "disk_reserved_free": 1.0,
            },
        },
        {
            "metadata": {"timestamp": "2023-11-14T00:00:00+00:00"},
            "data": {
                "name": "T1_US_A",
                "disk_usable": 20.0,
                "disk_experiment_use": 16.0,
                "disk_local_use": 2.0,
                "disk_reserved_free": 2.0,
            },
        },
    ]

    history = build_cmssst_history_by_group(cmssst_docs)
    tier_sums = resolve_cmssst_tier_sums_at_timestamp(history, 1700000000)

    assert len(history["T1_US_A"]) == 1
    assert tier_sums["T1"]["disk_usable"] == 20.0 * CMSSST_TB_TO_BYTES
