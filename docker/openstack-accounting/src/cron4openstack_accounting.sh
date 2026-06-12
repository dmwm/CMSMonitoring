#!/bin/bash
##H Script to create CMS Eos path sizes with conditions
##H CMSVOC and CMSMONITORING groups are responsible for this script.
set -e
TZ=UTC
myname=$(basename "$0")
script_dir="$(cd "$(dirname "$0")" && pwd)"
# Get nice util functions
. "${script_dir}"/utils.sh

if [ -z "$K8S_ENV" ]; then
    util4loge "K8S_ENV is not set. This script is intended to run in Kubernetes."
    exit 1
fi

# Do not change the order of "--output_file"([0],[1]) which is replaced in K8s run
py_input_args=(
    --output_file "/eos/user/c/cmsmonit/www/eos_openstack/openstack_accounting.html"
    --summary_json "/eos/cms/store/accounting/openstack_accounting_summary.json"
    --static_html_dir "${script_dir}/html"
)

# $1: output — replace static output file with user arg for testability.
py_input_args[1]=$1

export OTEL_SERVICE_NAME="${OTEL_SERVICE_NAME:-openstack-accounting}"

util4logi "${myname} is starting.."
util_cron_send_start "$myname" "1h"

util_kerberos_auth_with_keytab /etc/secrets/keytab
python3 "${script_dir}"/openstack_accounting.py "${py_input_args[@]}"

util_cron_send_end "$myname" "1h" "$?"
util4logi "${myname} successfully finished."
