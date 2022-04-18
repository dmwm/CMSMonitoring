#!/bin/bash
set -e
# Reference: some features/logics are copied from sqoop_utils.sh and cms-dbs3-full-copy.sh

# Imports CMS_RUCIO_PROD tables
BASE_PATH=/project/awg/cms/rucio/
JDBC_URL=jdbc:oracle:thin:@cms-nrac-scan.cern.ch:10121/CMSR_CMS_NRAC.cern.ch
SCHEMA="CMS_RUCIO_PROD"
RUCIO_TABLES="dids contents rules dataset_locks locks rses"

LOG_FILE=log/$(date +'%F_%H%m%S')_$(basename "$0")
TZ=UTC

####
trap 'onFailExit' ERR
onFailExit() {
    echo "Finished with error!" >>"$LOG_FILE".stdout
    echo "FAILED" >>"$LOG_FILE".stdout
    exit 1
}

####
if [ -f /etc/secrets/rucio ]; then
    USERNAME=$(grep username </etc/secrets/rucio | awk '{print $2}')
    PASSWORD=$(grep password </etc/secrets/rucio | awk '{print $2}')
else
    echo "[ERROR] Unable to read Rucio credentials" >>"$LOG_FILE".stdout
    exit 1
fi

# Check hadoop executable exist
if ! [ -x "$(command -v hadoop)" ]; then
    echo "[ERROR] It seems 'hadoop' is not exist in PATH! Exiting..." >>"$LOG_FILE".stdout
    exit 1
fi

# Full dump rucio table in avro format
sqoop_full_dump_rucio_cmd() {
    trap 'onFailExit' ERR
    kinit -R
    TABLE=$1
    echo "[INFO] ${SCHEMA}.${TABLE} : import starting.. "
    /usr/hdp/sqoop/bin/sqoop import \
        -Dmapreduce.job.user.classpath.first=true \
        -Doraoop.timestamp.string=false \
        -Dmapred.child.java.opts="-Djava.security.egd=file:/dev/../dev/urandom" \
        -Ddfs.client.socket-timeout=120000 \
        --username "$USERNAME" --password "$PASSWORD" \
        -z \
        --direct \
        --throw-on-error \
        --connect $JDBC_URL \
        --num-mappers 40 \
        --as-avrodatafile \
        --target-dir $BASE_PATH"$(date +%Y-%m-%d)"/"$TABLE" \
        --table "$SCHEMA"."$TABLE" \
        1>>"$LOG_FILE".stdout 2>>"$LOG_FILE".stderr
    hadoop fs -chmod -R o+rx $BASE_PATH"$(date +%Y-%m-%d)"
    echo "[INFO] ${SCHEMA}.${TABLE} : import finished successfully.. "
}

# Import all tables in order
for TABLE_NAME in $RUCIO_TABLES; do
    sqoop_full_dump_rucio_cmd "$TABLE_NAME" >>"$LOG_FILE".stdout 2>&1
done
