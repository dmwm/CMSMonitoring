#!/bin/bash

AM_BIN=./am/alertmanager
TEST_BIN=./CMSMonitoring/bin/test

if [ -f "$AM_BIN" ]; then
    nohup ./am/alertmanager --config.file=./am/alertmanager.yml </dev/null 2>&1 > AM.log &
else
	echo "AlertManager not found !! Testing Failed..."
    exit 1
fi

# Delay for Alertmanager so that it starts completely.
sleep 5

if [ -f "$TEST_BIN" ]; then
    ./CMSMonitoring/bin/test -config=config.json
else
	echo "Test script not found !! Testing Failed..."
    exit 1
fi