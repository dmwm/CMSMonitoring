# Popularity Plot

The popularity plot is created based on data available in HDFS, such as phedex dumps and dbs events data. 

There is a python script [scrutiny_plot.py](../../src/python/CMSMonitoring/scrutiny_plot.py) to create the popularity plot for a rolling year. To execute this script the enviroment need to be set up to use the analytix hadoop cluster.  The [cronPopularityPlot.sh](../../scripts/cronPopularityPlot.sh) will setup the environment in a lxplus7 like machine,  and will run the python script. 

All parameters are optional. You can run the script using:

`bash cronPopularityPlot.sh <end_date> <options>`

where `end_date` is the end of the period to be considered in format `yyyyMMdd`, if not end day is specified then it will default to *yesterday*.  The plot will be created for the period (end_date - 1y, end_date).

The options can be:

- `--outputFolder <folder>` folder to store the plots. Defaults to './output'
- `--outputFormat <png|pdf>` format to store the plots. Defaults to pdf. 

