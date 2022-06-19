VERSION=`git describe --tags`
flags=-ldflags="-s -w -X main.gitVersion=${VERSION}"

all: build

vet:
	go vet .

build:
	go clean; rm -rf pkg rucio-dataset-mon-go*; go build ${flags}

install:
	go install

clean:
	go clean; rm -rf pkg
