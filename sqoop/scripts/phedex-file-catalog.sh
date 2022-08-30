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
pushg_dump_start_time "$myname" "PHEDEX" "$SCHEMA"

# --------------------------------------------------------------------------------- START
JDBC_URL=$(sed '1q;d' /etc/secrets/cmsr_cstring)
USERNAME=$(sed '2q;d' /etc/secrets/cmsr_cstring)
PASSWORD=$(sed '3q;d' /etc/secrets/cmsr_cstring)

if [ -n "$1" ]; then
    START_DATE=$1
else
    START_DATE=$(date +'%F' -d "1 day ago")
fi
END_DATE=$(date +'%F' -d "$START_DATE + 1 day")

START_DATE_S=$(date +'%s' -d "$START_DATE")
END_DATE_S=$(date +'%s' -d "$END_DATE")

OUTPUT_FOLDER=$BASE_PATH/diff/date=$START_DATE
MERGED_FOLDER=$BASE_PATH/merged
{
    echo "Timerange: $START_DATE to $END_DATE"
    echo "Folder: $OUTPUT_FOLDER"
    echo "querying..."
} >>"$LOG_FILE".cron

sqoop import \
    -Dmapreduce.job.user.classpath.first=true -Ddfs.client.socket-timeout=120000 \
    --direct --connect "$JDBC_URL" --fetch-size 10000 --username "$USERNAME" --password "$PASSWORD" --target-dir "$OUTPUT_FOLDER" \
    -m 1 --query \
    "select ds.name as dataset_name, ds.id as dataset_id, ds.is_open as dataset_is_open, ds.time_create as dataset_time_create, bk.name as block_name, bk.id as block_id, bk.time_create as block_time_create, bk.is_open as block_is_open, f.logical_name as file_lfn, f.id as file_id, f.filesize, f.checksum, f.time_create as file_time_create from cms_transfermgmt.t_dps_dataset ds join cms_transfermgmt.t_dps_block bk on bk.dataset=ds.id join cms_transfermgmt.t_dps_file f on f.inblock=bk.id where f.time_create >= ${START_DATE_S} and f.time_create < ${END_DATE_S} and \$CONDITIONS" \
    --fields-terminated-by , --escaped-by \\ --optionally-enclosed-by '\"' \
    1>"$LOG_FILE".stdout 2>"$LOG_FILE".stderr

OUTPUT_ERROR=$(grep -E "ERROR tool.ImportTool: Error during import: Import job failed!" <"$LOG_FILE".stderr)
TRANSF_INFO=$(grep -E "INFO mapreduce.ImportJobBase: Transferred" <"$LOG_FILE".stderr)

if [[ $OUTPUT_ERROR == *"ERROR"* || ! $TRANSF_INFO == *"INFO"* ]]; then
    util4loge "Error occurred, check $LOG_FILE"
    exit 1
else
    hdfs dfs -cat "$OUTPUT_FOLDER"/part-m-00000 | hdfs dfs -appendToFile - "$MERGED_FOLDER"/part-m-00000
fi

# ---------------------------------------------------------------------------- STATISTICS
duration=$(($(date +%s) - START_TIME))
pushg_dump_end_time "$myname" "PHEDEX" "$SCHEMA"
pushg_dump_duration "$myname" "PHEDEX" "$SCHEMA" $duration
util4logi "all finished, time spent: $(util_secs_to_human $duration)"
