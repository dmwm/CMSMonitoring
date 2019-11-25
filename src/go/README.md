### CMS NATS clients
CMS [NATS](http://nats.io) is a cloud messaging service. Here you can find
basic Go-based clients for publisher and subscriber code. To compile the code
please following these steps:

```
# get dependency
go get github.com/codesuki/go-time-series

# build executable
go build nats-sub.go
go build nats-pub.go
```
