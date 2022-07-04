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
    - Builds `cmsmonitoring/cmsmon-rucio-ds-web` docker image using `src/go/rucio-dataset-mon-go`
- On tag :whale: `sqoop-*.*.*` :rocket: 
    - Builds `cmsmonitoring/sqoop` docker image and push to registry.cern.ch
- On tag `py-*.*.*`  
    - New release of CMSMonitoring PyPi module https://pypi.org/project/CMSMonitoring/ 
- Syntax check on special conditions
    - Check validations of json and yaml files only that kind of files are changed
