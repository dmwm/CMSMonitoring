VERSION=`git describe --tags`
flags=-ldflags "-s -w -extldflags '-static' -X main.gitVersion=${VERSION}"

all: build

vet:
	go vet .

build:
	go clean; rm -rf pkg rucio-dataset-monitoring*; CGO_ENABLED=0 go build ${flags}

install:
	go install

clean:
	go clean; rm -rf pkg
