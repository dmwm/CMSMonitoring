#!/bin/bash
# Utils for datasetmon scripts
# Copyright (c) 2022 - Ceyhun Uzunoglu <ceyhunuzngl AT gmail dot com>

function util4datasetmon_input_args_parser() {
    unset -v KEYTAB_SECRET HDFS_PATH ARG_MONGOHOST ARG_MONGOPORT ARG_MONGOUSER ARG_MONGOPASS ARG_MONGOWRITEDB ARG_MONGOAUTHDB PORT1 PORT2 K8SHOST WDIR help
    # Dictionary to keep variables
    declare -A arr

    PARSED_ARGS=$(getopt --unquoted --options v,h --name "$(basename -- "$0")" --longoptions keytab:,hdfs:,mongohost:,mongoport:,mongouser:,mongopass:,mongowritedb:,mongoauthdb:,p1:,p2:,host:,wdir:,help -- "$@")
    VALID_ARGS=$?
    if [ "$VALID_ARGS" != "0" ]; then
        util4loge "Given args not valid: $*"
        exit 1
    fi
    eval set -- "$PARSED_ARGS"
    while [[ $# -gt 0 ]]; do
        case "$1" in
        --keytab)       arr["KEYTAB_SECRET"]=$2     ; shift 2 ;;
        --hdfs)         arr["HDFS_PATH"]=$2         ; shift 2 ;;
        --mongohost)    arr["ARG_MONGOHOST"]=$2     ; shift 2 ;;
        --mongoport)    arr["ARG_MONGOPORT"]=$2     ; shift 2 ;;
        --mongouser)    arr["ARG_MONGOUSER"]=$2     ; shift 2 ;;
        --mongopass)    arr["ARG_MONGOPASS"]=$2     ; shift 2 ;;
        --mongowritedb) arr["ARG_MONGOWRITEDB"]=$2  ; shift 2 ;;
        --mongoauthdb)  arr["ARG_MONGOAUTHDB"]=$2   ; shift 2 ;;
        --p1)           arr["PORT1"]=$2             ; shift 2 ;;
        --p2)           arr["PORT2"]=$2             ; shift 2 ;;
        --host)         arr["K8SHOST"]=$2           ; shift 2 ;;
        --wdir)         arr["WDIR"]=$2              ; shift 2 ;;
        -h | --help)    arr["help"]=1               ; shift   ;;
        *)              break                                 ;;
        esac
    done

    for key in "${!arr[@]}"; do
        eval $key=${arr[$key]}
    done
}

#!/bin/bash
# Utils for cron scripts
# Author: Ceyhun Uzunoglu <ceyhunuzngl AT gmail [DOT] com>

# -------------------------------------- LOGGING UTILS --------------------------------------------
#######################################
# info log function
#######################################
function util4logi() {
    echo "$(date --rfc-3339=seconds) [INFO]" "$@"
}

#######################################
# warn log function
#######################################
function util4logw() {
    echo "$(date --rfc-3339=seconds) [WARN]" "$@"
}

#######################################
# error log function
#######################################
function util4loge() {
    echo "$(date --rfc-3339=seconds) [ERROR]" "$@"
}

#######################################
# util to print help message
#######################################
function util_usage_help() {
    grep "^##H" <"$0" | sed -e "s,##H,,g"
}
# -------------------------------------------------------------------------------------------------

# -------------------------------------- CHECK UTILS ----------------------------------------------
#######################################
# Util to check variables are defined
#  Arguments:
#    $1: variable name(s), meaning without $ sign. Supports multiple values
#  Usage:
#   util_check_vars ARG1 ARG2 ARG3
#  Returns:
#    success: 0
#    fail   : exits with exit-code 1
#######################################
function util_check_vars() {
    local var_check_flag var_names
    unset var_check_flag
    var_names=("$@")
    for var_name in "${var_names[@]}"; do
        [ -z "${!var_name}" ] && util4loge "$var_name is unset." && var_check_flag=true
    done
    [ -n "$var_check_flag" ] && exit 1
    return 0
}

#######################################
# Util to check shell command or executable exists
#  Arguments:
#    $1: command/executable name or full path
#  Usage:
#    util_check_cmd foo
#  Returns:
#    success: 0 and info log
#    fail   : exits with exit-code 1
#######################################
function util_check_cmd() {
    if which "$1" >/dev/null; then
        util4logi "Command $1 exists and in PATH."
    else
        util4loge "Please make sure you correctly set $1 executable path in PATH." && exit 1
    fi
}
# -------------------------------------------------------------------------------------------------

# ------------------------------------- PRE SETUP UTILS -------------------------------------------
#######################################
# check and set JAVA_HOME
#######################################
function util_set_java_home() {
    if [ -n "$JAVA_HOME" ]; then
        if [ -e "/usr/lib/jvm/java-1.8.0" ]; then
            export JAVA_HOME="/usr/lib/jvm/java-1.8.0"
        elif ! (java -XX:+PrintFlagsFinal -version 2>/dev/null | grep -E -q 'UseAES\s*=\s*true'); then
            util4loge "this script requires a java version with AES enabled"
            exit 1
        fi
    fi
}

