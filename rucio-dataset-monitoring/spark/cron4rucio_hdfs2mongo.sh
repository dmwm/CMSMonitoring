#!/bin/bash
# Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>
# 
# shellcheck disable=SC2068
set -e
##H Can only run in K8s, you may modify to run in local by arranging env vars
##H
##H cron4rucio_hdfs2mongo.sh
##H    Gets "datasets" and "detailed_datasets" HDFS results to LOCAL directory and send to MongoDB.
##H
##H Arguments:
##H   - keytab        : Kerberos auth file to connect Spark Analytix cluster (cmsmonit)
##H   - hdfs          : HDFS_PATH output path that include spark job results. Mongoimport will import them.
##H   - mongohost     : MongoDB host
##H   - mongoport     : MongoDB port
##H   - mongouser     : MongoDB user which has write access to required MongoDB database/collection
##H   - mongopass     : MongoDB user password
##H   - mongowritedb  : MongoDB database name that results will be written
##H   - mongoauthdb   : MongoDB database for authentication. Required for mongoimport `--authenticationDatabase` argument
##H   - wdir          : working directory
##H
##H Usage Example:
##H    ./cron4rucio_hdfs2mongo.sh --keytab ./keytab --hdfs /tmp/cmsmonit --mongohost $MONGO_HOST --mongoport $MONGO_PORT \
##H        --mongouser $MONGO_ROOT_USERNAME --mongopass $MONGO_ROOT_PASSWORD --mongowritedb rucio --mongoauthdb admin --wdir $WDIR
##H
##H How to test:
##H   - You can test just by giving different '--mongowritedb'.
##H   - OR, you can use totally different MongoDB instance
##H   - [!!!] You should to use same collection name and MongoDB instance in Go web service too.
##H
TZ=UTC
START_TIME=$(date +%s)
myname=$(basename "$0")
script_dir="$(cd "$(dirname "$0")" && pwd)"

# MongoDB collection names
col_main_datasets="main_datasets"
col_detailed_datasets="detailed_datasets"
col_datasets_in_tape_and_disk="datasets_in_tape_and_disk"

# Temporary HDFS result directory
hdfs_main_datasets="main"
hdfs_detailed_datasets="detailed"
hdfs_datasets_in_tape_and_disk="in_tape_and_disk"

# Source util functions
. "$script_dir"/utils.sh

if [ "$1" == "" ] || [ "$1" == "-h" ] || [ "$1" == "--help" ] || [ "$1" == "-help" ]; then
    util_usage_help
    exit 0
fi
util_cron_send_start "$myname" "1d"
unset -v KEYTAB_SECRET HDFS_PATH ARG_MONGOHOST ARG_MONGOPORT ARG_MONGOUSER ARG_MONGOPASS ARG_MONGOWRITEDB ARG_MONGOAUTHDB WDIR help
# ------------------------------------------------------------------------------------------------------------- PREPARE
util4datasetmon_input_args_parser $@

util4logi "Parameters: KEYTAB_SECRET:${KEYTAB_SECRET} HDFS_PATH:${HDFS_PATH} ARG_MONGOHOST:${ARG_MONGOHOST} ARG_MONGOPORT:${ARG_MONGOPORT} ARG_MONGOUSER:${ARG_MONGOUSER} ARG_MONGOWRITEDB:${ARG_MONGOWRITEDB} ARG_MONGOAUTHDB:${ARG_MONGOAUTHDB} WDIR:${WDIR}"
util_check_vars HDFS_PATH ARG_MONGOHOST ARG_MONGOPORT ARG_MONGOUSER ARG_MONGOPASS ARG_MONGOWRITEDB ARG_MONGOAUTHDB WDIR
util_setup_spark_k8s

# Check commands/CLIs exist
util_check_cmd mongoimport
util_check_cmd mongosh

KERBEROS_USER=$(util_kerberos_auth_with_keytab "$KEYTAB_SECRET")
util4logi "authenticated with Kerberos user: ${KERBEROS_USER}"

# ---------------------------------------------------------------------------------------------------- RUN MONGO IMPORT
# arg1: hdfs output directory
# arg2: mongodb collection name [datasets or detailed_datasets]
function run_mongo_import() {
    hdfs_out_dir=$1
    collection=$2
    # Local directory in K8s pod to store Spark results which will be copied from HDFS.
    # Create directory if not exist, delete file if exists.
    local_json_merge_dir=$WDIR/results
    mkdir -p "$local_json_merge_dir"
    local_json_merge_file=$local_json_merge_dir/"${collection}.json"
    rm -rf "$local_json_merge_file"

    # Copy files from HDFS to LOCAL directory as a single file
    hadoop fs -getmerge "$hdfs_out_dir"/part-*.json "$local_json_merge_file"

    mongoimport --drop --type=json \
        --host "$ARG_MONGOHOST" --port "$ARG_MONGOPORT" --username "$ARG_MONGOUSER" --password "$ARG_MONGOPASS" \
        --authenticationDatabase "$ARG_MONGOAUTHDB" --db "$ARG_MONGOWRITEDB" \
        --collection "$collection" --file "$local_json_merge_file"
    util4logi "Mongoimport finished. ${hdfs_out_dir} imported to collection: ${collection}"
}
# Remove trailing slash if exists
HDFS_PATH="${HDFS_PATH%/}/rucio_ds_for_mongo/$(date +%Y-%m-%d)"

###################### Import main datasets
# Arrange a temporary HDFS directory that current Kerberos user can use for datasets collection
run_mongo_import "${HDFS_PATH}/${hdfs_main_datasets}" "$col_main_datasets" 2>&1

###################### Import detailed datasets
run_mongo_import "${HDFS_PATH}/${hdfs_detailed_datasets}" "$col_detailed_datasets" 2>&1

###################### Import datasets in both Tape and Disk
run_mongo_import "${HDFS_PATH}/${hdfs_datasets_in_tape_and_disk}" "$col_datasets_in_tape_and_disk" 2>&1

# ---------------------------------------------------------------------------------------- SOURCE TIMESTAMP MONGOIMPORT
# Write current date to json file and import it to MongoDB "source_timestamp" collection for Go Web Page.
echo "{\"createdAt\": \"$(date +%Y-%m-%d)\"}" >source_timestamp.json
mongoimport --drop --type=json \
    --host "$ARG_MONGOHOST" --port "$ARG_MONGOPORT" --username "$ARG_MONGOUSER" --password "$ARG_MONGOPASS" \
    --authenticationDatabase "$ARG_MONGOAUTHDB" --db "$ARG_MONGOWRITEDB" \
    --collection "source_timestamp" --file source_timestamp.json

util4logi "source_timestamp collection is updated with current date"

# ---------------------------------------------------------------------------------------------- MongoDB INDEX CREATION
# Modify JS script
sed -i "s/_MONGOWRITEDB_/$ARG_MONGOWRITEDB/g" "$script_dir"/createindexes.js

mongosh --host "$ARG_MONGOHOST" --port "$ARG_MONGOPORT" --username "$ARG_MONGOUSER" --password "$ARG_MONGOPASS" \
    --authenticationDatabase "$ARG_MONGOAUTHDB" <"$script_dir"/createindexes.js
util4logi "MongoDB indexes are created for datasets and detailed_datasets collections"

# -------------------------------------------------------------------------------------------------------------- FINISH
# Print process wall clock time
duration=$(($(date +%s) - START_TIME))
util_cron_send_end "$myname" "1d" 0
util4logi "all finished, time spent: $(util_secs_to_human $duration)"
