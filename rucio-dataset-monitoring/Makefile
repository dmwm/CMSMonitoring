VERSION=`git describe --tags`
flags=-ldflags="-s -w -X main.gitVersion=${VERSION}"

all: build

vet:
	go vet .

build:
	go clean; rm -rf pkg rucio-dataset-monitoring*; go build -tags netgo ${flags}

install:
	go install

clean:
	go clean; rm -rf pkg
