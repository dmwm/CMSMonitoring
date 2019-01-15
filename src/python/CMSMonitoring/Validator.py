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

class ClassAdsValidator(Validator):
    def __init__(self, schema=None):
        if not schema:
            schema = os.environ.get('CLASSADS_SCHEMA', '')
        super(ClassAdsValidator, self).__init__(schema)
