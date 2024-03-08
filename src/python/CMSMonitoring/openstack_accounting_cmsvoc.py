#!/usr/bin/env python3
"""
germano.massullo@cern.ch

This script, meant to be run in acrontab under lxplus CMS VOC personal account
(not 'cmsvoc' service account), queries Openstacl CLI and retrieves metrics of
each project in space separated values, then transforms them into a JSON,
which is used to populate webpage
https://cmsdatapop.web.cern.ch/cmsdatapop/eos_openstack/openstack_accounting.html

Read
https://realpython.com/python-comments-guide/
for comments refactoring hints
"""

import collections
import os
import json
import tempfile


os.environ["OS_AUTH_URL"] = "https://keystone.cern.ch/v3"
os.environ["OS_PROJECT_DOMAIN_ID"] = "default"
os.environ["OS_AUTH_TYPE"] = "v3fedkerb"
os.environ["OS_MUTUAL_AUTH"] = "disabled"
os.environ["OS_IDENTITY_PROVIDER"] = "sssd"
os.environ["OS_PROTOCOL"] = "kerberos"
os.environ["OS_REGION_NAME"] = "cern"

def query_openstack_cli():
    accounting_file = tempfile.NamedTemporaryFile()
    try:
        # following line, originally was
        # os.system('openstack project list -f value -c Name | while read -r project ; do echo openstackProjectName $project ; OS_PROJECT_NAME="$project" openstack limits show --absolute -f value ; OS_PROJECT_NAME="$project" openstack role assignment list --project "$project" --names -f value ; echo ====== ; done > %s' % accounting_file.name)
        # all the env variables have been added in order to let this script correctly run as acrontab job. See ticket
        # RQF2243694 "can't create acrontab job"
        os.system('export OS_AUTH_URL=https://keystone.cern.ch/v3; export OS_AUTH_TYPE=v3fedkerb; export OS_PROTOCOL=kerberos; export OS_IDENTITY_PROVIDER=sssd; export OS_PROJECT_DOMAIN_ID=default; openstack project list -f value -c Name | while read -r project ; do echo openstackProjectName $project ; OS_PROJECT_NAME="$project" openstack limits show --absolute -f value ; OS_PROJECT_NAME="$project" openstack role assignment list --project "$project" --names -f value ; echo ====== ; done > %s' % accounting_file.name)
    except:
        print('Cannot get the Openstack accounting data')
    with open(accounting_file.name, 'r') as file:
        return file.readlines()

"""
query_openstack_cli() bash command takes various minutes to run. In case of
debug needs it is better to save that output into a file and just parse the file
"""
def read_file():
    with open('/afs/cern.ch/user/f/foo/openstack_accounting_summary.txt', 'r') as accounting_file:
        return accounting_file.readlines()


def get_openstack_accounting():
    dictionary_list = []
    openstack_project = {}
    lines = query_openstack_cli()
    #lines = read_file()
    for line in lines:
        if line.startswith("openstackProjectName"):
            # strip() is needed to remove \n symbol at the end of the project name
            openstack_project["openstackProjectName"] = line.partition("openstackProjectName")[2].strip()
        #
        #Details of a single Openstack project are delimited by a "======" line.
        #When the parser reaches it, it performs some cleanings in the output
        #and appends the openstack_project dictionary to dictionary_list
        #
        elif line.startswith("======"):
            # openstack_project["contacts"] is in the format of "foo@Default, bar@Default, ". Following line removes "@Default" occurrencies
            openstack_project["contacts"] = openstack_project["contacts"].replace("@Default", "")
            # openstack_project["contacts"] is in the format of "foo, bar, ". [:-2] removes the ", " at the end of the line
            openstack_project["contacts"] = openstack_project["contacts"][:-2]
            dictionary_list.append(openstack_project)
            openstack_project = {}
        elif (not line.startswith("openstackProjectName") and not line.startswith("======")):
            if line.startswith("coordinator") or line.startswith("Member") or line.startswith("operator") or line.startswith("owner"):
                line_chunks = line.split(' ')
                """
                Openstack CLI returns contacts list that sometimes have two spaces
                instead of one. This leads split(' ') to return "" as second chunk.
                If this happens, such chunk must be ignored.
                """
                # if second chunk is "", then skip to the third one
                if not line_chunks[1]:
                    # if openstack_project["contacts"] exists, append the new value
                    # to the old one
                    if "contacts" in openstack_project:
                        openstack_project["contacts"] += line_chunks[2] + ", "
                    else:
                        openstack_project["contacts"] = line_chunks[2] + ", "
                else:
                    # if openstack_project["contacts"] exists, append the new value
                    # to the old one
                    if "contacts" in openstack_project:
                        openstack_project["contacts"] += line_chunks[1] + ", "
                    else:
                        openstack_project["contacts"] = line_chunks[1] + ", "
            # the line contains a metric of the Openstack project
            else:
                attribute = line.split(" ")[0].strip()
                value = int(line.split(" ")[1].strip())
                openstack_project[attribute] = value
    # this block sums the metrics of all Openstack projects
    counter = collections.Counter()
    for d in dictionary_list:
        counter.update(d)
    total = dict(counter)
    total["openstackProjectName"] = "TOTAL"
    # cleans the huge list of contacts contained in total["contacts"]
    total["contacts"] = ""
    return dictionary_list + [total]


with open('/eos/cms/store/accounting/openstack_accounting_summary.json', 'w') as json_summary_output:
    json.dump(get_openstack_accounting(), json_summary_output, indent=4)
