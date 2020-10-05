# What is Rumble

- [Rumble](https://rumble.readthedocs.io) is a querying engine that allows you to query your large, messy datasets with ease and productivity.
- It supports **JSON-like** datasets including JSON, JSON Lines, **Parquet**, **Avro**, SVM, **CSV**, ROOT as well as text files, of any size from kB to at least the two-digit **TB range** (we have not found the limit yet).
- It runs on many local or distributed filesystems such as **HDFS**, **S3**, **Azure blob storage**, and HTTP (read-only), and of course your local drive as well. It leverages **Spark** automagically.
- The core of Rumble lies in **JSONiq**'s FLWOR expressions, the semantics of which map beautifully to DataFrames and Spark SQL

# How to start
- The interactive JSONiq tutorial that runs in your browser on a small server of Rumble team: [Try it on Colab Rumble Sandbox](https://colab.research.google.com/github/RumbleDB/rumble/blob/master/RumbleSandbox.ipynb)
- If you already have Spark on your machine, *Rumble is just a jar to download* and to use with spark-submit like you would use any other Spark application. A natural next step is thus to query files on your local drive.

***


## Useful Links

- [Getting Started Page](https://rumble.readthedocs.io/en/latest/Getting%20started/)
- [JSONiq language and its syntax](https://rumble.readthedocs.io/en/latest/JSONiq/)
- [Rumble Jar Releases Page]()
    - [Rumble Jar v1.8.0 for Spark 2.4.x, for analytix cluster](https://github.com/RumbleDB/rumble/releases/download/v1.8.0/spark-rumble-1.8.0.jar)
    - [Rumle Jar v1.8.0 for Spark 3](https://github.com/RumbleDB/rumble/releases/download/v1.8.0/spark-rumble-1.8.0-for-spark-3.jar)


---
---

# Examples

`Prerequisites`
- Runs on lxplus
- Access to ithdp-client02

### Example 1:

`Simple example of Rumble on WMArchive data in Analytix cluster`

---

##### Step by step

1. In lxplus, ssh to ithdp-client02:
```console
ssh ithdp-client02
```

2. Download Rumble jar:

```console
wget -O spark-rumble-1.8.0.jar https://github.com/RumbleDB/rumble/releases/download/v1.8.0/spark-rumble-1.8.0.jar
```

3. Create [JSONiq](https://rumble.readthedocs.io/en/latest/JSONiq/) query file:
> Simple jsoniq query which returns wmaid's of wmarchive document who has successful jobstate.
:
```js
for $doc in json-file("hdfs://analytix/project/monitoring/archive/wmarchive/raw/metric/2020/09/15/")
where $doc.data.meta_data.jobstate eq "success"
return $doc.data.wmaid
```

Create `rumble_example.jq` file
```console
cat <<< 'for $doc in json-file("hdfs://analytix/project/monitoring/archive/wmarchive/raw/metric/2020/09/15/")
where $doc.data.meta_data.jobstate eq "success"
return $doc.data.wmaid
' > rumble_example.jq
```

4. Run spark-submit job:
The result returns thousands of wmaids, so use `--result-size` parameter. And to not mess up the terminal with output, pipe output to a file:
> If your `rumble_example.jq` file in your home path, its afs path is like: `file:/afs/cern.ch/user/x/xxx/rumble_example.jq`. Spark job requires this as a requirement in analytix cluster. It takes approximately 4-5 minutes.

```console
spark-submit spark-rumble-1.8.0.jar \
--query-path file:/afs/cern.ch/user/x/xxx/rumble_example.jq \
--result-size 200000 > results.txt
```

