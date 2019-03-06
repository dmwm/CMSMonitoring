### How to setup Prometheus service on a custom node
This write-up is largely based on the following [blog](https://www.scaleway.com/docs/configure-prometheus-monitoring-with-grafana)


### Installation instructions
Here we describe how to download and install Prometheus server on local node
```
cd to main directory where we'll perform the setup
mkdir -p {usr/bin,etc/systemd/system,etc/prometheus,var/lib/prometheus,log}
curl -L -O https://github.com/prometheus/node_exporter/releases/download/v0.16.0/node_exporter-0.16.0.linux-amd64.tar.gz
tar xvf node_exporter-0.16.0.linux-amd64.tar.gz
cp node_exporter-0.16.0.linux-amd64/node_exporter usr/bin
rm -rf node_exporter-0.16.0.linux-amd64.tar.gz node_exporter-0.16.0.linux-amd64
```
Once installed we should determine path to node_exporter
```
ls -al $PWD/usr/bin/node_exporter
```
and create node_exporter service config file where you put proper path to node_exporter executable
```
# use your favorite editor, e.g.
vim etc/systemd/system/node_exporter.service
# and put the following context in this file
[Unit]
Description=Node Exporter
Wants=network-online.target
After=network-online.target

[Service]
User=node_exporter
Group=node_exporter
Type=simple
ExecStart=/data/cms/Prometheus/usr/bin/node_exporter --collectors.enabled meminfo,hwmon,entropy

[Install]
WantedBy=multi-user.target
```
Finally, we can start node_exporter on our node
```
nohup $PWD/usr/bin/node_exporter 2>&1 1>& $PWD/log/node_exporter.log < /dev/null &
```

Next step is to download and install prometheus server
```
curl -L -O https://github.com/prometheus/prometheus/releases/download/v2.2.1/prometheus-2.2.1.linux-amd64.tar.gz
tar xfz prometheus-*.tar.gz
cp prometheus-*/prometheus usr/bin/
cp prometheus-*/promtool usr/bin/
cp -r prometheus-*/consoles etc/prometheus
cp -r prometheus-*/console_libraries etc/prometheus/
rm -rf prometheus-*
```

First, we create new configuration for Prometheus server
```
# use your favorite editor
vim etc/prometheus/prometheus.yml
# and put the following context in this file
global:
  scrape_interval:     15s
  evaluation_interval: 15s

rule_files:

scrape_configs:
  - job_name: 'prometheus'
    scrape_interval: 5s
    static_configs:
      - targets: ['localhost:9090']
```

At this point, we should be able to start prometheus server using the following command:
```
prometheus --config.file $PWD/etc/prometheus/prometheus.yml \
    --storage.tsdb.path $PWD/var/lib/prometheus/ \
    --web.console.templates=$PWD/etc/prometheus/consoles \
    --web.console.libraries=$PWD/etc/prometheus/console_libraries
```

### cmsweb configuration
First, we need to setup proper iptables rules between node which will run our exporters, Prometheus server and grafana back-end servers. Here is an example of iptables rule to open port 9090 (our Prometheus server) that external clients (e.g. grafana servers) can access it:
```
sudo iptables -I INPUT -i eth0 -p tcp --dport 9090 -j ACCEPT
```
Eventually, we should tweak this rule such that we'll open port only for CERN MONIT backend server IPs.

Then, we need to add new Data Source in CMS monitroing grafana to support Prometheus end-point. For information how to do it please consult [this](https://prometheus.io/docs/visualization/grafana/) and [this](http://docs.grafana.org/features/datasources/prometheus/) pages, or go directly to CERN MONIT and add this Data Source.
