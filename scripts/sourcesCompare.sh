#!/bin/bash
# This script will check monit-es, monit influxdb, and cms-es, for differences related to the htcondor data flow. 

##################
#PARAMS
#################
_auth_token='This is not a valid token' #Before run the script you need to set the actual grafana token
threshold="0.01"
_status_list=("Completed" "Running" "Idle")
# grafana proxy urls to the datasources 
influx_db_proxy='https://monit-grafana.cern.ch/api/datasources/proxy/7731/query'
escms_db_proxy='https://monit-grafana.cern.ch/api/datasources/proxy/8983/_msearch'
es_db_proxy='https://monit-grafana.cern.ch/api/datasources/proxy/8787/_msearch'
_db='monit_production_cmsjm'
# queries
# influxdb
query='SELECT count("MemoryMB") FROM "raw"."condor" WHERE ("Status" {{status}}) AND time >= now() - 6h GROUP BY time(12m) fill(null)'
# ES:
escms_query='{"search_type":"query_then_fetch","ignore_unavailable":true,"index":["cms-20*"]}\n {"size":0,"query":{"bool":{"filter":[{"range":{"RecordTime":{"gte":"now-6h","lte":"now","format":"epoch_millis"}}},{"query_string":{"analyze_wildcard":true,"query":"Status:{{status}}"}}]}},"aggs":{"2":{"date_histogram":{"interval":"5m","field":"RecordTime","min_doc_count":0,"extended_bounds":{"min":"now-6h","max":"now"},"format":"epoch_millis"},"aggs":{"1":{"cardinality":{"field":"GlobalJobId"}}}}}}'
es_query='{"search_type":"query_then_fetch","ignore_unavailable":true,"index":["monit_prod_condor_raw_metric_v002*"]}\n {"size":0,"query":{"bool":{"filter":[{"range":{"data.RecordTime":{"gte":"now-6h","lte":"now","format":"epoch_millis"}}},{"query_string":{"analyze_wildcard":true,"query":"data.Status:{{status}}"}}]}},"aggs":{"2":{"date_histogram":{"interval":"5m","field":"data.RecordTime","min_doc_count":0,"extended_bounds":{"min":"now-6h","max":"now"},"format":"epoch_millis"},"aggs":{"1":{"cardinality":{"field":"data.GlobalJobId"}}}}}}'
#How to aggregate the query results:
_es_filter="[.responses[].aggregations[].buckets[].doc_count]|add"
#idb filter per status 
declare -A _status_filter=(["Completed"]="='Completed'" ["Running"]="='Running'" ['Idle']="=~/(Idle)/")

exit_code=0
declare -A _res_influx
declare -A _res_es
declare -A _res_escms

for _status in "${_status_list[@]}"
do
    #Query influxdb
	_curr_q="${query/\{\{status\}\}/${_status_filter[$_status]}}"
    _res_influx[$_status]=$(curl -G  $influx_db_proxy -H "Authorization: Bearer $_auth_token" -H "query_id: $(date +%S)" -H 'cache-control: no-cache' \
			 --data-urlencode db="$_db" --data-urlencode q="$_curr_q" 2>/dev/null\
			|jq "[.results[0].series[0].values[][1]]|add")
            
    #Query ES and CMS ES
    _curr_q_cmses="${escms_query/\{\{status\}\}/${_status}}"
    _curr_q_es="${es_query/\{\{status\}\}/${_status}}"

    _res_escms[$_status]=$(echo -e "$_curr_q_cmses" | curl -X POST --data-binary @- -H "Authorization: Bearer $_auth_token" "$escms_db_proxy" 2>/dev/null | jq "$_es_filter")
    _res_es[$_status]=$(echo -e "$_curr_q_es" | curl -X POST --data-binary @- -H "Authorization: Bearer $_auth_token" "$es_db_proxy" 2>/dev/null | jq "$_es_filter")
    
    #Create the output strings
    str_status="$str_status\t$_status"
    str_idb="$str_idb\t${_res_influx[$_status]}"
    str_es="$str_es\t${_res_es[$_status]}"
    str_cmses="$str_cmses\t${_res_escms[$_status]}"
    
    #ESCMS only have completed tasks, so compare only in that case:
    if [[ "$_status" = "Completed" ]]
    then        
        val_es=$(bc <<< "scale=4;(${_res_escms[$_status]} - ${_res_es[$_status]})/${_res_escms[$_status]}")

        if (( $(echo "${val_es#-} > $threshold"|bc -l) ))
        then 
            echo "Error: The completed jobs count is different above the threshold ($threshold) between monit-es and es-cms: $val_es"
            exit_code=$(( exit_code + 1 ))
        fi 
    fi
    val_es_idb=$(bc <<< "scale=4;(${_res_es[$_status]} - ${_res_influx[$_status]})/${_res_es[$_status]}")
    if (( $(echo "${val_es_idb#-} > $threshold"|bc -l) ))
    then 
        echo "Error: the values for the $_status are different between influxdb and es-monit above the threshold($threshold): $val_es_idb"
        exit_code=$(( exit_code + 1 ))
    fi
done 
#Write the results to the error output if there is an error. 
if [[ $exit_code -gt 0 ]]
then
(>&2 echo -e "Status:$str_status")
(>&2 echo -e "Influxdb:$str_idb")
(>&2 echo -e "escms:$str_cmses")
(>&2 echo -e "es:$str_es")
fi

exit $exit_code
