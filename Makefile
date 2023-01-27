test:
	go test ./... --race

proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-drpc_out=. --go-drpc_opt=paths=source_relative --proto_path=. actor/actor.proto

.PHONY: proto