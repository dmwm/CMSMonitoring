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

from CMSMonitoring.Validator import Validator, ClassAdsValidator

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

if __name__ == '__main__':
    unittest.main()
