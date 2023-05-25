#!/usr/bin/env python3
# -*- coding: utf-8 -*-
import os
import sys
import json
import time
import shutil
import tarfile
import requests
import subprocess
import click

baseUrl = 'https://monit-grafana.cern.ch/api'

def updatePathEnding(path):
    if not path.endswith("/"):
        path += "/"
    return path
def getGrafanaAuth(fname):
    if not os.path.exists(fname):
        print("File {} does not exist".format(fname))
        sys.exit(1)
    with open(fname, 'r+') as keyFile:
        SECRET_KEY = json.load(keyFile).get('SECRET_KEY')
    headers = {"Authorization": "Bearer %s" % SECRET_KEY}
    return headers


def createBackUpFiles(dashboard, folderTitle, headers):
    allJson = []

    dashboardUid = dashboard.get('uid')
    path = "grafana/dashboards/%s" % folderTitle

    if not os.path.exists(path):
        try:
            os.makedirs(os.path.join('grafana', 'dashboards', folderTitle))
            print("Successfully created the directory %s " % path)
        except OSError:
            print("Creation of the directory %s failed" % path)
    dashboardUrl = '%s/dashboards/uid/%s' % (baseUrl, dashboardUid)
    print('dashboardUrl', dashboardUrl)
    req = requests.get(dashboardUrl, headers=headers)
    dashboardData = req.json().get('dashboard')

    # Replace characters that might cause harm
    titleOfDashboard = dashboardData.get('title').replace(" ", "").replace(".", "_").replace("/", "_")
    uidOfDashboard = dashboardData.get('uid')

    filenameForFolder = '%s/%s-%s.json' % (path, titleOfDashboard, uidOfDashboard)
    print('filenameForFolder', filenameForFolder)
    with open(filenameForFolder, "w+") as jsonFile:
        json.dump(dashboardData, jsonFile)
    print('*' * 10 + '\n')
    allJson.append(dashboardData)

    filenameForAllJson = '%s/all.json' % path
    print('filenameForAllJson', filenameForAllJson)
    with open(filenameForAllJson, "w+") as jsonFile:
        json.dump(allJson, jsonFile)


def searchFoldersFromGrafana(headers):
    foldersUrl = 'https://monit-grafana.cern.ch/api/search?folderIds=0&orgId=11'

    req = requests.get(foldersUrl, headers=headers)
    if req.status_code != 200:
        print("Request {}, failed with reason: {}".format(foldersUrl, req.reason))
        sys.exit(1)
    foldersJson = req.json()

    for folder in foldersJson:
        folderId = folder.get('id')
        folderTitle = folder.get('title')

        if folderTitle not in ['Production', 'Development', 'Playground', 'Backup']:
            folderTitle = 'General'
            dashboard = folder
            createBackUpFiles(dashboard, folderTitle, headers)

        else:
            dashboardQuery = 'search?folderIds=%s&orgId=11&query=' % folderId
            dashboardQueryUrl = '%s/%s' % (baseUrl, dashboardQuery)

            print('individualFolderUrl', dashboardQueryUrl)

            req = requests.get(dashboardQueryUrl, headers=headers)
            folderData = req.json()

            pathToDb = os.path.abspath("grafana")
            if not os.path.exists(pathToDb):
                os.makedirs(pathToDb)

            filenameForFolder = 'grafana/%s-%s.json' % (folderId, folderTitle)
            with open(filenameForFolder, "w+") as jsonFile:
                json.dump(folderData, jsonFile)

            dashboardsAmount = len(folderData)
            for index, dashboard in enumerate(folderData):
                folderTitle = dashboard.get('folderTitle')
                print('Index/Amount: %s/%s' % (index, dashboardsAmount))

                createBackUpFiles(dashboard, folderTitle, headers)


def createTar(path, tar_name):
    with tarfile.open(tar_name, "w:gz") as tar_handle:
        for root, _, files in os.walk(path):
            for file in files:
                tar_handle.add(os.path.join(root, file))


def removeTempFiles(path):
    for root, dirs, files in os.walk(path):
        for f in files:
            os.unlink(os.path.join(root, f))
        for d in dirs:
            shutil.rmtree(os.path.join(root, d))
    if os.path.exists(path):
        os.rmdir(path)


def get_date():
    gmt = time.gmtime()
    year = str(gmt.tm_year)
    mon = gmt.tm_mon
    if len(str(mon)) == 1:
        mon = '0{}'.format(mon)
    else:
        mon = str(mon)
    day = gmt.tm_mday
    if len(str(day)) == 1:
        day = '0{}'.format(day)
    else:
        day = str(day)
    return day, mon, year


def copyToHDFS(archive, hdfs_path):
    day, mon, year = get_date()
    hdfs_path = updatePathEnding(hdfs_path)
    path = f'{hdfs_path}{year}/{mon}/{day}'
    print("Copy backup to {}".format(path))
    cmd = 'hadoop fs -mkdir -p {}'.format(path)
    output = subprocess.check_output(cmd, stderr=subprocess.STDOUT, shell=True)
    if output:
        print(output)
    cmd = 'hadoop fs -put {} {}'.format(archive, path)
    print('Execute: {}'.format(cmd))
    output = subprocess.check_output(cmd, stderr=subprocess.STDOUT, shell=True)
    if output:
        print(output)
    cmd = 'hadoop fs -ls {}'.format(path)
    output = subprocess.check_output(cmd, stderr=subprocess.STDOUT, shell=True)
    print("New backup: {}".format(output))


def copyToFileSystem(archive, base_dir):
    day, mon, year = get_date()
    base_dir = updatePathEnding(base_dir)
    path = f'{base_dir}{year}/{mon}/{day}'
    print("Copy backup to {}".format(path))
    cmd = 'mkdir -p {}'.format(path)
    output = subprocess.check_output(cmd, stderr=subprocess.STDOUT, shell=True)
    if output:
        print(output)
    cmd = 'cp {} {}'.format(archive, path)
    print('Execute: {}'.format(cmd))
    output = subprocess.check_output(cmd, stderr=subprocess.STDOUT, shell=True)
    if output:
        print(output)
    cmd = 'ls {}'.format(path)
    output = subprocess.check_output(cmd, stderr=subprocess.STDOUT, shell=True)
    print("New backup: {}".format(output))


@click.command()
@click.option("--token", "fname", required=True, help="Grafana token location JSON file")
@click.option("--hdfs-path", required=True, help="Path for Grafana backup in HDFS")
@click.option("--filesystem-path", required=True, help="Path for Grafana backup in EOS")
def main(fname, hdfs_path, filesystem_path):
    click.echo(f"Input Arguments: token:{fname}, hdfs_path:{hdfs_path}, filesystem_path:{filesystem_path}")
    headers = getGrafanaAuth(fname)
    searchFoldersFromGrafana(headers)
    createTar('./grafana', 'grafanaBackup.tar.gz')
    copyToHDFS('grafanaBackup.tar.gz', hdfs_path)
    copyToFileSystem('grafanaBackup.tar.gz', filesystem_path)
    removeTempFiles('./grafana/')


if __name__ == '__main__':
    main()
