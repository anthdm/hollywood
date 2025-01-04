test: build
	go test ./... -count=1 --race --timeout=5s

proto:
	protoc --go_out=. --go-vtproto_out=.  --go_opt=paths=source_relative --proto_path=. actor/actor.proto

build:
	go build -o bin/helloworld examples/helloworld/main.go 
	go build -o bin/hooks examples/middleware/hooks/main.go 
	go build -o bin/childprocs examples/childprocs/main.go 
	go build -o bin/request examples/request/main.go 
	go build -o bin/restarts examples/restarts/main.go 
	go build -o bin/eventstream examples/eventstream/main.go 
	go build -o bin/tcpserver examples/tcpserver/main.go 
	go build -o bin/metrics examples/metrics/main.go
	go build -o bin/chatserver examples/chat/server/main.go
	go build -o bin/chatclient examples/chat/client/main.go
	go build -o bin/cluster_member_1 examples/cluster/member_1/main.go
	go build -o bin/cluster_member_2 examples/cluster/member_2/main.go

bench:
	go run ./_bench/.

bench-profile:
	go test -bench='^BenchmarkHollywood$$' -run=NONE -cpuprofile cpu.prof -memprofile mem.prof ./_bench

.PHONY: proto
