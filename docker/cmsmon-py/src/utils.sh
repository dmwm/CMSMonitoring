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

#######################################
# util to exit function on fail
#######################################
function util_on_fail_exit() {
    util4loge "finished with error!"
    exit 1
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
# Util to check file exists
#  Arguments:
#    $1: file(s), full paths or names in current directory. Supports multiple files
#  Usage:
#    util_check_files $ARG_KEYTAB /etc/secrets/foo /data/config.json $FOO
#  Returns:
#    success: 0
#    fail   : exits with exit-code 1
#######################################
function util_check_files() {
    local var_check_flag file_names
    unset var_check_flag
    file_names=("$@")
    for file_name in "${file_names[@]}"; do
        [ -e "${!file_name}" ] && util4loge "$file_name file does not exist, please check given args." && var_check_flag=true
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

#######################################
# Util to check if directory exist and create if not
#  Arguments:
#    $1: directory path
#  Usage:
#    util_check_cmd /dir/foo
#  Returns:
#    success: 0 and info log
#    fail   : exits with exit-code 1
#######################################
function util_check_and_create_dir() {
    local dir=$1
    if [ -z "$dir" ]; then
        util4loge "Please provide output directory. Exiting.."
        util_usage_help
        exit 1
    fi
    if [ ! -d "$dir" ]; then
        util4logw "output directory does not exist, creating..: ${dir}}"
        if [[ "$(mkdir -p "$dir" >/dev/null)" -ne 0 ]]; then
            util4loge "cannot create output directory: ${dir}"
            exit 1
        fi
    fi
}
# -------------------------------------------------------------------------------------------------

# ------------------------------------- PRE SETUP UTILS -------------------------------------------
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
    cat <<EOF | curl -s -S --data-binary @- "$PUSHGATEWAY_URL"/metrics/job/cmsmon-cron-"${env}"/instance/"$(hostname)"
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
    cat <<EOF | curl -s -S --data-binary @- "$PUSHGATEWAY_URL"/metrics/job/cmsmon-cron-"${env}"/instance/"$(hostname)"
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
