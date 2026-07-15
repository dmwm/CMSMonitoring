#!/bin/bash
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

# $1: output, shift is mandatory because rucio/setup-py3.sh also waits for $1
output_=$1; shift

export OTEL_SERVICE_NAME="${OTEL_SERVICE_NAME:-rucio-quotas}"

util4logi "${myname} is starting.."
util_cron_send_start "$myname" "1h"

# Rucio API setup
export X509_USER_PROXY=/etc/proxy/proxy
export RUCIO_HOME=/cvmfs/cms.cern.ch/rucio/x86_64/rhel9/py3/current
util_kerberos_auth_with_keytab /etc/secrets/keytab
python3 "${script_dir}"/rucio_quotas.py \
    --output "$output_" \
    --template_dir "${script_dir}/rucio_quotas_html"

util_cron_send_end "$myname" "1h" "$?"
util4logi "${myname} successfully finished."
