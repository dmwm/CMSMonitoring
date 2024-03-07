# Germano Massullo - germano.massullo@cern.ch
# Dario Mapelli - dario.mapelli@cern.ch

# This script is running in acrontab in lxplus under the personal account (not 'cmsvoc' service account) of the CMS VOC, because such account has full permissions.
# A very extensive explanation of this script is available at
# https://its.cern.ch/jira/browse/CMSMONIT-521?focusedId=4724501&page=com.atlassian.jira.plugin.system.issuetabpanels:comment-tabpanel#comment-4724501
# Its text cannot be pasted here for both reserved comments and lack of layout capabilities in a Python file compared to a comment in a Jira ticket.
# A copy of this script is hosted for archival purposes at https://github.com/dmwm/CMSMonitoring

import os
import json
import logging
import tempfile


EXCLUDED_PATHS = ["/eos/cms/store/cmst3", "/eos/recovered", "/eos/totem"]

# many numbers comes from EOS tools in form of string. We need to convert them
# to numbers (integer or float)
# to be better handled by the CMS monitoring web page
def convert_to_terabytes_and_cast_from_string_to_number__non_ec(dictionary_list):
    for item in dictionary_list:
        item['usedbytes'] = float(item['usedbytes']) / (10**12)
        item['usedlogicalbytes'] = float(item['usedlogicalbytes']) / (10**12)
        item['maxbytes'] = float(item['maxbytes']) / (10**12)
        item['maxlogicalbytes'] = float(item['maxlogicalbytes']) / (10**12)
        try:
            item['used_logical_over_used_raw_percentage'] = item['usedlogicalbytes'] / item['usedbytes'] * 100
        except:
            item['used_logical_over_used_raw_percentage'] = None
        item['usedterabytes'] = item.pop('usedbytes')
        item['usedlogicalterabytes'] = item.pop('usedlogicalbytes')
        item['maxterabytes'] = item.pop('maxbytes')
        item['maxlogicalterabytes'] = item.pop('maxlogicalbytes')
        item['percentageusedbytes'] = float(item['percentageusedbytes'])
        item['percentageusedterabytes'] = item.pop('percentageusedbytes')
        item['usedfiles'] = int(item['usedfiles'])
        item['maxfiles'] = int(item['maxfiles'])
    return dictionary_list

# many numbers comes from EOS tools in form of string. We need to convert them
# to numbers (integer or float)
# to be better handled by the CMS monitoring web page
# =====
# the reason why following key names do not contain the word "terabytes"
# instead of "bytes" is # because the frontend will break as it expects
# the key names that are currently present in this function
def convert_to_terabytes_and_cast_from_string_to_number__ec(dictionary_list):
    for item in dictionary_list:
        item['max_logical_quota'] = float(item['max_logical_quota']) / (10**12)
        item['max_physical_quota'] = item['max_logical_quota'] * 12 / 10
        item['total_used_logical_bytes'] = float(item['total_used_logical_bytes']) / (10**12)
        item['logical_rep_bytes'] = float(item['logical_rep_bytes']) / (10**12)
        item['logical_ec_bytes'] = float(item['logical_ec_bytes']) / (10**12)
        item['total_used_physical_bytes'] = float(item['total_used_physical_bytes']) / (10**12)
        item['physical_rep_bytes'] = float(item['physical_rep_bytes']) / (10**12)
        item['physical_ec_bytes'] = float(item['physical_ec_bytes']) / (10**12)
        item['free_physical'] = float(item['free_physical']) / (10**12)
        item['free_physical_for_ec'] = float(item['free_physical_for_ec']) / (10**12)
        item['free_physical_for_rep'] = float(item['free_physical_for_rep']) / (10**12)
        item['free_logical'] = float(item['free_logical']) / (10**12)
        try:
            item['used_logical_over_used_raw_percentage'] = item['total_used_logical_bytes'] / item['total_used_physical_bytes'] * 100
        except:
            item['used_logical_over_used_raw_percentage'] = None
        item['total_used_logical_terabytes'] = item.pop('total_used_logical_bytes')
        item['logical_rep_terabytes'] = item.pop('logical_rep_bytes')
        item['logical_ec_terabytes'] = item.pop('logical_ec_bytes')
        item['total_used_physical_terabytes'] = item.pop('total_used_physical_bytes')
        item['physical_rep_terabytes'] = item.pop('physical_rep_bytes')
        item['physical_ec_terabytes'] = item.pop('physical_ec_bytes')
    return dictionary_list


def get_eos_ec_quota_dump():
    try:
        accounting_file = open("/eos/cms/store/accounting/cms_quota_dump.txt", "r")

    except:
        logging.exception('Cannot get the eos quota ls output from EOS')

    with open(accounting_file.name) as file:
        lines = file.readlines()
        dictionary_list = []
        for line in lines:
            line = line.strip()
            line = line.split(' ')
            keys_values_single_line = dict(s.split('=') for s in line)
            dictionary_list.append(keys_values_single_line)
    accounting_file.close()
    return dictionary_list


