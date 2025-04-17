#!/bin/bash
# shellcheck disable=SC2004
# Import Oracle CMS_ANALYSIS_REQMGR.TASKS table using Apache Sqoop tool.
#   - Imports full table without direct connection
#   - Dumped to HDFS in avro format
#

set -e
. "${WDIR}/sqoop/scripts/utils.sh"
TZ=UTC

export PATH="$PATH:/usr/hdp/sqoop/bin/"

# --------------------------------------------------------------------------------- PREPS
SCHEMA="CMS_ANALYSIS_REQMGR"
TABLE="TASKS"
NUM_MAPPERS=8

# ------------------------------------------------------------------------------- GLOBALS
myname=$(basename "$0")
BASE_PATH=$(util_get_config_val "$myname")
DAILY_PATH="${BASE_PATH}/$(date +%Y-%m-%d)"
START_TIME=$(date +%s)
LOG_FILE=log/$(date +'%F_%H%M%S')_$myname
pg_metric_db="TASKS"
util4logi "CMSSQOOP_ENV=${CMSSQOOP_ENV}, CMSSQOOP_CONFIGS=${CMSSQOOP_CONFIGS}." >>"$LOG_FILE".stdout

# Check hadoop executable exist
if ! [ -x "$(command -v hadoop)" ]; then
    echo "[ERROR] It seems 'hadoop' does not exist in PATH! Exiting..." >>"$LOG_FILE".stdout
    exit 1
fi

# -------------------------------------------------------------------------------- CHECKS
if [ -f /etc/secrets/cmsr_cstring ]; then
    JDBC_URL=$(sed '1q;d' /etc/secrets/cmsr_cstring)
    USERNAME=$(sed '2q;d' /etc/secrets/cmsr_cstring)
    PASSWORD=$(sed '3q;d' /etc/secrets/cmsr_cstring)
else
    util4loge "Unable to read the credentials" >>"$LOG_FILE".stdout
    exit 1
fi

# --------------------------------------------------------------------------------- UTILS
trap 'onFailExit' ERR
onFailExit() {
    util4loge "Finished with error! $(dirname "$0")" >>"$LOG_FILE".stdout
    util4loge "FAILED" >>"$LOG_FILE".stdout
    exit 1
}

# ------------------------------------------------------------------------- DUMP FUNCTION
# Dumps TASKS table in avro format
# shellcheck disable=SC2086
sqoop_dump_tasks_cmd() {
    local local_start_time
    local_start_time=$(date +%s)
    trap 'onFailExit' ERR
    kinit -R

    util4logi "${SCHEMA}.${TABLE} : import starting with num-mappers as $NUM_MAPPERS .."
    pushg_dump_start_time "$myname" "$pg_metric_db" "$SCHEMA" "$TABLE"
    #
    sqoop import \
      -Dmapreduce.job.user.classpath.first=true \
      -Ddfs.client.socket-timeout=120000 \
      -Dmapred.child.java.opts="-Djava.security.egd=file:/dev/../dev/urandom" \
      --username "$USERNAME" --password "$PASSWORD" -z --throw-on-error --connect "$JDBC_URL" \
      --num-mappers $NUM_MAPPERS \
      --split-by TM_START_TIME \
      --fetch-size 3000 \
      --map-column-java TM_TASK_FAILURE=String,TM_SPLIT_ARGS=String,TM_OUTFILES=String,TM_TFILE_OUTFILES=String,TM_EDM_OUTFILES=String,TM_ARGUMENTS=String,TM_OUTPUT_DATASET=String,TM_TASK_WARNINGS=String,TM_USER_FILES=String,TM_USER_CONFIG=String,TM_MULTIPUB_RULE=String \
      --target-dir "$DAILY_PATH" \
      --table \""$SCHEMA"."$TABLE"\" \
      --as-avrodatafile \
      1>"$LOG_FILE".stdout 2>"$LOG_FILE".stderr
    #
    util4logi "${SCHEMA}.${TABLE} : import finished successfully in $(util_secs_to_human "$(($(date +%s) - local_start_time))")"
    pushg_dump_end_time "$myname" "$pg_metric_db" "$SCHEMA" "$TABLE"
}

# ----------------------------------------------------------------------------------- RUN

# Run import
sqoop_dump_tasks_cmd >>"$LOG_FILE".stdout 2>&1

# Give read permission to the new folder and sub folders after the dump is finished
hadoop fs -chmod -R o+rx "$DAILY_PATH"

# ---------------------------------------------------------------------------- STATISTICS
# total duration
duration=$(($(date +%s) - START_TIME))
# Dumped tables total size in bytes
dump_size=$(util_hdfs_size "$DAILY_PATH")

# Pushgateway
pushg_dump_duration "$myname" "$pg_metric_db" "$SCHEMA" $duration
pushg_dump_size "$myname" "$pg_metric_db" "$SCHEMA" "$dump_size"

util4logi "all finished, time spent: $(util_secs_to_human $duration)" >>"$LOG_FILE".stdout
