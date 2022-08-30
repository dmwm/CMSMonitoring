#!/bin/bash

# Kerberos
keytab=/etc/secrets/keytab
principal=$(klist -k "$keytab" | tail -1 | awk '{print $2}')
echo "principal=$principal"
kinit "$principal" -k -t "$keytab"
if [ $? == 1 ]; then
    echo "Unable to perform kinit"
    exit 1
fi
klist -k "$keytab"

# Check configs.json is provided
if [ ! -e "${CMSSQOOP_CONFIGS}" ]; then
    echo "CMSSQOOP_CONFIGS variable is not defined or not a file. Exiting.."
    exit 1
fi

# execute given script
export PATH=$PATH:/usr/hdp/hadoop/bin:/data:/data/sqoop
ALERT_MANAGER_HOSTS="http://cms-monitoring.cern.ch:30093 http://cms-monitoring-ha1.cern.ch:30093 http://cms-monitoring-ha2.cern.ch:30093"

# Run all given inputs. To run them correctly, use $@; to get given input as string, use $*.
if "$@"; then
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
