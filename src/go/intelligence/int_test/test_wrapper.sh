#!/bin/sh
##H Script for automation of testing process of the intelligence module.

CMSMONITORING_REPO=$(pwd)/CMSMonitoring
AM_BIN=$(pwd)/am/alertmanager
TEST_CONFIG=$(pwd)/CMSMonitoring/src/go/intelligence/int_test/test_config.json
PID=$(ps auxwww | egrep "alertmanager" | grep -v grep | awk 'BEGIN{ORS=" "} {print $2}')
AM_URL="https://github.com/prometheus/alertmanager/releases/download/v0.21.0/alertmanager-0.21.0.linux-amd64.tar.gz"
CMSMONITORING_REPO_URL="https://github.com/dmwm/CMSMonitoring.git"

# function for starting AlertManager
start_am() {
    echo "Starting AlertManager in background."
    nohup ./am/alertmanager --config.file=./am/alertmanager.yml </dev/null 2>&1 >AM.log &
}

# function for stopping AlertManager
stop_am() {
    local PID=$(ps auxwww | egrep "alertmanager" | grep -v grep | awk 'BEGIN{ORS=" "} {print $2}')
    echo "Stopping Alertmanager. PID : ${PID}"
    if [ -n "${PID}" ]; then
        kill -9 ${PID}
    fi
}

# Logic for deploying the alertmanager
if [ ! -z "$PID" ]; then
    echo "AlertManager already running. PID : ${PID}"
else
    if [ -f "$AM_BIN" ]; then
        start_am
    else
        echo "AlertManager not found !!"
        if [ -x "$(command -v wget)" ]; then
            echo "Downloading AlertManager...."
            wget $AM_URL -O am.tar.gz
            if [ -x "$(command -v tar)" ]; then
                echo "Untar AlertManager..."
                tar -xzf am.tar.gz --one-top-level=am --strip-components 1
            else
                echo "Install tar to continue. Exiting.."
                exit 1
            fi
        else
            echo "Install wget to continue. Exiting.."
            exit 1
        fi
        start_am
    fi
fi

## building the intelligence module for testing
export GOPATH=$(pwd)/CMSMonitoring
export PATH=$(pwd);$(pwd)/bin:$PATH

if [ -x "$(command -v go)" ]; then
    echo "Building the int module...."
    go build go/intelligence/int_test
else
    echo "Install go to continue. Exiting.."
    exit 1
fi

# Delay for Alertmanager so that it starts completely.
sleep 5

if [ -x "$(command -v int_test)" ]; then
    int_test -config=$TEST_CONFIG
    stop_am
else
    echo "Test script not found !! Testing Failed..."
    exit 1
fi