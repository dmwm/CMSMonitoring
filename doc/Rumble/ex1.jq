for $doc in json-file("hdfs://analytix/project/monitoring/archive/wmarchive/raw/metric/2020/09/*")
where not $doc.data.meta_data.jobstate eq "success"
return $doc.data.wmaid
