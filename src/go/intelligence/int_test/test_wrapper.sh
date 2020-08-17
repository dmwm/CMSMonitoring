#!/bin/sh
##H Script for automation of testing process of the intelligence module.
##H Usage: test_wrapper.sh <work_directory>
##H

# If user provided the input parameter
if [ $# -ne 1 ]; then
    perl -ne '/^##H/ && do { s/^##H ?//; print }' <$0
    exit 1
fi

# Setup work directory based on user input
WDIR=$1
TOP=$(dirname $1)

if [ ! -d $WDIR ]; then
    echo "Provided work directory '$WDIR' does not exists. creating..."

    if [ ! -w $TOP ]; then
        echo "can not create $WDIR: $USER has no write permission for '$TOP' directory. exiting."
        exit 1
    fi

    if mkdir -p $WDIR; then
        echo "$WDIR created !"
    else
        echo "Unable to create $WDIR"
        exit 1
    fi
fi

#Variables
AM_BIN=$WDIR/am/alertmanager
AM_CONFIG=$WDIR/am/alertmanager.yml
TEST_CONFIG=$WDIR/test_config.json
AM_VERSION="alertmanager-0.21.0.linux-amd64"
AM_URL="https://github.com/prometheus/alertmanager/releases/download/v0.21.0/${AM_VERSION}.tar.gz"
CMSMONITORING_REPO_URL="https://github.com/indrarahul2013/CMSMonitoring.git"

PID=$(ps auxwww | egrep "alertmanager" | grep -v grep | awk 'BEGIN{ORS=" "} {print $2}')

# function for starting AlertManager
start_am() {
    echo "Starting AlertManager in background."
    nohup $AM_BIN --config.file=$AM_CONFIG </dev/null 2>&1 >AM.log &
}

# function for stopping AlertManager
stop_am() {
    local PID=$(ps auxwww | egrep "alertmanager" | grep -v grep | awk 'BEGIN{ORS=" "} {print $2}')
    echo "Stopping Alertmanager. PID : ${PID}"
    if [ -n "${PID}" ]; then
        kill -9 ${PID}
    fi
}

#function for clearing out working directory
delete_wdir() {
    echo "Deleting work directory : ${WDIR}"
    rm -rf $WDIR
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
            wget $AM_URL -O $WDIR/am.tar.gz
            if [ -x "$(command -v tar)" ]; then
                echo "Untar AlertManager..."
                tar -C $WDIR -xzf $WDIR/am.tar.gz
                if ! mv $WDIR/$AM_VERSION $WDIR/am; then
                    echo "Could not move. Exiting.."
                    delete_wdir
                    exit 1
                fi
            else
                echo "Install tar to continue. Exiting.."
                delete_wdir
                exit 1
            fi
        else
            echo "Install wget to continue. Exiting.."
            delete_wdir
            exit 1
        fi
        start_am
    fi
fi

## git clone the update CMSMonitoring repository in the working directory
if [ -x "$(command -v git)" ]; then
    echo "Cloning CMSMonitoring at ${WDIR}."
    cd $WDIR && git clone $CMSMONITORING_REPO_URL
else
    echo "Install git to continue. Exiting.."
    delete_wdir
    exit 1
fi

## building the intelligence module for testing
if ! mv $WDIR/CMSMonitoring/src/go/intelligence/int_test/test_config.json $WDIR; then
    echo "Could not move. Exiting.."
    delete_wdir
    exit 1
fi

if ! mv $WDIR/CMSMonitoring/src/go/intelligence/int_test/test_cases.json $TOP; then
    echo "Could not move. Exiting.."
    delete_wdir
    exit 1
fi

export GOPATH=$WDIR/CMSMonitoring
export PATH=$WDIR:$WDIR/bin:$PATH

if [ -x "$(command -v go)" ]; then
    echo "Building the int module...."
    go build -o $WDIR go/intelligence/int_test
else
    echo "Install go to continue. Exiting.."
    delete_wdir
    exit 1
fi

# Delay for Alertmanager so that it starts completely.
sleep 5

if [ -x "$(command -v int_test)" ]; then
    int_test -config=$TEST_CONFIG
    stop_am
    delete_wdir
else
    echo "Test script not found !! Testing Failed..."
    delete_wdir
    exit 1
fi
