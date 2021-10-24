#!/usr/bin/env python
# coding: utf-8

"""
File:              scrutiny_plot.py
Author:         Christian Ariza <christian.ariza AT gmail dot com>
Description: This script generate a data popularity plot based on dbs and
 phedex data on hdfs, Based on https://github.com/dmwm/CMSPopularity
"""

import os
import sys
import argparse
from datetime import datetime, timedelta
from dateutil.relativedelta import relativedelta

import pandas as pd
import numpy as np


from pyspark import SparkContext
from pyspark.sql import SparkSession
from pyspark.sql.functions import (
    input_file_name,
    regexp_extract,
    concat,
    col,
    when,
    lit,
    last,
    max as _max,
    min as _min,
    datediff,
    countDistinct,
    avg,
    unix_timestamp,
    from_unixtime,
)
from pyspark.sql.types import (
    StringType,
    StructField,
    StructType,
    LongType,
    IntegerType,
    DoubleType,
)
from pyspark.sql import Window
import matplotlib

matplotlib.use("Agg")
from matplotlib import pyplot
import seaborn as sns


class OptionParser:
    def __init__(self):
        "User based option parser"
        desc = """
This app create a data popularity plot (scrutiny plot)
based on phedex and dbs hdfs data.
               """
        self.parser = argparse.ArgumentParser("Scrutiny Plot", usage=desc)
        self.parser.add_argument(
            "end_date",
            help="Date in yyyyMMdd format.",
            nargs="?",
            type=str,
            default=datetime.strftime(datetime.now() - relativedelta(days=1), "%Y%m%d"),
        )
        self.parser.add_argument(
            "--outputFolder", help="Output folder path", default="./output"
        )
        self.parser.add_argument(
            "--outputFormat", help="Output format (png or pdf)", default="pdf"
        )
        self.parser.add_argument(
            "--allTiers",
            action="store_true",
            help="By default the plot only takes into account T1 and T2 sites.",
            default="False",
        )


def fill_nulls(df):
    """
    This script tries to fill the gid column, replacing the -1 values.

        1. if gid is -1 is replaced with None.
        2. if all columns in the row are null, then drop the row.
        3. If the gid is none then replace it with the last gid
            for that dataset in the same site.
    """
    df_na = df.na.replace(-1, None, ["gid"]).na.drop(how="all")
    ff = df_na.withColumn(
        "gid",
        when(col("gid").isNotNull(), col("gid")).otherwise(
            last("gid", True).over(
                Window.partitionBy("site", "dataset")
                .orderBy("date")
                .rowsBetween(-sys.maxsize, 0)
            )
        ),
    )
    return ff


def merge_phedex(start_date, end_date, spark_session, base="hdfs:///cms/phedex"):
    """
    Merge the phedex datasets for the given timerange to generate a dataframe with:
    site,dateset,min_date,max_date,min_rdate,max_rdate,min_size,max_size,days
    """
    _start = datetime.strptime(start_date, "%Y%m%d")
    _end = datetime.strptime(end_date, "%Y%m%d")
    _n_days = (_end - _start).days + 1
    _dates = [
        datetime.strftime(_start + timedelta(days=d), "%Y/%m/%d")
        for d in range(0, _n_days)
    ]
    _phedex_paths = ["{}/{}/part*".format(base, d) for d in _dates]
    sc = spark_session.sparkContext
    FileSystem = sc._gateway.jvm.org.apache.hadoop.fs.FileSystem
    URI = sc._gateway.jvm.java.net.URI
    Path = sc._gateway.jvm.org.apache.hadoop.fs.Path
    fs = FileSystem.get(URI("hdfs:///"), sc._jsc.hadoopConfiguration())
    l = [url for url in _phedex_paths if fs.globStatus(Path(url))]
    schema = StructType(
        [
            StructField("date", StringType()),
            StructField("site", StringType()),
            StructField("dataset", StringType()),
            StructField("size", LongType()),
            StructField("rdate", StringType()),
            StructField("gid", IntegerType()),
        ]
    )
    _agg_fields = ["date", "size"]
    _agg_func = [_min, _max]
    # if some column will be used as date, we can add
    # .option('dateFormat','yyyyMMdd')
    _df = (
        spark_session.read.option("basePath", base)
        .option("mode", "FAILFAST")
        .option("nullValue", "null")
        .option("emptyValue", "null")
        .csv(l, schema=schema)
    )
    _df = fill_nulls(_df)
    _grouped = (
        _df.groupby("site", "dataset", "rdate", "gid").agg(
            avg("size"),
            countDistinct("date"),
            *[fn(c) for c in _agg_fields for fn in _agg_func]
        )
    ).withColumnRenamed("count(DISTINCT date)", "days")
    _grouped = _grouped.selectExpr(
        *[
            "`{}` as {}".format(c, c.replace("(", "_").replace(")", ""))
            if "(" in c
            else c
            for c in _grouped.columns
        ]
    )
    _grouped = _grouped.withColumn("avg_size", col("avg_size").astype(LongType()))
    return _grouped


