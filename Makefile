run:
	go run main.go

test:
	go test ./... --race

proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --proto_path=. actor/actor.proto

.PHONY: proto