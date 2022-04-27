# testing-logstash

### What is this
This repo includes a Dockerfile which creates docker.elastic.co/logstash/logstash based image.

### Where can I find the latest Docker image
`docker pull mrceyhun/testing-logstash:8.1.3`

### How can I deploy
You can deploy simple `logstash.yaml` to your kubernetes cluster.

### What is included in 'testing-logstash' image
- `howtorun.txt` provides all details to run all services
- Logstash-8.1.3
- filebeat-8.1.3
- Example logstash.conf and filebeat.yaml
- Python3
- nc (Netcat)
- nano
- vim
- WORKDIR is /data
- logstash.yaml to deploy K8s ` kubectl apply -f logstash.yaml`

---


### How to validate configuration
You can use `-t` flag to validate configuration file without running Logstash:
- `logstash -t -f logstash.conf`
  - `Config Validation Result: OK. Exiting Logstash` means conf file is valid.
  - For detailed logs, use `--verbose` or `--log.level=debug` flag also, such as: `logstash --verbose -t -f logstash.conf`

### How to automatically get changes of logstash.conf
`logstash -r -f ...`


### Advanced testing
You may want to change `jvm.options`, `log4j2.properties`, `logstash-sample.conf`, `logstash.yml`, `pipelines.yml`, `startup.options` of Logstash. You can find all these files under 
`/etc/$


## References
- https://www.elastic.co/guide/en/logstash/current/input-plugins.html
- https://www.elastic.co/guide/en/logstash/current/output-plugins.html
- https://www.elastic.co/guide/en/logstash/current/filter-plugins.html
- https://www.elastic.co/guide/en/logstash/current/codec-plugins.html
- https://github.com/dmwm/CMSKubernetes/blob/master/kubernetes/cmsweb/monitoring/logstash.conf
- https://github.com/dmwm/CMSKubernetes/tree/master/kubernetes/cmsweb/monitoring

---
