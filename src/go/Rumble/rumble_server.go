package main

// rumble_server - Listen JSONiq requests and execute Rumble in spark cluster
//
// Copyright (c) 2021 - Ceyhun Uzunoglu <ceyhunuzngl@gmail.com>
//

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-playground/validator"
	"github.com/gofrs/flock"
	"github.com/gorilla/mux"
	"io"
	"io/fs"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Global variables
var portsInUseLockFname string

//go:embed "static"
var embeddedFiles embed.FS

// ErrorMessage Part of Response struct, provides detailed result
type ErrorMessage struct {
	Message    string `json:"message"`
	Command    string `json:"command"`
	Args       string `json:"spark_args"`
	SysError   string `json:"sys_error"`
	Stdout     string `json:"stdout"`
	VerboseMsg string `json:"verbose_stdout"`
}

// SparkConfRequest Optional part of Request, defines spark processing configurations
type SparkConfRequest struct {
	SparkExecutorMemory    string `json:"spark_executor_memory" validate:"alphanum,max=3,endswith=g"`
	SparkExecutorInstances string `json:"spark_executor_instances" validate:"number,max=3"`
	SparkExecutorCores     string `json:"spark_executor_cores" validate:"number,max=3"`
	SparkDriverMemory      string `json:"spark_driver_memory" validate:"alphanum,max=3,endswith=g"`
}

// Request user request which includes JSONiq query and other parameters
type Request struct {
	Query      string           `json:"query" validate:"required,min=3"`
	OutputPath string           `json:"output_path" validate:"required,min=3"`
	UserId     string           `json:"userid" validate:"required,min=3"`
	Verbose    int              `json:"verbose" validate:"oneof=0 1 2"`
	SparkConf  SparkConfRequest `json:"spark_conf"`
}

// Response provides query result and useful information like spark application id
type Response struct {
	UserId            string       `json:"userid" `
	SparkAppId        string       `json:"spark_app_id"`
	OutputPath        string       `json:"output_path"`
	ErrorMsg          ErrorMessage `json:"details"`
	InternalQueryFile string       `json:"internal_query_file"`
}

// runRumble
func runRumble(port1 string,
	port2 string,
	queryFile string,
	outputPath string,
	userId string,
	verbose int,
	sparkConf *SparkConfRequest) (Response, bool) {
	args := []string{
		"--conf", "spark.executor.memory=" + sparkConf.SparkExecutorMemory,
		"--conf", "spark.executor.instances=" + sparkConf.SparkExecutorInstances,
		"--conf", "spark.executor.cores=" + sparkConf.SparkExecutorCores,
		"--conf", "spark.driver.memory=" + sparkConf.SparkDriverMemory,
		"--conf", "spark.yarn.maxAppAttempts=2", // Set max retry to 2.
		"--conf", "spark.driver.bindAddress=0.0.0.0",
		"--conf", "spark.driver.host=" + os.Getenv("MY_NODE_NAME"),
		"--conf", fmt.Sprintf("spark.driver.port=%s", port1),
		"--conf", fmt.Sprintf("spark.driver.blockManager.port=%s", port2),
		os.Getenv("RUMBLE_JAR_FILE"),                      //"/data/spark-rumble.jar"
		"--query-path", fmt.Sprintf("file:%s", queryFile), // i.e. /data/queries/test.jq
		"--output-format", "json",
		"--materialization-cap", "-1",
		"--show-error-info", "yes",
		"--overwrite", "yes",
	}
	// Check output file path
	outputPath = sanitizeOutputPath(outputPath)
	log.Println("[INFO] HDFS output path:", outputPath)
	args = append(args, "--output-path", outputPath)

	command := "spark-submit"
	cmd := exec.Command(command, args...)
	log.Println("[INFO] Verbosity:", verbose)

	// For Debug purposes, does not returns any result.
	if verbose > 2 {
		log.Println("[----------- DEBUG MODE -----------]")
		stdout, _ := cmd.StdoutPipe()
		stderr, _ := cmd.StderrPipe()
		err := cmd.Start()
		if err != nil {
			log.Println(err)
			return Response{}, false // Do not return any response
		}
		_, err = io.Copy(os.Stdout, stdout)
		_, err = io.Copy(os.Stdout, stderr)

		err = cmd.Wait()
		if err != nil {
			log.Println(err)
			return Response{}, false // Do not return any response
		}
		return Response{}, true // Do not return any response
	} else {
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		responseErrMsg := ErrorMessage{
			Message: "",
			Command: command,
			Args:    fmt.Sprintf("%v", args),
		}
		log.Printf("[INFO] Spark-submit will start with these settings: %v", responseErrMsg)
		err := cmd.Run()
		responseErrMsg.SysError = fmt.Sprintf("%v", err)
		responseErrMsg.Stdout = out.String()
		appId := parseAppId(out.String())
		log.Println("[INFO] Stdout:", out.String())
		log.Println("[DEBUG] Stderr:", stderr.String())
		if verbose > 1 { // If user sets verbose greater than 1, send all errors.
			responseErrMsg.VerboseMsg = stderr.String()
		}
		if err != nil {
			responseErrMsg.Message = "Rumble execution failed. Please check your query! "
			log.Printf("[ERROR] Spark-submit failed: %v", responseErrMsg)
			return Response{
				ErrorMsg:   responseErrMsg,
				SparkAppId: appId,
			}, false
		}
		if verbose > 0 {
			responseErrMsg.Message = "Rumble execution SUCCESSFUL. You can reach your data from: " + outputPath
			return Response{
				UserId:            userId,
				SparkAppId:        appId,
				ErrorMsg:          responseErrMsg,
				OutputPath:        outputPath,
				InternalQueryFile: queryFile,
			}, true
		}
		return Response{
			UserId:     userId,
			SparkAppId: appId,
			OutputPath: outputPath,
			ErrorMsg: ErrorMessage{
				Message: "Rumble execution SUCCESSFUL. You can reach your data from: " + outputPath,
			},
		}, true
	}
}

