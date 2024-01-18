#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
File        : stepchain_cpu_eff.py
Author      : Author: Ceyhun Uzunoglu <ceyhunuzngl AT gmail [DOT] com>
Description : Generates StepChain tasks' CPU efficiency and
              sends results to MongoDB for a Go web service using intermediate steps.
"""

# system modules
import os
from datetime import date, datetime, timedelta

import click
import pandas as pd
# CMSSpark modules
from CMSSpark.spark_utils import get_spark_session, get_candidate_files
from pyspark.sql.functions import (array_distinct, col, collect_set, flatten, mean as _mean,
                                   size as _list_size, sum as _sum, when, )
from pyspark.sql.types import StructType, StructField, StringType, IntegerType, DoubleType, LongType, ArrayType

pd.options.display.float_format = "{:,.2f}".format
pd.set_option("display.max_colwidth", None)

# global variables
_DEFAULT_HDFS_FOLDER = "/project/monitoring/archive/wmarchive/raw/metric"
_VALID_DATE_FORMATS = ["%Y/%m/%d", "%Y-%m-%d", "%Y%m%d"]
_PROD_CMS_JOB_TYPES_FILTER = ["Production", "Processing", "Merge", "LogCollect", "Harvesting"]
_HOUR_DENOM = 60 * 60


def get_rdd_schema():
    """Final schema of steps"""
    return StructType(
        [
            StructField('ts', LongType(), nullable=False),
            StructField('Task', StringType(), nullable=False),
            StructField('fwjr_id', StringType(), nullable=True),
            StructField('JobType', StringType(), nullable=True),
            StructField('Site', StringType(), nullable=True),
            StructField('acquisition_era', ArrayType(StringType(), False), nullable=True),
            StructField('StepName', StringType(), nullable=True),
            StructField('nstreams', IntegerType(), nullable=True),
            StructField('nthreads', IntegerType(), nullable=True),
            StructField('cpu_time', DoubleType(), nullable=True),
            StructField('job_time', DoubleType(), nullable=True),
            StructField('threads_total_job_time', DoubleType(), nullable=True),
            StructField('era_length', IntegerType(), nullable=True),
            StructField('number_of_steps', IntegerType(), nullable=True),
            StructField('write_total_mb', DoubleType(), nullable=True),
            StructField('read_total_mb', DoubleType(), nullable=True),
            StructField('read_total_secs', DoubleType(), nullable=True),
            StructField('write_total_secs', DoubleType(), nullable=True),
            StructField('peak_rss', DoubleType(), nullable=True),
            StructField('peak_v_size', DoubleType(), nullable=True),
        ]
    )


def udf_step_extract(row):
    """
    Borrowed from wmarchive.py

    Helper function to extract useful data from WMArchive records.
    Returns list of step_res
    """
    result = []
    count = 0
    _task_name = row['task']
    _fwjr_id = row['meta_data']['fwjr_id']
    _jobtype = row['meta_data']['jobtype']
    _ts = row['meta_data']['ts']
    if 'steps' in row and row['steps']:
        for step in row['steps']:
            if ('name' in step) and step['name'].lower().startswith('cmsrun'):
                step_res = {'Task': _task_name, 'ts': _ts, 'fwjr_id': _fwjr_id, 'JobType': _jobtype}

                count += 1
                step_res['StepName'] = step['name']
                step_res['Site'] = step['site']
                step_res['nstreams'] = step['performance']['cpu']['NumberOfStreams']
                step_res['nthreads'] = step['performance']['cpu']['NumberOfThreads']
                step_res['cpu_time'] = step['performance']['cpu']['TotalJobCPU']
                step_res['job_time'] = step['performance']['cpu']['TotalJobTime']
                step_res['acquisition_era'] = []
                if step_res['nthreads'] and step_res['cpu_time'] and step_res['job_time']:
                    try:
                        step_res['threads_total_job_time'] = step_res['job_time'] * step_res['nthreads']
                    except Exception:
                        step_res['threads_total_job_time'] = None
                if step['output']:
                    for outx in step['output']:
                        if outx['acquisitionEra']:
                            step_res['acquisition_era'].append(outx['acquisitionEra'])
                if 'performance' in step:
                    performance = step['performance']
                    if 'storage' in performance:
                        if 'writeTotalMB' in performance['storage']:
                            step_res['write_total_mb'] = performance['storage']['writeTotalMB']
                        if 'readTotalMB' in performance['storage']:
                            step_res['read_total_mb'] = performance['storage']['readTotalMB']
                        if 'writeTotalSecs' in performance['storage'] and performance['storage']['writeTotalSecs']:
                            step_res['write_total_secs'] = float(performance['storage']['writeTotalSecs'])
                        if 'readTotalSecs' in performance['storage'] and performance['storage']['readTotalSecs']:
                            step_res['read_total_secs'] = float(performance['storage']['readTotalSecs'])
                    if 'memory' in performance:
                        if 'PeakValueRss' in performance['memory']:
                            step_res['peak_rss'] = performance['memory']['PeakValueRss']
                        if 'PeakValueVsize' in performance['memory']:
                            step_res['peak_v_size'] = performance['memory']['PeakValueVsize']
                # Get unique with set operations
                step_res['acquisition_era'] = list(set(step_res['acquisition_era']))
                step_res['era_length'] = len(step_res['acquisition_era'])
                result.append(step_res)
        if result is not None:
            [r.setdefault("number_of_steps", count) for r in result]
            return result


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
    click.echo("Stepchain cpu efficiency results to HDFS temp directory which will be imported to MongoDB from there")
    click.echo(f"Input Arguments: start_date:{start_date}, end_date:{end_date}, "
               f"last_n_days:{last_n_days}, hdfs_out_dir:{hdfs_out_dir}")

    mongo_collection_names = (
        "sc_task",
        "sc_task_cmsrun_jobtype",
        "sc_task_cmsrun_jobtype_site",
    )
    # HDFS output dict collection:hdfs path
    hdfs_out_collection_dirs = {c: hdfs_out_dir + "/" + c for c in mongo_collection_names}

    mongo_db = os.getenv("MONGO_WRITE_DB", "cpueff")
    mongo_auth_db = os.getenv("MONGO_AUTH_DB", "admin")
    mongo_host = os.getenv("MONGO_HOST")
    mongo_port = os.getenv("MONGO_PORT")
    mongo_u = os.getenv("MONGO_ROOT_USERNAME")
    mongo_p = os.getenv("MONGO_ROOT_PASSWORD")
    write_format, write_mode = 'json', 'overwrite'

    _yesterday = datetime.combine(date.today() - timedelta(days=1), datetime.min.time())
    if not (start_date or end_date):
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

    spark = get_spark_session(app_name='cms-stepchain-cpu-eff')
    # Set TZ as UTC. Also set in the spark-submit confs.
    spark.conf.set("spark.sql.session.timeZone", "UTC")

    df_raw = (
        spark.read.option("basePath", _DEFAULT_HDFS_FOLDER).json(
            get_candidate_files(start_date, end_date, spark, base=_DEFAULT_HDFS_FOLDER, day_delta=2)
        )
        .select(["data.*", "metadata.timestamp"])
        .filter(f"""data.meta_data.jobstate='success'
                  AND data.wmats >= {start_date.timestamp()}
                  AND data.wmats < {end_date.timestamp()}
                  """)
        .filter(col('data.meta_data.jobtype').isin(_PROD_CMS_JOB_TYPES_FILTER))
    )

    df_rdd = df_raw.rdd.flatMap(lambda r: udf_step_extract(r))
    df = spark.createDataFrame(df_rdd, schema=get_rdd_schema()).dropDuplicates().where(
        col("nstreams").isNotNull())

    df_task = (
        df.groupby(["Task"]).agg(
            (100 * _sum("cpu_time") / _sum("threads_total_job_time")).alias("CpuEfficiency"),
            _mean("number_of_steps").alias("NumberOfStep"),
            _mean("nthreads").alias("MeanThread"),
            _mean("nstreams").alias("MeanStream"),
            (_mean("cpu_time") / _HOUR_DENOM).alias("MeanCpuTimeHr"),
            (_sum("cpu_time") / _HOUR_DENOM).alias("TotalCpuTimeHr"),
            (_mean("job_time") / _HOUR_DENOM).alias("MeanJobTimeHr"),
            (_sum("job_time") / _HOUR_DENOM).alias("TotalJobTimeHr"),
            (_sum("threads_total_job_time") / _HOUR_DENOM).alias("TotalThreadJobTimeHr"),
            (_sum("write_total_secs") / _HOUR_DENOM).alias("WriteTotalHr"),
            (_sum("read_total_secs") / _HOUR_DENOM).alias("ReadTotalHr"),
            (100 * _sum("read_total_secs") / _sum("job_time")).alias("ReadTimePercentage"),
            _sum("write_total_mb").alias("WriteTotalMB"),
            _sum("read_total_mb").alias("ReadTotalMB"),
            _mean(when(col("peak_rss").isNotNull(), col("peak_rss"))).alias("MeanPeakRss"),
            _mean(when(col("peak_v_size").isNotNull(), col("peak_v_size"))).alias("MeanPeakVSize"),
            array_distinct(flatten(collect_set(col("acquisition_era")))).alias("AcquisitionEra"),
            collect_set("Site").alias("Sites"),
        )
        .withColumn("EraCount", _list_size(col("AcquisitionEra")))
        .withColumn("SiteCount", _list_size(col("Sites")))
        .sort(col("Task"))
    )

    df_task_cmsrun_jobtype = (
        df.groupby(["Task", "StepName", "JobType"]).agg(
            (100 * _sum("cpu_time") / _sum("threads_total_job_time")).alias("CpuEfficiency"),
            _mean("number_of_steps").alias("NumberOfStep"),
            _mean("nthreads").alias("MeanThread"),
            _mean("nstreams").alias("MeanStream"),
            (_mean("cpu_time") / _HOUR_DENOM).alias("MeanCpuTimeHr"),
            (_sum("cpu_time") / _HOUR_DENOM).alias("TotalCpuTimeHr"),
            (_mean("job_time") / _HOUR_DENOM).alias("MeanJobTimeHr"),
            (_sum("job_time") / _HOUR_DENOM).alias("TotalJobTimeHr"),
            (_sum("threads_total_job_time") / _HOUR_DENOM).alias("TotalThreadJobTimeHr"),
            (_sum("write_total_secs") / _HOUR_DENOM).alias("WriteTotalHr"),
            (_sum("read_total_secs") / _HOUR_DENOM).alias("ReadTotalHr"),
            (100 * _sum("read_total_secs") / _sum("job_time")).alias("ReadTimePercentage"),
            _sum("write_total_mb").alias("WriteTotalMB"),
            _sum("read_total_mb").alias("ReadTotalMB"),
            _mean(when(col("peak_rss").isNotNull(), col("peak_rss"))).alias("MeanPeakRss"),
            _mean(when(col("peak_v_size").isNotNull(), col("peak_v_size"))).alias("MeanPeakVSize"),
            array_distinct(flatten(collect_set(col("acquisition_era")))).alias("AcquisitionEra"),
            collect_set("Site").alias("Sites"),
        )
        .withColumn("EraCount", _list_size(col("AcquisitionEra")))
        .withColumn("SiteCount", _list_size(col("Sites")))
        .sort(col("Task"), col("StepName"), col("JobType"))
    )

    df_task_cmsrun_jobtype_site = (
        df.groupby(["Task", "StepName", "JobType", "Site"]).agg(
            (100 * _sum("cpu_time") / _sum("threads_total_job_time")).alias("CpuEfficiency"),
            _mean("number_of_steps").alias("NumberOfStep"),
            _mean("nthreads").alias("MeanThread"),
            _mean("nstreams").alias("MeanStream"),
            (_mean("cpu_time") / _HOUR_DENOM).alias("MeanCpuTimeHr"),
            (_sum("cpu_time") / _HOUR_DENOM).alias("TotalCpuTimeHr"),
            (_mean("job_time") / _HOUR_DENOM).alias("MeanJobTimeHr"),
            (_sum("job_time") / _HOUR_DENOM).alias("TotalJobTimeHr"),
            (_sum("threads_total_job_time") / _HOUR_DENOM).alias("TotalThreadJobTimeHr"),
            (_sum("write_total_secs") / _HOUR_DENOM).alias("WriteTotalHr"),
            (_sum("read_total_secs") / _HOUR_DENOM).alias("ReadTotalHr"),
            (100 * _sum("read_total_secs") / _sum("job_time")).alias("ReadTimePercentage"),
            _sum("write_total_mb").alias("WriteTotalMB"),
            _sum("read_total_mb").alias("ReadTotalMB"),
            _mean(when(col("peak_rss").isNotNull(), col("peak_rss"))).alias("MeanPeakRss"),
            _mean(when(col("peak_v_size").isNotNull(), col("peak_v_size"))).alias("MeanPeakVSize"),
            array_distinct(flatten(collect_set(col("acquisition_era")))).alias("AcquisitionEra"),
        )
        .withColumn("EraCount", _list_size(col("AcquisitionEra")))
        .sort(col("Task"), col("StepName"), col("JobType"), col("Site"))
    )

    # Write results to HDFS temporary location
    df_task.write.save(path=hdfs_out_collection_dirs['sc_task'], format=write_format, mode=write_mode)
    df_task_cmsrun_jobtype.write.save(path=hdfs_out_collection_dirs['sc_task_cmsrun_jobtype'],
                                      format=write_format, mode=write_mode)
    df_task_cmsrun_jobtype_site.write.save(path=hdfs_out_collection_dirs['sc_task_cmsrun_jobtype_site'],
                                           format=write_format, mode=write_mode)

    for mongo_col in mongo_collection_names:
        # Get hdfs results to single local json file with collection name
        os.system(f"hadoop fs -getmerge {hdfs_out_collection_dirs[mongo_col]}/part-*.json {mongo_col}.json")
        # Send local json file to MongoDB
        mongoimport_cmd_prefix = f"/data/mongoimport --drop --type=json --port {mongo_port} " \
                                 f"--host {mongo_host} --username {mongo_u} --password {mongo_p} " \
                                 f"--authenticationDatabase {mongo_auth_db} --db {mongo_db} "
        os.system(mongoimport_cmd_prefix + f"--collection {mongo_col} --file {mongo_col}.json")
        # Count lines in json file
        os.system(f"wc -l {mongo_col}.json")
        # Delete local json file
        os.system(f"rm -f {mongo_col}.json")


if __name__ == "__main__":
    main()
