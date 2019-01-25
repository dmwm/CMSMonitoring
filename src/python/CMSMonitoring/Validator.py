#!/usr/bin/env python
#-*- coding: utf-8 -*-
#pylint: disable=
"""
File       : Validator.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: set of utilities to validate CMS Monitoring schema(s)
"""

# system modules
import os
import sys
import json

def validate_schema(doc, schema):
    "Validate schema of a given document"
    base = 'SCHEMA_VALIDATOR'
    if not isinstance(doc, dict):
        return False
    for key, val in doc.items():
        if key not in schema:
            print("{}: key={} is not in a schema".format(base, key))
            return False
        if isinstance(val, dict):
            sub_schema = validate_schema(val, schema[key])
            if not sub_schema:
                print("{}: for sub schema={} val={} has wrong data-types".format(base, schema[key], val))
                return False
        expect = schema[key]
        if isinstance(val, list):
            types = set([type(x) for x in val])
            if len(types) != 1:
                print("{}: for key={} val={} has inconsisten data-types".format(base, key, val))
                return False
            if list(types)[0] != type(expect[0]):
                print("{}: for key={} val={} has incorrect data-type in list, found {} expect {}".format(base, key, val, type(val), type(expect)))
                return False
        if type(val) != type(expect):
            print("{}: for key={} val={} has incorrect data-type, found {} expect {}".format(base, key, val, type(val), type(expect)))
            return False
    return True

class Validator(object):
    def __init__(self, schema):
        if not schema:
            raise Exception('{}: unknown schema'.format(__file__))
        self.schema = json.load(open(schema))

    def validate(self, key, value):
        "Validate given key/value pair against the schema"
        if value not in self.schema.get(key, []):
            return False
        return True

    def validate_all(self, doc):
        "Validate all keys in a given document"
        if not isinstance(doc, dict):
            return False
        for key, val in doc.items():
            if not self.validate(key, val):
                return False
        return True

    def validate_schema(self, doc):
        "Validate schema of a given document"
        return validate_schema(doc, self.schema)

class ClassAdsValidator(Validator):
    def __init__(self, schema=None):
        if not schema:
            schema = os.environ.get('CLASSADS_SCHEMA', '')
        super(ClassAdsValidator, self).__init__(schema)
