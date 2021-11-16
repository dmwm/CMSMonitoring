#!/bin/bash

# This script runs a trivial http server which prints all incoming requests to stdout, ...
# ... using netcat and parsed with 'jq'.
# In order to use jq properly, you need to define a common word in your incoming request's json part.

# Ref for help part: https://github.com/dmwm/CMSKubernetes/blob/master/kubernetes/monitoring/deploy.sh

##H How to use
##H Usage: server.sh LISTEN_PORT COMMON_WORD_IN_JSON
##H Usage: server.sh (-v|--verbose) LISTEN_PORT
##H Usage: server.sh -h <help>                        Provides help to the current script
##H
##H Args:
##H     LISTEN_PORT          : nc(netcat) port to listen incoming requests
##H     COMMON_WORD_IN_JSON  : used in jq to catch json line in incoming request, and skips other lines which are not json object
##H     -v|--verbose         : [should be first argument if it'll used] prints raw incoming requests without json parsing
##H Examples:
##H     $ ./server.sh -h
##H     $ ./server.sh 7777 "message"
##H     $ ./server.sh 7777 "frontend"
##H     $ ./server.sh -v 7777
##H     $ ./server.sh --verbose 7777
##H
set -e # exit script if error occurs

# help definition
if [ "$1" == "-h" ] || [ "$1" == "-help" ] || [ "$1" == "--help" ] || [ "$1" == "help" ] || [ "$1" == "" ]; then
  perl -ne '/^##H/ && do { s/^##H ?//; print }' <"$0"
  exit 1
fi

VERBOSE=false
LISTEN_PORT=$1
COMMON_WORD_IN_JSON=$2

for opt in "$@"; do
  case "$opt" in
    -v|--verbose)
       VERBOSE=true
       LISTEN_PORT=$2
       ;;
  esac
done


function server_json_out () {
  echo "[server.sh] JSON output --->"
  echo "[server.sh] Listening port ${LISTEN_PORT} and will catch \"${COMMON_WORD_IN_JSON}\" in json "
  while true; do
    echo -e "HTTP/1.1 200 OK\n\n " | nc -l -w 1 "$LISTEN_PORT" | grep "$COMMON_WORD_IN_JSON" | jq
  done
}

function server_raw_out () {
  echo "[server.sh] Raw output --->"
  echo "[server.sh] Listening port ${LISTEN_PORT} ... "
  while true; do
    echo -e "HTTP/1.1 200 OK\n\n " | nc -l -w 1 "$LISTEN_PORT"
  done
}


if $VERBOSE
then
  server_raw_out 2>&1
else
  server_json_out
fi
