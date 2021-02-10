# How to inject data into MONIT

Please read the requirements from MONIT [first](https://monitdocs.web.cern.ch/monitdocs/ingestion/messaging.html).

You can also follow the step-by-step tutorial [here](https://github.com/dmwm/CMSMonitoring/blob/master/doc/training/data_injection.md) 
for data injection to a test topic, and data access and exploration.

## General instructions

In order to add your data to CERN MONIToring infrastructure please read the [documentation](http://monit-docs.web.cern.ch/monit-docs/ingestion/index.html), and follow these steps:
- Actions from your team: 
  - provide a sample of document which you'll intend to feed into CERN MONIT system. The data should be in JSON data-format.
  - you need to provide an estimate of data rates and volume you anticipate to feed into CERN MONIT system
- Actions from CERN MONIT team:
  - we need to request from them an entry point and collection where you'll send your data
  - they will ask for the following information:
    - What is the description of the type of data you are sending?
      - we suggest to use something like monit_prod_cms_yourproject
    - Do you have suggestions or preferences for the topic name?
      - use something like topic/cms_yourproject
    - Which email address (preferably an e-group) should be listed as responsible for the data producer?
      - this is not very high traffic, feel free to use or create your alert/ops egroup
    - How will you authenticate to the brokers (certificate or regular account)?
      - in our previous topics we used user/password authentication. The MONIT team provided us with credentials.
    - Who (i.e. which authenticated principals) should be authorized to send messages? to consume message?
      - in previous requests we explicitly listed nodes which will send the data, and a common account which is used to push the data. The consumer will be the CERN MONIT team.
  - once the entry point is created and you'll get your credentials and URLs, they will ask you to send few docs to verify the data flow
- Actions from your team:
  - you'll need to send data into CERN MONIT end-point via StompAMQ. The module exists in WMCore area, 
  see [StompAMQ.py module](https://github.com/dmwm/CMSMonitoring/tree/master/src/python/CMSMonitoring)
  and you may have a look at example how WMArchive does it [here](https://github.com/dmwm/WMArchive/blob/master/src/python/WMArchive/Tools/myspark.py#L425-L440),
  [here](https://github.com/dmwm/WMArchive/blob/master/src/python/WMArchive/Tools/myspark.py#L348-L356),
  [here](https://github.com/dmwm/WMArchive/blob/master/src/python/WMArchive/Tools/myspark.py#L348-L356) and 
  [here](https://github.com/dmwm/WMArchive/blob/master/src/python/WMArchive/Tools/myspark.py#L44).
  - Basically, you'll load StompAMQ module, parse your credential json file (CERN MONIT will give it to you) and grab
  your docs, wrap them up and send to MONIT via AMQ. In order to use StompAMQ module you'll need to load [py-stomp](https://pypi.org/project/stomp.py/)
  package which is available at PyPi  or use py2-stomp RPM from CMS comp repository, see [py2-stomp.spec](https://github.com/cms-sw/cmsdist/blob/comp_gcc630/py2-stomp.spec).
  - Each data-provider should provide a document describing their data attributes. A good example is [HTCondor](https://github.com/dmwm/cms-htcondor-es/blob/master/README.md).
  - The code used to feed data into MONIT should reside in some cms common repository (for example dmwm or cmssw)
  - you'll need to find your data in Kibana/Grafana and verify their content
    - Actions from us and/or CERN MONIT team:
       - we may help you to find your data in Kibana and show how to create a plot
       - we may help you to find your data in Grafana and create first dashboard
- Actions from your team:
  - you'll start feeding the data on a regular basis
  - provide links to code and documentation of attributes/schema
  - you'll create/support necessary dashboards to visualize your data

By the end of this process we'll include your dashboard into the central CMS Monitoring web page.

For more information about CERN ActiveMQ message passing please see this [page](http://monit-docs.web.cern.ch/monit-docs/ingestion/messaging.html).

Once your code is in production, you may use [this](https://github.com/dmwm/cms-htcondor-es/wiki/CMSMonitoring-Data-Flow-Test-procedure)
procedure to test new versions of the code.
