# Spider script instructions

The [CMS htcondor ES](https://github.com/dmwm/cms-htcondor-es) (a.k.a cms-htcondor-es or spider) is a set of scripts
which consumes CMS HTCondor ClassAds (Job processing docs), converts them to JSON documents and feeds them
via [CMSMonitoring/StompAMQ](https://github.com/dmwm/CMSMonitoring/blob/master/src/python/CMSMonitoring/StompAMQ.py)
module to CERN MONIT infrastructure. The service is running on `vocms0240`.
The full list of setup and maintenance instructions can be found
in [cms-htcondor-es wiki](https://github.com/dmwm/cms-htcondor-es/wiki/Setup-and-Instructions).

- the spider script feeding es-cms runs on the vocms0240 node
- common account to use is cmsjobmon (sudo su cmsjobmon)
- vocms0240, cmsjobmon user, crontab:
```crontab
*/12 * * * * /home/cmsjobmon/cms-htcondor-es/spider_cms_queues.sh
5-59/12 * * * * /home/cmsjobmon/cms-htcondor-es/spider_cms_history.sh
0 3 * * * /bin/bash "/home/cmsjobmon/cms-htcondor-es/cronAffiliation.sh"
```
- the cronAffiliation.sh script will refresh the affiliation dictionary used to add the AffiliationInstitute and AffiliationCountry to the documents. 
- logs in /home/cmsjobmon/cms-htcondor-es/log/spider_cms.log and 
/home/cmsjobmon/cms-htcondor-es/log_history/spider_cms.log
- code resides in [github](https://github.com/dmwm/cms-htcondor-es/)
- more information in this [ticket](https://its.cern.ch/jira/browse/CMSMONIT-4) and [this](https://its.cern.ch/jira/browse/CMSMONIT-17)

Logstash/filebeat on vocms0240 processes the spider_cms logs
- Data in [ES](https://es-cms-logmon.cern.ch/kibana/app/kibana#/discover) and prototype 
[dashboard1](https://es-cms-logmon.cern.ch/kibana/app/kibana::/dashboard/c8b59e70-4cb8-11e9-aa82-3bfc29c84269)
and [dashboard2](https://es-cms-logmon.cern.ch/kibana/app/kibana::/discover/8c4d6a70-afa1-11e9-b8f6-95a3ef32a7a6) (only visible inside CERN)
- documentation/instructions: [here](https://github.com/dmwm/cms-htcondor-es/wiki/Filebeat-Logstash-setup)
- config files: [here](https://github.com/dmwm/cms-htcondor-es/tree/master/doc/logstash)
