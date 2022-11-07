#!/bin/bash
# Sqoop utils

# -------------------------------------- LOGGING UTILS --------------------------------------------
# info log function
function util4logi() {
    echo "$(date --rfc-3339=seconds) [INFO]" "$@"
}
# warn log function
function util4logw() {
    echo "$(date --rfc-3339=seconds) [WARN]" "$@"
}
# error log function
function util4loge() {
    echo "$(date --rfc-3339=seconds) [ERROR]" "$@"
}
# -------------------------------------------------------------------------------------------------

# -------------------------------------- GET CONFIG UTILS -----------------------------------------
#######################################
# util to get values from sqoop/configs.json
#
# Uses $CMSSQOOP_CONFIGS json configs, can be defined as $WDIR/sqoop/configs.json but NO default for compatibility.
#
#  Arguments:
#    $1: Any key for $CMSSQOOP_CONFIGS JSON file
#  Returns:
#    if fails, exits with exit code 1
#######################################
function util_get_config_val() {
    local key
    key=$1
    # Get configs.json from environment variable
    # Safe for keys which include dot or any other reserved keywords
    value=$(jq --exit-status -r --arg k "$key" '.[$k]' <"$CMSSQOOP_CONFIGS")
    ec=$?
    if [ "$ec" -eq 0 ]; then
        echo "$value"
    else
        util4loge "Could not extract value -${key}- from ${CMSSQOOP_CONFIGS}. Exiting..."
        echo 0
    fi
}
# -------------------------------------------------------------------------------------------------

# ----------------------------------- PUSHGATEWAY UTILS --------------------------------------------
#######################################
# Send sqoop start time metric to pg
# Arguments:
#   arg1: cron job script name ($0)
#   arg2: database that dumped (Rucio/DBS)
#   arg3: schema of the tables that are dumped
#   arg4: table that is dumped
# Metric name schema: cms_sqoop_dump_start_${db}_${table}
#######################################
function pushg_dump_start_time() {
    local pushgateway_url env script db schema table value
    pushgateway_url=$(util_get_config_val PUSHGATEWAY_URL)
    env=${CMSSQOOP_ENV:test}
    script=$1
    db=$2
    schema=$3
    table=$4
    value=$(date +'%s')
    cat <<EOF | curl -s --data-binary @- "${pushgateway_url}/metrics/job/cms-sqoop/instance/$(hostname)"
# TYPE cms_sqoop_dump_start_${db}_${table} gauge
# HELP cms_sqoop_dump_start_${db}_${table} Dump start time in UTC seconds.
cms_sqoop_dump_start_${db}_${table}{env="${env}", script="${script}", db="${db}", schema="${schema}", table="${table}"} $value
EOF
}

#######################################
# Send sqoop end time metric to pg
# Arguments:
#   arg1: cron job script name ($0)
#   arg2: database that dumped (Rucio/DBS)
#   arg3: schema of the tables that are dumped
#   arg4: table that is dumped
# Metric name schema: cms_sqoop_dump_end_${db}_${table}
#######################################
function pushg_dump_end_time() {
    local pushgateway_url env script db schema table value
    pushgateway_url=$(util_get_config_val PUSHGATEWAY_URL)
    env=${CMSSQOOP_ENV:test}
    script=$1
    db=$2
    schema=$3
    table=$4
    value=$(date +'%s')
    cat <<EOF | curl -s --data-binary @- "${pushgateway_url}/metrics/job/cms-sqoop/instance/$(hostname)"
# TYPE cms_sqoop_dump_end_${db}_${table} gauge
# HELP cms_sqoop_dump_end_${db}_${table} Dump end time in UTC seconds.
cms_sqoop_dump_end_${db}_${table}{env="${env}", script="${script}", db="${db}", schema="${schema}", table="${table}"} $value
EOF
}

#######################################
# Send sqoop dump duration metric to pg
# Arguments:
#   arg1: cron job script name ($0)
#   arg2: database that dumped (Rucio/DBS)
#   arg3: schema of the tables that are dumped
#   arg4: value duration in seconds
#######################################
function pushg_dump_duration() {
    local pushgateway_url env script db schema value
    pushgateway_url=$(util_get_config_val PUSHGATEWAY_URL)
    env=${CMSSQOOP_ENV:test}
    script=$1
    db=$2
    schema=$3
    value=$4
    cat <<EOF | curl -s --data-binary @- "${pushgateway_url}/metrics/job/cms-sqoop/instance/$(hostname)"
# TYPE cms_sqoop_dump_duration gauge
# HELP cms_sqoop_dump_duration Total duration of sqoop dump in seconds.
cms_sqoop_dump_duration{env="${env}", script="${script}", db="${db}", schema="${schema}"} $value
EOF
}

