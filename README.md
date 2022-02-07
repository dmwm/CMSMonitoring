### CMSMonitoring
CMSMonitoring repository contains code, files related to CMS Monitoring
infrastructure.

### Documentation

- Available [here](https://cmsmonit-docs.web.cern.ch/)
- source [code](https://gitlab.cern.ch/cmsmonitoring/cmsmonit-docs) 


### Git Workflows

- On tag `go-*.*.*` 
    - Builds go executables and release `cmsmon-tools`
    - Builds `cmsmonitoring/cmsmonit-int` docker image and push to registry.cern.ch
    - Builds `cmsmonitoring/cmsmonit-alert` docker image and push to registry.cern.ch
- On tag `sqoop-*.*.*` 
    - Builds `cmsmonitoring/sqoop` docker image and push to registry.cern.ch
- Syntaxcheck on specical condition
    - Check validations of json and yaml files only that kind of files are changed
