#!/usr/bin/env python

"""
Standard python setup.py file for CMSMonitoring
"""
from __future__ import print_function
__author__ = "Valentin Kuznetsov"

import os
import sys
import subprocess

from distutils.core import setup

def version():
    "Return git tag version of the package or custom version"
    cmd = 'git tag --list | tail -1'
    ver = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE).stdout.read().replace('\n', '')
    return ver if ver else 'development'

def main():
    with open("README.md", "r") as fh:
        long_description = fh.read()

    dist = setup(
        name                 = 'CMSMonitoring',
        version              = version(),
        description          = 'CMS Monitoring utilities',
        long_description     = long_description,
        keywords             = ['CMS experiment', 'CERN', 'Monitoring'],
        packages             = ['CMSMonitoring'],
        package_dir          = {'CMSMonitoring': 'src/python/CMSMonitoring'},
        requires             = ['jsonschema (>=2.6.0)', ' genson (>=1.0.2)', 'stomp.py (==4.1.21)'],
        classifiers          = ["Operating System :: OS Independent",
            "License :: OSI Approved :: MIT License",
            "Programming Language :: Python", "Topic :: Monitoring"],
        scripts              = ['bin/%s'%s for s in os.listdir('bin')],
        author               = 'Valentin Kuznetsov',
        author_email         = 'vkuznet@gmail.com',
        url                  = 'https://github.com/dmwm/CMSMonitoring',
        license              = license,
    )

if __name__ == "__main__":
    main()

