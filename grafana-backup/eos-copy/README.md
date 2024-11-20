## Grafana backup
Gets json definitions of all CMSMonitoring related dashboards using Grafana API.

Then compresses and puts them to EOS folder `/eos/cms/store/group/offcomp_monit/grafana_backup`.


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

- Run the file:
```sh
python3 dashboard-exporter.py --token keys.json --filesystem-path /path/to/fs_backup/
```
- Set file executable:
```sh
chmod +x ./dashboard-exporter.py
```
- Set crontab for preferred timeframe
- See `run.sh` file