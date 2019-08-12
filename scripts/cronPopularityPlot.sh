#!/bin/bash

if ! klist -s
then
    (>&2 echo -e "This application requires a valid kerberos ticket")
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "/cvmfs/sft.cern.ch/lcg/views/LCG_96/x86_64-centos7-gcc8-opt/setup.sh"
source "/cvmfs/sft.cern.ch/lcg/etc/hadoop-confext/hadoop-swan-setconf.sh" analytix
cd "$SCRIPT_DIR" || exit
spark-submit --master yarn --driver-memory 10g --num-executors 48  --executor-memory 6g --conf spark.driver.extraClassPath='/eos/project/s/swan/public/hadoop-mapreduce-client-core-2.6.0-cdh5.7.6.jar' "$SCRIPT_DIR/../src/python/CMSMonitoring/scrutiny_plot.py" "$@"