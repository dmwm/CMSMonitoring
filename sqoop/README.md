# CMS Monitoring Sqoop dumps

Main repository for sqoop dumps

## Deployment requirements

#### Environment variables

- `WDIR`: working directory. `sqoop` base directory should be `$WDIR/sqoop`.
- `CMSSQOOP_CONFIGS`: full path of **configs.json** file which contains **HDFS paths** for each script. It can
  be `$WDIR/sqoop/configs.json`, `$WDIR/sqoop/configs-test.json` or any other json(via ConfgiMap may be). If it is not provided, it will be set according to `CMSSQOOP_ENV` value.
- `CMSSQOOP_ENV`: for production, it should be `prod`. For tests, it should be `test` or should not be set because it will be set as **test**.
  - It will be used in PushGateway "$env" tag.


> **How HDFS output paths of scripts are defined. Check run.sh for the logic.**
>
> - If `$CMSSQOOP_CONFIGS` is provided, in any case, HDFS output paths will be read from that JSON file.
> - If `$CMSSQOOP_CONFIGS` is NOT provided:
>     - If `$CMSSQOOP_ENV` is provided as `prod`, `$CMSSQOOP_CONFIGS` will look to `~/sqoop/configs.json`.
>     - If `$CMSSQOOP_ENV` is NOT provided OR NOT `prod`, `$CMSSQOOP_CONFIGS` will look to `~/sqoop/configs-test.json`.
> Why this logic: It should be both parametrized and secure.


#### Other requirements

- `configs.json`(HDFS output paths and PG url) can be provided via `$CMSSQOOP_CONFIGS` env var, see `util_get_config_val` function
  in `scripts/utils.sh`.
- Please provide all secrets in `/etc/secrets/` directory (**[TODO]** this will be made configurable after new full dbs dump deployments)
- Required secrets (`$secrets` refers to cmsmonitoring/secrets, `$cmsmon-configs` refers to
  cmsmonitoring/cmsmon-configs repositories in gitlab):
    - rucio : $secrets/rucio/rucio
    - cms-es-size.json : $secrets/sqoop/cms-es-size.json
    - cmsr_cstring : $secrets/sqoop/cmsr_cstring
    - keytab : $secrets/sqoop/keytab
    - lcgr_cstring : $secrets/sqoop/lcgr_cstring
    - token : $secrets/sqoop/token
    - hdfs.json : $cmsmon-configs/sqoop/hdfs.json

#### New cron job requirements

- A cron job should be testable.
- A cron job must send start/end/duration metrics to Prometheus through PushGateway.
- If cron job writes data to some location, it should be put into `configs.json`, NO hardcoded path/config should be
  defined because they make testing difficult.
- For almost all multi-usage logics, there is a util function. For readability and consistency concerns, usage of util
  functions is strongly suggested. For new util functions, it is expected to be complied
  with [Google Style Guide](https://google.github.io/styleguide/shellguide.html).

## How to test

Everything should be testable.

- Set `CMSSQOOP_ENV` as `test` or any other value than `prod`. Then HDFS output paths will be defined by `sqoop/configs-test.json`.
- OR, provide test directories for sqoop dumps via `$CMSSQOOP_CONFIGS`. See `sqoop/configs-test.json` for dev-test values.
- Do not set `CMSSQOOP_ENV` as `prod` and do not set `$CMSSQOOP_CONFIGS` as `sqoop/configs.json` which is production configurations.

## Special dump implementation for full DBS dumps

For `CMS_DBS3_PROD_*` schemas tables dumps takes so long time in normal ways. For that reason, their dumps are specialized as:

- Import kind is full table dump with direct connection.
- Dump all tables to HDFS in compressed(-z) CSV format which is compatible with
  [CMSSpark/src/python/CMSSpark/schemas.py](https://github.com/dmwm/CMSSpark/blob/master/src/python/CMSSpark/schemas.py)
  .
    - schemas.py is compatible with both compressed and raw CSV.
- Table dump processes run in parallel which means total process time is equal to the max table dump time, so
  to `FILE_LUMIS` or `FILE_PARENTS`.
- Before dumping tables, each table is checked of having data or not in DB, because full/direct dump fails if there is
  no data
  in the table.
- Table data checks run sequentially using simple SQL query

## TODO: Discuss degrading CMS_DBS3_PROD_PHYS* table dumps

- It was suggested in October 2022 O&C week.
