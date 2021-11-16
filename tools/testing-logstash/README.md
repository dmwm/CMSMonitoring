# testing-logstash

### What is this
This repo includes a Dockerfile which creates CentOS based image with prebuild Logstash and testing tools.

### Where can I find the latest Docker image
`docker push mrceyhun/testing-logstash:latest`

### How can I deploy
You can deploy simple `logstash.yaml` to your kubernetes cluster.

### What is included in 'testing-logstash' image
- Logstash-7.x
- Python3
- nc (Netcat)
- nano
- vim
- `logstash` alias to `/usr/share/logstash/bin/logstash`
- WORKDIR is /data

## How can I test Logstash config
There are 2 testing scenarios you can find here: `stdout` output plugin and `http` output plugin examples.

First of all, deploy kubernetes deployment to your cluster:
`kubectl apply -f logstash.yaml`

---
1. **Testing with `stdout` output plugin**<br />
- Copy logstash config file and input logs file ("frontend.log" is used in logstash-ex2.conf's input plugin) to k8s pod: 
  - `kubectl cp examples/logstash-ex2.conf logstash-b8d54cb84-j7fxx:/data -n default`
  - `kubectl cp frontend.log logstash-b8d54cb84-j7fxx:/data -n default`
- Get a shell in kubernetes pod:
  - `kubectl -n default exec --stdin --tty logstash-b8d54cb84-j7fxx -- /bin/bash`
- Run logstash with necessary flags (please see their meanings: logstash --help ):
  - `logstash -r -f logstash-ex2.conf`
- Both output results and Logstash info logs will be written to stdout, not to "/usr/share/logstash/logs".
- You can use `json` or `rubydebug` codec in output plugin
- You don't need to start and stop logstash because it takes long time to start. `-r` flag allows to reread input files when there is a change in config file
- So, I suggest that keep running logstash, open new shell in k8s pod and edit logstash-ex2.conf and save. These 2 configurations in input plugin will allow to read input file from the beginning again  with the new configuration of course:
  - `start_position => "beginning"` 
  - `sincedb_path => "/dev/null"`

---

2. **Testing with `http` output plugin**<br />
"http" output plugin posts http requests to defined `url => 'host:port'`. We need to catch these requests, because they are the final version of our parsing results that are sent to server. To catch these requests, you need a server which listens incoming requests. We will run a basic http server in localhost to print incoming requests data which are the final version that we're sending to our server. So, we'll use `http://127.0.0.1:PORT` in `url` part of the `http` output plugin.

> Not: If you define "format => json" in "http" output plugin, you will get exactly one json body for each line of input logs. Besides, if you define "format => json_batch" in "http" output plugin, you will get a body like array of jsons which represents multiple line of input logs. We use "json_batch" in our config.

**Let's test**
- Copy logstash config file and input logs file ("frontend.log" is used in logstash-ex1.conf's input plugin) to k8s pod:
  - `kubectl cp examples/logstash-ex1.conf logstash-b8d54cb84-j7fxx:/data -n default`
  - `kubectl cp frontend.log logstash-b8d54cb84-j7fxx:/data -n default`
- Get a shell in kubernetes pod:
  - `kubectl -n default exec --stdin --tty logstash-b8d54cb84-j7fxx -- /bin/bash`
- Change port number of "http" output plugin in logstash-ex1.conf. I used 7777, see `http://127.0.0.1:7777` line. Use same port as argument to ./server.py 
- Run server.py either with
  - <pre> ./server.py PORT                           # Prints raw data of incoming requests </pre>
  - or
  - <pre> ./server.py PORT 2>&1 | grep -F '[{' | jq  # Prints body of the incoming requests in pretty json format </pre>
- Open another k8s pod shell in differnt terminal tab and run logstash:
  - `logstash -r -f logstash-ex1.conf`
- You will see final data(array of jsons; try copying, you'll see brackets) that you will send to your production server in the output of "server.py".
- To continue testing without restarting logstash, you may open a 3rd shell in kubernetes pod and make changes in your configuration. After saving the config file, logstash will reread input data and send to defined host:port.
---

### How to validate configuration
You can use `-t` flag to validate configuration file without running Logstash:
- `logstash -t -f logstash.conf`
  - `Config Validation Result: OK. Exiting Logstash` means conf file is valid. 
  - For detailed logs, use `--verbose` or `--log.level=debug` flag also, such as: `logstash --verbose -t -f logstash.conf`

### Advanced testing
You may want to change `jvm.options`, `log4j2.properties`, `logstash-sample.conf`, `logstash.yml`, `pipelines.yml`, `startup.options` of Logstash. You can find all these files under `/etc/logstash` directory. Keep in your mind that `/etc/logstash/conf.d` is the location for configuration files of Logstash, if you run it as service without `-f` flag. 


## References
- https://www.elastic.co/guide/en/logstash/current/input-plugins.html
- https://www.elastic.co/guide/en/logstash/current/output-plugins.html
- https://www.elastic.co/guide/en/logstash/current/filter-plugins.html
- https://www.elastic.co/guide/en/logstash/current/codec-plugins.html
- https://github.com/dmwm/CMSKubernetes/blob/master/kubernetes/cmsweb/monitoring/logstash.conf
- https://github.com/dmwm/CMSKubernetes/tree/master/kubernetes/cmsweb/monitoring
