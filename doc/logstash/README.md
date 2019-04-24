### Logstash for cmsweb

We use [Logstash](https://www.elastic.co/products/logstash) software to process
and ingest data from cmsweb logs to CERN MONIT infrastructure. The cmsweb logs
are located on dedicated node which collects all cmsweb logs from production
services and store them into directory structure based on host names. Here is
its layout

```
/path
  |
  |- host1
  |   |- services1
  |   |- services2
  |- host2
  |   |- services1
  |   |- services2
...
```

This directory contains all configuration files required to run logstash
process. 

```
# Step 1: download required binary distribution from
#         https://www.elastic.co/downloads/logstash
#
# Step 2: write own configuration files, e.g. cmsweb.conf
#
# Step 3: run the following process
#
bin/logstash -f cmsweb.conf
```
