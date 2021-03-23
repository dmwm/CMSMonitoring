package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Record represent hdfs record
type Record struct {
	Size      int64
	Timestamp int64
	Path      string
}

func run(path, pattern string, timeoffset, verbose int) ([]Record, error) {
	var records []Record
	args := []string{"fs", "-ls"}
	command := "hadoop"
	now := time.Now().Unix() - int64(timeoffset)
	tnow := time.Unix(now, 0)
	sdate := fmt.Sprintf("%d/%02d/%02d", tnow.Year(), tnow.Month(), tnow.Day())
	hpath := fmt.Sprintf("%s/%s", path, sdate)
	if verbose > 0 {
		log.Println("look-up hdfs", hpath)
	}
	args = append(args, hpath)
	cmd := exec.Command(command, args...)
	stdout, err := cmd.Output()
	if err != nil {
		msg := fmt.Sprintf("%v %v %v", command, args, err)
		log.Println(msg)
		return records, errors.New(msg)
	}
	if verbose > 0 {
		log.Println(string(stdout))
	}
	for _, line := range strings.Split(string(stdout), "\n") {
		var arr []string
		for _, a := range strings.Split(line, " ") {
			if a != "" {
				arr = append(arr, a)
			}
		}
		if len(arr) < 8 {
			if verbose > 0 {
				log.Printf("output: '%s' length %d\n", line, len(arr))
			}
		} else {
			size, err := strconv.ParseInt(fmt.Sprintf("%v", arr[4]), 10, 64)
			if err != nil {
				return records, err
			}
			tdate, err := time.Parse("2006-01-02 15:04", fmt.Sprintf("%s %s", arr[5], arr[6]))
			if err != nil {
				return records, err
			}
			path := fmt.Sprintf("%s", arr[7])
			records = append(records, Record{Size: size, Path: path, Timestamp: tdate.Unix()})
		}
	}
	if strings.Contains(string(stdout), pattern) {
		return records, nil
	}
	return records, nil
}

// Exporter represents Prometheus exporter structure
type Exporter struct {
	Path           string
	Pattern        string
	Timeoffset     int
	Verbose        int
	mutex          sync.Mutex
	scrapeFailures prometheus.Counter
	size           *prometheus.Desc
	timestamp      *prometheus.Desc
	//     path           *prometheus.Desc
}

func NewExporter(ns, path, pattern string, timeoffset, verbose int) *Exporter {
	var labels = []string{"path"}
	return &Exporter{
		Path:           path,
		Pattern:        pattern,
		Timeoffset:     timeoffset,
		Verbose:        verbose,
		scrapeFailures: prometheus.NewCounter(prometheus.CounterOpts{}),
		size: prometheus.NewDesc(
			prometheus.BuildFQName(ns, "", "size"),
			fmt.Sprintf("Size of the record on path %s", path),
			labels,
			nil),
		timestamp: prometheus.NewDesc(
			prometheus.BuildFQName(ns, "", "timestamp"),
			fmt.Sprintf("Timestamp of the record at path: %s", path),
			labels,
			nil),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.size
	ch <- e.timestamp
	//     ch <- e.path
}

// Collect performs metrics collectio of exporter attributes
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock() // To protect metrics from concurrent collects.
	defer e.mutex.Unlock()
	if err := e.collect(ch); err != nil {
		log.Printf("Error scraping: %s\n", err)
		e.scrapeFailures.Inc()
		e.scrapeFailures.Collect(ch)
	}
	return
}

// helper function which collects exporter attributes
func (e *Exporter) collect(ch chan<- prometheus.Metric) error {

	// extract records
	records, err := run(e.Path, e.Pattern, e.Timeoffset, e.Verbose)
	if err != nil {
		//         ch <- prometheus.MustNewConstMetric(e.size, prometheus.CounterValue, 0)
		//         ch <- prometheus.MustNewConstMetric(e.timestamp, prometheus.CounterValue, 0)
		return err
	}

	for _, r := range records {
		size := float64(r.Size)
		timestamp := r.Timestamp
		labels := []string{r.Path}
		ch <- prometheus.MustNewConstMetric(e.size, prometheus.CounterValue, float64(size), labels...)
		ch <- prometheus.MustNewConstMetric(e.timestamp, prometheus.CounterValue, float64(timestamp), labels...)
	}
	return nil
}

func main() {
	var timeoffset int
	flag.IntVar(&timeoffset, "timeoffset", 0, "time offset in seconds to look at")
	var path string
	flag.StringVar(&path, "path", "", "HDFS path")
	var pattern string
	flag.StringVar(&pattern, "pattern", "", "HDFS pattern to watch")
	var namespace string
	flag.StringVar(&namespace, "namespace", "hdfs", "namespace to use for exporter")
	var verbose int
	flag.IntVar(&verbose, "verbose", 0, "verbose output")
	var listeningAddress string
	flag.StringVar(&listeningAddress, "address", ":18000", "address of web interface.")
	var endpoint string
	flag.StringVar(&endpoint, "endpoint", "/metrics", "Path under which to expose metrics.")
	flag.Parse()
	// log time, filename, and line number
	if verbose > 0 {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}

	exporter := NewExporter(namespace, path, pattern, timeoffset, verbose)
	prometheus.MustRegister(exporter)

	log.Printf("Starting Server: %s\n", listeningAddress)
	http.Handle(endpoint, promhttp.Handler())
	log.Fatal(http.ListenAndServe(listeningAddress, nil))
}
