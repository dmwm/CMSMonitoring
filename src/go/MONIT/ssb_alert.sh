#!/bin/bash
source config

while true; 
do
echo "Infinite Loop!! Press Ctrl+C for halting...." 
eval "$ssb_query" > $input_file
eval "$alert_cmd"
sleep $interval; 
done