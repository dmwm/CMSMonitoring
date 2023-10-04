## Grafana backup
Gets json definitions of all CMSMonitoring related dashboards using Grafana API.

Then compresses and puts them to the EOS folder `/eos/cms/store/group/offcomp_monit/`.


## Requirements
- Create file to your Grafana authentication token and name it `keys.json`

```
    {
        "SECRET_KEY": "YOUR_SECRET_KEY"
    }
```

- Get `amtool` executable
```
curl -ksLO https://github.com/prometheus/alertmanager/releases/download/v0.23.0/alertmanager-0.23.0.linux-amd64.tar.gz && \
tar xfz alertmanager-0.23.0.linux-amd64.tar.gz && \
mv alertmanager-0.23.0.linux-amd64/amtool . && \
rm -rf alertmanager-0.23.0.linux-amd64*
```

## How to use
- Build the `Go` executable:
```sh
go build dashboard-exporter.go
```
- Run the file:
```sh
./dashboard-exporter --token keys.json --filesystem-path /path/to/fs_backup/
```
- Set file executable:
```sh
chmod +x ./dashboard-exporter.py
```
- Set crontab for preferred timeframe
- See `run.sh` file