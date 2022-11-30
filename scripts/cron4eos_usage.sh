#!/bin/bash
##H Script to create CMS Eos path sizes with conditions
##H CMSVOC and CMSMONITORING groups are responsible for this script.

set -e
TZ=UTC
myname=$(basename "$0")
script_dir="$(cd "$(dirname "$0")" && pwd)"

# Do not change the order of "--output_file"([0],[1]) which is replaced in K8s run
py_input_args=(
    --output_file "/eos/user/c/cmsmonit/www/eos-path-size/size.html"
    --non_ec_json "/eos/cms/store/accounting/eos_non_ec_accounting.json"
    --ec_json "/eos/cms/store/accounting/eos_ec_accounting.json"
    --summary_json "/eos/cms/store/accounting/eos_accounting_summary.json"
    --static_html_dir "${script_dir}/../src/html/eos_path_size"
)

# ---------------------------------------------------------------------------------------------------------- Run in K8S
if [ -n "$K8S_ENV" ]; then
    # $1: output
    # Replace static output file with user arg for testability.
    py_input_args[1]=$1

    # Get nice utils from CMSSpark
    curl -ksLO https://raw.githubusercontent.com/dmwm/CMSSpark/master/bin/utils/common_utils.sh
    . common_utils.sh

    util4logi "${myname} is starting.."
    util_cron_send_start "$myname" "1h"

    util_kerberos_auth_with_keytab /etc/secrets/keytab
    python3 "${script_dir}"/../src/python/CMSMonitoring/eos_path_size.py "${py_input_args[@]}" 2>&1

    util_cron_send_end "$myname" "1h" "$?"
    util4logi "${myname} successfully finished."
    exit 0
    # break
fi
# ---------------------------------------------------------------------------------------------------------------------

. /cvmfs/sft.cern.ch/lcg/views/LCG_101/x86_64-centos7-gcc8-opt/setup.sh

# Catch output to not print successful jobs stdout to email, print when failed
output=$(pip install --user schema 2>&1)
ec=$?
if [ $ec -ne 0 ]; then
    echo "$output" - exit code: $ec
    exit $ec
fi

if ! [ "$(python -c 'import sys; print(sys.version_info.major)')" = 3 ]; then
    echo "It seem python version is not 3.X! Exiting..."
    exit 1
fi

# Catch	output to not print successful jobs stdout to email, print when failed
output=$(python "$HOME"/CMSMonitoring/src/python/CMSMonitoring/eos_path_size.py "${py_input_args[@]}" 2>&1)
ec=$?
if [ $ec -ne 0 ]; then
    echo "$output" - exit code: $ec
    exit $ec
fi
