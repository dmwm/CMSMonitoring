#!/bin/bash

unset crondpid
crondpid=$(cat /var/run/crond.pid)

# Kerberos
keytab=/etc/secrets/keytab
principal=$(klist -k "$keytab" | tail -1 | awk '{print $2}')
kinit "$principal" -k -t "$keytab"
if [ $? == 1 ]; then
    echo "$(date --rfc-3339=seconds) [ERROR] <$*> Unable to perform kinit, exiting" >/proc/$crondpid/fd/1 2>&1
    exit 1
fi

# get script dir
script_dir="$(
    cd -- "$(dirname "$0")" >/dev/null 2>&1
    pwd -P
)"

# ------------------------------------------------------------------------------------------------- Check $CMSSQOOP_ENV
if [[ -z $CMSSQOOP_ENV ]]; then
    # CMSSQOOP_ENV variable is NOT defined, setting it as 'test'
    export CMSSQOOP_ENV=test
fi

# --------------------------------------------------------------------------------------------- Check $CMSSQOOP_CONFIGS
# Check configs.json is provided via env variable
if [ ! -e "${CMSSQOOP_CONFIGS}" ]; then
    echo "$(date --rfc-3339=seconds) [INFO] CMSSQOOP_CONFIGS variable is not defined or not a file, will check CMSSQOOP_ENV variable to set."
    if [ $CMSSQOOP_ENV = "prod" ]; then
        # If no CMSSQOOP_CONFIGS provided and CMSSQOOP_ENV is provided as prod, set HDFS output paths via configs.json
        export CMSSQOOP_CONFIGS=$script_dir/configs.json
    else
        # If no CMSSQOOP_CONFIGS and CMSSQOOP_ENV is provided, set dev HDFS output paths via configs-dev.json
        export CMSSQOOP_CONFIGS=$script_dir/configs-dev.json
    fi
fi

echo "$(date --rfc-3339=seconds) [INFO] CMSSQOOP_ENV=${CMSSQOOP_ENV}, CMSSQOOP_CONFIGS=${CMSSQOOP_CONFIGS}." >/proc/$crondpid/fd/1 2>&1
# ---------------------------------------------------------------------------------------------------------------------

# execute given script
export PATH=$PATH:/usr/hdp/hadoop/bin:/data:/data/sqoop
ALERT_MANAGER_HOSTS="http://cms-monitoring.cern.ch:30093 http://cms-monitoring-ha1.cern.ch:30093 http://cms-monitoring-ha2.cern.ch:30093"

# Run all given inputs. To run them correctly, use $@; to get given input as string, use $*.
if "$@"; then
     echo "$(date --rfc-3339=seconds) [INFO]" "<$*> successfully finished" >/proc/$crondpid/fd/1 2>&1
else
    echo "$(date --rfc-3339=seconds) [ERROR]" "<$*> failed" >/proc/$crondpid/fd/1 2>&1
    expire=$(date -d '+2 hour' --rfc-3339=ns | tr ' ' 'T')
    for amhost in $ALERT_MANAGER_HOSTS; do
        amtool alert add sqoop_failure alertname='sqoop job failure' \
            job="$*" \
            host="$(hostname)" \
            severity=high \
            tag=k8s \
            alert=amtool \
            kind=cluster \
            service=sqoop \
            --end="$expire" \
            --annotation=summary="Sqoop job failure" \
            --annotation=date="$(date)" \
            --alertmanager.url="$amhost"
    done
fi
