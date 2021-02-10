### WMArchive wiki pages
WMArchive is CMS data-service to store FrameWork JobReport (FWJR) documents.
It provides a set of tools to store FWJR into short-term (MongoDB) and long-term (HDFS) storages,
to convert FWJR from JSON to AVRO data-formats and end-user tools (myspark) to run a spark job to find FWJR on HDFS.

### Managing the service
WMArchive has two nodes, `vocms0181` (pre-production) and `vocms0182` (production). To start, stop the service you need only two steps:

```
# login to wma account
sudo -i -u wma /bin/bash

# if you need to upgrade, please use either deploy script or wmarch_config.py files
# in /data/cfg/wmarchive area and then redeploy the service via install command
/data/vm_manage.sh install

# service command (start|stop|status)
/data/vm_manage.sh start
```

If you do not want to use this wrapper, then you can use the standard commands for managing
services in our VMs, like:
```
(A=/data/cfg/admin; cd /data; $A/InstallDev -s status:wmarchive)
(A=/data/cfg/admin; cd /data; $A/InstallDev -s stop:wmarchive)
(A=/data/cfg/admin; cd /data; $A/InstallDev -s start:wmarchive)
```


