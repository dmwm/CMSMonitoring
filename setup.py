#!/usr/bin/env python

"""
Standard python setup.py file for CMSMonitoring
"""
import os
import sys
import fnmatch
import subprocess

from distutils.core import setup

def datafiles(dir, pattern=None):
    """Return list of data files in provided relative dir"""
    files = []
    for dirname, dirnames, filenames in os.walk(dir):
        for subdirname in dirnames:
            files.append(os.path.join(dirname, subdirname))
        for filename in filenames:
            if  filename[-1] == '~':
                continue
            # match file name pattern (e.g. *.css) if one given
            if pattern and not fnmatch.fnmatch(filename, pattern):
                continue
            files.append(os.path.join(dirname, filename))
    return files

def version():
    "Return git tag version of the package or custom version"
    cmd = 'git tag --list | tail -1'
    ver = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE).stdout.read().replace('\n', '')
    return ver if ver else 'development'

def main():
    ver = sys.version.split(' ')[0]
    pver = '.'.join(ver.split('.')[:-1])
    lpath = 'lib/python{}/site-packages'.format(pver)
    dist = setup(
        name                 = 'CMSMonitoring',
        version              = version(),
        author               = 'Valentin Kuznetsov',
        author_email         = 'vkuznet@gmail.com',
        description          = 'CMS Monitoring utilities',
        long_description     = 'CMS Monitoring utilities',
        packages             = ['CMSMonitoring'],
        package_dir          = {'CMSMonitoring': 'src/python/CMSMonitoring'},
        install_requires     = ['jsonschema>=2.6.0', 'genson>=1.0.2', 'stomp.py==4.1.21'],
        scripts              = ['bin/%s'%s for s in os.listdir('bin')],
        url                  = 'https://github.com/dmwm/CMSMonitoring',
        data_files           = [
            ('{}/CMSMonitoring/schemas'.format(lpath), datafiles('schemas', '*.json')),
            ('{}/CMSMonitoring/jsonschemas'.format(lpath), datafiles('jsonschemas', '*.schema')),
            ],
        classifiers          = [
            "Programming Language :: Python",
            "Operating System :: OS Independent",
            "License :: OSI Approved :: MIT License",
            ],
    )

if __name__ == "__main__":
    main()
