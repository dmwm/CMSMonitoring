#!/bin/bash
##H Script to create CMS Eos path sizes with conditions
##H CMSVOC and CMSMONITORING groups are responsible for this script.

. /cvmfs/sft.cern.ch/lcg/views/LCG_101/x86_64-centos7-gcc8-opt/setup.sh

# Catch output to not print successful jobs stdout to email, print when failed
if ! output=$(pip install --user schema 2>&1); then
    echo "$output"
    exit $?
fi

if ! [ "$(python -c 'import sys; print(sys.version_info.major)')" = 3 ]; then
    echo "It seem python version is not 3.X! Exiting..."
    exit 1
fi

script_dir=$(
    cd -- "$(dirname "$0")" >/dev/null 2>&1 || exit
    pwd -P
)

py_input_args=(
    --output_file "/eos/user/c/cmsmonit/www/eos-path-size/size.html"
    --non_ec_json "/eos/cms/store/accounting/eos_non_ec_accounting.json"
    --ec_json "/eos/cms/store/accounting/eos_ec_accounting.json"
    --static_html_dir "${script_dir}/../src/html/eos_path_size"
)

# Catch	output to not print successful jobs stdout to email, print when failed
if ! output=$(
    python "$HOME"/CMSMonitoring/src/python/CMSMonitoring/eos_path_size.py "${py_input_args[@]}" 2>&1
); then
    echo "$output"
    exit $?
fi
