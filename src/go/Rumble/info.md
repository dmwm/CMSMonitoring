 ---
 
# Rumble Usage 

###### Attention:
- Rumble getting started: https://rumble.readthedocs.io/en/latest/Getting%20started/
- JSONiq query language: https://rumble.readthedocs.io/en/latest/JSONiq/
- More advance queries: https://www.jsoniq.org/docs/JSONiq-usecases/html-single/index.html
- Please define output hdfs path in a public place i.e. `/tmp/rumble/username/123`.
- You can reach your data with this command: `hadoop fs -cat $output_path/*`
- Be careful while copy-pasting below examples. New lines should be exactly same.
- Please provide satisfying spark parameters considering your query.
- While using `curl`, you need to sringify multiline JSONiq query. You can do it in: https://onlinetexttools.com/json-stringify-text

###### Examples
1. Example JSONiq query
```
for $doc in json-file("hdfs://analytix/project/monitoring/archive/wmarchive/raw/metric/2020/09/15/*")
where $doc.data.meta_data.jobstate eq "success"
return $doc.data.wmaid
```

2. Example JSONiq query
```
for $doc in json-file("hdfs://analytix/project/monitoring/archive/wmarchive/raw/metric/2020/12/21/part-00000-8e64eeb5-2dbd-457e-bee0-ccd623641950-c000.json.gz")
where $doc.data.meta_data.jobstate ne "success"
let $job := $doc.data.meta_data.jobtype
group by $job
return { $job : count( $doc.data.wmaid ) }
```

3. Curl Example:
```
curl --location --request POST 'http://cuzunogl-k8s-yjc6wuzsnjes-node-0:30000/rumble-server' \ --header 'Content-Type: application/json' \ --data-raw '{
"query": "for $doc in json-file(\"hdfs://analytix/project/monitoring/archive/wmarchive/raw/metric/2020/09/15/*\")\nwhere $doc.data.meta_data.jobstate eq \"success\"\nreturn $doc.data.wmaid",
"output_path": "/tmp/rumle/cuzunogl/1122",
"userid": "ceyhun uzunoglu",
"verbose": 1,
"spark_conf": {
    "spark_executor_memory": "4g",
    "spark_executor_instances": "4",
    "spark_executor_cores": "4",
    "spark_driver_memory": "4g"
}
}'
```

4. rumble_client:
```
$> git clone https://github.com/mrceyhun/CMSMonitoring.git
$> cd src/go/Rumble
$> go build rumble_client.go
$> cat ./test.jq >
for $doc in json-file("hdfs://analytix/project/monitoring/archive/wmarchive/raw/metric/2020/09/15/*")
where $doc.data.meta_data.jobstate eq "success"
return $doc.data.wmaid
$> ./rumble_client -user cuzunogl -output /tmp/rumble/cuzunogl/test -query=./test.jq -verbose 1 -timeout 10 -executor-memory-g 2 -executor-instances 4 -executor-cores 2 -driver-memory-g 2
```
