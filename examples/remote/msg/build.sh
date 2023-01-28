
protoc -I actor/actor.proto --go_out=. --go_opt=paths=source_relative --proto_path=. examples/remote/msg/message.proto