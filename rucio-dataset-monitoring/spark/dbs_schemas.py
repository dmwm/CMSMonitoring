#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
File        : dbs_schemas.py
Author      : Valentin Kuznetsov <vkuznet AT gmail [DOT] com>, Ceyhun Uzunoglu <ceyhunuzngl AT gmail [DOT] com>
Description : Schema module for DBS/PhEDEx/AAA/EOS/CMSSW/JobMonitoring meta-data on HDFS
"""

# spark modules
from pyspark.sql.types import DoubleType, IntegerType, LongType, StructType, StructField, StringType


def schema_datasets():
    """DBS DATASETS table schema

    DATASET_ID NOT NULL NUMBER(38)
    DATASET NOT NULL VARCHAR2(700)
    IS_DATASET_VALID NOT NULL NUMBER(38)
    PRIMARY_DS_ID NOT NULL NUMBER(38)
    PROCESSED_DS_ID NOT NULL NUMBER(38)
    DATA_TIER_ID NOT NULL NUMBER(38)
    DATASET_ACCESS_TYPE_ID NOT NULL NUMBER(38)
    ACQUISITION_ERA_ID NUMBER(38)
    PROCESSING_ERA_ID NUMBER(38)
    PHYSICS_GROUP_ID NUMBER(38)
    XTCROSSSECTION FLOAT(126)
    PREP_ID VARCHAR2(256)
    CREATION_DATE NUMBER(38)
    CREATE_BY VARCHAR2(500)
    LAST_MODIFICATION_DATE NUMBER(38)
    LAST_MODIFIED_BY VARCHAR2(500)

    :returns: StructType consisting StructField array
    """
    return StructType([
        StructField("DATASET_ID", LongType(), True),
        StructField("DATASET", StringType(), True),
        StructField("IS_DATASET_VALID", IntegerType(), True),
        StructField("PRIMARY_DS_ID", LongType(), True),
        StructField("PROCESSED_DS_ID", LongType(), True),
        StructField("DATA_TIER_ID", IntegerType(), True),
        StructField("DATASET_ACCESS_TYPE_ID", IntegerType(), True),
        StructField("ACQUISITION_ERA_ID", IntegerType(), True),
        StructField("PROCESSING_ERA_ID", IntegerType(), True),
        StructField("PHYSICS_GROUP_ID", IntegerType(), True),
        StructField("XTCROSSSECTION", DoubleType(), True),
        StructField("PREP_ID", StringType(), True),
        StructField("CREATION_DATE", DoubleType(), True),
        StructField("CREATE_BY", StringType(), True),
        StructField("LAST_MODIFICATION_DATE", DoubleType(), True),
        StructField("LAST_MODIFIED_BY", StringType(), True)
    ])


def schema_blocks():
    """DBS BLOCKS table schema

    BLOCK_ID NOT NULL NUMBER(38)
    BLOCK_NAME NOT NULL VARCHAR2(500)
    DATASET_ID NOT NULL NUMBER(38)
    OPEN_FOR_WRITING NOT NULL NUMBER(38)
    ORIGIN_SITE_NAME NOT NULL VARCHAR2(100)
    BLOCK_SIZE NUMBER(38)
    FILE_COUNT NUMBER(38)
    CREATION_DATE NUMBER(38)
    CREATE_BY VARCHAR2(500)
    LAST_MODIFICATION_DATE NUMBER(38)
    LAST_MODIFIED_BY VARCHAR2(500)

    :returns: StructType consisting StructField array
    """
    return StructType([
        StructField("BLOCK_ID", LongType(), True),
        StructField("BLOCK_NAME", StringType(), True),
        StructField("DATASET_ID", LongType(), True),
        StructField("OPEN_FOR_WRITING", IntegerType(), True),
        StructField("ORIGIN_SITE_NAME", StringType(), True),
        StructField("BLOCK_SIZE", DoubleType(), True),
        StructField("FILE_COUNT", LongType(), True),
        StructField("CREATION_DATE", DoubleType(), True),
        StructField("CREATE_BY", StringType(), True),
        StructField("LAST_MODIFICATION_DATE", DoubleType(), True),
        StructField("LAST_MODIFIED_BY", StringType(), True)
    ])


def schema_files():
    """DBS FILES table schema

    FILE_ID NOT NULL NUMBER(38)
    LOGICAL_FILE_NAME NOT NULL VARCHAR2(500)
    IS_FILE_VALID NOT NULL NUMBER(38)
    DATASET_ID NOT NULL NUMBER(38)
    BLOCK_ID NOT NULL NUMBER(38)
    FILE_TYPE_ID NOT NULL NUMBER(38)
    CHECK_SUM NOT NULL VARCHAR2(100)
    EVENT_COUNT NOT NULL NUMBER(38)
    FILE_SIZE NOT NULL NUMBER(38)
    BRANCH_HASH_ID NUMBER(38)
    ADLER32 VARCHAR2(100)
    MD5 VARCHAR2(100)
    AUTO_CROSS_SECTION FLOAT(126)
    CREATION_DATE NUMBER(38)
    CREATE_BY VARCHAR2(500)
    LAST_MODIFICATION_DATE NUMBER(38)
    LAST_MODIFIED_BY VARCHAR2(500)

    :returns: StructType consisting StructField array
    """
    return StructType([
        StructField("FILE_ID", LongType(), True),
        StructField("LOGICAL_FILE_NAME", StringType(), True),
        StructField("IS_FILE_VALID", IntegerType(), True),
        StructField("DATASET_ID", LongType(), True),
        StructField("BLOCK_ID", LongType(), True),
        StructField("FILE_TYPE_ID", IntegerType(), True),
        StructField("CHECK_SUM", StringType(), True),
        StructField("EVENT_COUNT", LongType(), True),
        StructField("FILE_SIZE", DoubleType(), True),
        StructField("BRANCH_HASH_ID", LongType(), True),
        StructField("ADLER32", StringType(), True),
        StructField("MD5", StringType(), True),
        StructField("AUTO_CROSS_SECTION", DoubleType(), True),
        StructField("CREATION_DATE", DoubleType(), True),
        StructField("CREATE_BY", StringType(), True),
        StructField("LAST_MODIFICATION_DATE", DoubleType(), True),
        StructField("LAST_MODIFIED_BY", StringType(), True)
    ])
