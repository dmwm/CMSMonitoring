#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
File        : datasets.py
Author      : Ceyhun Uzunoglu <ceyhunuzngl AT gmail [DOT] com>
Description : This Spark job creates datasets summary results by aggregating Rucio&DBS tables and
                save result to HDFS directory as a source to MongoDB of go web service
"""

# system modules
from datetime import datetime

import click
import pandas as pd
from pyspark import SparkContext
from pyspark.sql import SparkSession
from pyspark.sql.functions import (
    col, collect_list, concat_ws, countDistinct, first, greatest, lit, when, broadcast, lower,
    avg as _avg,
    count as _count,
    hex as _hex,
    max as _max,
    min as _min,
    split as _split,
    sum as _sum,
)
from pyspark.sql.types import LongType

# Local
import dbs_schemas

# global variables
TODAY = datetime.today().strftime('%Y-%m-%d')
# Rucio
HDFS_RUCIO_RSES = f'/project/awg/cms/rucio/{TODAY}/rses/part*.avro'
HDFS_RUCIO_REPLICAS = f'/project/awg/cms/rucio/{TODAY}/replicas/part*.avro'
HDFS_RUCIO_DIDS = f'/project/awg/cms/rucio/{TODAY}/dids/part*.avro'
# DBS
HDFS_DBS_DATASETS = f'/project/awg/cms/dbs/PROD_GLOBAL/{TODAY}/DATASETS/*.gz'
HDFS_DBS_FILES = f'/project/awg/cms/dbs/PROD_GLOBAL/{TODAY}/FILES/*.gz'
HDFS_SUB_DIR = "main"

pd.options.display.float_format = '{:,.2f}'.format
pd.set_option('display.max_colwidth', None)


def get_spark_session(app_name):
    """Get or create the spark context and session.
    """
    sc = SparkContext(appName=app_name)
    return SparkSession.builder.config(conf=sc._conf).getOrCreate()


def get_csvreader(spark):
    """CSV reader for DBS csv gz format"""
    return (
        spark.read.format("csv").option("nullValue", "null").option("mode", "FAILFAST")
    )


def get_df_rses(spark):
    """Get pandas dataframe of RSES
    """
    df_rses = spark.read.format("avro").load(HDFS_RUCIO_RSES) \
        .filter(col('DELETED_AT').isNull()) \
        .withColumn('rse_id', lower(_hex(col('ID')))) \
        .withColumn('rse_tier', _split(col('RSE'), '_').getItem(0)) \
        .withColumn('rse_country', _split(col('RSE'), '_').getItem(1)) \
        .withColumn('rse_kind',
                    when((col("rse").endswith('Temp') | col("rse").endswith('temp') | col("rse").endswith('TEMP')),
                         'temp')
                    .when((col("rse").endswith('Test') | col("rse").endswith('test') | col("rse").endswith('TEST')),
                          'test')
                    .otherwise('prod')
                    ) \
        .select(['rse_id', 'RSE', 'RSE_TYPE', 'rse_tier', 'rse_country', 'rse_kind'])
    return broadcast(df_rses)


def get_df_replicas(spark):
    """Create main replicas dataframe by selecting only Disk or Tape RSEs in Rucio REPLICAS table
    """
    return spark.read.format('avro').load(HDFS_RUCIO_REPLICAS) \
        .filter(col('SCOPE') == 'cms') \
        .filter(col('STATE') == 'A') \
        .withColumn('rse_id', lower(_hex(col('RSE_ID')))) \
        .withColumn('f_size_replicas', col('BYTES').cast(LongType())) \
        .withColumnRenamed('NAME', 'f_name') \
        .withColumnRenamed('ACCESSED_AT', 'rep_accessed_at') \
        .withColumnRenamed('CREATED_AT', 'rep_created_at') \
        .select(['f_name', 'rse_id', 'f_size_replicas', 'rep_accessed_at', 'rep_created_at'])


def get_df_dids_files(spark):
    """Create spark dataframe for DIDS table by selecting only Files in Rucio DIDS table.
    """
    return spark.read.format('avro').load(HDFS_RUCIO_DIDS) \
        .filter(col('DELETED_AT').isNull()) \
        .filter(col('HIDDEN') == '0') \
        .filter(col('SCOPE') == 'cms') \
        .filter(col('DID_TYPE') == 'F') \
        .withColumnRenamed('NAME', 'f_name') \
        .withColumnRenamed('ACCESSED_AT', 'dids_accessed_at') \
        .withColumnRenamed('CREATED_AT', 'dids_created_at') \
        .withColumn('f_size_dids', col('BYTES').cast(LongType())) \
        .select(['f_name', 'f_size_dids', 'dids_accessed_at', 'dids_created_at'])


def get_df_dbs_f_d(spark):
    """Create a dataframe for FILE-DATASET membership/ownership map
    """
    dbs_files = get_csvreader(spark).schema(dbs_schemas.schema_files()).load(HDFS_DBS_FILES) \
        .withColumnRenamed('LOGICAL_FILE_NAME', 'f_name') \
        .select(['f_name', 'DATASET_ID'])
    dbs_datasets = get_csvreader(spark).schema(dbs_schemas.schema_datasets()).load(HDFS_DBS_DATASETS) \
        .filter(col('IS_DATASET_VALID') == '1') \
        .select(['DATASET_ID', 'DATASET'])
    df_dbs_f_d = dbs_files.join(dbs_datasets, ["DATASET_ID"], how='left') \
        .withColumnRenamed('DATASET_ID', 'dataset_id') \
        .withColumnRenamed('DATASET', 'dataset') \
        .select(['dataset_id', 'f_name', 'dataset'])
    return df_dbs_f_d


def get_df_ds_general_info(spark):
    """Calculate real size and total file counts of dataset: RealSize, TotalFileCnt
    """
    dbs_files = get_csvreader(spark).schema(dbs_schemas.schema_files()).load(HDFS_DBS_FILES) \
        .select(['DATASET_ID', 'FILE_SIZE', 'LOGICAL_FILE_NAME'])
    dbs_datasets = get_csvreader(spark).schema(dbs_schemas.schema_datasets()).load(HDFS_DBS_DATASETS) \
        .select(['DATASET_ID'])
    return dbs_datasets.join(dbs_files, ['DATASET_ID'], how='left') \
        .groupby('DATASET_ID') \
        .agg(_sum('FILE_SIZE').alias('RealSize'),
             countDistinct(col('LOGICAL_FILE_NAME')).alias('TotalFileCnt')
             ) \
        .withColumnRenamed('DATASET_ID', 'Id').select(['Id', 'RealSize', 'TotalFileCnt'])


def get_df_replicas_j_dids(df_replicas, df_dids_files):
    """Left join of df_replicas and df_dids_files to fill the RSE_ID, f_size and accessed_at, created_at for all files.

    Be aware that there are 2 columns for each f_size, accessed_at, created_at
    They will be combined in get_df_file_rse_ts_size

    Columns:
        comes from DID:       file, dids_accessed_at, dids_created_at, f_size_dids,
        comes from REPLICAS:  file, rse_id, f_size_replicas, rep_accessed_at, rep_created_at
   """
    return df_replicas.join(df_dids_files, ['f_name'], how='left')


def get_df_file_rse_ts_size(df_replicas_j_dids):
    """Combines columns to get filled and correct values from join of DIDS and REPLICAS

    Firstly, REPLICAS size value will be used. If there are files with no size values, DIDS size values will be used:
    see 'when' function order. For accessed_at and created_at, their max values will be got.

    df_file_rse_ts_size: files and their rse_id, size and access time are completed
    """
    return df_replicas_j_dids \
        .withColumn('f_size',
                    when(col('f_size_replicas').isNotNull(), col('f_size_replicas'))
                    .when(col('f_size_dids').isNotNull(), col('f_size_dids'))
                    ) \
        .withColumn('accessed_at',
                    greatest(col('dids_accessed_at'), col('rep_accessed_at'))
                    ) \
        .withColumn('created_at',
                    greatest(col('dids_created_at'), col('rep_created_at'))
                    ) \
        .select(['f_name', 'rse_id', 'accessed_at', 'f_size', 'created_at'])


def get_df_dataset_file_rse_ts_size(df_file_rse_ts_size, df_dbs_f_d):
    """ Left join df_file_rse_ts_size and df_dbs_f_d to get dataset names of files.

    In short: adds 'dataset' names to 'df_file_rse_ts_size' dataframe by joining DBS tables

    Columns: block(from df_contents_f_to_b), file, rse_id, accessed_at, f_size
    """
    df_dataset_file_rse_ts_size = df_file_rse_ts_size \
        .join(df_dbs_f_d, ['f_name'], how='left') \
        .fillna("UnknownDatasetNameOfFiles_MonitoringTag", subset=['dataset']) \
        .select(['dataset_id', 'dataset', 'f_name', 'rse_id', 'accessed_at', 'created_at', 'f_size'])

    return df_dataset_file_rse_ts_size


def get_df_enr_with_rse_info(df_dataset_file_rse_ts_size, df_rses):
    """Add RSE type, name, kind, tier, country by joining RSE ID"""
    return df_dataset_file_rse_ts_size.join(df_rses, ['rse_id'], how='left') \
        .select(['dataset_id', 'dataset', 'f_name', 'rse_id', 'accessed_at', 'created_at', 'f_size',
                 'RSE', 'RSE_TYPE', 'rse_tier', 'rse_country', 'rse_kind'])


# --------------------------------------------------------------------------------
# Main dataset functions
# --------------------------------------------------------------------------------

def get_df_sub_rse_details(df_enr_with_rse_info):
    """Get dataframe of datasets that are not read since N months for sub details htmls

    Group by 'dataset' and 'rse_id' of get_df_dataset_file_rse_ts_size

    Filters:
        - If a dataset contains EVEN a single file with null accessed_at, filter out

    Access time filter logic:
        - If 'last_access_time_of_dataset_in_all_rses' is less than 'n_months_filter', ...
          ... set 'is_not_read_since_{n_months_filter}_months' column as True

    Columns:
        - 'dataset_size_in_rse_gb'
                Total size of a Dataset in an RSE.
                Produced by summing up datasets' all files in that RSE.
        - 'last_access_time_of_dataset_in_rse'
                Last access time of a Dataset in an RSE.
                Produced by getting max `accessed_at`(represents single file's access time) of a dataset in an RSE.
        - '#files_with_null_access_time_of_dataset_in_rse'
                Number of files count, which have NULL `accessed_at` values, of a Dataset in an RSE.
                This is important to know to filter out if there is any NULL `accessed_at` value of a Dataset.
        - '#files_of_dataset_in_rse'
                Number of files count of a Dataset in an RSE
        - '#distinct_files_of_dataset_in_rse'
                Number of unique files count of dataset in an RSE

    df_main_datasets_and_rses: RSE name, dataset and their size and access time calculations
    """
    return df_enr_with_rse_info \
        .groupby(['rse_id', 'dataset']) \
        .agg(_sum(col('f_size')).alias('SizeInRseBytes'),
             _max(col('accessed_at')).alias('LastAccessInRse'),
             _count(lit(1)).alias('FileCnt'),
             _sum(when(col('accessed_at').isNull(), 0).otherwise(1)).alias('AccessedFileCnt'),
             first(col('dataset_id')).alias('dataset_id'),
             first(col('RSE_TYPE')).alias('RseType'),
             first(col('RSE')).alias('RSE'),
             first(col('rse_tier')).alias('rse_tier'),
             first(col('rse_country')).alias('rse_country'),
             first(col('rse_kind')).alias('rse_kind'),
             ) \
        .withColumnRenamed('dataset', 'Dataset') \
        .select(['dataset_id', 'RseType', 'RSE', 'Dataset', 'SizeInRseBytes',
                 'LastAccessInRse', 'FileCnt', 'AccessedFileCnt', ])


def get_df_main_datasets(df_sub_rse_details, df_ds_general_info):
    """Get dataframe of datasets not read since N months for main htmls.

    Get last access of dataframe in all RSE(s)
    """
    df = df_sub_rse_details \
        .groupby(['RseType', 'Dataset']) \
        .agg(_max(col('SizeInRseBytes')).cast(LongType()).alias('Max'),
             _min(col('SizeInRseBytes')).cast(LongType()).alias('Min'),
             _avg(col('SizeInRseBytes')).cast(LongType()).alias('Avg'),
             _sum(col('SizeInRseBytes')).cast(LongType()).alias('Sum'),
             _max(col('LastAccessInRse')).alias('LastAccessMs'),
             concat_ws(', ', collect_list('RSE')).alias('RSEs'),
             first(col('dataset_id')).cast(LongType()).alias('Id'),
             ) \
        .withColumn('LastAccess', (col('LastAccessMs') / 1000).cast(LongType())) \
        .select(['Id', 'RseType', 'Dataset', 'LastAccess', 'Max', 'Min', 'Avg', 'Sum', 'RSEs'])

    return df.join(df_ds_general_info, ['Id'], how='left') \
        .select(['Id', 'RseType', 'Dataset', 'LastAccess', 'Max', 'Min', 'Avg', 'Sum', 'RealSize',
                 'TotalFileCnt', 'RSEs'])


@click.command()
@click.option('--hdfs_out_dir', default=None, type=str, required=True,
              help='I.e. /tmp/${KERBEROS_USER}/rucio_ds_mongo/$(date +%Y-%m-%d) ')
def main(hdfs_out_dir):
    """Main function that run Spark dataframe creations and save results to HDFS directory as JSON lines
    """
    hdfs_out_dir = hdfs_out_dir + "/" + HDFS_SUB_DIR
    # HDFS output file format. If you change, please modify bin/cron4rucio_ds_mongo.sh accordingly.
    write_format = 'json'
    write_mode = 'overwrite'

    spark = get_spark_session(app_name='cms-monitoring-rucio-datasets')
    # Set TZ as UTC. Also set in the spark-submit confs.
    spark.conf.set("spark.sql.session.timeZone", "UTC")

    df_rses = get_df_rses(spark)
    df_dbs_f_d = get_df_dbs_f_d(spark)
    df_ds_general_info = get_df_ds_general_info(spark)
    df_replicas = get_df_replicas(spark)
    df_dids_files = get_df_dids_files(spark)
    df_replicas_j_dids = get_df_replicas_j_dids(df_replicas, df_dids_files)
    df_file_rse_ts_size = get_df_file_rse_ts_size(df_replicas_j_dids)
    df_dataset_file_rse_ts_size = get_df_dataset_file_rse_ts_size(df_file_rse_ts_size, df_dbs_f_d)
    df_enr_with_rse_info = get_df_enr_with_rse_info(df_dataset_file_rse_ts_size, df_rses)
    df_sub_rse_details = get_df_sub_rse_details(df_enr_with_rse_info)
    df_main_datasets = get_df_main_datasets(df_sub_rse_details, df_ds_general_info)
    df_main_datasets.write.save(path=hdfs_out_dir, format=write_format, mode=write_mode)


if __name__ == '__main__':
    main()
