# Installation

- Table of contents
  * [Setup](#setup)
  * [Build](#build)
    - [Main](#main)
    - [Test](#test)
  * [Run](#run)
  * [Test](#test-1)
    - [Manual](#manual)
    - [Automation](#automation)

## Setup

If you want the compiled binary to reside in CMSMonitoring/bin then set the $GOPATH at /CMSMonitoring by this command.

```  export GOPATH=<WORK_DIR>/CMSMonitoring ```

You may also want to to include following into the PATH variable.

```$GOPATH/bin```

However, You can chose any other path anytime by providing path to $GOPATH, setting the PATH variable.

## Build

#### Main
To build the intelligence module run the following command. Upon completion of build the binary file will reside in $GOPATH/bin.

`go build go/intelligence`

#### Test

Build the test binary residing [here](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/intelligence/int_test/test.go) by running :-

`go build go/intelligence/int_test`

## Run
```
Usage: intelligence [options]
  -config string
    	Config File path
  -iter int
    	Custom defined no. of iterations for premature termination
  -verbose int
    	Verbosity Level, can be overwritten in config
```
As $GOPATH/bin has already been set in PATH variable, you can run the intelligence module binary by executing the command below. Config file path flag (-config) is mandatory. However, -verbose and -iter flags are optional.  

`intelligence -config <path-to-config-file>`

## Test
[/src/go/intelligence/int_test](https://github.com/dmwm/CMSMonitoring/tree/master/src/go/intelligence/int_test)

For testing purpose of intelligence module, we have provide following :-
- [test.go](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/intelligence/int_test/test.go) - testing module 
- [test_cases.json](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/intelligence/int_test/test_cases.json) - Fake alerts mimicing SSB and GGUS alerts for testing purpose.
- [test_config.json](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/intelligence/int_test/test_config.json) - dedicated config file for testing (similar to the main config file with few changes).
- [test_wrapper.sh](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/intelligence/int_test/test_wrapper.sh) - Bash script for automating the testing process.

Testing can be done in two ways :-
- Manual
    - You run an instance of Alertmanager manually and then run the test binary with required values in [test_config.json](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/intelligence/int_test/test_config.json).
- Automation
    - You just run the [test_wrapper.sh](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/intelligence/int_test/test_wrapper.sh) bash script which will automate all process from environment setup to running the test.

Testing config changes and points to consider :- 
- CMSMON url must be changed to the url of Testing instance of AlertManager
- "testfile" can take two values in different scenario. 
    - Manual -> <WORK_DIR>/CMSMonitoring/src/go/intelligence/int_test/test_cases.json
    - Automation -> default value is /tmp/test_cases.json if you are testing in /tmp/<your_name> directory. It can be configured though.

Let's see how we can test in two scenarios. 

##### Manual

It is expected that a testing instance of Alertmanager is running in the system. On running the command below the testing of the whole pipeline starts. It pushes test alerts to Alertmanager, changes their severity value, annotates the Grafana dashboards, silences unwanted alerts and at the end outputs the test results.

`int_test -config  <path-to-test-config-file>`

##### Automation

All test scenario required for intelligence module testing has been automated using the [test_wrapper.sh](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/intelligence/int_test/test_wrapper.sh). 

```
 Script for automation of testing process of the intelligence module.
 Usage: test_wrapper.sh <config-file-path> <wdir>  

 config  test config file path      (mandatory)
 wdir    work directory             default: /tmp/$USER

 Options:
 help    help manual
```

Follow the following steps :-
1) Clone the repository at a specific directory

```$ git clone https://github.com/dmwm/CMSMonitoring.git```

2) Set the PATH variable

```$ export PATH=/<PATHTOCMSMonitoring>/src/go/intelligence/int_test/:$PATH```

3) Edit [CMSMonitoring/src/go/intelligence/int_test/test_config.json](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/intelligence/int_test/test_config.json) and set "testfile" to: <WORK_DIR>/test_cases.json 

where WORKDIR is the directory where you want to run the test (if you use the default WORK_DIR  /tmp/$USER, change "testfile" to : /tmp/$USER/test_cases.json.
If different WORK_DIR, then change "testfile" to: <WORK_DIR>/test_cases.json).

4) Run [test_wrapper.sh](https://github.com/dmwm/CMSMonitoring/blob/master/src/go/intelligence/int_test/test_wrapper.sh) at default directory (i.e. /tmp/$USER)

```$ test_wrapper.sh <test-config-file-path>``` 

Run this command for testing at different directory. 

```$ test_wrapper.sh <test-config-file-path> <WORK_DIR>``` 

##### *For LXPLUS USERS* - If you want to test on lxplus VM, you don't need to deploy alerting services (GGUS & SSB). There are some fake alerts which are similar to GGUS and SSB ticketing services which are pushed into Alertmanager before starting the test. However, you can run alerting services to include realtime alerts in testing process too. Wondering how to run alerting services ? Go [here](https://github.com/dmwm/CMSMonitoring/blob/master/doc/AlertManagement/installation.md).
