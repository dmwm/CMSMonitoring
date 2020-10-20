# MONIT go client query examples

## 1. How to use
1. You can use monit tool directly from `/cvmfs/cms.cern.ch/cmsmon/monit --help`
2. You can build your own monit go tool.
    > *If golang does not exist in your platform:*
    > * Download golang tar ball (for your platform) from [golang.org](https://golang.org/) 
    > * Unpack and setup PATH and GOPATH. The PATH should include path to go executable, e.g. /usr/local/bin or whatever path you'll choose on your system, and GOPATH can be any area on your node which will contain all go dependencies installed in your user space. 
    > * For instance, you may create your gopath in your home area: <br /> `mkdir -p $HOME/gopath` <br /> and then setup GOPATH points to it, e.g.: <br />`export GOPATH=$HOME/gopath`

    - Firstly, download requirements: 
    
        `go get github.com/go-stomp/stomp && go get github.com/vkuznet/x509proxy`
    - Now you can buil your own monit go tool:
        
        `git clone https://github.com/dmwm/CMSMonitoring.git && cd CMSMonitoring/src/go/MONIT`
        
        `go build monit.go`
    - You can use the tool in current path like:
        
        `monit --help`

> Get your GRAFANA API token from [cms-monit](https://its.cern.ch/jira/projects/CMSMONIT)

> You may add your token to env vars: `export GRAFANA_TOKEN="<Grafana-Api-Token>"` 

---


## Elasticsearch query examples


> Keeping queries in files is a desired way. Elasticsearch queries can be long and string query may complicate things.


1. **Query1 in pseudocode:** Get top 10 IPs in last 2 days from "monit-timber-cmswebk8s-apache" where (http response code is 400 or 404 and greater than 99) AND (metadata type is frontend) AND (cmsweb environment is k8s-prod) AND (cmsweb cluster is cmsweb) Ordered By descending.
    - Create the query file
        
        `cat query1.json`

        ```json
        {"size":0,"query":{"bool":{"filter":[{"range":{"data.rec_date":{"gte":"now-2d/d","lte":"now/d","format":"epoch_millis"}}},{"query_string":{"analyze_wildcard":true,"query":"data.code:(\"400\" OR \"404\") AND data.system:/.*/ AND metadata.type:(\"frontend\") AND data.cmsweb_env:/k8s-prod/ AND data.cmsweb_cluster:/cmsweb/"}},{"range":{"data.code":{"gt":"99"}}}]}},"aggs":{"clients_agg":{"terms":{"field":"data.clientip","size":10,"order":{"_count":"desc"},"min_doc_count":1}}}}
        ```
    - Run monit tool

        `monit -token $GRAFANA_TOKEN -query=query1.json -dbname=monit-timber-cmswebk8s-apache`
    - Response 

        ```json
        {"responses":[{"_shards":{"failed":0,"skipped":0,"successful":30,"total":30},"aggregations":{"clients_agg":{"buckets":[{"doc_count":128,"key":"10.100.246.192"},{"doc_count":120,"key":"10.100.249.0"},{"doc_count":80,"key":"137.138.122.135"},{"doc_count":64,"key":"137.138.120.114"},{"doc_count":56,"key":"10.100.109.64"},{"doc_count":48,"key":"10.100.165.192"},{"doc_count":24,"key":"10.100.250.128"},{"doc_count":16,"key":"137.138.45.225"},{"doc_count":16,"key":"137.138.77.53"},{"doc_count":16,"key":"188.184.103.188"}],"doc_count_error_upper_bound":0,"sum_other_doc_count":0}},"hits":{"hits":[],"max_score":0,"total":568},"status":200,"timed_out":false,"took":7}]}
        ```

2. **Query2 in pseudocode:** Get sum of total failed jobs in last 1 day from monit_es_condor where (Status of job is "Completed") and (for all ScheddNames) Group By "CMS Campaign types" Order By descending Limit 50
    - Create the query file
        
        `cat query2.json`

        ```json
        {"size":0,"query":{"bool":{"filter":[{"range":{"data.RecordTime":{"gte":"now-1d/d","lte":"now/d","format":"epoch_millis"}}},{"query_string":{"analyze_wildcard":true,"query":"data.Status:\"Completed\" AND data.ScheddName:*"}}]}},"aggs":{"campaign_type_agg":{"terms":{"field":"data.CMS_CampaignType","size":50,"order":{"_key":"asc"},"min_doc_count":1,"missing":"0"},"aggs":{"sum_failed_job_agg":{"sum":{"field":"data.JobFailed"}}}}}}
        ```
    - Run monit tool

        `monit -token $GRAFANA_TOKEN -query=query2.json -dbname=monit_es_condor`
    - Response 

        ```json
        {"responses":[{"_shards":{"failed":0,"skipped":624,"successful":656,"total":656},"aggregations":{"campaign_type_agg":{"buckets":[{"doc_count":571981,"key":"Analysis","sum_failed_job_agg":{"value":53949}},{"doc_count":229528,"key":"MC Ultralegacy","sum_failed_job_agg":{"value":50144}},{"doc_count":4357,"key":"Phase2 requests","sum_failed_job_agg":{"value":38}},{"doc_count":33,"key":"RelVal","sum_failed_job_agg":{"value":0}},{"doc_count":55340,"key":"Run2 requests","sum_failed_job_agg":{"value":10643}},{"doc_count":452,"key":"Run3 requests","sum_failed_job_agg":{"value":13}},{"doc_count":33784,"key":"UNKNOWN","sum_failed_job_agg":{"value":1850}}],"doc_count_error_upper_bound":0,"sum_other_doc_count":0}},"hits":{"hits":[],"max_score":null,"total":{"relation":"gte","value":10000}},"status":200,"timed_out":false,"took":150}],"took":150}
        ```

> For advance queries, we would be very happy to help you : [cms-monit](https://its.cern.ch/jira/projects/CMSMONIT) 