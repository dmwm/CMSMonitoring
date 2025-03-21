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
# Returns left part of the dot containing string
# Arguments:
#   arg1: string
#######################################
function util_dotless_name() {
    echo "$1" | cut -f1 -d"."
}

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
    local pushgateway_url env script db schema value dotless_script_name
    pushgateway_url=$(util_get_config_val PUSHGATEWAY_URL)
    env=${CMSSQOOP_ENV:test}
    script=$1
    db=$2
    schema=$3
    value=$4
    dotless_script_name=$(util_dotless_name "$script")
    cat <<EOF | curl -s --data-binary @- "${pushgateway_url}/metrics/job/cms-sqoop/instance/$(hostname)"
# TYPE cms_sqoop_dump_duration_${dotless_script_name} gauge
# HELP cms_sqoop_dump_duration_${dotless_script_name} Total duration of sqoop dump in seconds.
cms_sqoop_dump_duration_${dotless_script_name}{env="${env}", script="${script}", db="${db}", schema="${schema}"} $value
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
    local pushgateway_url env script db schema value dotless_script_name
    pushgateway_url=$(util_get_config_val PUSHGATEWAY_URL)
    env=${CMSSQOOP_ENV:test}
    script=$1
    db=$2
    schema=$3
    value=$4
    dotless_script_name=$(util_dotless_name "$script")
    cat <<EOF | curl -s --data-binary @- "${pushgateway_url}/metrics/job/cms-sqoop/instance/$(hostname)"
# TYPE cms_sqoop_dump_size_${dotless_script_name} gauge
# HELP cms_sqoop_dump_size_${dotless_script_name} Total HDFS size of sqoop dumps in bytes.
cms_sqoop_dump_size_${dotless_script_name}{env="${env}", script="${script}", db="${db}", schema="${schema}"} $value
EOF
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
