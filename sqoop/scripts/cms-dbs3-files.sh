#!/bin/bash
. "${WDIR}/sqoop/scripts/sqoop_utils.sh"
. "${WDIR}/sqoop/scripts/utils.sh"
setJava

TZ=UTC
myname=$(basename "$0")
BASE_PATH=$(util_get_config_val "$myname")
START_TIME=$(date +%s)
SCHEMA="CMS_DBS3_PROD_GLOBAL_OWNER"
LOG_FILE=log/$(date +'%F_%H%M%S')_$myname
# pushgateway table name
custom_pg_table_name="FILES_custom"
pushg_dump_start_time "$myname" "DBS" "$SCHEMA" "$custom_pg_table_name"

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
    "SELECT B.BLOCK_ID, F.LOGICAL_FILE_NAME, F.FILE_SIZE, F.ADLER32 FROM CMS_DBS3_PROD_GLOBAL_OWNER.BLOCKS B JOIN CMS_DBS3_PROD_GLOBAL_OWNER.FILES F ON F.BLOCK_ID = B.BLOCK_ID where ( F.creation_date >= ${START_DATE_S} or F.LAST_MODIFICATION_DATE >= ${START_DATE_S} ) and ( F.creation_date < ${END_DATE_S} and F.LAST_MODIFICATION_DATE < ${END_DATE_S} ) AND \$CONDITIONS" \
    --fields-terminated-by , --escaped-by \\ --optionally-enclosed-by '\"' \
    1>"$LOG_FILE".stdout 2>"$LOG_FILE".stderr

OUTPUT_ERROR=$(grep -E "ERROR tool.ImportTool: Error during import: Import job failed!" <"$LOG_FILE".stderr)
TRANSF_INFO=$(grep -E "INFO mapreduce.ImportJobBase: Transferred" <"$LOG_FILE".stderr)

if [[ $OUTPUT_ERROR == *"ERROR"* || ! $TRANSF_INFO == *"INFO"* ]]; then
    util4loge "Error occurred, check $LOG_FILE"
    exit 1
else
    hdfs dfs -cat $OUTPUT_FOLDER/part-m-00000 | hdfs dfs -appendToFile - $MERGED_FOLDER/part-m-00000
fi

# ---------------------------------------------------------------------------- STATISTICS
duration=$(($(date +%s) - START_TIME))
pushg_dump_duration "$myname" "DBS" "$SCHEMA" $duration
pushg_dump_end_time "$myname" "DBS" "$SCHEMA" "$custom_pg_table_name"
util4logi "all finished, time spent: $(util_secs_to_human $duration)" >>"$LOG_FILE".stdout
