package remote

import (
	"github.com/anthdm/hollywood/actor"
	"google.golang.org/protobuf/proto"
)

type Marshaler interface {
	proto.Message
	MarshalVT() ([]byte, error)
}

type Unmarshaler interface {
	proto.Message
	UnmarshalVT([]byte) error
}

func makeEnvelope(streams []writeToStream) (*Envelope, error) {
	protos := make([]*Message, len(streams))
	for i := 0; i < len(streams); i++ {
		msg := streams[i]
		pmsg, err := serialize(msg.pid, msg.sender, msg.msg)
		if err != nil {
			return nil, err
		}
		protos[i] = pmsg
	}
	return &Envelope{
		Messages: protos,
	}, nil
}

func serialize(pid *actor.PID, sender *actor.PID, msg Marshaler) (*Message, error) {
	b, err := msg.MarshalVT()
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
	msg, err := registryGetType(typeName)
	if err != nil {
		return nil, err
	}
	if err := msg.UnmarshalVT(data); err != nil {
		return nil, err
	}
	return msg, nil

	// n, err := protoregistry.GlobalTypes.FindMessageByName(protoreflect.FullName(typeName))
	// if err != nil {
	// 	return nil, err
	// }
	// pm := n.New().Interface()
	// err = proto.Unmarshal(data, pm)
	// return pm, err
}
