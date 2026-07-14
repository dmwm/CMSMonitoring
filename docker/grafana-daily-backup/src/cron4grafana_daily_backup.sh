#!/bin/bash
##H Script to export Grafana dashboards and archive them to EOS
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

py_input_args=(
    --token "/etc/grafana-secrets/grafana-token.json"
    --filesystem-path "/eos/cms/store/group/offcomp_monit/grafana_backup"
)

export OTEL_SERVICE_NAME="${OTEL_SERVICE_NAME:-grafana-daily-backup}"

util4logi "${myname} is starting.."
util_cron_send_start "$myname" "1d"

util_kerberos_auth_with_keytab /etc/secrets/keytab
python3 "${script_dir}"/daily_backup.py "${py_input_args[@]}"

util_cron_send_end "$myname" "1d" "$?"
util4logi "${myname} successfully finished."