#######################################
# Util to authenticate with keytab and to return Kerberos principle name
#  Arguments:
#    $1: keytab file
#  Usage:
#    principle=$(util_kerberos_auth_with_keytab /foo/keytab)
#  Returns:
#    success: principle name before '@' part. If principle is 'johndoe@cern.ch, will return 'johndoe'
#    fail   : exits with exit-code 1
#######################################
function util_kerberos_auth_with_keytab() {
    local principle
    principle=$(klist -k "$1" | tail -1 | awk '{print $2}')
    # run kinit and check if it fails or not
    if ! kinit "$principle" -k -t "$1" >/dev/null; then
        util4loge "Exiting. Kerberos authentication failed with keytab:$1"
        exit 1
    fi
    # remove "@" part from the principle name
    echo "$principle" | grep -o '^[^@]*'
}

#######################################
# setup hadoop and spark in k8s
#######################################
function util_setup_spark_k8s() {
    # check hava home
    util_set_java_home

    hadoop-set-default-conf.sh analytix 'hadoop spark' 3.2
    source hadoop-setconf.sh analytix 3.2 spark3
    export SPARK_LOCAL_IP=127.0.0.1
    export PYSPARK_PYTHON=/cvmfs/sft.cern.ch/lcg/releases/Python/3.9.6-b0f98/x86_64-centos7-gcc8-opt/bin/python3
    # until IT changes this setting, we need to turn off info logs in this way. Don't try spark.sparkContext.setLogLevel('WARN'), doesn't work, since they are not spark logs but spark-submit logs.
    sed -i 's/rootLogger.level = info/rootLogger.level = warn/g' "$SPARK_CONF_DIR"/log4j2.properties
}
# -------------------------------------------------------------------------------------------------

# ----------------------------------- PUSHGATEWAY UTILS -------------------------------------------
#######################################
# Returns left part of the dot containing string
# Arguments:
#   arg1: string
#######################################
function util_dotless_name() {
    echo "$1" | cut -f1 -d"."
}

#######################################
# Util to send cronjob start time to pushgateway, depends on K8S_ENV
#  Arguments:
#    $1: cronjob name
#    $2: period (10m, 1h, 1d, 1M)
#  Usage:
#    util_cron_send_start foo 1d
#######################################
function util_cron_send_start() {
    local script_name_with_extension env
    if [ -z "$PUSHGATEWAY_URL" ]; then
        util4loge "PUSHGATEWAY_URL variable is not defined. Exiting.."
        exit 1
    fi
    script_name_with_extension=$1
    period=$2
    # If K8S_ENV is not set, use default tag
    env=${K8S_ENV:-default}
    script=$(util_dotless_name "$script_name_with_extension")
    cat <<EOF | curl --data-binary @- "$PUSHGATEWAY_URL"/metrics/job/cmsmon-cron-"${env}"/instance/"$(hostname)"
# TYPE cmsmon_cron_start_${env}_${script} gauge
# HELP cmsmon_cron_start_${env}_${script} cronjob START Unix time
cmsmon_cron_start_${env}_${script}{script="${script}", env="${env}", period="${period}"} $(date +%s)
EOF
}

#######################################
# Util to send cronjob end time to pushgateway, depends on K8S_ENV
#  Arguments:
#    $1: cronjob name
#    $2: period (10m, 1h, 1d, 1M)
#    $3: cronjob exit status
#  Usage:
#    util_cron_send_end foo 1d 210
#######################################
function util_cron_send_end() {
    local script_name_with_extension env exit_code
    if [ -z "$PUSHGATEWAY_URL" ]; then
        util4loge "PUSHGATEWAY_URL variable is not defined. Exiting.."
        exit 1
    fi
    script_name_with_extension=$1
    period=$2
    exit_code=$3
    # If K8S_ENV is not set, use default tag
    env=${K8S_ENV:-default}
    script=$(util_dotless_name "$script_name_with_extension")
    cat <<EOF | curl --data-binary @- "$PUSHGATEWAY_URL"/metrics/job/cmsmon-cron-"${env}"/instance/"$(hostname)"
# TYPE cmsmon_cron_end_${env}_${script} gauge
# HELP cmsmon_cron_end_${env}_${script} cronjob END Unix time
cmsmon_cron_end_${env}_${script}{script="${script}", env="${env}", period="${period}", status="${exit_code}"} $(date +%s)
EOF
}
# -------------------------------------------------------------------------------------------------

# ------------------------------------------- OTHER UTILS -----------------------------------------
#######################################
# util to convert seconds to h, m, s format used in logging
#  Arguments:
#    $1: seconds in integer
#  Usage:
#    util_secs_to_human SECONDS
#    util_secs_to_human 1000 # returns: 0h 16m 40s
#  Returns:
#    '[\d+]h [\d+]m [\d+]s' , assuming [\d+] integer values
#######################################
function util_secs_to_human() {
    echo "$((${1} / 3600))h $(((${1} / 60) % 60))m $((${1} % 60))s"
}
# -------------------------------------------------------------------------------------------------
