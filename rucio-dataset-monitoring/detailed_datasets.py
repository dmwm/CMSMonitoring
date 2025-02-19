#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
File        : detailed_datasets.py
Author      : Ceyhun Uzunoglu <ceyhunuzngl AT gmail [DOT] com>
Description : This Spark job creates detailed datasets(in each RSEs) results by aggregating Rucio&DBS tables and
                save result to HDFS directory as a source to MongoDB of go web service
"""
import time
# system modules
from datetime import datetime
from decimal import Decimal

import click
import pandas as pd
from pyspark import SparkContext
from pyspark.sql import DataFrame, DataFrameReader
from pyspark.sql import SparkSession
from pyspark.sql.functions import (
    coalesce,
    broadcast,
    col,
    collect_set,
    countDistinct,
    first,
    greatest,
    lit,
    lower,
    sort_array,
    when,
    hex as _hex,
    max as _max,
    split as _split,
    sum as _sum,
    unix_timestamp
)
from pyspark.sql.types import IntegerType, LongType, DecimalType

# Local
import dbs_schemas
import osearch

pd.options.display.float_format = "{:,.2f}".format
pd.set_option("display.max_colwidth", None)

# global variables
TODAY = datetime.today().strftime("%Y-%m-%d")
# Rucio
HDFS_RUCIO_RSES = f"/project/awg/cms/rucio/{TODAY}/rses/part*.avro"
HDFS_RUCIO_REPLICAS = f"/project/awg/cms/rucio/{TODAY}/replicas/part*.avro"
HDFS_RUCIO_DIDS = f"/project/awg/cms/rucio/{TODAY}/dids/part*.avro"
HDFS_RUCIO_DLOCKS = f"/project/awg/cms/rucio/{TODAY}/dataset_locks/part*.avro"
# DBS
HDFS_DBS_DATASETS = f"/project/awg/cms/dbs/PROD_GLOBAL/{TODAY}/DATASETS/*.gz"
HDFS_DBS_BLOCKS = f"/project/awg/cms/dbs/PROD_GLOBAL/{TODAY}/BLOCKS/*.gz"
HDFS_DBS_FILES = f"/project/awg/cms/dbs/PROD_GLOBAL/{TODAY}/FILES/*.gz"
PROD_ACCOUNTS = [
    "transfer_ops",
    "wma_prod",
    "wmcore_output",
    "wmcore_transferor",
    "crab_tape_recall",
    "sync",
]
SYNC_PREFIX = "sync"

HDFS_SUB_DIR_DETAILED = "detailed"


def get_spark_session(app_name: str) -> SparkSession:
    """Get or create the spark context and session."""
    sc = SparkContext(appName=app_name)
    return SparkSession.builder.config(conf=sc._conf).getOrCreate()


# --------------------------------------------------------------------------------------------------------------- DBS
def get_csvreader(spark: SparkSession) -> DataFrameReader:
    """CSV reader for DBS csv gz format"""
    return (
        spark.read.format("csv").option("nullValue", "null").option("mode", "FAILFAST")
    )


def get_df_dbs_files(spark: SparkSession) -> DataFrame:
    """Create DBS Files table dataframe"""
    return (
        get_csvreader(spark)
        .schema(dbs_schemas.schema_files())
        .load(HDFS_DBS_FILES)
        .filter(col("IS_FILE_VALID") == "1")
        .withColumnRenamed("LOGICAL_FILE_NAME", "FILE_NAME")
        .select(["FILE_NAME", "DATASET_ID", "BLOCK_ID", "FILE_SIZE"])
    )


def get_df_dbs_blocks(spark: SparkSession) -> DataFrame:
    """Create DBS Blocks table dataframe"""
    return (
        get_csvreader(spark)
        .schema(dbs_schemas.schema_blocks())
        .load(HDFS_DBS_BLOCKS)
        .select(["BLOCK_NAME", "BLOCK_ID", "DATASET_ID", "FILE_COUNT"])
    )


def get_df_dbs_datasets(spark: SparkSession) -> DataFrame:
    """Create DBS Datasets table dataframe"""
    return (
        get_csvreader(spark)
        .schema(dbs_schemas.schema_datasets())
        .load(HDFS_DBS_DATASETS)
        .filter(col("IS_DATASET_VALID") == "1")
        .select(["DATASET_ID", "DATASET"])
    )


def get_df_dbs_f_d_map(spark: SparkSession) -> DataFrame:
    """Create dataframe for DBS dataset:file map"""
    dbs_files = get_df_dbs_files(spark)
    dbs_datasets = get_df_dbs_datasets(spark)
    return dbs_files.join(dbs_datasets, ["DATASET_ID"], how="inner").select(
        ["DATASET_ID", "DATASET", "FILE_NAME"]
    )


def get_df_dbs_b_d_map(spark: SparkSession) -> DataFrame:
    """Create dataframe for DBS dataset:block map"""
    dbs_blocks = get_df_dbs_blocks(spark)
    dbs_datasets = get_df_dbs_datasets(spark)
    return dbs_blocks.join(dbs_datasets, ["DATASET_ID"], how="inner").select(
        ["DATASET_ID", "DATASET", "BLOCK_NAME"]
    )


def get_df_ds_file_and_block_cnt(spark: SparkSession) -> DataFrame:
    """Calculate total file and block count of a DBS dataset

    It will be used as a reference to define if dataset is fully replicated in RSE.
    """
    file_count_df = (
        get_df_dbs_f_d_map(spark)
        .groupby(["DATASET"])
        .agg(countDistinct(col("FILE_NAME")).alias("TOT_FILE_CNT"))
        .select(["DATASET", "TOT_FILE_CNT"])
    )
    block_count_df = (
        get_df_dbs_b_d_map(spark)
        .groupby(["DATASET"])
        .agg(countDistinct(col("BLOCK_NAME")).alias("TOT_BLOCK_CNT"))
        .select(["DATASET", "TOT_BLOCK_CNT"])
    )

    return file_count_df.join(block_count_df, ["DATASET"], how="inner").select(
        ["DATASET", "TOT_BLOCK_CNT", "TOT_FILE_CNT"]
    )


# ---------------------------------------------------------------------------------------------------------------------


def get_df_rses(spark: SparkSession) -> DataFrame:
    """Create rucio RSES table dataframe with some rse tag calculations"""
    df_rses = (
        spark.read.format("avro")
        .load(HDFS_RUCIO_RSES)
        .filter(col("DELETED_AT").isNull())
        .withColumn("rse_id", lower(_hex(col("ID"))))
        .withColumn("rse_tier", _split(col("RSE"), "_").getItem(0))
        .withColumn("rse_country", _split(col("RSE"), "_").getItem(1))
        .withColumn(
            "rse_kind",
            when(
                (
                    col("rse").endswith("Temp")
                    | col("rse").endswith("temp")
                    | col("rse").endswith("TEMP")
                ),
                "temp",
            )
            .when(
                (
                    col("rse").endswith("Test")
                    | col("rse").endswith("test")
                    | col("rse").endswith("TEST")
                ),
                "test",
            )
            .otherwise("prod"),
        )
        .select(["rse_id", "RSE", "RSE_TYPE", "rse_tier", "rse_country", "rse_kind"])
    )
    return broadcast(df_rses)


def get_df_replicas(spark: SparkSession) -> DataFrame:
    """Create rucio Replicas table dataframe"""
    return (
        spark.read.format("avro")
        .load(HDFS_RUCIO_REPLICAS)
        .filter(col("SCOPE") == "cms")
        .filter(col("STATE") == "A")
        .withColumn("rse_id", lower(_hex(col("RSE_ID"))))
        .withColumn("f_size_replicas", col("BYTES").cast(LongType()))
        .withColumnRenamed("NAME", "f_name")
        .withColumnRenamed("ACCESSED_AT", "rep_accessed_at")
        .withColumnRenamed("CREATED_AT", "rep_created_at")
        .select(
            ["f_name", "rse_id", "f_size_replicas", "rep_accessed_at", "rep_created_at"]
        )
    )


def get_df_dids_files(spark: SparkSession) -> DataFrame:
    """Create rucio DIDS table dataframe for files"""
    return (
        spark.read.format("avro")
        .load(HDFS_RUCIO_DIDS)
        .filter(col("DELETED_AT").isNull())
        .filter(col("HIDDEN") == "0")
        .filter(col("SCOPE") == "cms")
        .filter(col("DID_TYPE") == "F")
        .withColumnRenamed("NAME", "f_name")
        .withColumnRenamed("ACCESSED_AT", "dids_accessed_at")
        .withColumnRenamed("CREATED_AT", "dids_created_at")
        .withColumn("f_size_dids", col("BYTES").cast(LongType()))
        .select(["f_name", "f_size_dids", "dids_accessed_at", "dids_created_at"])
    )


def get_df_dlocks(spark: SparkSession) -> DataFrame:
    """Create rucio DatasetLocks table dataframe

    - If account is sync, use sync prefix for all of them. Filter only production accounts.
    - DatasetLocks will be used to calculate number of locked blocks. If all blocks of dataset are locked, dataset  will
    be set as locked.

    - "sync-rules" represents all SYNC account rules. It can 1 or many. Because of their quantity, this solution is
    applied to provide better visualization.
    """
    df_dlocks = (
        spark.read.format("avro")
        .load(HDFS_RUCIO_DLOCKS)
        .filter(col("SCOPE") == "cms")
        .withColumn("rse_id", lower(_hex(col("RSE_ID"))))
        .withColumn("rule_id", lower(_hex(col("RULE_ID"))))
        .withColumn(
            "account",
            when(col("ACCOUNT").startswith(SYNC_PREFIX), lit(SYNC_PREFIX)).otherwise(
                col("ACCOUNT")
            ),
        )
        .filter(col("account").isin(PROD_ACCOUNTS))
        .withColumnRenamed("NAME", "dlocks_block_name")
        .select(["rse_id", "dlocks_block_name", "account", "rule_id"])
    )

    # Map locks(block) to datasets
    df_dbs_b_d_map = get_df_dbs_b_d_map(spark)
    df_dlocks = df_dlocks.join(
        df_dbs_b_d_map,
        df_dlocks.dlocks_block_name == df_dbs_b_d_map.BLOCK_NAME,
        how="left",
    ).select(["rse_id", "dlocks_block_name", "account", "rule_id", "DATASET_ID"])

    # Change SYNC rule_ids to "sync-rules", because there are too many of them
    df_dlocks = df_dlocks.withColumn(
        "rule_id",
        when(col("account") == SYNC_PREFIX, "sync-rules").otherwise(col("rule_id")),
    )

    # Group by DATASET and RSE to gather dataset lock accounts and rule counts
    df_dlocks = (
        df_dlocks.groupby(["rse_id", "DATASET_ID"])
        .agg(
            sort_array(collect_set("account")).alias("ProdAccounts"),
            collect_set("rule_id").alias("BlockRuleIDs"),
            countDistinct(col("dlocks_block_name")).alias("ProdLockedBlockCount"),
        )
        .select(
            [
                "rse_id",
                "DATASET_ID",
                "ProdAccounts",
                "BlockRuleIDs",
                "ProdLockedBlockCount",
            ]
        )
    )

    # Get RSE from its id
    df_rses = get_df_rses(spark)
    return (
        df_dlocks.join(df_rses.select(["rse_id", "RSE"]), ["rse_id"], how="left")
        .withColumnRenamed("RSE", "dlocks_RSE")
        .select(
            [
                "dlocks_RSE",
                "DATASET_ID",
                "ProdAccounts",
                "BlockRuleIDs",
                "ProdLockedBlockCount",
            ]
        )
    )


def get_df_files_enriched(spark: SparkSession) -> DataFrame:
    """Enriched files with REPLICAS and DIDS

    Add replica size, access time and creation of replica files of each RSE.
    """
    df_replicas = get_df_replicas(spark)
    df_dids_files = get_df_dids_files(spark)
    df_rses = get_df_rses(spark)
    df_rep_enr_dids = (
        df_replicas.join(df_dids_files, ["f_name"], how="left")
        .withColumn(
            "f_size",
            when(col("f_size_replicas").isNotNull(), col("f_size_replicas")).when(
                col("f_size_dids").isNotNull(), col("f_size_dids")
            ),
        )
        .withColumn(
            "accessed_at", greatest(col("dids_accessed_at"), col("rep_accessed_at"))
        )
        .withColumn(
            "created_at", greatest(col("dids_created_at"), col("rep_created_at"))
        )
        .select(["f_name", "rse_id", "accessed_at", "f_size", "created_at"])
    )

    return df_rep_enr_dids.join(
        df_rses.select(["rse_id", "RSE"]), ["rse_id"], how="left"
    ).select(["f_name", "RSE", "f_size", "accessed_at", "created_at"])


def get_df_datasets_files_phase1(spark: SparkSession) -> DataFrame:
    """Files with dataset name

    Map dataset-RSE and files with enriched values.
    """
    df_files_enriched = get_df_files_enriched(spark)
    df_dbs_f_d_map = get_df_dbs_f_d_map(spark)
    return (
        df_files_enriched.join(
            df_dbs_f_d_map,
            df_files_enriched.f_name == df_dbs_f_d_map.FILE_NAME,
            how="left",
        )
        .fillna("UnknownDatasetNameOfFiles_MonitoringTag", subset=["DATASET"])
        .fillna("0", subset=["DATASET_ID"])
        .withColumnRenamed("DATASET_ID", "d_id")
        .withColumnRenamed("DATASET", "d_name")
        .select(
            ["d_id", "d_name", "RSE", "f_name", "f_size", "accessed_at", "created_at"]
        )
    )


def get_df_main_datasets_in_each_rse(spark: SparkSession) -> DataFrame:
    """Main"""
    # Add last access and size of dataset for each RSE
    # ( Dataset, RSE ) + ( FileCount, AccessedFileCount, LastAccess, LastCreate, SizeBytes )
    df_datasets_files_phase1 = get_df_datasets_files_phase1(spark)
    df = (
        df_datasets_files_phase1.groupby(["RSE", "d_name"])
        .agg(
            countDistinct(col("f_name")).alias("FileCount"),
            _max(col("accessed_at")).alias("LastAccessMs"),
            _max(col("created_at")).alias("LastCreateMs"),
            _sum(col("f_size")).alias("SizeBytes"),
            _sum(when(col("accessed_at").isNull(), 0).otherwise(1)).alias(
                "AccessedFileCount"
            ),
            first(col("d_id")).alias("d_id"),
        )
        .withColumn("LastAccess", (col("LastAccessMs") / 1000).cast(LongType()))
        .withColumn("LastCreate", (col("LastCreateMs") / 1000).cast(LongType()))
    )

    # Add file counts and block counts with additional tags like IsFullyReplicated, FilePercentage
    # + ( FilePercentage, IsFullyReplicated, TOT_BLOCK_CNT->BlockCount )
    df_ds_file_and_block_cnt = get_df_ds_file_and_block_cnt(spark)
    df = (
        df.join(
            df_ds_file_and_block_cnt,
            df.d_name == df_ds_file_and_block_cnt.DATASET,
            how="left",
        )
        .drop("DATASET")
        .withColumn(
            "FilePercentage",
            (100 * col("FileCount") / col("TOT_FILE_CNT")).cast(DecimalType(6, 2)),
        )
        .withColumn(
            "IsFullyReplicated",
            when(col("FileCount") == col("TOT_FILE_CNT"), lit(True)).otherwise(
                lit(False)
            ),
        )
        .withColumn("Id", col("d_id").cast(LongType()))
        .withColumnRenamed("d_name", "Dataset")
        .withColumnRenamed("TOT_BLOCK_CNT", "BlockCount")
        .select(
            [
                "RSE",
                "Dataset",
                "Id",
                "SizeBytes",
                "LastAccess",
                "LastCreate",
                "FileCount",
                "AccessedFileCount",
                "IsFullyReplicated",
                "FilePercentage",
                "BlockCount",
            ]
        )
    )

    df_dlocks = get_df_dlocks(spark)
    # if      ( BlockCount/TOT_BLOCK_CNT == ProdLockedBlockCount ): FULLY
    # else if ( ProdLockedBlockCount >=1                         ): PARTIAL
    # else                                                        : DYNAMIC
    df = (
        df.join(
            df_dlocks,
            on=((df.Id == df_dlocks.DATASET_ID) & (df.RSE == df_dlocks.dlocks_RSE)),
            how="left",
        )
        .withColumn(
            "IsLocked",
            when(col("BlockCount") == col("ProdLockedBlockCount"), lit("FULLY"))
            .when((col("ProdLockedBlockCount") >= 1), lit("PARTIAL"))
            .otherwise(lit("DYNAMIC")),
        )
        .withColumn(
            "ProdLockedBlockCount",
            coalesce(col("ProdLockedBlockCount"), lit(0)).cast(IntegerType()),
        )
        .select(
            [
                "RSE",
                "Dataset",
                "Id",
                "SizeBytes",
                "LastAccess",
                "LastCreate",
                "IsFullyReplicated",
                "IsLocked",
                "FilePercentage",
                "FileCount",
                "AccessedFileCount",
                "BlockCount",
                "ProdLockedBlockCount",
                "ProdAccounts",
                "BlockRuleIDs",
            ]
        )
    )

    df_rses = get_df_rses(spark)
    return (
        df.join(df_rses, ["RSE"], how="left")
        .withColumnRenamed("RSE_TYPE", "Type")
        .withColumnRenamed("rse_tier", "Tier")
        .withColumnRenamed("rse_country", "C")
        .withColumnRenamed("rse_kind", "RseKind")
        .fillna(0, subset=["LastAccess"])
        .select(
            [
                "Type",
                "Dataset",
                "RSE",
                "Tier",
                "C",
                "RseKind",
                "SizeBytes",
                "LastAccess",
                "LastCreate",
                "IsFullyReplicated",
                "IsLocked",
                "FilePercentage",
                "FileCount",
                "AccessedFileCount",
                "BlockCount",
                "ProdLockedBlockCount",
                "ProdAccounts",
                "BlockRuleIDs",
            ]
        )
        .withColumn("timestamp", unix_timestamp())
    )


def get_index_schema():
    return {
        "settings": {"index": {"number_of_shards": 1, "number_of_replicas": 1}},
        "mappings": {
            "properties": {
                "Type": {"type": "keyword"},
                "Dataset": {"type": "keyword"},
                "RSE": {"type": "keyword"},
                "Tier": {"type": "keyword"},
                "C": {"type": "keyword"},
                "RseKind": {"type": "keyword"},
                "SizeBytes": {"type": "long"},
                "LastAccess": {"format": "epoch_second", "type": "date"},
                "LastCreate": {"format": "epoch_second", "type": "date"},
                "IsFullyReplicated": {"type": "boolean"},
                "IsLocked": {"type": "keyword"},
                "FilePercentage": {"type": "float"},
                "FileCount": {"type": "integer"},
                "AccessedFileCount": {"type": "integer"},
                "BlockCount": {"type": "integer"},
                "ProdLockedBlockCount": {"type": "integer"},
                "ProdAccounts": {"type": "keyword"},
                "BlockRuleIDs": {"type": "keyword"},
                "timestamp": {"format": "epoch_second", "type": "date"},
            }
        },
    }


def drop_nulls_in_dict(d):
    def convert_value(value):
        if value is None:
            return None
        if isinstance(value, Decimal):
            return float(value)
        return value

    return {k: convert_value(v) for k, v in d.items() if convert_value(v) is not None}


def send(part, opensearch_host, es_secret_file, es_index_template):
    """Send given data to OpenSearch"""
    client = osearch.get_es_client(opensearch_host, es_secret_file, get_index_schema())
    # Monthly index format: index_mod="M"
    idx = client.get_or_create_index(
        timestamp=time.time(), index_template=es_index_template, index_mod="D"
    )
    client.send(idx, part, metadata=None, batch_size=10000, drop_nulls=False)


@click.command()
@click.option(
    "--hdfs_out_dir",
    default=None,
    type=str,
    required=True,
    help="I.e. /tmp/${KERBEROS_USER}/rucio_ds_detailed/$(date +%Y-%m-%d) ",
)
@click.option(
    "--es_host",
    required=True,
    default=None,
    type=str,
    help="OpenSearch host name without port: os-cms.cern.ch/os",
)
@click.option(
    "--es_secret_file",
    required=True,
    default=None,
    type=str,
    help='OpenSearch secret file that contains "user:pass" only',
)
@click.option(
    "--es_index",
    required=True,
    default=None,
    type=str,
    help='OpenSearch index template (prefix), i.e.: "test-wmarchive-agent-count"',
)
def main(hdfs_out_dir: str, es_host=None, es_secret_file=None, es_index=None) -> None:
    """Main function that run Spark dataframe creations and save results to HDFS directory as JSON lines"""
    hdfs_out_dir_detailed = hdfs_out_dir + "/" + HDFS_SUB_DIR_DETAILED

    # HDFS output file format. If you change, please modify bin/cron4rucio_ds_mongo.sh accordingly.
    write_format = "parquet"
    write_mode = "overwrite"

    spark = get_spark_session(app_name="cms-monitoring-rucio-detailed-datasets")
    # Set TZ as UTC. Also set in the spark-submit confs.
    spark.conf.set("spark.sql.session.timeZone", "UTC")

    # Detailed datasets
    df_datasets_in_each_rse = get_df_main_datasets_in_each_rse(spark).cache()
    df_datasets_in_each_rse.write.save(
        path=hdfs_out_dir_detailed, format=write_format, mode=write_mode
    )

    for part in df_datasets_in_each_rse.rdd.mapPartitions(
        lambda p: [[drop_nulls_in_dict(x.asDict()) for x in p]]
    ).toLocalIterator():
        part_size = len(part)
        print(f"Length of partition: {part_size}")
        send(
            part,
            opensearch_host=es_host,
            es_secret_file=es_secret_file,
            es_index_template=es_index,
        )


if __name__ == "__main__":
    main()
