#!/bin/bash
# shellcheck disable=SC2004
# Import Oracle CMS_DBS3_PROD_GLOBAL_OWNER tables using Apache Sqoop tool.
#   - Import kind is full table dump with direct connection
#   - Dumped to HDFS in compressed(-z) CSV format which is compatible with dmwm/CMSSpark/src/python/CMSSpark/schemas.py
#     - schemas.py is compatible with both compressed and raw CSV
#

set -e
. "${WDIR}/sqoop/scripts/utils.sh"
TZ=UTC

# --------------------------------------------------------------------------------- PREPS
SCHEMA="CMS_DBS3_PROD_GLOBAL_OWNER"
# Sorted in ascending size order which is the suggested order to decrease run time
DBS_TABLES_SMALL="DATASET_ACCESS_TYPES FILE_DATA_TYPES \
 PRIMARY_DS_TYPES APPLICATION_EXECUTABLES PHYSICS_GROUPS DATA_TIERS PROCESSING_ERAS RELEASE_VERSIONS ACQUISITION_ERAS \
 PARAMETER_SET_HASHES PRIMARY_DATASETS PROCESSED_DATASETS OUTPUT_MODULE_CONFIGS DATASET_PARENTS DATASET_OUTPUT_MOD_CONFIGS \
 DATASETS BLOCK_PARENTS "
DBS_TABLES_BIG="BLOCKS FILE_OUTPUT_MOD_CONFIGS FILES FILE_LUMIS "
# Index-organized tables are not suitable for more than 1 mapper even their size is big.
# In order to iterate table, sqoop run query in each iteration to find max/min unique which takes so much time. That's why we'll set --num-mappers(-m) as 1 in these tables.
DBS_TABLES_BIG_NUM_MAPPERS_1="FILE_PARENTS"

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

# Keep error count
error_count=0

# ------------------------------------------------------------------------- DUMP FUNCTION
# Dumps full dbs table in compressed CSV format
# arg1: TABLE
# arg2: NUM_MAPPERS
# shellcheck disable=SC2086
sqoop_dump_dbs_cmd() {
    local local_start_time TABLE NUM_MAPPERS
    kinit -R
    local_start_time=$(date +%s)
    TABLE=$1
    NUM_MAPPERS=$2

    util4logi "${SCHEMA}.${TABLE} : import starting with num-mappers as $NUM_MAPPERS .."
    pushg_dump_start_time "$myname" "$pg_metric_db" "$SCHEMA" "$TABLE"
    #
    /usr/hdp/sqoop/bin/sqoop import -Dmapreduce.job.user.classpath.first=true -Doraoop.timestamp.string=false \
        -Dmapred.child.java.opts="-Djava.security.egd=file:/dev/../dev/urandom" -Ddfs.client.socket-timeout=120000 \
        --fetch-size 10000 --fields-terminated-by , --escaped-by \\ --optionally-enclosed-by '\"' \
        -z --direct --throw-on-error --num-mappers $NUM_MAPPERS \
        --connect "$jdbc_url" --username "$username" --password "$password" \
        --target-dir "$DAILY_BASE_PATH"/"$TABLE" --table "$SCHEMA"."$TABLE" \
        1>>"$LOG_FILE".stdout 2>>"$LOG_FILE".stderr
    error_count=$(($error_count + $?))
    #
    util4logi "${SCHEMA}.${TABLE} : import finished successfully in $(util_secs_to_human "$(($(date +%s) - local_start_time))")"
    pushg_dump_end_time "$myname" "$pg_metric_db" "$SCHEMA" "$TABLE"
}

# ----------------------------------------------------------------------------------- RUN

# Run imports parallel for small tables with num_mappers=1
for TABLE_NAME in $DBS_TABLES_SMALL; do
    sqoop_dump_dbs_cmd "$TABLE_NAME" 1 >>"$LOG_FILE".stdout 2>&1 &
    sleep 5
done

# wait to finish parallel jobs
wait

# Run imports sequentially for big tables with num_mappers=50
for TABLE_NAME in $DBS_TABLES_BIG; do
    sqoop_dump_dbs_cmd "$TABLE_NAME" 50 >>"$LOG_FILE".stdout 2>&1
    sleep 5
done

# Run imports sequentially for big tables bu mappers should be 1
for TABLE_NAME in $DBS_TABLES_BIG_NUM_MAPPERS_1; do
    sqoop_dump_dbs_cmd "$TABLE_NAME" 1 >>"$LOG_FILE".stdout 2>&1
done

# Give read permission to the new folder and sub folders after all dumps finished
hadoop fs -chmod -R o+rx "$DAILY_BASE_PATH"
error_count=$(($error_count + $?))

# Copy daily results to legacy production folder
if [ "$CMSSQOOP_ENV" = "prod" ]; then
    copy_to_legacy_folders "$DAILY_BASE_PATH" "$LEGACY_PROD_PATH" "$LOG_FILE"
    error_count=$(($error_count + $?))
fi
# ---------------------------------------------------------------------------- STATISTICS
# total duration
duration=$(($(date +%s) - START_TIME))
# Dumped tables total size in bytes
dump_size=$(util_hdfs_size "$DAILY_BASE_PATH")

# Pushgateway
pushg_dump_duration "$myname" "$pg_metric_db" "$SCHEMA" $duration
pushg_dump_size "$myname" "$pg_metric_db" "$SCHEMA" "$dump_size"

util4logi "error cont: ${error_count}" >>"$LOG_FILE".stdout
util4logi "all finished, time spent: $(util_secs_to_human $duration)" >>"$LOG_FILE".stdout
