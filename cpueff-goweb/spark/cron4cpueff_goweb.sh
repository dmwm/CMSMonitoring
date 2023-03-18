#!/bin/bash
# Copyright (c) 2023 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>
#
# shellcheck disable=SC2068
set -e
##H Can only run in K8s, you may modify to run in local by arranging env vars
##H
##H cron4cpueff_goweb.sh
##H
##H Arguments:
##H   - keytab              : Kerberos auth file: secrets/kerberos
##H   - p1, p2, host, wdir  : [ALL FOR K8S] p1 and p2 spark required ports(driver and blockManager), host is k8s node dns alias, wdir is working directory
##H   - test                : Use test HDFS output directory
##H
##H Usage Example:
##H    ./cron4cpueff_goweb.sh --keytab ./keytab --last_n_days 30 --wdir $WDIR
##H    ./cron4cpueff_goweb.sh --keytab ./keytab --last_n_days 30 --wdir $WDIR --test
##H
##H How to test:
##H   - You can test just by giving different '--mongowritedb'.
##H   - OR, you can use totally different MongoDB instance
##H
TZ=UTC
START_TIME=$(date +%s)
myname=$(basename "$0")
script_dir="$(cd "$(dirname "$0")" && pwd)"

# Source util functions
. "$script_dir"/utils.sh

if [ "$1" == "" ] || [ "$1" == "-h" ] || [ "$1" == "--help" ] || [ "$1" == "-help" ]; then
    util_usage_help
    exit 0
fi
util_cron_send_start "$myname" "1d"
unset -v KEYTAB_SECRET PORT1 PORT2 K8SHOST WDIR IS_TEST
# ------------------------------------------------------------------------------------------------------------- PREPARE
util_input_args_parser $@

util4logi "Parameters: KEYTAB_SECRET:${KEYTAB_SECRET} PORT1:${PORT1} PORT2:${PORT2} K8SHOST:${K8SHOST} WDIR:${WDIR} IS_TEST:${IS_TEST}"
util_check_vars PORT1 PORT2 K8SHOST MONGO_HOST MONGO_PORT MONGO_ROOT_USERNAME MONGO_ROOT_PASSWORD
util_setup_spark_k8s

# Check commands/CLIs exist
util_check_cmd mongoimport
util_check_cmd mongosh

KERBEROS_USER=$(util_kerberos_auth_with_keytab "$KEYTAB_SECRET")
util4logi "authenticated with Kerberos user: ${KERBEROS_USER}"

# Remove trailing slash if exists
HDFS_PATH="/tmp/${KERBEROS_USER%/}/prod/cpueff"
if [ "$IS_TEST" != "" ]; then
    HDFS_PATH="/tmp/${KERBEROS_USER%/}/test/cpueff"
fi

# Details of environment variables
export MONGO_WRITE_DB="cpueff"
export MONGO_AUTH_DB="admin"

#export MONGO_HOST
#export MONGO_PORT
#export MONGO_ROOT_USERNAME
#export MONGO_ROOT_PASSWORD
# ------------------------------------------------------------------------------------------------------- RUN SPARK JOB
# arg1: PySpark python file to run
# arg2: hdfs output directory
function run_spark() {
    export PYTHONPATH=$script_dir:${WDIR}/CMSSpark/src/python:$PYTHONPATH
    # Required for Spark job in K8s
    spark_py_file=$1
    hdfs_out_dir=$2
    # 1 day buffer and last 30 days of data
    end_date=$(date -d '1 day ago' +"%Y-%m-%d")
    start_date=$(date -d '31 day ago' +"%Y-%m-%d")

    util4logi "Spark Job for ${spark_py_file} starting... between ${start_date} - ${end_date}"
    spark_submit_args=(
        --master yarn --conf spark.ui.showConsoleProgress=false --conf spark.sql.session.timeZone=UTC
        --conf "spark.driver.bindAddress=0.0.0.0" --conf "spark.driver.host=${K8SHOST}"
        --conf "spark.driver.port=${PORT1}" --conf "spark.driver.blockManager.port=${PORT2}"
        --driver-memory=8g --executor-memory=8g --packages org.apache.spark:spark-avro_2.12:3.3.1
    )

    py_input_args=(--hdfs_out_dir "$hdfs_out_dir" --start_date "$start_date" --end_date "$end_date")

    # Run
    spark-submit "${spark_submit_args[@]}" "${spark_py_file}" "${py_input_args[@]}" 2>&1
    # ------------------------------------------------------------------------------------ POST OPS
    # Give read access to new dumps for all users
    #hadoop fs -chmod -R o+rx "$hdfs_out_dir"/
    util4logi "Spark results are written to ${hdfs_out_dir}"
    util4logi "Spark Job for ${spark_py_file} finished."
}

run_spark "$script_dir/cpueff_condor_goweb.py" "$HDFS_PATH" 2>&1
run_spark "$script_dir/cpueff_stepchain_goweb.py" "$HDFS_PATH" 2>&1

# ------------------------------------------------------------------------------------ MONGODB INDEX CREATION
# Modify JS script
sed -i "s/_MONGOWRITEDB_/$MONGO_WRITE_DB/g" "$script_dir"/createindexes.js

mongosh --host "$MONGO_HOST" --port "$MONGO_PORT" --username "$MONGO_ROOT_USERNAME" --password "$MONGO_ROOT_PASSWORD" \
    --authenticationDatabase "$MONGO_AUTH_DB" <"$script_dir"/createindexes.js
util4logi "MongoDB indexes are created for condor_main, condor_detailed, sc_task and sc_task_cmsrun_jobtype_site collections"

# Print process wall clock time
duration=$(($(date +%s) - START_TIME))
util_cron_send_end "$myname" "1d" 0
util4logi "all finished, time spent: $(util_secs_to_human $duration)"
