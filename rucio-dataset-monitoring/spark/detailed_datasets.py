#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
File        : detailed_datasets.py
Author      : Ceyhun Uzunoglu <ceyhunuzngl AT gmail [DOT] com>
Description : This Spark job creates detailed datasets(in each RSEs) results by aggregating Rucio&DBS tables and
                save result to HDFS directory as a source to MongoDB of go web service
"""

# system modules
from datetime import datetime

import click
import pandas as pd
from pyspark import SparkContext
from pyspark.sql import SparkSession
from pyspark.sql.functions import (
    coalesce, col, collect_set, countDistinct, first, greatest, lit, lower, sort_array, when,
    hex as _hex,
    max as _max,
    size as _size,
    split as _split,
    sum as _sum,
)
from pyspark.sql.types import IntegerType, LongType, DecimalType

# Local
import dbs_schemas

pd.options.display.float_format = '{:,.2f}'.format
pd.set_option('display.max_colwidth', None)

# global variables
TODAY = datetime.today().strftime('%Y-%m-%d')
# Rucio
HDFS_RUCIO_RSES = f'/project/awg/cms/rucio/{TODAY}/rses/part*.avro'
HDFS_RUCIO_REPLICAS = f'/project/awg/cms/rucio/{TODAY}/replicas/part*.avro'
HDFS_RUCIO_DIDS = f'/project/awg/cms/rucio/{TODAY}/dids/part*.avro'
HDFS_RUCIO_DLOCKS = f'/project/awg/cms/rucio/{TODAY}/dataset_locks/part*.avro'
# DBS
HDFS_DBS_DATASETS = f'/project/awg/cms/dbs/PROD_GLOBAL/{TODAY}/DATASETS/*.gz'
HDFS_DBS_BLOCKS = f'/project/awg/cms/dbs/PROD_GLOBAL/{TODAY}/BLOCKS/*.gz'
HDFS_DBS_FILES = f'/project/awg/cms/dbs/PROD_GLOBAL/{TODAY}/FILES/*.gz'
PROD_ACCOUNTS = ['transfer_ops', 'wma_prod', 'wmcore_output', 'wmcore_transferor', 'crab_tape_recall', 'sync']
SYNC_PREFIX = 'sync'

HDFS_SUB_DIR_DETAILED = "detailed"
HDFS_SUB_DIR_IN_BOTH = "in_tape_and_disk"


def get_spark_session(app_name):
    """Get or create the spark context and session."""
    sc = SparkContext(appName=app_name)
    return SparkSession.builder.config(conf=sc._conf).getOrCreate()


# --------------------------------------------------------------------------------------------------------------- DBS
def get_csvreader(spark):
    """CSV reader for DBS csv gz format"""
    return spark.read.format("csv").option("nullValue", "null").option("mode", "FAILFAST")


def get_df_dbs_files(spark):
    """Create DBS Files table dataframe
    """
    # Get DBS-Files required fields
    return get_csvreader(spark).schema(dbs_schemas.schema_files()).load(HDFS_DBS_FILES) \
        .filter(col('IS_FILE_VALID') == '1') \
        .withColumnRenamed('LOGICAL_FILE_NAME', 'FILE_NAME') \
        .select(['FILE_NAME', 'DATASET_ID', 'BLOCK_ID', 'FILE_SIZE'])


def get_df_dbs_blocks(spark):
    """Create DBS Blocks table dataframe
    """
    # Get DBS-Blocks required fields
    return get_csvreader(spark).schema(dbs_schemas.schema_blocks()).load(HDFS_DBS_BLOCKS) \
        .select(['BLOCK_NAME', 'BLOCK_ID', 'DATASET_ID', 'FILE_COUNT'])


def get_df_dbs_datasets(spark):
    """Create DBS Datasets table dataframe
    """
    # Get DBS-Datasets required fields
    return get_csvreader(spark).schema(dbs_schemas.schema_datasets()).load(HDFS_DBS_DATASETS) \
        .filter(col('IS_DATASET_VALID') == '1') \
        .select(['DATASET_ID', 'DATASET'])


def get_df_dbs_f_d_map(spark):
    """Create dataframe for DBS dataset:file map
    """
    # Get DBS Datasets-Files map with required fields
    dbs_files = get_df_dbs_files(spark).withColumnRenamed('DATASET_ID', 'F_DATASET_ID')
    dbs_datasets = get_df_dbs_datasets(spark)
    return dbs_files.join(dbs_datasets, dbs_files.F_DATASET_ID == dbs_datasets.DATASET_ID, how='inner') \
        .select(['DATASET_ID', 'DATASET', 'FILE_NAME'])


def get_df_dbs_b_d_map(spark):
    """Create dataframe for DBS dataset:block map
    """
    # Get DBS Datasets-Blocks map with required fields
    dbs_blocks = get_df_dbs_blocks(spark).withColumnRenamed('DATASET_ID', 'B_DATASET_ID')
    dbs_datasets = get_df_dbs_datasets(spark)
    return dbs_blocks.join(dbs_datasets, dbs_blocks.B_DATASET_ID == dbs_datasets.DATASET_ID, how='inner') \
        .select(['DATASET_ID', 'DATASET', 'BLOCK_NAME'])


def get_df_ds_file_and_block_cnt(spark):
    """Calculate total file and block count of a dataset
    """
    # DBS Dataset file count. It will be used as a reference to define if dataset is fully replicated in RSE.
    _df1 = get_df_dbs_f_d_map(spark) \
        .groupby(['DATASET']) \
        .agg(countDistinct(col('FILE_NAME')).alias('TOT_FILE_CNT')) \
        .select(['DATASET', 'TOT_FILE_CNT'])
    # DBS Dataset block count.
    _df2 = get_df_dbs_b_d_map(spark) \
        .groupby(['DATASET']) \
        .agg(countDistinct(col('BLOCK_NAME')).alias('TOT_BLOCK_CNT')) \
        .withColumnRenamed('DATASET', 'B_DATASET') \
        .select(['B_DATASET', 'TOT_BLOCK_CNT'])

    return _df1.join(_df2, _df1.DATASET == _df2.B_DATASET, how='inner') \
        .select(['DATASET', 'TOT_BLOCK_CNT', 'TOT_FILE_CNT'])


# ---------------------------------------------------------------------------------------------------------------------


def get_df_rses(spark):
    """Create rucio RSES table dataframe with some rse tag calculations
    """
    # Rucio Rses required fields
    return spark.read.format("avro").load(HDFS_RUCIO_RSES) \
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


def get_df_replicas(spark):
    """Create rucio Replicas table dataframe
    """
    # Rucio Replicas required fields
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
    """Create rucio DIDS table dataframe with just for files
    """
    # Rucio Dids only files required fields
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


def get_df_dids_datasets(spark):
    """Create rucio DIDS table dataframe with just for datasets
    """
    # Rucio Dids only datasets required fields
    return spark.read.format('avro').load(HDFS_RUCIO_DIDS) \
        .filter(col('DELETED_AT').isNull()) \
        .filter(col('HIDDEN') == '0') \
        .filter(col('SCOPE') == 'cms') \
        .filter(col('DID_TYPE') == 'C') \
        .withColumnRenamed('NAME', 'f_name') \
        .withColumnRenamed('ACCESSED_AT', 'dids_accessed_at') \
        .withColumnRenamed('CREATED_AT', 'dids_created_at') \
        .withColumn('f_size_dids', col('BYTES').cast(LongType())) \
        .select(['f_name', 'f_size_dids', 'dids_accessed_at', 'dids_created_at'])


def get_df_dlocks(spark):
    """Create rucio DatasetLocks table dataframe

    - If account is sync, use sync prefix for all of them. Filter only production accounts.
    - DatasetLocks will be used to calculate number of locked blocks. If all blocks of dataset are locked, dataset  will
    be set as locked.

    - "sync-rules" represents all SYNC account rules. It can 1 or many. Because of their quantity, this solution is
    applied to provide better visualization.
    """
    # Get DatasetLocks required fields
    df_dlocks = spark.read.format('avro').load(HDFS_RUCIO_DLOCKS) \
        .filter(col('SCOPE') == 'cms') \
        .withColumn('rse_id', lower(_hex(col('RSE_ID')))) \
        .withColumn('rule_id', lower(_hex(col('RULE_ID')))) \
        .withColumn('account',
                    when(col('ACCOUNT').startswith(SYNC_PREFIX), lit(SYNC_PREFIX)).otherwise(col('ACCOUNT'))) \
        .filter(col('account').isin(PROD_ACCOUNTS)) \
        .withColumnRenamed('NAME', 'dlocks_block_name') \
        .select(['rse_id', 'dlocks_block_name', 'account', 'rule_id'])

    # Map locks(block) to datasets
    df_dbs_b_d_map = get_df_dbs_b_d_map(spark)
    df_dlocks = df_dlocks.join(df_dbs_b_d_map, df_dlocks.dlocks_block_name == df_dbs_b_d_map.BLOCK_NAME, how='left') \
        .select(['rse_id', 'dlocks_block_name', 'account', 'rule_id', 'DATASET_ID'])

    # Change SYNC rule_ids to "sync-rules", because there are so many of them
    df_dlocks = df_dlocks.withColumn("rule_id",
                                     when(col("account") == SYNC_PREFIX, "sync-rules").otherwise(col("rule_id")))

    # Group by DATASET and RSE to gather dataset lock accounts and rule counts
    df_dlocks = df_dlocks.groupby(['rse_id', 'DATASET_ID']) \
        .agg(sort_array(collect_set('account')).alias('ProdAccounts'),
             collect_set('rule_id').alias('BlockRuleIDs'),
             countDistinct(col('dlocks_block_name')).alias('ProdLockedBlockCount')
             ) \
        .select(['rse_id', 'DATASET_ID', 'ProdAccounts', 'BlockRuleIDs', 'ProdLockedBlockCount'])

    # Get RSE from its id
    df_rses = get_df_rses(spark)
    return df_dlocks.join(df_rses.select(['rse_id', 'RSE']), ['rse_id'], how='left') \
        .withColumnRenamed('RSE', 'dlocks_RSE') \
        .select(['dlocks_RSE', 'DATASET_ID', 'ProdAccounts', 'BlockRuleIDs', 'ProdLockedBlockCount'])


def get_df_files_enriched(spark):
    """Enriched files with REPLICAS and DIDS

    Add replica size, access time and creation of replica files of each RSE.
    """
    df_replicas = get_df_replicas(spark)
    df_dids_files = get_df_dids_files(spark)
    df_rses = get_df_rses(spark)
    df_rep_enr_dids = df_replicas.join(df_dids_files, ['f_name'], how='left') \
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

    return df_rep_enr_dids.join(df_rses.select(['rse_id', 'RSE']), ['rse_id'], how='left') \
        .select(['f_name', 'RSE', 'f_size', 'accessed_at', 'created_at'])


def get_df_datasets_files_phase1(spark):
    """Files with dataset name

    Map dataset-RSE and files with enriched values.
    """
    df_files__enriched = get_df_files_enriched(spark)
    df_dbs_f_d_map = get_df_dbs_f_d_map(spark)
    return df_files__enriched \
        .join(df_dbs_f_d_map, df_files__enriched.f_name == df_dbs_f_d_map.FILE_NAME, how='left') \
        .fillna("UnknownDatasetNameOfFiles_MonitoringTag", subset=['DATASET']) \
        .fillna("0", subset=['DATASET_ID']) \
        .withColumnRenamed('DATASET_ID', 'd_id') \
        .withColumnRenamed('DATASET', 'd_name') \
        .select(['d_id', 'd_name', 'RSE', 'f_name', 'f_size', 'accessed_at', 'created_at'])


def get_df_main_datasets_in_each_rse(spark):
    """Main
    """
    # Add last access and size of dataset for each RSE
    # ( Dataset, RSE ) + ( FileCount, AccessedFileCount, LastAccess, LastAccess, SizeBytes )
    df_datasets_files_phase1 = get_df_datasets_files_phase1(spark)
    df = df_datasets_files_phase1.groupby(['RSE', 'd_name']) \
        .agg(countDistinct(col('f_name')).alias('FileCount'),
             _max(col('accessed_at')).alias('LastAccessMs'),
             _sum(col('f_size')).alias('SizeBytes'),
             _sum(when(col('accessed_at').isNull(), 0).otherwise(1)).alias('AccessedFileCount'),
             first(col('d_id')).alias('d_id')
             ) \
        .withColumn('LastAccess', (col('LastAccessMs') / 1000).cast(LongType()))

    # Add file counts and block counts with additional tags like IsFullyReplicated, FilePercentage
    # + ( FilePercentage, IsFullyReplicated, TOT_BLOCK_CNT->BlockCount )
    df_ds_file_and_block_cnt = get_df_ds_file_and_block_cnt(spark)
    df = df.join(df_ds_file_and_block_cnt, df.d_name == df_ds_file_and_block_cnt.DATASET, how='left').drop('DATASET') \
        .withColumn('FilePercentage',
                    (100 * col('FileCount') / col('TOT_FILE_CNT')).cast(DecimalType(6, 2))) \
        .withColumn('IsFullyReplicated',
                    when(col('FileCount') == col('TOT_FILE_CNT'), lit(True)).otherwise(lit(False))) \
        .withColumn('Id', col('d_id').cast(LongType())) \
        .withColumnRenamed('d_name', 'Dataset') \
        .withColumnRenamed('TOT_BLOCK_CNT', 'BlockCount') \
        .select(['RSE', 'Dataset', 'Id', 'SizeBytes', 'LastAccess', 'FileCount', 'AccessedFileCount',
                 'IsFullyReplicated', 'FilePercentage', 'BlockCount'])

    # Add Block level locks tags
    df_dlocks = get_df_dlocks(spark)
    # if      ( BlockCount/TOT_BLOCK_CNT == ProdLockedBlockCount ): FULLY
    # else if ( ProdLockedBlockCount >=1                         ): PARTIAL
    # else                                                        : DYNAMIC
    df = df.join(df_dlocks, on=((df.Id == df_dlocks.DATASET_ID) & (df.RSE == df_dlocks.dlocks_RSE)), how='left') \
        .withColumn('IsLocked',
                    when(col('BlockCount') == col('ProdLockedBlockCount'), lit('FULLY'))
                    .when((col('ProdLockedBlockCount') >= 1), lit('PARTIAL'))
                    .otherwise(lit('DYNAMIC'))
                    ) \
        .withColumn('ProdLockedBlockCount', coalesce(col('ProdLockedBlockCount'), lit(0)).cast(IntegerType())) \
        .select(['RSE', 'Dataset', 'Id', 'SizeBytes', 'LastAccess', 'IsFullyReplicated', 'IsLocked',
                 'FilePercentage', 'FileCount', 'AccessedFileCount', 'BlockCount', 'ProdLockedBlockCount',
                 'ProdAccounts', 'BlockRuleIDs'])

    # Add RSE tags
    df_rses = get_df_rses(spark)
    return df.join(df_rses, ['RSE'], how='left') \
        .withColumnRenamed('RSE_TYPE', 'Type') \
        .withColumnRenamed('rse_tier', 'Tier') \
        .withColumnRenamed('rse_country', 'C') \
        .withColumnRenamed('rse_kind', 'RseKind') \
        .fillna(0, subset=['LastAccess']) \
        .select(['Type', 'Dataset', 'RSE', 'Tier', 'C', 'RseKind', 'SizeBytes', 'LastAccess',
                 'IsFullyReplicated', 'IsLocked', 'FilePercentage', 'FileCount', 'AccessedFileCount', 'BlockCount',
                 'ProdLockedBlockCount', 'ProdAccounts', 'BlockRuleIDs'])


def get_df_main_datasets_in_both_disk_and_tape(df_main_datasets_in_each_rse):
    """Produce datasets that are in both DISK and TAPE with additional fields
    """
    # Use only production RSEs with selected fields
    df = df_main_datasets_in_each_rse \
        .filter(col('RseKind') == 'prod') \
        .select(['Type', 'Dataset', 'RSE', 'RseKind', 'SizeBytes',
                 'IsFullyReplicated', 'IsLocked', 'FileCount', 'BlockCount'])
    # Find datasets in both TAPE and DISK
    df_datasets_in_both = df.select(['Dataset', 'Type']) \
        .groupby(['Dataset']) \
        .agg(collect_set(col('Type')).alias('TypeSet')) \
        .filter(_size(col("TypeSet")) > 1) \
        .select(['Dataset'])

    # Use only datasets are in both Tape and Disk; filter out others
    df = df_datasets_in_both.join(df, ['Dataset'], how='left')

    df = df.groupby(['Dataset']) \
        .agg(_max(col('SizeBytes')).alias('MaxSize'),
             # Tape
             sort_array(collect_set(when(col('Type') == 'TAPE', col("RSE")))).alias("TapeRseSet"),
             countDistinct(when(col('Type') == 'TAPE', col("RSE"))).alias("TapeRseCount"),
             _sum(when(col('Type') == 'TAPE', col("IsFullyReplicated").cast("long"))
                  ).alias("TapeFullyReplicatedRseCount"),
             _sum(when(col('Type') == 'TAPE', col('IsLocked').eqNullSafe('FULLY').cast('long'))
                  ).alias('TapeFullyLockedRseCount'),
             # Disk
             sort_array(collect_set(when(col('Type') == 'DISK', col("RSE")))).alias("DiskRseSet"),
             countDistinct(when(col('Type') == 'DISK', col("RSE"))).alias("DiskRseCount"),
             _sum(when(col('Type') == 'DISK', col("IsFullyReplicated").cast("long"))
                  ).alias("DiskFullyReplicatedRseCount"),
             _sum(when(col('Type') == 'DISK', col('IsLocked').eqNullSafe('FULLY').cast('long'))
                  ).alias('DiskFullyLockedRseCount'),
             ) \
        .select(['Dataset', 'MaxSize',
                 'TapeFullyReplicatedRseCount', 'DiskFullyReplicatedRseCount',
                 'TapeFullyLockedRseCount', 'DiskFullyLockedRseCount',
                 'TapeRseCount', 'DiskRseCount',
                 'TapeRseSet', 'DiskRseSet'])
    return df


@click.command()
@click.option('--hdfs_out_dir', default=None, type=str, required=True,
              help='I.e. /tmp/${KERBEROS_USER}/rucio_ds_mongo/$(date +%Y-%m-%d) ')
def main(hdfs_out_dir):
    """Main function that run Spark dataframe creations and save results to HDFS directory as JSON lines
    """
    hdfs_out_dir_detailed = hdfs_out_dir + "/" + HDFS_SUB_DIR_DETAILED
    hdfs_out_dir_in_both = hdfs_out_dir + "/" + HDFS_SUB_DIR_IN_BOTH

    # HDFS output file format. If you change, please modify bin/cron4rucio_ds_mongo.sh accordingly.
    write_format = 'json'
    write_mode = 'overwrite'

    spark = get_spark_session(app_name='cms-monitoring-rucio-detailed-datasets-for-mongo')
    # Set TZ as UTC. Also set in the spark-submit confs.
    spark.conf.set("spark.sql.session.timeZone", "UTC")

    # Detailed datasets
    df_datasets_in_each_rse = get_df_main_datasets_in_each_rse(spark)
    df_datasets_in_each_rse.write.save(path=hdfs_out_dir_detailed, format=write_format, mode=write_mode)

    # Datasets in both Tape and Disk
    df_datasets_in_both_disk_and_tape = get_df_main_datasets_in_both_disk_and_tape(df_datasets_in_each_rse)
    df_datasets_in_both_disk_and_tape.write.save(path=hdfs_out_dir_in_both, format=write_format, mode=write_mode)


if __name__ == '__main__':
    main()
