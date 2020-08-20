#!/bin/sh
##H Script for automation of testing process of the intelligence module.
##H Usage: test_wrapper.sh <config-file-path> <wdir>  
##H

case ${1:-status} in
help )
    cat $0 | grep "^##H" | sed -e "s,##H,,g"
    echo " config  test config file path      (mandatory)"
    echo " wdir    work directory             default: /tmp/${USER}"
    echo ""
    echo " Options:"
    echo " help    help manual"
    exit 1
    ;;
esac

TEST_CONFIG=${1}
if [ -z "$TEST_CONFIG" ]; then
    echo "Pass the Config File Path. Testing Failed. Exiting.."
    exit
fi

# Setup work directory based on user input
WDIR=${2:-"/tmp/$USER"}
TOP=$(dirname $WDIR)

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
AM_VERSION="alertmanager-0.21.0.linux-amd64"
AM_URL="https://github.com/prometheus/alertmanager/releases/download/v0.21.0/${AM_VERSION}.tar.gz"
CMSMONITORING_REPO_URL="https://github.com/dmwm/CMSMonitoring.git"

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
        command -v wget
        if [ $? -eq 0 ]; then
            echo "Downloading AlertManager using wget"
            wget $AM_URL -O $WDIR/am.tar.gz
        else
            command -v curl
            if [ $? -eq 0 ]; then
                echo "Downloading AlertManager using curl"
                curl $AM_URL >$WDIR/am.tar.gz
            else
                echo "Install wget or curl to continue and try again. Exiting.."
                exit 1
            fi
        fi
        command -v tar
        if [ $? -eq 0 ]; then
            echo "Untar AlertManager..."
            tar -C $WDIR -xzf $WDIR/am.tar.gz
            mv $WDIR/$AM_VERSION $WDIR/am
            if [ $? -eq 0 ]; then
                echo "Successfully renamed ${WDIR}/${AM_VERSION} to ${WDIR}/am"
            else
                echo "Could not rename. Exiting.."
                delete_wdir
                exit 1
            fi
        else
            echo "Install tar to continue and try again. Exiting.."
            exit 1
        fi
        start_am
    fi
fi

## git clone the update CMSMonitoring repository in the working directory
command -v git
if [ $? -eq 0 ]; then
    echo "Cloning CMSMonitoring at ${WDIR}."
    cd $WDIR && git clone $CMSMONITORING_REPO_URL
else
    echo "Install git to continue and try again. Exiting.."
    exit 1
fi

## building the intelligence module for testing
mv $WDIR/CMSMonitoring/src/go/intelligence/int_test/test_cases.json $WDIR
if [ $? -eq 0 ]; then
    echo "Successfully moved ${WDIR}/CMSMonitoring/src/go/intelligence/int_test/test_cases.json to $WDIR."
else
    echo "Could not move. Exiting.."
    delete_wdir
    exit 1
fi

export GOPATH=$WDIR/CMSMonitoring
export PATH=$WDIR:$WDIR/bin:$PATH

command -v go
if [ $? -eq 0 ]; then
    echo "Building the int module...."
    go build -o $WDIR go/intelligence/int_test
else
    echo "Install go to continue and try again. Exiting.."
    exit 1
fi

# Delay for Alertmanager so that it starts completely.
sleep 5

command -v int_test
if [ $? -eq 0 ]; then
    int_test -config=$TEST_CONFIG
    stop_am
    delete_wdir
else
    echo "Test script not found !! Testing Failed..."
    delete_wdir
    exit 1
fi
