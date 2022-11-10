#!/bin/bash
set -e
# Import Oracle CMS_DBS3_PROD_GLOBAL_OWNER tables using Apache Sqoop tool.
#   - Import kind is full table dump with direct connection
#   - Dumped to HDFS in compressed(-z) CSV format which is compatible with dmwm/CMSSpark/src/python/CMSSpark/schemas.py
#     - schemas.py is compatible with both compressed and raw CSV
#   - Table dump processes run in parallel which means total process time is equal to the max table dump time, so as to 'FILE_LUMIS'
#   - Before dumping table, each table is checked of having data or not, because full/direct dump fails if there is no data in the table
#   - Table data checks run sequentially using simple SQL query

. "${WDIR}/sqoop/scripts/utils.sh"
TZ=UTC

# --------------------------------------------------------------------------------- PREPS
SCHEMA="CMS_DBS3_PROD_GLOBAL_OWNER"
# Sorted in ascending size order which is the suggested order to decrease run time
DBS_TABLES="ASSOCIATED_FILES BRANCH_HASHES DATASET_RUNS MIGRATION_BLOCKS MIGRATION_REQUESTS DATASET_ACCESS_TYPES FILE_DATA_TYPES \
 PRIMARY_DS_TYPES APPLICATION_EXECUTABLES PHYSICS_GROUPS DATA_TIERS PROCESSING_ERAS RELEASE_VERSIONS ACQUISITION_ERAS \
 PARAMETER_SET_HASHES PRIMARY_DATASETS PROCESSED_DATASETS OUTPUT_MODULE_CONFIGS DATASET_PARENTS DATASET_OUTPUT_MOD_CONFIGS \
 DATASETS BLOCK_PARENTS BLOCKS FILE_OUTPUT_MOD_CONFIGS FILE_PARENTS FILES FILE_LUMIS "

# index-organized tables are not suitable for 40 mappers. In order to iterate table, sqoop run query in each iteration to find max/min unique if
# that's why we'll set --num-mappers(-m) as 1 in these tables.
NUM_MAPPER_1_TABLES="FILE_PARENTS"

# ------------------------------------------------------------------------------- GLOBALS
myname=$(basename "$0")
BASE_PATH=$(util_get_config_val "$myname")
DAILY_BASE_PATH="${BASE_PATH}/$(date +%Y-%m-%d)"
LOG_FILE=log/$(date +'%F_%H%M%S')_$myname

# Daily data will be copied to this folder until all our users switch to daily folders usage.
LEGACY_PROD_PATH="/project/awg/cms/CMS_DBS3_PROD_GLOBAL"
START_TIME=$(date +%s)
pg_metric_db="DBS_GLOBAL"
util4logi "CMSSQOOP_ENV=${CMSSQOOP_ENV}, CMSSQOOP_CONFIGS=${CMSSQOOP_CONFIGS}." >>"$LOG_FILE".stdout

# -------------------------------------------------------------------------------- CHECKS
if [ -f /etc/secrets/cmsr_cstring ]; then
    jdbc_url=$(sed '1q;d' /etc/secrets/cmsr_cstring)
    username=$(sed '2q;d' /etc/secrets/cmsr_cstring)
    password=$(sed '3q;d' /etc/secrets/cmsr_cstring)
else
    util4loge "Unable to read DBS credentials" >>"$LOG_FILE".stdout
    exit 1
fi

# ------------------------------------------------------------------------- DUMP FUNCTION
# Dumps full dbs table in compressed CSV format
sqoop_dump_dbs_cmd() {
    local local_start_time TABLE num_mappers
    kinit -R
    local_start_time=$(date +%s)
    TABLE=$1
    num_mappers=40
    if [[ $TABLE == *"$NUM_MAPPER_1_TABLES"* ]]; then
        num_mappers=1
    fi
    util4logi "${SCHEMA}.${TABLE} : import starting with num-mappers as $num_mappers .."
    pushg_dump_start_time "$myname" "$pg_metric_db" "$SCHEMA" "$TABLE"
    #
    /usr/hdp/sqoop/bin/sqoop import -Dmapreduce.job.user.classpath.first=true -Doraoop.timestamp.string=false \
        -Dmapred.child.java.opts="-Djava.security.egd=file:/dev/../dev/urandom" -Ddfs.client.socket-timeout=120000 \
        --fetch-size 10000 --fields-terminated-by , --escaped-by \\ --optionally-enclosed-by '\"' \
        -z --direct --throw-on-error --num-mappers $num_mappers \
        --connect "$jdbc_url" --username "$username" --password "$password" \
        --target-dir "$DAILY_BASE_PATH"/"$TABLE" --table "$SCHEMA"."$TABLE" \
        1>>"$LOG_FILE".stdout 2>>"$LOG_FILE".stderr
    #
    util4logi "${SCHEMA}.${TABLE} : import finished successfully in $(util_secs_to_human "$(($(date +%s) - local_start_time))")"
    pushg_dump_end_time "$myname" "$pg_metric_db" "$SCHEMA" "$TABLE"
}

# ----------------------------------------------------------------------------------- RUN
# successful table dump counter
tables_success_counter=0

# Import all tables in parallel after table data check
for TABLE_NAME in $DBS_TABLES; do
    ec=$(check_table_exist "${SCHEMA}.${TABLE_NAME}" "$jdbc_url" "$username" "$password")
    case "$ec" in
    0)
        util4logi "${SCHEMA}.${TABLE_NAME} : table check OKAY" >>"$LOG_FILE".stdout
        # Run in background
        sqoop_dump_dbs_cmd "$TABLE_NAME" >>"$LOG_FILE".stdout 2>&1 &
        # Increment table count
        tables_success_counter=$((tables_success_counter + 1))
        ;;
    1) util4logw "${SCHEMA}.${TABLE_NAME} : table check NO DATA, skipping" >>"$LOG_FILE".stdout ;;
    esac
done

# Wait to finish all background jobs
wait

# Give read permission to the new folder and sub folders after all dumps finished
hadoop fs -chmod -R o+rx "$DAILY_BASE_PATH"

# Copy daily results to legacy production folder
if [ "$CMSSQOOP_ENV" = "prod" ]; then
    copy_to_legacy_folders "$DAILY_BASE_PATH" "$LEGACY_PROD_PATH" "$LOG_FILE"
fi
# ---------------------------------------------------------------------------- STATISTICS
# total duration
duration=$(($(date +%s) - START_TIME))
# Dumped tables total size in bytes
dump_size=$(util_hdfs_size "$DAILY_BASE_PATH")

# Pushgateway
pushg_dump_duration "$myname" "$pg_metric_db" "$SCHEMA" $duration
pushg_dump_size "$myname" "$pg_metric_db" "$SCHEMA" "$dump_size"
pushg_dump_table_count "$myname" "$pg_metric_db" "$SCHEMA" $tables_success_counter

util4logi "all finished, time spent: $(util_secs_to_human $duration)" >>"$LOG_FILE".stdout