#######################################
# Send total dumped tables size in HDFS metric to pg
# Arguments:
#   arg1: cron job script name ($0)
#   arg2: database that dumped (Rucio/DBS)
#   arg3: schema of the tables that are dumped
#   arg4: value size in bytes
#######################################
function pushg_dump_size() {
    local pushgateway_url env script db schema value
    pushgateway_url=$(util_get_config_val PUSHGATEWAY_URL)
    env=${CMSSQOOP_ENV:test}
    script=$1
    db=$2
    schema=$3
    value=$4
    cat <<EOF | curl -s --data-binary @- "${pushgateway_url}/metrics/job/cms-sqoop/instance/$(hostname)"
# TYPE cms_sqoop_dump_size gauge
# HELP cms_sqoop_dump_size Total HDFS size of sqoop dumps in bytes.
cms_sqoop_dump_size{env="${env}", script="${script}", db="${db}", schema="${schema}"} $value
EOF
}

#######################################
# Send number of dumped tables metric to pg
# Arguments:
#   arg1: cron job script name ($0)
#   arg2: database that dumped (Rucio/DBS)
#   arg3: schema of the tables that are dumped
#   arg4: value number of tables dumped
#######################################
function pushg_dump_table_count() {
    local pushgateway_url env script db schema value
    pushgateway_url=$(util_get_config_val PUSHGATEWAY_URL)
    env=${CMSSQOOP_ENV:test}
    script=$1
    db=$2
    schema=$3
    value=$4
    cat <<EOF | curl -s --data-binary @- "${pushgateway_url}/metrics/job/cms-sqoop/instance/$(hostname)"
# TYPE cms_sqoop_dump_table_count gauge
# HELP cms_sqoop_dump_table_count Total number of tables are dumped.
cms_sqoop_dump_table_count{env="${env}", script="${script}", db="${db}", schema="${schema}"} $value
EOF
}

#######################################
# Test function to test metrics
#######################################
function test_send_metrics_to_pg() {
    # dump duration is 10 minutes
    pushg_dump_duration "test.sh" "TESTDB" "TEST_SCHEMA" 600

    # dumped file size in HDFS : hadoop fs -du -s /FOO | awk -F' ' '{print $1}'
    pushg_dump_size "test.sh" "TESTDB" "TEST_SCHEMA" 1000000

    # number of tables are dumped in this script is 11
    pushg_dump_table_count "test.sh" "TESTDB" "TEST_SCHEMA" 11
}
# -------------------------------------------------------------------------------------------------

# ----------------------------------------- OTHER UTILS -------------------------------------------
#######################################
# util to convert seconds to h, m, s format used in logging
#  Arguments:
#    arg1: seconds in integer
#  Usage:
#    util_secs_to_human SECONDS
#    util_secs_to_human 1000 # returns: 0h 16m 40s
#  Returns:
#    '[\d+]h [\d+]m [\d+]s' , assuming [\d+] integer values
#######################################
function util_secs_to_human() {
    echo "$((${1} / 3600))h $(((${1} / 60) % 60))m $((${1} % 60))s"
}

#######################################
# util to get HDFS directory size in bytes
#  Arguments:
#    $1: HDFS directory
#  Returns:
#    if fails, returns 0, else it returns HDFS folder size in bytes
#######################################
function util_hdfs_size() {
    local size
    size="$(hadoop fs -du -s "$1" | awk -F' ' '{print $1}' 2>/dev/null)"
    ec=$?
    if [ "$ec" -eq 0 ]; then
        echo "$size"
    else
        echo 0
    fi
}

#######################################
# util to check if table exists and contains data using special select count query
#  Arguments:
#    $1: SCHEMA.TABLE
#    $2: jdbc_url
#    $3: username
#    $4: password
#  Usage:
#    check_table_exist SCHEMA.TABLE JDBC_URL USERNAME PASSWORD
#  Returns:
#    0: OKAY, 1: NO DATA IN TABLE or ERROR
#######################################
check_table_exist() {
    local table jdbc_url username password result
    table=$1
    jdbc_url=$2
    username=$3
    password=$4
    # Redirect error logs to /dev/null. If count query returns 1, it means there is data, else no data or error.
    result=$(/usr/hdp/sqoop/bin/sqoop eval --connect "$jdbc_url" --username "$username" --password "$password" --query "select count(*) from ${table} where rownum<=1" 2>/dev/null)
    if echo "$result" | grep -q "1"; then
        echo 0
    else
        echo 1
    fi
}
# -------------------------------------------------------------------------------------------------
