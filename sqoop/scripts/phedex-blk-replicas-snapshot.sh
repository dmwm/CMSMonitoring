#!/bin/bash
. "${WDIR}/sqoop/scripts/sqoop_utils.sh"
. "${WDIR}/sqoop/scripts/utils.sh"
setJava

TZ=UTC
myname=$(basename "$0")
BASE_PATH=$(util_get_config_val "$myname")
START_TIME=$(date +%s)
SCHEMA="CMS_TRANSFERMGMT"
LOG_FILE=log/$(date +'%F_%H%M%S')_$myname
# PG will keep table name as script name since this is a custom import
pushg_dump_start_time "$myname" "PHEDEX" "$SCHEMA" "CMS_TRANSFERMGMT_replicas"

# --------------------------------------------------------------------------------- START
#BASE_PATH="transfermgmt"
JDBC_URL=$(sed '1q;d' /etc/secrets/cmsr_cstring)
USERNAME=$(sed '2q;d' /etc/secrets/cmsr_cstring)
PASSWORD=$(sed '3q;d' /etc/secrets/cmsr_cstring)

TIME=$(date +'%Hh%mm%Ss')

OUTPUT_FOLDER=$BASE_PATH/time=$(date +'%F')_${TIME}
{
    echo "Folder: $OUTPUT_FOLDER"
    echo "querying..."
} >>"$LOG_FILE".cron

sqoop import \
    -Dmapreduce.job.user.classpath.first=true -Ddfs.client.socket-timeout=120000 \
    --direct --connect "$JDBC_URL" --fetch-size 10000 --username "$USERNAME" --password "$PASSWORD" --target-dir "$OUTPUT_FOLDER" \
    -m 1 --query \
    "select cms_transfermgmt.now, ds.name as dataset_name, ds.id as dataset_id, ds.is_open as dataset_is_open, ds.time_create as dataset_time_create, ds.time_update as dataset_time_update, bk.name as block_name, bk.id as block_id, bk.files as block_files, bk.bytes as block_bytes, bk.is_open as block_is_open, bk.time_create as block_time_create, bk.time_update as block_time_update, n.name as node_name, n.id as node_id, br.is_active, br.src_files, br.src_bytes, br.dest_files, br.dest_bytes, br.node_files, br.node_bytes, br.xfer_files, br.xfer_bytes, br.is_custodial, br.user_group, br.time_create as replica_time_create, br.time_update as replica_time_update from cms_transfermgmt.t_dps_dataset ds join cms_transfermgmt.t_dps_block bk on bk.dataset=ds.id join cms_transfermgmt.t_dps_block_replica br on br.block=bk.id join cms_transfermgmt.t_adm_node n on n.id=br.node and \$CONDITIONS" \
    --fields-terminated-by , --escaped-by \\ --optionally-enclosed-by '\"' \
    1>"$LOG_FILE".stdout 2>"$LOG_FILE".stderr

OUTPUT_ERROR=$(grep -E "ERROR tool.ImportTool: Error during import: Import job failed!" <"$LOG_FILE".stderr)
TRANSF_INFO=$(grep -E "INFO mapreduce.ImportJobBase: Transferred" <"$LOG_FILE".stderr)

if [[ $OUTPUT_ERROR == *"ERROR"* || ! $TRANSF_INFO == *"INFO"* ]]; then
    util4loge "Error occurred, check $LOG_FILE"
    exit 1
fi

# ---------------------------------------------------------------------------- STATISTICS
duration=$(($(date +%s) - START_TIME))
pushg_dump_duration "$myname" "PHEDEX" "$SCHEMA" $duration
pushg_dump_end_time "$myname" "PHEDEX" "$SCHEMA" "CMS_TRANSFERMGMT_replicas"
util4logi "all finished, time spent: $(util_secs_to_human $duration)" >>"$LOG_FILE".stdout
