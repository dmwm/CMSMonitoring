#!/bin/bash
set -e
# Imports CMS_RUCIO_PROD tables
# Reference: some features/logics are copied from sqoop_utils.sh and cms-dbs3-full-copy.sh
. "${WDIR}/sqoop/scripts/utils.sh"

export PATH="$PATH:/usr/hdp/sqoop/bin/"

TZ=UTC
SCHEMA="CMS_RUCIO_PROD"
# Suggested order
RUCIO_TABLES="replicas dids contents dataset_locks locks rules rules_history requests requests_history subscriptions rses accounts account_limits bad_replicas deleted_dids rse_attr_map"

myname=$(basename "$0")
BASE_PATH=$(util_get_config_val "$myname")
DAILY_BASE_PATH=${BASE_PATH}/$(date +%Y-%m-%d)
START_TIME=$(date +%s)
LOG_FILE=log/$(date +'%F_%H%M%S')_$(basename "$0")
pg_metric_db="RUCIO_PROD"
util4logi "CMSSQOOP_ENV=${CMSSQOOP_ENV}, CMSSQOOP_CONFIGS=${CMSSQOOP_CONFIGS}." >>"$LOG_FILE".stdout

# --------------------------------------------------------------------------------- UTILS
trap 'onFailExit' ERR
onFailExit() {
    util4loge "Finished with error! $(dirname "$0")" >>"$LOG_FILE".stdout
    util4loge "FAILED" >>"$LOG_FILE".stdout
    exit 1
}

# ------------------------------------------------------------------------- DUMP FUNCTION
# Full dump rucio table in avro format
sqoop_full_dump_rucio_cmd() {
    local local_start_time
    local_start_time=$(date +%s)
    trap 'onFailExit' ERR
    kinit -R
    TABLE=$1
    util4logi "${SCHEMA}.${TABLE} : import starting.. "
    pushg_dump_start_time "$myname" "$pg_metric_db" "$SCHEMA" "$TABLE"
    #
    sqoop import \
        -Dmapreduce.job.user.classpath.first=true \
        -Doraoop.timestamp.string=false \
        -Dmapred.child.java.opts="-Djava.security.egd=file:/dev/../dev/urandom" \
        -Ddfs.client.socket-timeout=120000 \
        --username "$USERNAME" --password "$PASSWORD" \
        -z \
        --direct \
        --throw-on-error \
        --connect "$JDBC_URL" \
        --num-mappers 40 \
        --as-avrodatafile \
        --target-dir "${DAILY_BASE_PATH}/${TABLE}" \
        --table "$SCHEMA"."$TABLE" \
        1>>"$LOG_FILE".stdout 2>>"$LOG_FILE".stderr
    #
    util4logi "${SCHEMA}.${TABLE} : import finished successfully in $(util_secs_to_human "$(($(date +%s) - local_start_time))") "
    pushg_dump_end_time "$myname" "$pg_metric_db" "$SCHEMA" "$TABLE"
}
# ---------------------------------------------------------------------------------------
if [ -f /etc/secrets/rucio ]; then
    USERNAME=$(grep username </etc/secrets/rucio | awk '{print $2}')
    PASSWORD=$(grep password </etc/secrets/rucio | awk '{print $2}')
    JDBC_URL=$(grep jdbc_url </etc/secrets/rucio | awk '{print $2}')
else
    echo "[ERROR] Unable to read Rucio credentials" >>"$LOG_FILE".stdout
    exit 1
fi

# Check hadoop executable exist
if ! [ -x "$(command -v hadoop)" ]; then
    echo "[ERROR] It seems 'hadoop' is not exist in PATH! Exiting..." >>"$LOG_FILE".stdout
    exit 1
fi

# Import all tables in order
for TABLE_NAME in $RUCIO_TABLES; do
    sqoop_full_dump_rucio_cmd "$TABLE_NAME" >>"$LOG_FILE".stdout 2>&1
done

# Give read permission to the new folder and sub folders after all dumps finished
hadoop fs -chmod -R o+rx "${DAILY_BASE_PATH}"

# Dumped tables total size in bytes
dump_size=$(util_hdfs_size "$DAILY_BASE_PATH")

# ---------------------------------------------------------------------------- STATISTICS
duration=$(($(date +%s) - START_TIME))
pushg_dump_duration "$myname" "$pg_metric_db" "$SCHEMA" $duration
pushg_dump_size "$myname" "$pg_metric_db" "$SCHEMA" "$dump_size"
util4logi "all finished, time spent: $(util_secs_to_human $duration)" >>"$LOG_FILE".stdout
