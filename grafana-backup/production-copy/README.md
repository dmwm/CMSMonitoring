
## Production Copy
This script copies Grafana dashboards from the folder "Production" to "Production Copy" using the Grafana API.

## Requirements
- Create a file for your Grafana authentication token and name it `keys.json`

```
{
    "production_copy_token": "YOUR_SECRET_KEY"
}
```

- Get `amtool` executable
```
curl -ksLO https://github.com/prometheus/alertmanager/releases/download/v0.27.0/alertmanager-0.27.0.linux-amd64.tar.gz && \
tar xfz alertmanager-0.27.0.linux-amd64.tar.gz && \
mv alertmanager-0.27.0.linux-amd64/amtool . && \
rm -rf alertmanager-0.27.0.linux-amd64*
```

## How to use

- Run the file:
```sh
python3 dashboard-copy.py --url https://grafana-url.com --token keys.json
```
- Set the file executable:
```sh
chmod +x ./dashboard-copy.py
```
- Set crontab for preferred timeframe
- See `run.sh` file