func parseAppId(out string) string {
	flagStart, flagEnd := "ApplicationID:[", "]APP_ID_FLAG"
	if strings.Contains(out, flagStart) {
		return strings.Split(strings.Split(out, flagStart)[1], flagEnd)[0]
	}
	return ""
}

func sanitizeOutputPath(file string) string {
	checkStr := []string{"hdfs:///analytix", "hdfs://analytix", "hdfs://", "hdfs:/", "hdfs:", "hdfs"}
	for _, substr := range checkStr {
		if strings.Contains(file, substr) {
			file = strings.Replace(file, substr, "", 1)
			break
		}
	}
	if !strings.HasPrefix(file, "/") {
		return "hdfs://analytix/" + file
	}
	return "hdfs://analytix" + file
}

// writeQueryToFile Write JSONiq query to a file with unix timestamp.
func writeQueryToFile(query string, userId string) (string, error) {
	baseQueryPath := "/data/queries/"
	// Create if it does not exist, else do nothing
	if err := os.MkdirAll(baseQueryPath, os.ModeDir); err != nil {
		log.Println("[ERROR] could not create temp file, reason:", err)
		return "", err
	}
	userId = strings.ReplaceAll(userId, " ", "") // Clear whitespace
	jsoniqFileName := fmt.Sprintf("%s%s-%d.jq", baseQueryPath, userId, time.Now().UnixNano())
	if err := ioutil.WriteFile(jsoniqFileName, []byte(query), 0666); err != nil {
		log.Println("[ERROR] could not create temp file, reason:", err)
		return "", err
	}
	return jsoniqFileName, nil
}

func writeMapToFile(portsInUse map[string]bool, lockFname string) error {
	data, err := json.Marshal(portsInUse)
	if err != nil {
		log.Println("[ERROR] unable to write ports map to file, reason:", err)
		return errors.New("unable to write ports map to file")
	}
	if err := ioutil.WriteFile(lockFname, data, 0644); err != nil {
		log.Println("[ERROR] unable to write ports map to file, reason:", err)
		return errors.New("unable to write ports map to file")
	}
	return nil
}

func readMapFromFile(lockFname string) map[string]bool {
	file, _ := ioutil.ReadFile(lockFname)
	data := map[string]bool{}
	if err := json.Unmarshal(file, &data); err != nil {
		log.Fatal("[ERROR]: could not read json file")
	}
	return data
}

