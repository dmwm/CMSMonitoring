### CMSMonitoring training
This document describe necessary steps to inject your data into CERN MONIT
infrastructure. To do that you need to have the following items:
- a copy of CMSMonitoring codebase
- prepare your documents in JSON data-format
  - it is advisiable to have a flat
    key-value structure of your documents rather then nesting one. 
  - all documents should contain an identical set of keys, i.e. you
  cannot have one document with `key1` and `key2`, while another document
  will have totally different set of keys, e.g. `key3` and `key4`.
  In this case we say that two documents schema do not overlap. But it is
  possible that you may omit a certain keys, in this case a default value
  may be used in MONIT for them
  - always try to add your own timestamp in your JSON document, the timestamp
  data-format should be in UNIX since epoch since it is easy to parse and
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
  accordingly. In a ticket you should specify the following **mandatory** items:
  - a data volume you foresee from your data-provider to CERN MONIT, an
  approximate numbers are sufficient, e.g. size of JSON document times
  number of docs per certain time
  - an approximate injection rate, e.g. 1K docs per day
  - you should provide desired topic name where your docs will appear, e.g.
  cms-my-topic (try always use cms prefix followed by your topic name)
  - your preference for authentication, password based or X509 certificate
  - provide an e-group which will be used for communication between your
  team and CERN MONIT

#### CMS Monitoring setup
```
cd workdir

git clone git@github.com:dmwm/CMSMonitoring.git

# install all dependencies
pip install -r requirements.txt

# go to python codebase area
cd CMSMonitoring/src/python

# export PYTHONPATH
export PYTHONPATH=$PWD
```

At this point your environment is set to inject data into CERN MONIT.
Next, we'll show how to write a simple code to do that:
```
#!/usr/bin/env python

from CMSMonitoring.StompAMQ import StompAMQ

def records():
    "example of function which can generate JSON records"
    for i in range(10):
        doc = {"key1":i}
        yield doc

def credentials(fname=None):
    "Read credentials from WMA_BROKER environment"
    if  not fname:
        fname = os.environ.get('WMA_BROKER', '')
    if  not os.path.isfile(fname):
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
creds = credentials(opts.amq)
host, port = creds['host_and_ports'].split(':')
port = int(port)
# create instance of StompAMQ object with your credentials
amq = StompAMQ(creds['username'], creds['password'],
               creds['producer'], creds['topic'],
               validation_schema=None, host_and_ports=[(host, port)])

# loop over your document records and create notification documents
# we will send to MONIT
data = []
for doc in documents:
    # every document should be hash id
    hid = doc.get("hash", 1) # replace this line with your hash id generation
    notification, _, _ = amq.make_notification(doc, hid)
    data.append(notification)

# send our data to MONIT
results = amq.send(data)
```
