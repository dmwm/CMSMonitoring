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
    ver = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE).stdout.read()
    return ver if ver else 'development'

version      = version()
name         = "CMSMonitoring"
description  = "CMS Monitoring utilities"
readme       ="""
CMSMonitoring: https://github.com/dmwm/CMSMonitoring
"""
author       = "Valentin Kuznetsov",
author_email = "vkuznet@gmail.com",
url          = "https://github.com/dmwm/CMSMonitoring",
keywords     = ["CMS", "Monitoring"]
package_dir  = {'CMSMonitoring': 'src/python/CMSMonitoring'}
packages     = ['CMSMonitoring']
license      = "CMS experiment software"
classifiers  = [
    "Development Status :: 3 - Production/Beta",
    "Intended Audience :: Developers",
    "License :: OSI Approved :: CMS/CERN Software License",
    "Operating System :: MacOS :: MacOS X",
    "Operating System :: Microsoft :: Windows",
    "Operating System :: POSIX",
    "Programming Language :: Python",
    "Topic :: Monitoring"
]

def main():
    dist = setup(
        name                 = name,
        version              = version,
        description          = description,
        long_description     = readme,
        keywords             = keywords,
        packages             = packages,
        package_dir          = package_dir,
        requires             = ['jsonschema (>=2.6.0)', ' genson (>=1.0.2)', 'stomp.py (==4.1.21)'],
        classifiers          = classifiers,
        scripts              = ['bin/%s'%s for s in os.listdir('bin')],
        author               = author,
        author_email         = author_email,
        url                  = url,
        license              = license,
    )

if __name__ == "__main__":
    main()

