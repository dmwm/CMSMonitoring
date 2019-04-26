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

#### References
1. [Logstash](https://www.elastic.co/guide/en/logstash/current/introduction.html)
2. [Logstash CLI](https://www.elastic.co/guide/en/logstash/current/running-logstash-command-line.html)
3. [Logstash options](https://www.elastic.co/guide/en/logstash/current/plugins-inputs-file.html)
4. [Filebeat](https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-getting-started.html)

##### Configuration rerefences
1. [Centralized pipelines](https://www.elastic.co/guide/en/logstash/current/configuring-centralized-pipelines.html)
2. [logstash.yml](https://www.elastic.co/guide/en/logstash/current/logstash-settings-file.html)
3. [dir-layout](https://www.elastic.co/guide/en/logstash/current/dir-layout.html)
4. [Get started](https://www.elastic.co/guide/en/elastic-stack-get-started/7.0/get-started-elastic-stack.html#logstash-setup)
