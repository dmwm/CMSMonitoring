#!/bin/bash
set -e
# Reference: some features&logics are copied from sqoop_utils.sh and cms-dbs3-full-copy.sh

##H rucio_ds_daily_stats.sh imports required tables for Rucio datasets daily statistics time series data
##H
##H Usage: cd /data/sqoop; ./run.sh ./scripts/rucio_ds_daily_stats.sh
##H
##H Script actions:
##H   None
##H
##H Requirements:
##H   - cmsr_cstring [file]
##H       should be exist in /data/sqoop directory
##H       K8s secret name: rucio-secrets
##H       Origin: secrets/sqoop/cmsr_cstring
##H   - /etc/secrets/rucio [file]
##H       K8s secret name: rucio-secrets
##H       Origin: secrets/rucio/rucio
##H   - cmssqoop user kerberos ticket to access /tmp/cmssqoop hdfs directory
##H

if [ ! -f cmsr_cstring ]; then
    echo "cmsr_cstring file does not exist in the path!"
    exit 1
fi
if [ ! -f /etc/secrets/rucio ]; then
    echo "/etc/secrets/rucio does not exist!"
    exit 1
fi

BASE_PATH_PREFIX=/tmp/cmssqoop/rucio_daily_stats
BASE_PATH="$BASE_PATH_PREFIX"-$(date +%Y-%m-%d)

# DBS and Rucio use same jdbc url
JDBC_URL=$(sed '1q;d' cmsr_cstring)

# Create lof folder in current directory if not exist
mkdir -p log
LOG_FILE=log/$(date +'%F_%H%m%S')_$(basename "$0")

trap 'onFailExit' ERR
onFailExit() {
    echo "$(date --rfc-3339=seconds)" "[ERROR] Finished with error! See logs: ${LOG_FILE}.stdout and ${LOG_FILE}.stderr" >>"$LOG_FILE".stdout
    exit 1
}

function dbs_dumps() {
    trap 'onFailExit' ERR
    dbs_username=$(sed '2q;d' cmsr_cstring)
    dbs_password=$(sed '3q;d' cmsr_cstring)
    SCHEMA=CMS_DBS3_PROD_GLOBAL_OWNER

    # The tables that split-by column set as Null and -m=1 are so small tables, no need to the parallel import
    tables_dict="table split_column num_mappers
FILES,dataset_id,40
DATASETS,dataset_id,4
DATA_TIERS,Null,1
PHYSICS_GROUPS,Null,1
ACQUISITION_ERAS,Null,1
DATASET_ACCESS_TYPES,Null,1"

    {
        # Skip first line
        read -r
        while IFS=, read -r table split_column num_mappers; do
            # If split-by column set as Null, use only num_mapper
            if [[ "$split_column" == "Null" ]]; then
                extra_args=(-m "$num_mappers")
            else
                extra_args=(-m "$num_mappers" --split-by "$split_column")
            fi

            echo "$(date --rfc-3339=seconds)" [INFO] Start: "$SCHEMA"."$table" with "${extra_args[@]}"
            /usr/hdp/sqoop/bin/sqoop import \
                -Dmapreduce.job.user.classpath.first=true \
                -Doraoop.timestamp.string=false \
                -Dmapred.child.java.opts="-Djava.security.egd=file:/dev/../dev/urandom" \
                -Ddfs.client.socket-timeout=120000 \
                --username "$dbs_username" --password "$dbs_password" \
                --direct \
                --compress \
                --throw-on-error \
                --connect "$JDBC_URL" \
                --as-avrodatafile \
                --target-dir "${BASE_PATH}/${table}/" \
                --table "$SCHEMA"."$table" \
                "${extra_args[@]}" \
                1>>"$LOG_FILE".stdout 2>>"$LOG_FILE".stderr
            echo "$(date --rfc-3339=seconds)" [INFO] End: "$SCHEMA"."$table"
        done
    } <<<"$tables_dict"
    hadoop fs -chmod -R o+rx "$BASE_PATH"/
}

function rucio_dumps() {
    trap 'onFailExit' ERR
    rucio_username=$(grep username </etc/secrets/rucio | awk '{print $2}')
    rucio_password=$(grep password </etc/secrets/rucio | awk '{print $2}')

    SCHEMA=CMS_RUCIO_PROD

    tables_dict="table split_column num_mappers
REPLICAS,rse_id,40
CONTENTS,Null,40"

    {
        # Skip first line
        read -r
        while IFS=, read -r table split_column num_mappers; do
            # If split-by column set as Null, use only num_mapper
            if [[ "$split_column" == "Null" ]]; then
                extra_args=(-m "$num_mappers")
            else
                extra_args=(-m "$num_mappers" --split-by "$split_column")
            fi

            echo "$(date --rfc-3339=seconds)" [INFO] Start: "$SCHEMA"."$table" with "${extra_args[@]}"
            /usr/hdp/sqoop/bin/sqoop import \
                -Dmapreduce.job.user.classpath.first=true \
                -Doraoop.timestamp.string=false \
                -Dmapred.child.java.opts="-Djava.security.egd=file:/dev/../dev/urandom" \
                -Ddfs.client.socket-timeout=120000 \
                --username "$rucio_username" --password "$rucio_password" \
                --direct \
                --compress \
                --throw-on-error \
                --connect "$JDBC_URL" \
                --as-avrodatafile \
                --target-dir "${BASE_PATH}/${table}/" \
                --table "$SCHEMA"."$table" \
                "${extra_args[@]}" \
                1>>"$LOG_FILE".stdout 2>>"$LOG_FILE".stderr

            echo "$(date --rfc-3339=seconds)" [INFO] End: "$SCHEMA"."$table"
        done
    } <<<"$tables_dict"
    hadoop fs -chmod -R o+rx "$BASE_PATH"/
}

# Run parallel
dbs_dumps >>"$LOG_FILE".stdout 2>&1 &
rucio_dumps >>"$LOG_FILE".stdout 2>&1 &
wait

# Delete yesterdays dumps
path_of_yesterday="$BASE_PATH_PREFIX"-"$(date -d "yesterday" '+%Y-%m-%d')"
hadoop fs -rm -r -f -skipTrash "$path_of_yesterday"
echo "$(date --rfc-3339=seconds)" "[INFO] Dump of yesterday is deleted ${path_of_yesterday}" >>"$LOG_FILE".stdout 2>&1

echo "$(date --rfc-3339=seconds)" "[INFO] All finished" >>"$LOG_FILE".stdout 2>&1