def get_eos_quota_ls_output():
    dictionary_list_temp = []
    dictionary_list = []
    accounting_file = tempfile.NamedTemporaryFile()
    try:
        # export EOSHOME="" is needed to avoid getting the following two messages everytime the command is run
        # =====
        # pre-configuring default route to /eos/user/c/cmsvoc/
        # -use $EOSHOME variable to override
        # =====
        os.system('export EOSHOME="" && eos -r 103074 1399 quota ls -m > %s' % accounting_file.name)

    except:
        logging.exception('Cannot get the eos quota ls output from EOS')

    with open(accounting_file.name) as file:
        lines = file.readlines()
        for line in lines:
            line = line.strip()
            line = line.split(' ')
            keys_values_single_line = dict(s.split('=') for s in line)
            dictionary_list_temp.append(keys_values_single_line)
    accounting_file.close()
    i = 0
    """
    each xrdcp entry, when in eos quota ls, it either has gid=all or gid=project, attribute not both of them
    Concerning this, Jaroslav Guenther said:
    "That is expected, it is a special quota type which does not
    allow any other quota node to be defined on the same path. Project
    quota books all volume/inode usage under the project subtree to a single
    project account (gid 99). E.g. the recycle bin uses this quota type."
    """
    while i < len(dictionary_list_temp):
        if( ("gid" in dictionary_list_temp[i]) and ( (dictionary_list_temp[i]["gid"] == "ALL") or (dictionary_list_temp[i]["gid"] == "project") ) ):
            dictionary_list.append(dictionary_list_temp[i])
        i = i + 1

    # "eos quota ls" uses 'space' instead of 'path' as attribute name for folders. The following cycle changes this to 'path', so that later is
    # easier to write code to compare various outputs
    for x in dictionary_list:
        x['path'] = x['space']
        del x['space']
    return dictionary_list


def get_xrdcp_output():
    accounting_file = tempfile.NamedTemporaryFile()
    try:
        # export EOSHOME="" is needed to avoid getting the following two messages everytime the command is run
        # =====
        # pre-configuring default route to /eos/user/c/cmsvoc/
        # -use $EOSHOME variable to override
        # =====
        
        #os.system('export EOSHOME="" && xrdcp root://eoscms.cern.ch//eos/cms/proc/accounting - > %s' % accounting_file.name)
        os.system('XRD_CPUSEPGWRTRD=0 xrdcp --nopbar root://eoscms.cern.ch//eos/cms/proc/accounting - > %s' % accounting_file.name)

    except:
        logging.exception('Cannot get the xrdcp output from EOS')

    with open(accounting_file.name) as json_file:
        json_data_temp = json.load(json_file)
    accounting_file.close()
    data = json_data_temp['storageservice']['storageshares']
    """
    Due how EOS returns JSON output, each data element contains a 'path' key
    which value is in form of
    ["foo"] (so a list) instead of "foo" (so a string).
    This adds useless complexity, so must be removed.
    item['path'][0] returns "foo", instead item['path'] returns ["foo"]
    That's why [0] is used
    """
    for item in data:
        item['path'] = item['path'][0]
    return data


def match_xrdcp_and_eos_quota_ls_entries():
    xrdcp_data = get_xrdcp_output()
    eos_quota_ls_data = get_eos_quota_ls_output()
    eos_ec_quota = get_eos_ec_quota_dump()

    xrdcp_data_paths = [ i['path'] for i in xrdcp_data]
    eos_quota_ls_data_paths = [ j['path'] for j in eos_quota_ls_data]
    eos_ec_quota_paths = [ k['quota_node'] for k in eos_ec_quota]

    xrdcp_paths_set = set(xrdcp_data_paths)
    eos_quota_ls_paths_set = set(eos_quota_ls_data_paths)
    eos_ec_quota_paths_set = set(eos_ec_quota_paths)

    print(eos_quota_ls_paths_set - xrdcp_paths_set)
    print("eos_quota_ls_paths_set length is",len(eos_quota_ls_paths_set))
    print("xrdcp_paths_set length is",len(xrdcp_paths_set))
    print("eos_ec_quota_paths_set length is",len(eos_ec_quota_paths_set))
    print("xrdcp & eos_ec_quota_paths_set length is",len(xrdcp_paths_set & eos_ec_quota_paths_set))


def get_non_ec_statistics():
    xrdcp_data = get_xrdcp_output()
    eos_quota_ls_data = get_eos_quota_ls_output()
    eos_ec_quota = get_eos_ec_quota_dump()

    xrdcp_data_paths = [ i['path'] for i in xrdcp_data]
    eos_quota_ls_data_paths = [ j['path'] for j in eos_quota_ls_data]
    eos_ec_quota_paths = [ k['quota_node'] for k in eos_ec_quota]

    xrdcp_paths_set = set(xrdcp_data_paths)
    eos_quota_ls_paths_set = set(eos_quota_ls_data_paths)
    eos_ec_quota_paths_set = set(eos_ec_quota_paths)
    paths = xrdcp_paths_set - eos_ec_quota_paths_set

    results = list(filter(lambda x: x['path'] in paths, eos_quota_ls_data))
    return convert_to_terabytes_and_cast_from_string_to_number__non_ec(results)


