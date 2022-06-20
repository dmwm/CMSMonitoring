#!/usr/bin/env python
# -*- coding: utf-8 -*-
# pylint: disable=C0301,R0903,R0912,W0703,C0103
"""
File       : Validator.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: set of utilities to validate CMS Monitoring schema(s)
The API to use is validate_schema. We provide two implementation:
- based on jsonschema [1] and genson [2] packages
- pure python based implementation

[1] https://github.com/Julian/jsonschema
[2] https://github.com/wolverdude/genson/
"""
from __future__ import print_function

# system modules
import os
import json
import time
import hashlib
import logging

try:
    import jsonschema

    JSONSCHEMA = True
except ImportError:
    JSONSCHEMA = False


def md5hash(rec):
    """Return md5 hash of given query"""
    if not isinstance(rec, dict):
        raise NotImplementedError
    # discard timestamp fields from hash calculations since they're dynamic
    record = dict(rec)
    rec = json.JSONEncoder(sort_keys=True).encode(record)
    keyhash = hashlib.md5()
    try:
        keyhash.update(rec)
    except TypeError:  # python3
        keyhash.update(rec.encode('ascii'))
    return keyhash.hexdigest()


# global pool of validators
VALIDATORS = {}


class JsonSchemaValidator(object):
    "Python 2.X implementation of validator based on jsonschema package"

    def __init__(self, schema, logger=None):
        logging.basicConfig(level=logging.WARNING)
        self.logger = logger if logger else logging.getLogger('JsonSchemaValidator')
        shash = md5hash(schema)
        if shash in VALIDATORS:
            self.validator = VALIDATORS[shash]
        else:
            self.validator = jsonschema.validators.validator_for(schema)(schema)
            self.validator.check_schema(schema)
            VALIDATORS[shash] = self.validator

    def validate_schema(self, doc):
        "validation document agains validator schema"
        try:
            self.validator.validate(doc)
        except Exception as exp:
            msg = "Fail schema validation: {}".format(str(exp))
            self.logger(msg)
            return False
        return True


class Schemas(object):
    "Schemas object provides access to CMSMonitoring schema files"

    def __init__(self, update=3600, jsonschemas=False):
        self.tstamp = time.time()
        self.update = update
        self.sdict = {}
        self.jsonschemas = jsonschemas

    def schemas(self):
        "Return all known CMSMonitoring schemas"
        if self.sdict and (time.time() - self.tstamp) < self.update:
            return self.sdict
        if 'CMSMONITORING_SCHEMAS' in os.environ:
            fdir = os.environ['CMSMONITORING_SCHEMAS']
        else:
            code_dir = '/'.join(__file__.split('/')[:-1])
            if os.path.join(code_dir, 'schemas'):
                fdir = os.path.join(code_dir, 'schemas')
            else:
                fname = __file__.split('CMSMonitoring/Validator.py')[0].split('CMSMonitoring')[0]
                fdir = '{}/CMSMonitoring/schemas'.format(fname)
        if self.jsonschemas:
            if 'CMSMONITORING_JSONSCHEMAS' in os.environ:
                fdir = os.environ['CMSMONITORING_JSONSCHEMAS']
            else:
                code_dir = '/'.join(__file__.split('/')[:-1])
                if os.path.join(code_dir, 'jsonschemas'):
                    fdir = os.path.join(code_dir, 'jsonschemas')
                else:
                    fdir = '{}/CMSMonitoring/jsonschemas'.format(fname)
        snames = []
        try:
            snames = os.listdir(fdir)
        except OSError:
            raise Exception('Invalid CMSMonitoring schemas area: {}'.format(fdir))
        except Exception as exp:
            raise Exception('Invalid CMSMonitoring schemas area: {}, error={}'.format(fdir, str(exp)))
        for sname in snames:
            self.sdict[sname] = json.load(open(os.path.join(fdir, sname)))
        return self.sdict


def validate_jsonschema(schema, doc, logger=None):
    """
    Jsonschema implementation of validate schema of a given document
    :param schema: schema to be used
    :param doc: document to be validated
    """
    validator = JsonSchemaValidator(schema, logger)
    return validator.validate_schema(doc)


def etype(val):
    "Helper function to deduce type of given value either from python based type or jsonschema"
    if isinstance(val, dict) and 'type' in val:
        return val['type']

    for _type in [str, bool]:
        if isinstance(val, _type):
            return _type

    if isinstance(val, (int)):
        return (int)

    if isinstance(val, (float)):
        return (float, int)

    return type(val)


def _validate_schema(schema, doc, logger):
    """
    Python based implementation to validate schema of a given document
    :param schema: schema to be used
    :param doc: document to be validated
    """
    base = 'SCHEMA_VALIDATOR'
    if not isinstance(doc, dict):
        return None, None

    offending_keys = []
    unknown_keys = []

    for key, val in doc.items():
        if key not in schema:
            msg = "{}: unknown key={}, val={}".format(base, key, repr(val))
            logger.warn(msg)
            unknown_keys.append(key)
            continue

        if isinstance(val, dict):
            sub_offending, _ = _validate_schema(schema[key], val, logger)
            if sub_offending:
                msg = "{}: for sub schema={} val={} has wrong data-types".format(base, schema[key], repr(val))
                logger.warn(msg)
                offending_keys.append(key)
                continue

        expect = schema[key]
        if isinstance(val, list):
            if len(set([type(x) for x in val])) != 1:
                msg = "{}: for key={} val={} has inconsistent data-types".format(base, key, repr(val))
                logger.warn(msg)
                offending_keys.append(key)
                continue

            if not isinstance(val[0], etype(expect[0])):
                msg = "{}: for key={} val={} has incorrect data-type in list, found {} expect {}".format(base, key,
                                                                                                         repr(val),
                                                                                                         type(val[0]),
                                                                                                         etype(
                                                                                                             expect[0]))
                logger.warn(msg)
                offending_keys.append(key)
                continue

        if val != None and not isinstance(val, etype(expect)):
            msg = "{}: for key={} val={} has incorrect data-type, found {} expect {}".format(base, key, repr(val),
                                                                                             type(val), etype(expect))
            logger.warn(msg)
            offending_keys.append(key)
            continue

    return offending_keys, unknown_keys


def validate_schema(schema, doc, logger):
    """
    validates given document against given shema
    :param schema: schema to be used
    :param doc: document to be validated
    """
    if '$schema' in schema:
        logger.debug("using jsonschema validator")
        return validate_jsonschema(schema, doc, logger)
    logger.debug("using python based validator")
    return _validate_schema(schema, doc, logger)


class Validator(object):
    """
    Validator class
    """

    def __init__(self, schema, logger=None):
        if not schema:
            raise Exception('{}: unknown schema'.format(__file__))
        self.schema = json.load(open(schema))
        logging.basicConfig(level=logging.WARNING)
        self.logger = logger if logger else logging.getLogger('Validator')

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
        return validate_schema(self.schema, doc, self.logger)


class ClassAdsValidator(Validator):
    """
    ClassAdsValidation class
    """

    def __init__(self, schema=None):
        if not schema:
            schema = os.environ.get('CLASSADS_SCHEMA', '')
        super(ClassAdsValidator, self).__init__(schema)
