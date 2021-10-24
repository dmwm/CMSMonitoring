#!/usr/bin/env python
from CMSMonitoring.StompAMQ import validate
import os
import json

print("Test")
for fname in os.listdir('schemas'):
    if fname.endswith('json'):
        data=json.load(open('schemas/{}'.format(fname)))
        res=validate(data, verbose=False)
        print("doc: {} {}".format(fname, res))
