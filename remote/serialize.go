package remote

import (
	"github.com/anthdm/hollywood/actor"
	"google.golang.org/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

func serialize(pid *actor.PID, sender *actor.PID, msg proto.Message) (*Message, error) {
	b, err := proto.Marshal(msg)
	if err != nil {
		return nil, err
	}
	m := &Message{
		Data:     b,
		TypeName: string(proto.MessageName(msg)),
		Target:   pid,
		Sender:   sender,
	}
	return m, nil
}

func deserialize(data []byte, typeName string) (any, error) {
	n, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(typeName))
	if err != nil {
		return nil, err
	}
	pm := n.New().Interface()
	err = proto.Unmarshal(data, pm)
	return pm, err
}
