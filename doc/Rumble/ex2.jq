subsequence(
  for $doc in json-file("hdfs://analytix/project/monitoring/archive/wmarchive/raw/metric/2020/09/15/")
  return $doc.data.wmaid,
  1,
  5
)
