
protoc -I=../actor --go_out=. --go-vtproto_out=. --go_opt=paths=source_relative  --proto_path=. cluster.proto