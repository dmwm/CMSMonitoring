//package main
//
//rumble_client - CLI client for rumble
//
//Copyright (c) 2021 - Ceyhun Uzunoglu <ceyhunuzngl@gmail.com>
//
//
//import (
//   "bytes"
//   "encoding/json"
//   "flag"
//   "fmt"
//   "io/ioutil"
//   "log"
//   "net/http"
//   "os"
//   "strconv"
//   "time"
//)
//
//type SparkConfRequest struct {
//   SparkExecutorMemory    string `json:"spark_executor_memory"`
//   SparkExecutorInstances string `json:"spark_executor_instances"`
//   SparkExecutorCores     string `json:"spark_executor_cores"`
//   SparkDriverMemory      string `json:"spark_driver_memory"`
//}
//
//type Request struct {
//   Query      string           `json:"query"`
//   OutputPath string           `json:"output_path"`
//   UserId     string           `json:"userid"`
//   Verbose    int              `json:"verbose"`
//   SparkConf  SparkConfRequest `json:"spark_conf"`
//}
//
//// getQuery: helper function to either read file content.
//func getQuery(r string) string {
//   if _, err := os.Stat(r); err != nil {
//       log.Println("[INFO] Query string is provided.")
//       return r
//   }
//   b, e := ioutil.ReadFile(r)
//   if e != nil {
//       log.Fatalf("[ERROR] Unable to read data from file: %s, error: %s", r, e)
//   }
//   return string(b)
//}
//
//func sendRequest(server string, verbose int, userId string, output string, query string, timeout int,
//   sparkExecutorMemory int, sparkExecutorInstances int, sparkExecutorCores int, sparkDriverMemory int) {
//   var req *http.Request
//   queryStr := getQuery(query)
//   payload := Request{UserId: userId, OutputPath: output, Query: queryStr, Verbose: verbose,
//       SparkConf: SparkConfRequest{
//           SparkExecutorMemory:    fmt.Sprintf("%dg", sparkExecutorMemory),
//           SparkExecutorInstances: strconv.Itoa(sparkExecutorInstances),
//           SparkExecutorCores:     strconv.Itoa(sparkExecutorCores),
//           SparkDriverMemory:      fmt.Sprintf("%dg", sparkDriverMemory),
//       }}
//   data, e := json.Marshal(payload)
//   log.Println("[INFO] Rumble server:", server)
//   log.Print("[INFO] JSON payload:\n", string(data), "\n")
//   if e != nil {
//       log.Fatalln("[ERROR] Unable to marshall request.")
//       return
//   }
//   req, e = http.NewRequest("POST", server, bytes.NewBuffer(data))
//   if e != nil {
//       log.Fatalf("Unable to make request to %s, error: %s", server, e)
//   }
//   req.Header.Set("Accept", "application/json")
//   req.Header.Set("Content-Type", "application/json")
//
//   client := &http.Client{Timeout: time.Minute * time.Duration(timeout)}
//   resp, e := client.Do(req)
//   if e != nil {
//       log.Fatalf("Unable to get response from %s, error: %s", server, e)
//   }
//   defer resp.Body.Close()
//
//   log.Println("Response Status:", resp.Status)
//   if verbose > 0 {
//       log.Println("Response Headers:", resp.Header)
//   }
//   if resp.StatusCode == http.StatusOK {
//       log.Println(resp) // Print response
//   }
//}
//
//func main() {
//   url := "http://cuzunogl-k8s-yjc6wuzsnjes-node-0:30000/rumble-server"
//   var server string
//   flag.StringVar(&server, "server", url, "Rumble server URL")
//   var verbose int
//   flag.IntVar(&verbose, "verbose", 0, "verbosity level")
//   var userId string
//   flag.StringVar(&userId, "user", "", "userId")
//   var output string
//   flag.StringVar(&output, "output", "", "HDFS output path.")
//   var query string
//   flag.StringVar(&query, "query", "", "JSONiq query string or JSONiq query file")
//   var timeout int
//   flag.IntVar(&timeout, "timeout", 10, "timeout minutes, default is 10 minutes.")
//   var sparkExecutorMemory int
//   flag.IntVar(&sparkExecutorMemory, "executor-memory-g", 0, "-executor-memory-g 8")
//   var sparkExecutorInstances int
//   flag.IntVar(&sparkExecutorInstances, "executor-instances", 0, "-executor-instances 4")
//   var sparkExecutorCores int
//   flag.IntVar(&sparkExecutorCores, "executor-cores", 0, "-executor-cores 2")
//   var sparkDriverMemory int
//   flag.IntVar(&sparkDriverMemory, "driver-memory-g", 0, "-driver-memory-g 2")
//
//   flag.Usage = func() {
//       fmt.Println("Usage: rumble_client [options]")
//       fmt.Println("   # \"output\", \"user\" and \"query\" are mandatory fields.")
//       fmt.Println("   # \"output\" should be a public HDFS path.")
//       flag.PrintDefaults()
//       fmt.Println("Examples:")
//       fmt.Println("   # simple query")
//       fmt.Println("   rumble_client -user john-doe -output /tmp/rumble/userid/123 -query=\"1+1\"")
//       fmt.Println("")
//       fmt.Println("   # query with all parameters")
//       fmt.Println("   rumble_client -user john-doe -output /tmp/rumble/userid/123 -query=./test.jq " +
//           "-verbose 1 -timeout 5 -executor-memory-g 2 -executor-instances 4 -executor-cores 2 -driver-memory-g 2")
//       fmt.Println("")
//   }
//   flag.Parse()
//   if verbose > 0 {
//       log.SetFlags(log.LstdFlags | log.Lshortfile)
//   } else {
//       log.SetFlags(log.LstdFlags)
//   }
//   sendRequest(server, verbose, userId, output, query, timeout, sparkExecutorMemory, sparkExecutorInstances, sparkExecutorCores, sparkDriverMemory)
//}
//
