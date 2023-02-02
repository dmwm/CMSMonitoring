#!/bin/bash
set -e
TZ=UTC
myname=$(basename "$0")
script_dir="$(cd "$(dirname "$0")" && pwd)"
# Get nice util functions
. "${script_dir}"/utils.sh

# ---------------------------------------------------------------------------------------------------------- Run in K8S
if [ -n "$K8S_ENV" ]; then
    # $1: output, shift is mandatory because rucio/setup-py3.sh also waits for $1
    output_=$1; shift

    util4logi "${myname} is starting.."
    util_cron_send_start "$myname" "1h"

    # Rucio API setup
    export X509_USER_PROXY=/etc/proxy/proxy
    source /cvmfs/cms.cern.ch/rucio/setup-py3.sh
    util_kerberos_auth_with_keytab /etc/secrets/keytab
    python3 "${script_dir}"/../src/python/CMSMonitoring/rucio_quotas.py \
        --output "$output_" \
        --template_dir "${script_dir}/../src/html/rucio_quotas" 2>&1

    util_cron_send_end "$myname" "1h" "$?"
    util4logi "${myname} successfully finished."
    exit 0
    # break
fi
# Run in LxPlus for test ----------------------------------------------------------------------------------------------

source /cvmfs/cms.cern.ch/cmsset_default.sh >/dev/null
source /cvmfs/cms.cern.ch/rucio/setup-py3.sh >/dev/null
output=$(voms-proxy-init -voms cms -rfc -valid 192:00 2>&1)
ec=$?
if [ $ec -ne 0 ]; then
    echo "$output" - exit code: $ec
    exit $ec
fi

py_input_args=(
    --output "/eos/user/c/cmsmonit/www/rucio/quotas.html"
    --template_dir "${script_dir}/../src/html/rucio_quotas"
)

# Catch	output to not print successful jobs stdout to email, print when failed
output=$(
    "${script_dir}"/../venv/bin/python3 \
        "${script_dir}"/../src/python/CMSMonitoring/rucio_quotas.py "${py_input_args[@]}" 2>&1
)
ec=$?
if [ $ec -ne 0 ]; then
    echo "$output" - exit code: $ec
    exit $ec
fi
