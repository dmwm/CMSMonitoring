#!/bin/bash
set -e

source /cvmfs/cms.cern.ch/cmsset_default.sh
source /cvmfs/cms.cern.ch/rucio/setup-py3.sh
voms-proxy-init -voms cms -rfc -valid 192:00

script_dir="$(
    cd -- "$(dirname "$0")" >/dev/null 2>&1 || exit
    pwd -P
)"

py_input_args=(
    --output "/eos/user/c/cmsmonit/www/rucio/quotas.html"
    --template_dir "${script_dir}/../src/html/rucio_quotas"
)

# Catch	output to not print successful jobs stdout to email, print when failed
if ! output=$(
    "${script_dir}"/../venv/bin/python3 \
        "${script_dir}"/../src/python/CMSMonitoring/rucio_quotas.py "${py_input_args[@]}" 2>&1
); then
    echo "$output"
    exit $?
fi
