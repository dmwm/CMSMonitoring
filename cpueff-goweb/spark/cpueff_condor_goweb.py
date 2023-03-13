#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
File        : cpueff_condor_goweb.py
Author      : Ceyhun Uzunoglu <ceyhunuzngl AT gmail [DOT] com>
Description : Generate aggregated information about workflows/requests cpu efficiency for the workflows/request matching
              the parameters. And send results to MongoDB for a Go web service using intermediate steps.
"""
import json

import click
import os
from datetime import datetime, date, timedelta

from pyspark.sql.functions import array_max, col, collect_set, lit, upper, when, sum as _sum
from pyspark.sql.types import StructType, LongType, StringType, StructField, DoubleType, IntegerType

# CMSSpark modules
from CMSSpark.spark_utils import get_candidate_files, get_spark_session

# global variables
_VALID_TYPES = ["analysis", "production", "folding@home", "test"]
_VALID_DATE_FORMATS = ["%Y/%m/%d", "%Y-%m-%d", "%Y%m%d"]
_BASE_HDFS_CONDOR = "/project/monitoring/archive/condor/raw/metric"


def _get_schema():
    return StructType(
        [
            StructField(
                "data",
                StructType(
                    [
                        StructField("GlobalJobId", StringType(), nullable=False),
                        StructField("Workflow", StringType(), nullable=False),
                        StructField("WMAgent_RequestName", StringType(), nullable=True),
                        StructField("ScheddName", StringType(), nullable=True),
                        StructField("WMAgent_JobID", StringType(), nullable=True),
                        StructField("RecordTime", LongType(), nullable=False),
                        StructField("JobFailed", LongType(), nullable=False),
                        StructField("Status", StringType(), nullable=True),
                        StructField("Site", StringType(), nullable=True),
                        StructField("Tier", StringType(), nullable=True),
                        StructField("Type", StringType(), nullable=True),
                        StructField("WallClockHr", DoubleType(), nullable=False),
                        StructField("CpuTimeHr", DoubleType(), nullable=True),
                        StructField("RequestCpus", DoubleType(), nullable=True),
                        StructField("CpuEff", DoubleType(), nullable=True),
                        StructField("CpuEffOutlier", IntegerType(), nullable=True),
                    ]
                ),
            ),
        ]
    )


@click.command()
@click.option("--start_date", type=click.DateTime(_VALID_DATE_FORMATS))
@click.option("--end_date", type=click.DateTime(_VALID_DATE_FORMATS))
@click.option("--hdfs_out_dir", default=None, type=str, required=True, help='I.e. /tmp/${KERBEROS_USER}/prod/cpueff ')
@click.option("--last_n_days", type=int, default=30, help="Last n days data will be used")
def main(start_date, end_date, hdfs_out_dir, last_n_days):
    """Main

    Examples:
        --mongo_host "cmsmon-test-2tzv4rdqsho2-node-1" --mongo_port 32017 --mongo_u admin --mongo_p admin
    """
    hdfs_out_dir = hdfs_out_dir.rstrip("/")
    click.echo("Condor cpu efficiency results to HDFS temp directory which will be imported to MongoDB from there")
    click.echo(f"Input Arguments: start_date:{start_date}, end_date:{end_date}, "
               f"last_n_days:{last_n_days}, hdfs_out_dir:{hdfs_out_dir}")

    mongo_collection_names = (
        "condor_main",
        "condor_detailed",
        "condor_tiers",
    )
    mongo_collection_source_timestamp = "source_timestamp"
    # HDFS output dict collection:hdfs path
    hdfs_out_collection_dirs = {c: hdfs_out_dir + "/" + c for c in mongo_collection_names}
    write_format, write_mode = 'json', 'overwrite'

    mongo_db = os.getenv("MONGO_WRITE_DB", "cpueff")
    mongo_auth_db = os.getenv("MONGO_AUTH_DB", "admin")
    mongo_host = os.getenv("MONGO_HOST")
    mongo_port = os.getenv("MONGO_PORT")
    mongo_u = os.getenv("MONGO_ROOT_USERNAME")
    mongo_p = os.getenv("MONGO_ROOT_PASSWORD")

    _yesterday = datetime.combine(date.today() - timedelta(days=1), datetime.min.time())
    if not (start_date or end_date):
        # defaults to the last 30 days with 3 days offset.
        # Default: (today-33days to today-3days)
        end_date = _yesterday
        start_date = end_date - timedelta(days=last_n_days)
    elif not start_date:
        start_date = end_date - timedelta(days=last_n_days)
    elif not end_date:
        end_date = min(start_date + timedelta(days=last_n_days), _yesterday)
    if start_date > end_date:
        raise ValueError(
            f"start date ({start_date}) should be earlier than end date({end_date})"
        )

    # Should be a list, used also in dataframe merge conditions.
    spark = get_spark_session(app_name="cms-cpu-efficiency")
    # Set TZ as UTC. Also set in the spark-submit confs.
    spark.conf.set("spark.sql.session.timeZone", "UTC")

    schema = _get_schema()
    raw_df = (
        spark.read.option("basePath", _BASE_HDFS_CONDOR).json(
            get_candidate_files(start_date, end_date, spark, base=_BASE_HDFS_CONDOR, day_delta=1),
            schema=schema,
        ).select("data.*").filter(
            f"""Status='Completed'
          AND JobFailed=0
          AND RecordTime >= {start_date.timestamp() * 1000}
          AND RecordTime < {end_date.timestamp() * 1000}
          """
        )
        .drop_duplicates(["GlobalJobId"])
        .fillna(value="NULL", subset=["WMAgent_RequestName"])
    )
    raw_df = raw_df.withColumn("Tier", upper(col("Tier")))
    raw_df = (
        raw_df
        .withColumn("RequestCpus",
                    when(col("RequestCpus").isNotNull(), col("RequestCpus")).otherwise(lit(1)))
        .withColumn("CoreTimeHr",
                    col("WallClockHr") * col("RequestCpus"))
        .withColumn("WastedCpuTimeHr",
                    ((col("RequestCpus") * col("WallClockHr")) - col("CpuTimeHr")))
        .withColumn("Tier", upper(col("Tier")))
        .withColumn("WmagentRequestName", col("WMAgent_RequestName"))
    ).cache()

    # Order is important
    group_by_cols_detailed = ["Type", "Workflow", "WmagentRequestName", "Site", "Tier", "CpuEffOutlier"]
    df_detailed = raw_df.groupby(group_by_cols_detailed).agg(
        (100 * _sum("CpuTimeHr") / _sum("CoreTimeHr")).alias("CpuEff"),
        _sum("RequestCpus").alias("Cpus"),
        _sum("CpuTimeHr").alias("CpuTimeHr"),
        _sum("WallClockHr").alias("WallClockHr"),
        _sum("CoreTimeHr").alias("CoreTimeHr"),
        _sum("WastedCpuTimeHr").alias("WastedCpuTimeHr"),
        collect_set("ScheddName").alias("Schedds"),
        array_max(collect_set("WMAgent_JobID")).alias("MaxWmagentJobId"),
    ).sort(col("Type").desc(), col("Workflow"), col("WmagentRequestName"), col("Site"), col("CoreTimeHr"))

    T1T2_Tiers = ("T1", "T2")
    group_by_cols_main = ["Type", "Workflow", "WmagentRequestName", "CpuEffOutlier"]
    df_main = raw_df.groupby(group_by_cols_main).agg(
        (100 * _sum("CpuTimeHr") / _sum("CoreTimeHr")).alias("CpuEff"),
        _sum("RequestCpus").alias("Cpus"),
        _sum("CpuTimeHr").alias("CpuTimeHr"),
        _sum("WallClockHr").alias("WallClockHr"),
        _sum("CoreTimeHr").alias("CoreTimeHr"),
        _sum("WastedCpuTimeHr").alias("WastedCpuTimeHr"),
        (100 *
         _sum(when(col("Tier").isin(*T1T2_Tiers), col("CpuTimeHr"))) /
         _sum(when(col("Tier").isin(*T1T2_Tiers), col("CoreTimeHr")))
         ).alias("CpuEffT1T2"),
        _sum(when(col("Tier").isin(*T1T2_Tiers), col("RequestCpus"))).alias("CpusT1T2"),
        _sum(when(col("Tier").isin(*T1T2_Tiers), col("CpuTimeHr"))).alias("CpuTimeHrT1T2"),
        _sum(when(col("Tier").isin(*T1T2_Tiers), col("WallClockHr"))).alias("WallClockHrT1T2"),
        _sum(when(col("Tier").isin(*T1T2_Tiers), col("CoreTimeHr"))).alias("CoreTimeHrT1T2"),
        _sum(when(col("Tier").isin(*T1T2_Tiers), col("WastedCpuTimeHr"))).alias("WastedCpuTimeHrT1T2"),
    ).sort(col("Type").desc(), col("Workflow"), col("WmagentRequestName"), col("CoreTimeHr"))

    df_tiers_overview = raw_df.filter(col("Tier") != "UNKNOWN").groupby("Tier", "Type").agg(
        (100 * _sum("CpuTimeHr") / _sum("CoreTimeHr")).alias("TierCpuEff"),
        _sum("RequestCpus").alias("TierCpus"),
        _sum("CpuTimeHr").alias("TierCpuTimeHr"),
        _sum("WallClockHr").alias("TierWallClockHr"),
    )

    df_main.write.save(path=hdfs_out_collection_dirs['condor_main'], format=write_format, mode=write_mode)
    df_detailed.write.save(path=hdfs_out_collection_dirs['condor_detailed'], format=write_format, mode=write_mode)
    df_tiers_overview.write.save(path=hdfs_out_collection_dirs['condor_tiers'], format=write_format, mode=write_mode)

    mongoimport_cmd_prefix = f"/data/mongoimport --drop --type=json --port {mongo_port} " \
                             f"--host {mongo_host} --username {mongo_u} --password {mongo_p} " \
                             f"--authenticationDatabase {mongo_auth_db} --db {mongo_db} "
    for mongo_col in mongo_collection_names:
        # Get hdfs results to single local json file with collection name
        os.system(f"hadoop fs -getmerge {hdfs_out_collection_dirs[mongo_col]}/part-*.json {mongo_col}.json")
        # Send local json file to MongoDB
        os.system(mongoimport_cmd_prefix + f"--collection {mongo_col} --file {mongo_col}.json")
        # Count lines in json file
        os.system(f"wc -l {mongo_col}.json")
        # Delete local json file
        os.system(f"rm -f {mongo_col}.json")

    # Import source timestamp
    source_timestamp_json_file = 'source_timestamp.json'
    with open(source_timestamp_json_file, 'w+') as f:
        f.write(json.dumps({
            "createdAt": end_date.strftime('%Y-%m-%d'),
            "startDate": start_date.strftime('%Y-%m-%d'),
            "endDate": end_date.strftime('%Y-%m-%d')
        }))
    os.system(mongoimport_cmd_prefix +
              f"--collection {mongo_collection_source_timestamp} --file {source_timestamp_json_file}")


if __name__ == "__main__":
    main()
