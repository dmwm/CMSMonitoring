# CRSG storage report

## Purpose

This directory builds the `crsg-storage-report` image in the CERN `cmsmonitoring` registry.

The job produces **monthly CRSG storage snapshot** documents by:

1. Reading **daily Rucio usage** from OpenSearch on `RUCIO_USAGE_OS_HOST`, aggregating locked vs dynamic bytes per `(tier, rse_type)` for one UTC day in the reporting month.
2. Reading **CMSSST** capacity metrics from OpenSearch on `CMSSST_OS_HOST`, taking the latest document per RSE at each month-end instant and summing disk fields **by tier**.
3. Merging the two sources and bulk-indexing rows into `OPENSEARCH_HOST`.

All data access is via OpenSearch (Kerberos). There is **no Promxy** query in this image; legacy filenames (`promxy_to_opensearch.py`, `src/promxy.py`) are historical.

## Deployment

Run the container with `/app/entrypoint.sh`, which performs Kerberos `kinit` from a keytab and executes `promxy_to_opensearch.py`.

Typical image reference:

`registry.cern.ch/cmsmonitoring/crsg-storage-report:<tag>`

Wire a Kubernetes `CronJob` in [CMSKubernetes](https://github.com/dmwm/CMSKubernetes) with env vars below, a keytab volume, and `KRB5_CLIENT_KTNAME` pointing at the mounted keytab. The legacy [`promxy-to-opensearch.yaml`](https://github.com/dmwm/CMSKubernetes/blob/master/kubernetes/monitoring/crons/promxy-to-opensearch.yaml) cron still targets the old Promxy-based image and is **not** compatible with this codebase.

```bash
export KUBECONFIG=<path_to_kubeconfig>
kubectl apply -f <path_to_cronjob_yaml>
```

### Environment variables

**Destination (write):**

- `OPENSEARCH_HOST`: OpenSearch URL for bulk upload. Default: `https://os-cms.cern.ch:443/os`
- `OPENSEARCH_INDEX`: Destination index name. Default: `rucio-used-space-1h`
- `CERT_PATH`: CA bundle for TLS. Default: `/etc/pki/tls/certs/ca-bundle.trust.crt`

**CMSSST source (read on `CMSSST_OS_HOST`):**

- `CMSSST_OS_HOST`: Default: `https://monit-opensearch-lt.cern.ch:443/os`
- `CMSSST_OS_INDEX`: Index pattern. Default: `monit_prod_cmssst*`
- `CMSSST_OS_FILTER_FIELD` / `CMSSST_OS_FILTER_VALUE`: Doc filter (defaults `metadata.monit_hdfs_path` / `scap15min`)
- `CMSSST_OS_GROUP_FIELD`: Latest-doc grouping field. Default: `data.name`
- `CMSSST_OS_SORT_FIELD`: Recency field. Default: `metadata.timestamp`
- `CMSSST_OS_EXCLUDE_FIELD`: Field matched by exclude regex. Default: `data.name`
- `EXCLUDE_RSE_REGEX`: Regex excluding test/temp RSEs. Default: `.*(Test|test|_Temp|_temp)`

**Rucio daily usage source (read on `RUCIO_USAGE_OS_HOST`):**

- `RUCIO_USAGE_OS_HOST`: Default: `https://monit-opensearch.cern.ch:443/os`
- `RUCIO_USAGE_OS_INDEX`: Index pattern for daily Rucio stats. Default: `monit_prod_cms_rucio_raw_daily_stats*` (must be non-empty)
- `RUCIO_USAGE_SUM_FIELD`: Field summed in composite agg. Default: `data.dbs_size`
- `RUCIO_USAGE_TIME_FIELD`: Time field for day window (epoch seconds). Default: `metadata.timestamp`
- `RUCIO_USAGE_RSE_TIER_FIELD` / `RUCIO_USAGE_RSE_TYPE_FIELD`: Tier and type bucket fields (defaults `data.rse_tier`, `data.rse_type`). Use `.keyword` subfields if mappings require it.
- `RUCIO_USAGE_RSE_IDENTITY_FIELD`: Per-RSE bucket before tier rollup. Default: `data.rse`
- `RUCIO_USAGE_LOCK_FIELD`: Lock dimension. Default: `data.dbs_is_d_locked`
- `RUCIO_USAGE_LOCK_VALUE_LOCKED` / `RUCIO_USAGE_LOCK_VALUE_DYNAMIC`: Filter values (defaults `locked`, `dynamic`)
- `RUCIO_USAGE_COMPOSITE_BATCH_SIZE`: Composite aggregation page size. Default: `500`
- `RUCIO_USAGE_SUM_UNIT`: `bytes` or `tb` — if `tb`, sums are converted to bytes like CMSSST tier totals. Default: `bytes`

**Reporting schedule (regular mode):**

- `RUCIO_UPLOAD_DAY_OF_MONTH`: Day of month when the monthly report is expected to run (1–31). Default: `3`
- `RUCIO_REPORTING_MONTH_OFFSET`: Months to subtract from the anchor month to get the reporting month. Default: `1` (anchor month M → report on M−1)
- `QUERY_DATE`: Optional UTC date (`YYYY-MM-DD`) treated as start-of-day for the run clock. Default: now (UTC). Used with upload day/offset to pick the reporting month.

**Backfill:**

- `BACKFILL_ENABLED`: If `true`, process every UTC month from `BACKFILL_START_DATE` through `BACKFILL_END_DATE` (inclusive) instead of a single reporting month.
- `BACKFILL_START_DATE` / `BACKFILL_END_DATE`: `YYYY-MM-DD`. If omitted when backfill is enabled, start defaults to one year ago and end to now.

**Runtime:**

- `KRB5_CLIENT_KTNAME`: Keytab path for `kinit` in `entrypoint.sh`. Default: `/etc/krb5.keytab`
- `DEBUG_MODE`, `DRY_RUN`: Verbose logging; generate documents but skip OpenSearch upload.

## Behavior

### Month selection

Each processed month uses a snapshot instant at the **last second of that UTC month**.

- **Regular run:** One reporting month from `QUERY_TIME` (now or `QUERY_DATE`), `RUCIO_UPLOAD_DAY_OF_MONTH`, and `RUCIO_REPORTING_MONTH_OFFSET`. If the run day is before the upload day, the anchor month is the previous calendar month; the reporting month is anchor minus the offset.
- **Backfill:** One snapshot per month in the backfill date range.

### Rucio aggregation

For each reporting month, the job selects a **single UTC calendar day**:

1. Try the last day of the month.
2. If the composite aggregation returns no buckets, walk backward day by day for up to **10** days.

Within that day, OpenSearch sums `RUCIO_USAGE_SUM_FIELD` with a composite aggregation on `(tier, rse_type, rse identity, lock state)`, applying dashboard-aligned filters (`_exists_:data.dataset`, `data.is_dataset_valid:1`, `data.dbs_has_ds_name:True`) and separate queries for locked vs dynamic.

Usage attributed to RSE `T2_CH_CERN` is subtracted from tier `T2` and added to tier `T0` for the same `rse_type`.

### CMSSST aggregation

For each month-end snapshot timestamp, fetch the **latest CMSSST document per RSE** (grouped by `CMSSST_OS_GROUP_FIELD`) at or before that instant, excluding names matching `EXCLUDE_RSE_REGEX`.

Disk fields (`disk_pledge`, `disk_usable`, `disk_experiment_use`, `disk_local_use`, `disk_reserved_free`) are converted from TB to bytes and summed by **tier** (first two characters of `data.name`). Special cases:

- `T2_CH_CERN`: selected disk fields roll into tier `T0`; pledge-related fields stay on `T2`.
- `T0_CH_CERN`: `disk_reserved_free` is not added to tier `T0`.

### Merge and upload

For each `(tier, rse_type)` present in Rucio totals at a snapshot:

- Join CMSSST tier disk sums.
- Compute derived fields (see output schema).
- Bulk index with deterministic document IDs from `(snapshot_month, tier, rse_type)` so re-runs are idempotent.

With `DRY_RUN=true`, the job logs sample documents and a short validation checklist instead of uploading.

## Output documents

Each row includes:

| Field | Description |
| --- | --- |
| `timestamp` | Month-end snapshot instant (ISO-8601 UTC) |
| `snapshot_month` | `YYYY-MM` of the reporting month |
| `tier`, `rse_type` | Join keys |
| `disk_pledge`, `disk_usable`, `disk_experiment_use`, `disk_local_use`, `disk_reserved_free` | CMSSST tier totals (bytes) |
| `rucio_locked`, `rucio_dynamic` | Rucio usage (bytes) |
| `rucio_locally_managed` | Set to `disk_local_use` |
| `rucio_available` | `disk_experiment_use − disk_reserved_free − rucio_locked − rucio_dynamic` |
| `non_rucio_buffers` | `disk_usable − disk_experiment_use + disk_reserved_free`, plus `rucio_available` if negative |

## Tests

From this directory:

```bash
python3 -m pytest tests/
```
