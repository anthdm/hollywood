test: build
	go test ./... --race

proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-drpc_out=. --go-drpc_opt=paths=source_relative --proto_path=. actor/actor.proto

build:
	go build -o bin/helloworld examples/helloworld/main.go 
	go build -o bin/hooks examples/middleware/hooks/main.go 
	go build -o bin/childprocs examples/childprocs/main.go 
	go build -o bin/request examples/request/main.go 
	go build -o bin/restarts examples/restarts/main.go 
	go build -o bin/eventstream examples/eventstream/main.go 

.PHONY: proto