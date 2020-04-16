### CMSMonitoring training
This document describes the necessary steps to inject your data into the CERN MONIT
infrastructure. To do that you need to have the following items:
- a copy of CMSMonitoring codebase
- prepare your documents in JSON data-format
  - it is advisable to have a flat
    key-value structure of your documents rather then nesting one. 
  - all documents should contain an identical set of keys, i.e. you
  cannot have one document with `key1` and `key2`, while another document
  will have totally different set of keys, e.g. `key3` and `key4`.
  In this case we say that two documents schema do not overlap. But it is
  possible that you may omit a certain keys, in this case a default value
  may be used in MONIT for them
  - always try to add your own timestamp in your JSON document, the timestamp
  data-format should be in UNIX epoch since it is easy to parse and
  convert into different time zones if necessary
  - try to stick with proper data-types, e.g. if you supply a run number
  make sure it is an integer and not a string data-type. This will simplify
  tramendously data look-up and aggregation across different data-providers
- you should obtain a proper end-point from CERN MONIT team where data will
  float. You can arrange this by opening either [CMSMONIT](https://its.cern.ch/jira/browse/CMSMONIT)
  or [SNOW](https://cern.service-now.com/service-portal/home.do) ticket.
  The former will be directed to CMS Monitoring group and we'll coordinate
  its progress with CERN MONIT team, the later will go directly to
  CERN MONIT line of support and bypass CMS Monitoring. Please choose
  accordingly. In the ticket you should specify the following **mandatory** items:
  - a data volume you foresee from your data-provider to CERN MONIT, an
  approximate numbers are sufficient, e.g. size of JSON document times
  number of docs per certain time
  - an approximate injection rate, e.g. 1K docs per day
  - you should provide desired topic name where your docs will appear, e.g.
  cms-my-topic (try always to use the cms prefix followed by your topic name)
  - your preference for authentication, password based or X509 certificate
  - provide an e-group which will be used for communication between your
  team and CERN MONIT

#### CMS Monitoring setup
Due to authentication policy at CERN MONIT infrastructure
we should either proceed with training from
`lxplus` node and/or user must request a new end-point
with proper credentials from the CERN MONIT group. For that purpose
please open up a [SNOW
ticket](https://cern.service-now.com/service-portal/home.do).

Otherwise please login to `lxplus` and proceed with example
below:
```
# create a working directory
cd workdir

# Create and activate a python virtual environment
virtualenv venv
source ./venv/bin/activate

# Clone the CMSMonitoring repository
git clone https://github.com/dmwm/CMSMonitoring.git

# install all dependencies
pip install -r CMSMonitoring/requirements.txt

# go to python codebase area
cd CMSMonitoring/src/python

# export PYTHONPATH
export PYTHONPATH="$PWD:$PYTHONPATH"
```
we also need to setup our broker credentials for that we'll use a dedicated file, training.json.  Please create it with the following content (for production needs you'll have similar file), in your working folder.
```json
{
    "producer":"cms-training",
    "topic":"/topic/cms.training",
    "host_and_ports":"cms-test-mb.cern.ch:61323"
}
```
You can use this instructions to do so and set the `MONIT_BROKER` environment variable.
```bash
cat <<EOF > training.json
{
    "producer":"cms-training",
    "topic":"/topic/cms.training",
    "host_and_ports":"cms-test-mb.cern.ch:61323"
}
EOF
# once this file is in place you'll need to setup an environment variable
export MONIT_BROKER=$PWD/training.json
```

#### Data injection
At this point your environment is set to inject data into CERN MONIT.
Next, we'll show how to write a simple code to do that:
```python
#!/usr/bin/env python

# system modules
import os
import json

# CMSMonitoring modules
from CMSMonitoring.StompAMQ import StompAMQ

def records():
    "example of function which can generate JSON records"
    for i in range(100):
        doc = {"key":i, "data":"data-{}".format(i), "user":os.getenv("USER")}
        yield doc

def credentials(fname=None):
    "Read credentials from MONIT_BROKER environment"
    if  not fname:
        fname = os.environ.get('MONIT_BROKER', '')
    if  not os.path.isfile(fname):
        raise Exception("Unable to locate MONIT credentials, please setup MONIT_BROKER")
        return {}
    with open(fname, 'r') as istream:
        data = json.load(istream)
    return data

# REPLACE THIS line with list of your JSON records
# you may fetch them from a file, from DB, from other source
# but you should be able to provide this list. In this example
# we will use our custom records function which provides them.
documents = records()

# read our credentials
creds = credentials()
host, port = creds['host_and_ports'].split(':')
port = int(port)
# we will authenticate with our grid certificate
# but we still need to pass username and password to StompAMQ
# therefore we'll use empty strings for them
username = ""
password = ""
# please do not copy certificates anywhere, they are only valid for training
ckey = '/afs/cern.ch/user/c/cmsmonit/public/.globus/robot-training-key.pem'
cert = '/afs/cern.ch/user/c/cmsmonit/public/.globus/robot-training-cert.pem'
producer = creds['producer']
topic = creds['topic']
print("producer: {}, topic {}".format(producer, topic))
print("ckey: {}, cert: {}".format(ckey, cert))
# create instance of StompAMQ object with your credentials
amq = StompAMQ(username, password,
               producer, topic,
               key=ckey, cert=cert,
               validation_schema=None, host_and_ports=[(host, port)])

# loop over your document records and create notification documents
# we will send to MONIT
data = []
for doc in documents:
    # every document should be hash id
    hid = doc.get("hash", 1) # replace this line with your hash id generation
    notification, _, _ = amq.make_notification(doc,"training_document", docId=hid)
    data.append(notification)

# send our data to MONIT
results = amq.send(data)
print("results", results)
```
The output of this script will yield the following output
```
producer: cms-training, topic /topic/cms.training
ckey: /home/cmspopdb/.globus/userkey.pem, cert: /home/cmspopdb/.globus/usercert.pem
WARNING:StompAMQ:No document validation performed!
WARNING:stomp.py:[Errno 0] Error
('results', [])
```
And, the data will appear in ElasticSearch under
[monit_prod_cms-training](https://es-monit.cern.ch/kibana/goto/67aafadf62076462a8c2c7b5bfdf1a5b)
index.

##### Data injection and look-up using monit Go-tool
We also provide Go tool `monit` which can be used for data injection and data
look-up. This section provides basic steps how to use it. You may get it either
under `/cvmfs/cms.cern.ch/cmsmonit-tools` area or build it on your own.
```
# to inject data to MONIT you need

# a file with your data, e.g. doc.json
{"country":"XYZ","id":1,"metadata":{"uuid":"77ae63ce-a648-425f-a09d-cb7ed6f1457f"},"requestedCores":3.39,"requestedMB":1054,"siteName":"T3_US_XXX","tier":"T0","training_username":"test-user","usedMB":1156}

# credential file, e.g. cat creds.json
{
    "producer":"cms-training",
    "topic":"/topic/cms.training",
    "key": "/path/robot-training-key.pem",
    "cert": "/path/robot-training-cert.pem",
    "host_and_ports":"cms-test-mb.cern.ch:61323"
}

# run the following command
monit -creds=creds_training.json -inject=doc.json
```
And, if you want to look-up the data use the following:
```
# a file with your ES query, e.g. cat query.json
{"query":{"match":{"data.payload.status": "Available"}}, "from": 0, "size": 10}

# run the following command, here you should specify the dbname to use for your query
monit -token token -query=query.json -dbname=monit_prod_wmagent

# all datasources (databases) can be found by using this command
monit -datasources
```

#### How to update your documents in MONIT/ES
It is possible to update document(s) in MONIT ES. For that user need to re-run
injection with StompAMQ (as shown in section above), but this time the updated
document should contain an additional `_id` key-value along with actual data.
The value of `_id` attributed will be used by ES to create a new version
of the document.

#### How to visualize your data
The injected data can be visualized either in ES/Kibana
or grafana. For former, you need to visit `Visualize` section
of 
[monit_prod_cms-training](https://es-monit.cern.ch/kibana/goto/67aafadf62076462a8c2c7b5bfdf1a5b)
page and choose appropriate Chart.

For later, please visit
[monit-grafana.cern.ch](https://monit-grafana.cern.ch/d/000000530/cms-monitoring-project?orgId=11)
page and select CMS monitoring project (orgId=11). You can
either create a new dashboard with
[monit_prod_cms-training](https://monit-grafana.cern.ch/datasources/edit/9411/)
data-source or 
just visit [CMS training](https://monit-grafana.cern.ch/d/Cp1mIXJWk/cms-training?orgId=11)
dashboard and play around with it.

#### How to check the data in HDFS

From a lxplus machine setup the environment to use HADOOP:
```bash
 source "/cvmfs/sft.cern.ch/lcg/views/LCG_96python3/x86_64-centos7-gcc8-opt/setup.sh"
 source "/cvmfs/sft.cern.ch/lcg/etc/hadoop-confext/hadoop-swan-setconf.sh" analytix
```
Now you can list the files for this topic. The folder structure is: `/project/monitoring/archive/<producer>/raw/metric/<yyyy>/<MM>/<dd>`. 
The MonIT data flow will use a tmp folder for the last day, as they have a compaction process daily.
So, to list the files created for today, using the cms-training producer, you can use:
```bash
hadoop fs -ls /project/monitoring/archive/cms-training/raw/metric/$(date +%Y/%m/%d).tmp
```
And see the content using cat:
```bash
hadoop fs -cat /project/monitoring/archive/cms-training/raw/metric/$(date +%Y/%m/%d).tmp/*
```
