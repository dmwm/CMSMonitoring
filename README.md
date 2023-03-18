### CMSMonitoring
CMSMonitoring repository contains code, files related to CMS Monitoring
infrastructure.

### Documentation

- Available [here](https://cmsmonit-docs.web.cern.ch/)
- source [code](https://gitlab.cern.ch/cmsmonitoring/cmsmonit-docs) 


### Git Workflows

- On tag :whale: `go-*.*.*` :rocket:
    - Builds go executables and release `cmsmon-tools`
    - Builds `cmsmonitoring/cmsmonit-int` docker image and push to registry.cern.ch
    - Builds `cmsmonitoring/cmsmonit-alert` docker image and push to registry.cern.ch
    - Builds `cmsmonitoring/nats-sub` docker image and push to registry.cern.ch
- On tag :whale: `rgo-*.*.*` :rocket:
    - Builds `cmsmonitoring/rucio-mon-goweb` docker image using `rucio-dataset-monitoring`
    - Builds `cmsmonitoring/spark2mng` docker image using `rucio-dataset-monitoring`
- On tag :whale: `cpueff-*.*.*` :rocket:
    - Builds `cmsmonitoring/cpueff-goweb` docker image using `cpueff-goweb`
    - Builds `cmsmonitoring/cpueff-spark` docker image using `cpueff-goweb`
- On tag :whale: `sqoop-*.*.*` :rocket: 
    - Builds `cmsmonitoring/sqoop` docker image and push to registry.cern.ch
- On tag `py-*.*.*`  
    - New release of CMSMonitoring PyPi module https://pypi.org/project/CMSMonitoring/
- On tag `drpy-*.*.*`
    - Builds `cmsmonitoring/cmsmon-py` docker image and push to registry.cern.ch
- Syntax check on special conditions
    - Check validations of json and yaml files only that kind of files are changed