func initializePortsInUse() string {
	portsInUse := make(map[string]bool)
	lockFile, err := ioutil.TempFile(os.TempDir(), "rumble-ports-*.json")
	if err != nil {
		log.Fatalln("[ERROR] unable to create rumble-ports lock file, reason:", err)
	}
	for _, v := range os.Environ() {
		if strings.Contains(v, "RUMBLE_SERVICE_PORT_PORT_") {
			kv := strings.Split(v, "=")
			portsInUse[kv[1]] = false
		}
	}
	if len(portsInUse) < 2 {
		log.Fatal("[ERROR] no ports in RUMBLE_SERVICE_PORT_PORT_N format.")
	}
	if err = writeMapToFile(portsInUse, lockFile.Name()); err != nil {
		log.Fatalln("[ERROR]", err)
	}
	log.Println("[INFO] total number of spark-submit ports :", len(portsInUse))
	log.Printf("[INFO] initial ports in use: %v", fmt.Sprint(portsInUse))
	log.Println("[INFO] rumble-ports lock file name:", lockFile.Name())
	return lockFile.Name()
}

func releasePorts(port1 string, port2 string, lockFname string) error {
	fileLock := flock.New(lockFname)
	var locked bool
	var lockerr error
	for i := 1; i <= 10; i++ { // Try to get lock with 5 seconds
		locked, lockerr = fileLock.TryLock()
		if lockerr == nil && locked {
			fmt.Println("[INFO]: got the lock")
			portsInUse := readMapFromFile(lockFname)
			portsInUse[port1], portsInUse[port2] = false, false
			if err := writeMapToFile(portsInUse, lockFname); err != nil {
				_ = fileLock.Unlock()
				return err
			}
			_ = fileLock.Unlock()
			return nil
		}
		time.Sleep(1 * time.Second)
	}
	return errors.New("could not get the lock")

}

// Check available ports and return if there are any. 2 ports for per spark-submit.
func reservePorts(lockFname string) (string, string, error) {
	fileLock := flock.New(lockFname)
	var locked bool
	var lockerr error
	var port1, port2 string
	for i := 1; i <= 5; i++ { // Try to get lock with 5 seconds
		locked, lockerr = fileLock.TryLock()
		if lockerr == nil && locked {
			fmt.Println("[INFO]]: got the lock")
			portsInUse := readMapFromFile(lockFname)
			for key, value := range portsInUse {
				if key != "" && value == false {
					port1 = key
				}
			}
			for key, value := range portsInUse {
				if key != "" && key != port1 && value == false {
					port2 = key
				}
			}
			if port1 == "" || port2 == "" {
				_ = fileLock.Unlock()
				return "", "", errors.New("no available ports")
			}
			portsInUse[port1], portsInUse[port2] = true, true
			if err := writeMapToFile(portsInUse, lockFname); err != nil {
				_ = fileLock.Unlock()
				return "", "", err
			}
			_ = fileLock.Unlock()
			return port1, port2, nil
		}
		time.Sleep(1 * time.Second)
	}
	log.Println("[ERROR]: could not get lock!")
	return "", "", errors.New("could not get lock")
}

func processRumbleServerRequest(w http.ResponseWriter, request *Request, verbose int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Accept", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	jsoniqFileName, err := writeQueryToFile(request.Query, request.UserId)
	if err != nil {
		log.Println("[ERROR] could not create temp JSONiq file, reason:", err)
		w.WriteHeader(http.StatusBadRequest)
		if encodeErr := json.NewEncoder(w).Encode(Response{
			ErrorMsg: ErrorMessage{
				Message:  "Internal error. Please try again!",
				SysError: fmt.Sprintf("%v", err),
			},
		}); encodeErr != nil {
			log.Println("[ERROR] could not marshal response, reason:", encodeErr)
		}
		return
	}
	port1, port2, err := reservePorts(portsInUseLockFname)
	if err != nil {
		log.Println("[ERROR]", err)
		w.WriteHeader(http.StatusTooManyRequests)
		if encodeErr := json.NewEncoder(w).Encode(Response{
			ErrorMsg: ErrorMessage{
				Message:  "All workers are occupied! Please try again!",
				SysError: fmt.Sprintf("%v", err),
			},
		}); encodeErr != nil {
			log.Println("[ERROR] could not marshal response, reason:", encodeErr)
		}
		return
	}
	log.Printf("[INFO] reserved ports for current request: [%v] - [%v]", port1, port2)
	if request.Verbose > 0 {
		verbose = request.Verbose
	}
	log.Println("[INFO] verbosity:", verbose)
	userId := strings.ReplaceAll(request.UserId, " ", "") // Clear whitespace
	response, ok := runRumble(port1, port2, jsoniqFileName, request.OutputPath, userId, verbose, &request.SparkConf)
	log.Printf("[INFO] response: %#v", response)
	// Release reserved ports
	if err := releasePorts(port1, port2, portsInUseLockFname); err != nil {
		log.Println("[ERROR] will exit, ports could not released", port1, port2)
		log.Fatalln("[ERROR]: ", err)
	}
	log.Println("[INFO] ports released", port1, port2)
	if !ok {
		log.Println("[INFO] FAIL")
		w.WriteHeader(http.StatusBadRequest)
		if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
			log.Println("[ERROR] could not marshal response, reason:", encodeErr)
		}
		return
	}
	log.Println("[INFO] SUCCESS")
	w.WriteHeader(http.StatusOK)
	if encodeErr := json.NewEncoder(w).Encode(response); encodeErr != nil {
		log.Println("[ERROR] could not marshal response, reason:", encodeErr)
	}
}