# ## The weight of the dataset in the given period is a weigthed average of the size of the replicas in that period
#
def weigthed_size(min_date, max_date, begin, end):
    """
    A vectorized approach to calcule the weigth of a size in a given period.
    @param x spark dataframe
    @param begin first day of the period (as lit column).
    @param end last day of the period (as lit column).
    """
    _start = when(min_date >= begin, min_date).otherwise(begin)
    _end = when((max_date < end), max_date).otherwise(end)
    delta = datediff(
        from_unixtime(unix_timestamp(_end, "yyyyMMdd")),
        from_unixtime(unix_timestamp(_start, "yyyyMMdd")),
    ) + lit(1)
    delta = when((max_date < begin) | (min_date > end), lit(0)).otherwise(delta)
    period = datediff(
        from_unixtime(unix_timestamp(end, "yyyyMMdd")),
        from_unixtime(unix_timestamp(begin, "yyyyMMdd")),
    ) + lit(1)
    x = delta.cast(DoubleType()) / period.cast(DoubleType())
    return x


def generate_scrutiny_plot(
    end_date,
    output_format="pdf",
    output_folder="./output",
    eventsInputFolder="hdfs:///cms/dbs_events",
    basePath_dbs="hdfs:///cms/dbs_condor/dataset",
    onlyT1_T2=True,
):
    """
     param onlyT1_T2: Take into account only replicas in T1 and T2 sites.
    """
    start_date_d = datetime.strptime(end_date, "%Y%m%d") - relativedelta(
        months=12, days=-1
    )
    start_date = datetime.strftime(start_date_d, "%Y%m%d")
    midterm = datetime.strftime(start_date_d + relativedelta(months=6), "%Y%m%d")
    lastQuarter = datetime.strftime(start_date_d + relativedelta(months=9), "%Y%m%d")
    dbsInput = "{}/{{{},{}}}/*/*/part-*".format(
        basePath_dbs, start_date[:4], end_date[:4]
    )
    sc = SparkContext(appName="scrutinyPlotReplicated")
    spark = SparkSession.builder.config(conf=sc._conf).getOrCreate()
    phedex_df = merge_phedex(start_date, end_date, spark)
    if onlyT1_T2:
        phedex_df = phedex_df.filter(
            col("site").startswith("T1_") | col("site").startswith("T2_")
        )
    # ## Calculate the effective average size of each dataset in the given periods
    #  size of each dataset in each of the time periods
    phedex_df = phedex_df.withColumn(
        "weight_6Month",
        weigthed_size(
            phedex_df.min_date, phedex_df.max_date, lit(midterm), lit(end_date)
        ),
    )
    phedex_df = phedex_df.withColumn(
        "weighted_size_6Month", col("avg_size") * col("weight_6Month")
    )

    phedex_df = phedex_df.withColumn(
        "weight_3Month",
        weigthed_size(
            phedex_df.min_date, phedex_df.max_date, lit(lastQuarter), lit(end_date)
        ),
    )
    phedex_df = phedex_df.withColumn(
        "weighted_size_3Month", col("avg_size") * col("weight_3Month")
    )

    phedex_df = phedex_df.withColumn(
        "weight_12Month",
        weigthed_size(
            phedex_df.min_date, phedex_df.max_date, lit(start_date), lit(end_date)
        ),
    )
    phedex_df = phedex_df.withColumn(
        "weighted_size_12Month", col("avg_size") * col("weight_3Month")
    )

    phedex_df = phedex_df.withColumn("min_date", col("rdate"))
    _df_dsSzDur = (
        phedex_df.groupby("dataset")
        .agg(
            {
                "min_date": "min",
                "max_date": "max",
                "weighted_size_3Month": "sum",
                "weighted_size_6Month": "sum",
                "weighted_size_12Month": "sum",
            }
        )
        .toPandas()
    )
    del phedex_df
    # # Read  dbs_condor dataset
    #
    # This dataset, stored in hdfs, will be the base to determine the use of the datasets.

    dbs_df = (
        spark.read.option("basePath", basePath_dbs)
        .csv(dbsInput, header=True)
        .select("dataset", "sum_evts")
        .withColumn("filename", input_file_name())
    )
    # ## Filter the dataset
    #
    # We are only interested on records with datasets. There should be no records with dataset and without events (but currently there are).
    # Are there records with dataset but without events (empty sum_evts in the original files)?
    # - By default, spark takes empty string as null.
    # - In the current version there are rendered as the "null" string instead of null value (this will change on another versions).

    dbs_df = dbs_df.filter('dataset != "null" AND sum_evts !="null" AND sum_evts != ""')
    zero = dbs_df.filter('sum_evts = "0.0"')
    dbs_df = dbs_df.subtract(zero)
    dbs_df = dbs_df.withColumn("events", dbs_df.sum_evts.cast("double") * 1000)
    dbs_df = dbs_df.withColumn(
        "days",
        concat(
            regexp_extract(dbs_df.filename, ".*/([0-9]{4})/([0-9]{2})/([0-9]{2})", 1),
            regexp_extract(dbs_df.filename, ".*/([0-9]{4})/([0-9]{2})/([0-9]{2})", 2),
            regexp_extract(dbs_df.filename, ".*/([0-9]{4})/([0-9]{2})/([0-9]{2})", 3),
        ),
    )

    dbs_df = dbs_df.filter("days between {} AND {}".format(start_date, end_date))

    # # Use of each dataset per day

    _df_agg = (
        dbs_df.groupBy("dataset", "days").sum("events").alias("sum_events").toPandas()
    )
    _plain = _df_agg.rename(columns={"days": "day", "sum(events)": "sum_events"})

    del dbs_df
    del _df_agg
    _plain[_plain.sum_events == 0].head()
    _events_hadoop = spark.read.option("basePath", eventsInputFolder).csv(
        "{}/part*.csv".format(eventsInputFolder), header=True
    )
    _events = _events_hadoop.select("dataset", "nevents")

    df_dsSzDur = pd.merge(_df_dsSzDur, _events.toPandas(), on="dataset")

    df_dsSzDur = df_dsSzDur.rename(
        columns={
            "sum(weighted_size_12Month)": "size12month",
            "sum(weighted_size_3Month)": "size3month",
            "sum(weighted_size_6Month)": "size6month",
            "max(max_date)": "end",
            "min(min_date)": "begin",
            "nevents": "nEvents",
        }
    )

    # ## Join the datasets
    #
    # A inner join to keep only the used datasets.

    _merged = pd.merge(df_dsSzDur, _plain, on="dataset", sort=True)

    # Rate of the events used over the number of events in the file
    _merged["rate"] = _merged.sum_events / _merged.nEvents.astype(float)

    # ## Create the desired datasets.
    #
    # The datasets sixMnts, threeMnts and twelveMnts contains only data for datasets that where used at least once in the given period.

    _merged.day = _merged.day.astype("str")
    full = _merged
    sixMnts = full[full.day >= midterm][["dataset", "size6month", "day", "rate"]]
    threeMnts = full[(full.day >= lastQuarter)][
        ["dataset", "size3month", "day", "rate"]
    ]
    twelveMnts = full[["dataset", "size12month", "day", "rate"]][
        np.logical_not(np.isnan(full.rate))
    ]
    # ## Sum the event usage rate
    #

    sum_3mth = threeMnts.groupby(["dataset", "size3month"]).agg({"rate": "sum"})
    sum_6mth = sixMnts.groupby(["dataset", "size6month"]).agg({"rate": "sum"})
    sum_12mth = twelveMnts.groupby(["dataset", "size12month"]).agg({"rate": "sum"})
    types = {"3 months": sum_3mth, "6 months": sum_6mth, "full year": sum_12mth}
    cols = {
        "3 months": "size3month",
        "6 months": "size6month",
        "full year": "size12month",
    }
    bdates = {"3 months": lastQuarter, "6 months": midterm, "full year": start_date}
    gp = None
    for _type in list(types.keys()):
        _sum = types[_type].reset_index()

        # positive values <1 belong to the first bin  (one accesss).
        _sum.rate = np.where(np.logical_and(_sum.rate < 1, _sum.rate > 0), 1, _sum.rate)
        # if there are 0 or  negative values they should be in another bin (-currently there are none-).
        _sum.rate = np.where(_sum.rate <= 0, -1, _sum.rate)
        _sum["rtceil"] = np.round(_sum.rate).astype(np.int)

        _sum.rtceil = np.where(_sum.rtceil > 14, 15, _sum.rtceil)
        _sum.rtceil = _sum.rtceil.astype(str)
        _sum.rtceil = _sum.rtceil.map(lambda x: x.rjust(2, "0"))

        # Group higher values
        _sum.rtceil.values[_sum.rtceil == "15"] = "15+"

        #
        _sum.rtceil.values[_sum.rtceil == "-1"] = "-Evts"

        # agregate per bin
        _gp = _sum.groupby("rtceil").agg({cols[_type]: ["sum", "count"]})
        _gp.columns = _gp.columns.droplevel(0)
        _gp = _gp.reset_index()
        _gp.columns = ["bin", "f_size", "count_ds"]

        # Unused data:
        # Unused data is data that exists (had a size>0) in the given period but is not in the sum dataframe
        # (it was not used in the given period).
        _unused = df_dsSzDur[
            np.logical_and(
                np.logical_not(df_dsSzDur.dataset.isin(_sum.dataset)),
                df_dsSzDur[cols[_type]] > 0,
            )
        ]
        # old
        # Unused old data is unused data that existed before the period begins
        _unused_old = _unused.loc[_unused.begin.astype(str) < bdates[_type]]
        # new
        # Unused new data is unused data created on this period
        _unused_new = _unused.loc[_unused.begin.astype(str) >= bdates[_type]]

        _gp = _gp.append(
            {
                "bin": "0-old",
                "f_size": np.sum(_unused_old[cols[_type]]),
                "count_ds": np.unique(_unused_old["dataset"]).size,
            },
            ignore_index=True,
        )
        _gp = _gp.append(
            {
                "bin": "00-new",
                "f_size": np.sum(_unused_new[cols[_type]]),
                "count_ds": np.unique(_unused_new["dataset"]).size,
            },
            ignore_index=True,
        )

        # We want values in PB
        _gp.f_size = _gp.f_size * (1024 ** -5)
        # And we want the total in the legend
        _gp["type"] = "{} {:.2f} PB".format(_type, _gp.f_size.sum())

        if gp is None:
            gp = _gp
        else:
            gp = pd.concat([gp, _gp], ignore_index=True)
    values = gp.type.unique()
    values.sort()

    x_order = gp.bin.unique()
    x_order.sort()

    a4_dims = (11.7, 8.27)
    fig, ax = pyplot.subplots(figsize=a4_dims)
    sns.set_palette(sns.color_palette("Blues"))

    plot = sns.barplot(
        x="bin", y="f_size", order=x_order, hue="type", hue_order=values, data=gp, ax=ax
    )

    pyplot.xlabel("Number of accesses")
    pyplot.ylabel("Aggregated data size [PB]")
    pyplot.suptitle("Scrutiny plot (linear)")
    pyplot.title(
        """
    Number of accesses vs total size in the period ({},{})
    """.format(
            start_date, end_date
        )
    )
    plot.legend(title="Dataset Accesses over Period")
    _directory = output_folder
    if not os.path.exists(_directory):
        os.makedirs(_directory)
    _filename = os.path.join(
        _directory, "scrutiny{}-{}.{}".format(start_date, end_date, output_format)
    )
    fig.savefig(
        _filename, format=output_format,
    )
    return _filename


def main():
    """
    Main function
    """
    optmgr = OptionParser()
    opts = optmgr.parser.parse_args()
    filename = generate_scrutiny_plot(
        opts.end_date, output_format=opts.outputFormat, output_folder=opts.outputFolder
    )
    print(filename)


if __name__ == "__main__":
    main()
