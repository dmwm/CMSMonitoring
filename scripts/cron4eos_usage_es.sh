#!/bin/bash
##H Script to send CMS EOS usage summary data to MONIT
##H CMS-VOC and CMSMONITORING groups are responsible for this script.
##H Arguments:
##H   $1: AMQ credentials json file path (secrets/cms-eos-mon/amq-broker.json)
##H
set -e
TZ=UTC
myname=$(basename "$0")
script_dir="$(cd "$(dirname "$0")" && pwd)"
# Get nice util functions
. "${script_dir}"/utils.sh

if [ "$1" == "" ] || [ "$1" == "-h" ] || [ "$1" == "--help" ] || [ "$1" == "-help" ]; then
    util_usage_help
    exit 0
fi

py_input_args=(
    --creds "$1"
    --summary_json "/eos/cms/store/accounting/eos_accounting_summary.json"
)
# ---------------------------------------------------------------------------------------------------------- Run in K8S
if [ -n "$K8S_ENV" ]; then
    util4logi "${myname} is starting.."
    util_cron_send_start "$myname" "1h"

    util_kerberos_auth_with_keytab /etc/secrets/keytab
    python3 "${script_dir}"/../src/python/CMSMonitoring/eos_usage_es.py "${py_input_args[@]}" 2>&1

    util_cron_send_end "$myname" "1h" "$?"
    util4logi "${myname} successfully finished."
    exit 0
fi
