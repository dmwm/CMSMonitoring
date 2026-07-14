#!/bin/bash
##H Script to copy Grafana dashboards from Production to Production Copy
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
    --url "https://monit-grafana.cern.ch"
    --token "/etc/grafana-secrets/grafana-token.json"
)

export OTEL_SERVICE_NAME="${OTEL_SERVICE_NAME:-grafana-production-copy}"

util4logi "${myname} is starting.."
util_cron_send_start "$myname" "1d"

python3 "${script_dir}"/production_copy.py "${py_input_args[@]}"

util_cron_send_end "$myname" "1d" "$?"
util4logi "${myname} successfully finished."