def get_ec_statistics():
    results = get_eos_ec_quota_dump()
    return convert_to_terabytes_and_cast_from_string_to_number__ec(results)

# this function uses the nomenclature of non EC JSON
def produce_summary():
    ec_statistics = get_ec_statistics()
    non_ec_statistics = get_non_ec_statistics()
    total = {'path' : 'TOTAL', 'usedterabytes' : 0.0, 'usedlogicalterabytes' : 0.0, 'maxlogicalterabytes' : 0.0, 'maxphysicalterabytes' : 0.0, 'used_logical_over_used_raw_percentage' : 0.0, 'used_logical_space_percentage' : 0.0}

    for item in non_ec_statistics:
        if (any(item['path'].startswith(x) for x in EXCLUDED_PATHS)):
            del non_ec_statistics[non_ec_statistics.index(item)]
            continue
        del item['gid']
        del item['maxfiles']
        del item['quota']
        del item['usedfiles']
        del item['statusbytes']
        del item['statusfiles']
        item['used_logical_space_percentage'] = item.pop('percentageusedterabytes')
        item['maxphysicalterabytes'] = item.pop('maxterabytes')
        total['usedterabytes'] += item['usedterabytes']
        total['usedlogicalterabytes'] += item['usedlogicalterabytes']
        total['maxlogicalterabytes'] += item['maxlogicalterabytes']
        total['maxphysicalterabytes'] += item['maxphysicalterabytes']
    for item in ec_statistics:
        item['path'] = item.pop('quota_node')
        if (any(item['path'].startswith(x) for x in EXCLUDED_PATHS)):
            del ec_statistics[ec_statistics.index(item)]
            continue
        item['maxlogicalterabytes'] = item.pop('max_logical_quota')
        item['maxphysicalterabytes'] = item.pop('max_physical_quota')
        item['usedlogicalterabytes'] = item.pop('total_used_logical_terabytes')
        del item['logical_rep_terabytes']
        del item['logical_ec_terabytes']
        item['usedterabytes'] = item.pop('total_used_physical_terabytes')
        del item['physical_rep_terabytes']
        del item['physical_ec_terabytes']
        del item['free_physical']
        del item['free_physical_for_ec']
        del item['free_physical_for_rep']
        del item['free_logical']
        try:
            item['used_logical_space_percentage'] = item['usedlogicalterabytes'] * 100 / item['maxlogicalterabytes']
        except:
            item['used_logical_space_percentage'] = None
        total['usedterabytes'] += item['usedterabytes']
        total['usedlogicalterabytes'] += item['usedlogicalterabytes']
        total['maxlogicalterabytes'] += item['maxlogicalterabytes']
        total['maxphysicalterabytes'] += item['maxphysicalterabytes']
    try:
        total['used_logical_over_used_raw_percentage'] = total['usedlogicalterabytes'] / total['usedterabytes'] * 100
    except:
        total['used_logical_over_used_raw_percentage'] = None
    try:
        total['used_logical_space_percentage'] = total['usedlogicalterabytes'] * 100 / total['maxlogicalterabytes']
    except:
        total['used_logical_space_percentage'] = None

    for item in non_ec_statistics:
        if (any(item['path'].startswith(x) for x in EXCLUDED_PATHS)):
            del non_ec_statistics[non_ec_statistics.index(item)]
    for item in non_ec_statistics:
        if (any(item['path'].startswith(x) for x in EXCLUDED_PATHS)):
            del non_ec_statistics[non_ec_statistics.index(item)]
    for item in non_ec_statistics:
        if (any(item['path'].startswith(x) for x in EXCLUDED_PATHS)):
            del non_ec_statistics[non_ec_statistics.index(item)]
    for item in non_ec_statistics:
        if (any(item['path'].startswith(x) for x in EXCLUDED_PATHS)):
            del non_ec_statistics[non_ec_statistics.index(item)]
    # the [] brackets around total are needed to convert it into a list,
    # otherwise the addition operator will not work among a list and a dictionary
    return non_ec_statistics + ec_statistics + [total]

with open('/eos/cms/store/accounting/eos_ec_accounting.json', 'w') as json_ec_output:
    json.dump(get_ec_statistics(), json_ec_output, indent=4)

with open('/eos/cms/store/accounting/eos_non_ec_accounting.json', 'w') as json_non_ec_output:
    json.dump(get_non_ec_statistics(), json_non_ec_output, indent=4)

with open('/eos/cms/store/accounting/eos_accounting_summary.json', 'w') as json_summary_output:
    json.dump(produce_summary(), json_summary_output, indent=4)
