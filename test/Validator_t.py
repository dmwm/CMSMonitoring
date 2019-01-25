#!/usr/bin/env python
#-*- coding: utf-8 -*-
#pylint: disable=
"""
File       : Validator_t.py
Author     : Valentin Kuznetsov <vkuznet AT gmail dot com>
Description: unit test code for CMS Monitoring Validator
"""

# system modules
import os
import sys
import unittest

from CMSMonitoring.Validator import Validator, ClassAdsValidator, validate_schema

class testDASCore(unittest.TestCase):
    def setUp(self):
        fpath = __file__.split('test')[0]
        if not fpath:
            fpath = os.getcwd()
        schema = "{}/schemas/ClassAds.json".format(fpath)
        self.validator = ClassAdsValidator(schema)

    def testValidate(self):
        "Test validate method of our validator instance"
        result = self.validator.validate('Pool', 'Global')
        self.assertEqual(result, True)
        result = self.validator.validate('Pool', 'Bla')
        self.assertEqual(result, False)
        result = self.validator.validate('P', '')
        self.assertEqual(result, False)

    def testValidateSchema(self):
        "Test validate_schema function"
        schema = {'s':'s', 'i':1, 'd':{'ds':'s', 'di':1}, 'l':[1,2]}
        result = validate_schema(schema, schema)
        self.assertEqual(result, True)

        doc = {'s': 1}
        result = validate_schema(doc, schema)
        self.assertEqual(result, False)

        doc = {'s':'s', 'i':1, 'd':{'ds':'s', 'di':1}, 'l':[1,'2']}
        result = validate_schema(doc, schema)
        self.assertEqual(result, False)

        doc = {'s':'s', 'i':1, 'd':{'ds':1, 'di':1}, 'l':[1,2]}
        result = validate_schema(doc, schema)
        self.assertEqual(result, False)

if __name__ == '__main__':
    unittest.main()
