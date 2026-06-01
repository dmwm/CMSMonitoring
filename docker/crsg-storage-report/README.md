# Promxy to Opensearch

## Purpose

This is the codebase for the `promxy-to-opensearch` docker image inside the cmsmonitoring repository of the CERN docker registry.

The job loads **monthly Rucio storage usage** from an OpenSearch index on **`CMSSST_OS_HOST`**, merges it with **CMSSST** disk aggregates from the same cluster, and writes merged documents to **`OPENSEARCH_HOST`**.

## Deployment

The code for deploying the cronjob in charge of the data loads is in the [CMSKubernetes repository](https://github.com/dmwm/CMSKubernetes/blob/master/kubernetes/monitoring/crons/promxy-to-opensearch.yaml).

To deploy or update the cronjob using the yaml, set the proper `kubectl` context and apply the yaml:

``` bash
export KUBECONFIG=<path_to_kubeconfig>
kubectl apply -f <path_to_cronjob_yaml>
```

### Environment variables

**Destination (write):**

- `OPENSEARCH_HOST`: OpenSearch URL for bulk upload. Default: `https://os-cms.cern.ch:443/os`
- `OPENSEARCH_INDEX`: Destination index name.

**Sources on `CMSSST_OS_HOST` (read):**

- `CMSSST_OS_HOST`: Default: `https://monit-opensearch-lt.cern.ch:443/os`
- `CMSSST_OS_INDEX`: CMSSST index pattern. Default: `monit_prod_cmssst*`
- `CMSSST_OS_FILTER_FIELD` / `CMSSST_OS_FILTER_VALUE`: Filter for CMSSST docs (default `metadata.monit_hdfs_path` / `scap15min`).
- `CMSSST_OS_GROUP_FIELD` / `CMSSST_OS_SORT_FIELD` / `CMSSST_OS_EXCLUDE_FIELD`: Grouping, recency, and exclude regex field for latest-per-RSE selection.

**Rucio monthly usage index (required):**

- `RUCIO_USAGE_OS_INDEX`: Index pattern on `CMSSST_OS_HOST` for monthly Rucio storage documents (**required**).
- `RUCIO_USAGE_SUM_FIELD`: Numeric field for `sum` aggregation (default `data.used_bytes`).
- `RUCIO_USAGE_TIME_FIELD`: Time field for reporting-month window (default `metadata.timestamp`, epoch seconds).
- `RUCIO_USAGE_RSE_TIER_FIELD` / `RUCIO_USAGE_RSE_TYPE_FIELD`: Bucket fields (defaults `data.rse_tier`, `data.rse_type`). Use `.keyword` subfields if mappings require it.
- `RUCIO_USAGE_LOCK_FIELD`: Lock dimension field (default `data.dbs_is_d_locked`).
- `RUCIO_USAGE_LOCK_VALUE_LOCKED` / `RUCIO_USAGE_LOCK_VALUE_DYNAMIC`: Filter values (defaults `locked`, `dynamic`).
- `RUCIO_USAGE_MONTH_START_BUFFER_DAYS` / `RUCIO_USAGE_MONTH_END_BUFFER_DAYS`: Widen the epoch range around each reporting month (defaults `0` and `10`) so documents stamped in early **M+1** are included.
- `RUCIO_USAGE_COMPOSITE_BATCH_SIZE`: Composite aggregation page size (default `500`).
- `RUCIO_USAGE_SUM_UNIT`: `bytes` or `tb` — if `tb`, sums are converted to bytes like CMSSST tier totals (default `bytes`).

**Scheduling / backfill:**

- `QUERY_DATE`: Optional UTC date (`YYYY-MM-DD`) for the reference month in regular mode.
- `BACKFILL_ENABLED`, `BACKFILL_START_DATE`, `BACKFILL_END_DATE`: Monthly snapshots from month-end times in the inclusive month range.
- `MONTHLY_LATE_ARRIVAL_LOOKBACK_MONTHS`, `MONTHLY_INCLUDE_PARTIAL_CURRENT_MONTH`: Regular mode month selection.
- `KRB5_CLIENT_KTNAME`: Kerberos keytab path (default `/etc/secrets/keytab`).
- `DEBUG_MODE`, `DRY_RUN`: Logging verbosity and skip upload.

## Behavior

- For each processing month, the job queries Rucio docs whose `RUCIO_USAGE_TIME_FIELD` falls in an epoch window from the first day of that month (minus start buffer) through the end of the extended window after month boundary (end buffer), then **sums** `RUCIO_USAGE_SUM_FIELD` per `(rse_tier, rse_type)` with filters **locked** vs **dynamic**.
- CMSSST **latest doc per RSE** at each month-end snapshot instant is aggregated **by tier** (TB → bytes) and joined using the same **tier** as the Rucio rows.
- Output documents include `timestamp`, `snapshot_month`, `tier`, `rse_type`, `rucio_locked`, `rucio_dynamic`, `rucio_locally_managed`, `rucio_available`, `non_rucio_buffers` (no `rucioInstance`, no legacy `promxy_raw_*` fields).