// RumbleServerRequestHandler we should only receive POST request
func RumbleServerRequestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Accept", "application/json")
	w.Header().Set("Content-Type", "application/json")
	var request Request
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&request)
	log.Printf("[INFO] Request - Method: %v, Host: %v, RemoteAddr: %v, Args: %v, URI: %v",
		r.Method, r.Host, r.RemoteAddr, r.URL, r.RequestURI)
	if err != nil {
		tempBody, readErr := ioutil.ReadAll(r.Body)
		if readErr != nil {
			log.Printf("[ERROR] could not get and even read request body, reasons: %#v, %#v", err, readErr)
		}
		log.Printf("[ERROR] could not get request body, reason: %v, Body:%#v", err, tempBody)
		w.WriteHeader(http.StatusBadRequest)
		if encodeErr := json.NewEncoder(w).Encode(Response{
			ErrorMsg: ErrorMessage{
				Message:  "Please check your query, json could not be decoded!",
				SysError: fmt.Sprintf("%v", err),
			},
		}); encodeErr != nil {
			log.Println("[ERROR] could not marshal response, reasons:", encodeErr)
		}
		return
	}
	log.Printf("[INFO] Request Body:%#v", request)

	validate := validator.New()
	if validationErr := validate.Struct(request); validationErr != nil {
		log.Println("[ERROR] request body could not be validated", validationErr)
		w.WriteHeader(http.StatusBadRequest)
		if encodeErr := json.NewEncoder(w).Encode(Response{
			ErrorMsg: ErrorMessage{
				Message:    "Please check your request, given values are not valid!",
				SysError:   fmt.Sprintf("%v", err),
				VerboseMsg: fmt.Sprintf("%v", validationErr),
			},
		}); encodeErr != nil {
			log.Println("[ERROR] could not marshal response, reasons:", encodeErr)
		}
		return
	}

	// Set defaults
	verbose := 0
	if request.SparkConf.SparkExecutorMemory == "" {
		request.SparkConf.SparkExecutorMemory = "2g"
	}
	if request.SparkConf.SparkExecutorInstances == "" {
		request.SparkConf.SparkExecutorInstances = "2"
	}
	if request.SparkConf.SparkExecutorCores == "" {
		request.SparkConf.SparkExecutorCores = "2"
	}
	if request.SparkConf.SparkDriverMemory == "" {
		request.SparkConf.SparkDriverMemory = "2g"
	}
	processRumbleServerRequest(w, &request, verbose)
}

func handlers() *mux.Router {
	router := mux.NewRouter()
	fsys, err := fs.Sub(embeddedFiles, "static")
	if err != nil {
		log.Fatalln("[ERROR] embedded files could not initialized, reason:", err)
	}
	router.Handle("/", http.FileServer(http.FS(fsys)))
	router.HandleFunc("/rumble-server", RumbleServerRequestHandler).Methods("POST", "OPTIONS")
	return router
}

func main() {
	portsInUseLockFname = initializePortsInUse()
	if os.Getenv("MY_NODE_NAME") == "" || os.Getenv("RUMBLE_JAR_FILE") == "" {
		log.Fatal("[ERROR] MY_NODE_NAME and RUMBLE_JAR_FILE env variables are not defined.")
	}
	http.Handle("/", handlers())
	log.Fatal(http.ListenAndServe(":8080", nil))
}
