#!/usr/bin/env python

"""
Standard python setup.py file for CMSMonitoring
"""
import os
import sys
import shutil
import fnmatch
import subprocess

from distutils.core import setup


def parse_requirements(requirements_file):
    """Create a list for the 'install_requires' component of the setup function by parsing a requirements file"""
    if os.path.exists(requirements_file):
        # return a list that contains each line of the requirements file
        return open(requirements_file, 'r').read().splitlines()
    else:
        print("ERROR: requirements file " + requirements_file + " not found.")
        sys.exit(1)


def datafiles(dir, pattern=None):
    """Return list of data files in provided relative dir"""
    files = []
    for dirname, dirnames, filenames in os.walk(dir):
        for subdirname in dirnames:
            files.append(os.path.join(dirname, subdirname))
        for filename in filenames:
            if filename[-1] == '~':
                continue
            # match file name pattern (e.g. *.css) if one given
            if pattern and not fnmatch.fnmatch(filename, pattern):
                continue
            files.append(os.path.join(dirname, filename))
    return files


def get_version_str(init_file):
    """Parses __init__.py to get version in case of version import exception"""
    with open(init_file) as f:
        for line in f.readlines():
            line = line.replace('\n', '')
            if line.startswith('__version__'):
                return str(line.split('=')[-1]).strip().replace("'", "").replace('"', '')


def main():
    """Main setup"""
    # this try/except block is taking care of setting up proper
    # version either by reading it from CMSMonitoring package
    # this part is used by pip to get the package version
    try:
        import CMSMonitoring
        version = CMSMonitoring.__version__
    except Exception as e:
        version = '0.0.0'
        init_file = 'CMSMonitoring/__init__.py'
        if os.path.exists(init_file):
            version = get_version_str(init_file)

    ver = sys.version.split(' ')[0]
    pver = '.'.join(ver.split('.')[:-1])
    lpath = 'lib/python{}/site-packages'.format(pver)
    root = os.getcwd().replace('/src/python', '')
    schema_path = '{}/schemas'.format(root)
    json_path = '{}/jsonschemas'.format(root)
    static_path = '{}/static'.format(root)
    data_files = [
        ('jsonschemas', datafiles(json_path, '*.schema')),
    ]
    for pair in data_files:
        path = 'CMSMonitoring/{}'.format(pair[0])
        if not os.path.exists(path):
            os.makedirs(path)
            for fname in pair[1]:
                shutil.copy(fname, path)
    data_files = [
        ('jsonschemas', datafiles('CMSMonitoring/jsonschemas', '*.schema')),
    ]

    dist = setup(
        name='CMSMonitoring',
        version=version,
        author='Valentin Kuznetsov',
        author_email='vkuznet@gmail.com',
        license='MIT',
        description='CMS Monitoring utilities',
        long_description='CMS Monitoring utilities',
        packages=['CMSMonitoring'],
        package_dir={'CMSMonitoring': 'CMSMonitoring'},
        install_requires=parse_requirements("requirements.txt"),
        scripts=['bin/%s' % s for s in os.listdir('bin')],
        url='https://github.com/dmwm/CMSMonitoring',
        include_package_data=True,
        data_files=data_files,
        classifiers=[
            "Programming Language :: Python",
            "Operating System :: OS Independent",
            "License :: OSI Approved :: MIT License",
        ],
    )

    # remove jsonschemas from CMSMonitoring
    if os.path.exists('CMSMonitoring/jsonschemas'):
        shutil.rmtree('CMSMonitoring/jsonschemas')


def read_init(init_file):
    "Read package init file and return its content"
    init = None
    with open(init_file) as istream:
        init = istream.read()
    return init


def write_version(init_file, init_content):
    "Write package init file with given content"
    if not init_content:
        init_content = \
            """
            __version__ = '0.0.0'
            __all__ = []
            """
    if init_file:
        with open(init_file, 'w') as ostream:
            ostream.write(init_content)


def version():
    "Return git tag version of the package or custom version"
    cmd = 'git tag --list | tail -1'
    ver = subprocess.Popen(cmd, shell=True, stdout=subprocess.PIPE).stdout.read()
    ver = str(ver.decode("utf-8")).replace('\n', '')
    ver = ver if ver else '0.0.0'
    return ver


if __name__ == "__main__":
    # This part is used by `python setup.py sdist` to build package tar-ball
    # read git version
    # ver = version()
    # read package init file
    # init_file = 'CMSMonitoring/__init__.py'
    # init = read_init(init_file)
    # replace package init file with our version
    # write_version(init_file, init.replace('0.0.0', ver))
    # execute setup main
    main()
    # put back original content of package init file
    # write_version(init_file, init.replace(ver, '0.0.0'))
