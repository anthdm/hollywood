package remote

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

type Serializer interface {
	Serialize(msg any) ([]byte, error)
	TypeName(any) string
}

type Deserializer interface {
	Deserialize([]byte, string) (any, error)
}

type VTMarshaler interface {
	proto.Message
	MarshalVT() ([]byte, error)
}

type VTUnmarshaler interface {
	proto.Message
	UnmarshalVT([]byte) error
}

// Todo: delete this or state why it isn't deleted.
// type DefaultSerializer struct{}

// func (DefaultSerializer) Serialize(msg any) ([]byte, error) {
// 	switch msg.(type) {
// 	case VTMarshaler:
// 		return VTProtoSerializer{}.Serialize(msg)
// 	case proto.Message:
// 		return ProtoSerializer{}.Serialize(msg)
// 	default:
// 		return nil, fmt.Errorf("unsupported message type (%v) for serialization", reflect.TypeOf(msg))
// 	}
// }

// func (DefaultSerializer) Deserialize(data []byte, mtype string) (any, error) {
// 	switch msg.(type) {
// 	case VTMarshaler:
// 		return VTProtoSerializer{}.Serialize(msg)
// 	case proto.Message:
// 		return ProtoSerializer{}.Serialize(msg)
// 	default:
// 		return nil, fmt.Errorf("unsupported message type (%v) for serialization", reflect.TypeOf(msg))
// 	}
// }

type ProtoSerializer struct{}

func (ProtoSerializer) Serialize(msg any) ([]byte, error) {
	return proto.Marshal(msg.(proto.Message))
}

func (ProtoSerializer) Deserialize(data []byte, tname string) (any, error) {
	pname := protoreflect.FullName(tname)
	n, err := protoregistry.GlobalTypes.FindMessageByName(pname)
	if err != nil {
		return nil, err
	}
	pm := n.New().Interface()
	err = proto.Unmarshal(data, pm)
	return pm, err
}

func (ProtoSerializer) TypeName(msg any) string {
	return string(proto.MessageName(msg.(proto.Message)))
}

type VTProtoSerializer struct{}

func (VTProtoSerializer) TypeName(msg any) string {
	return string(proto.MessageName(msg.(proto.Message)))
}

func (VTProtoSerializer) Serialize(msg any) ([]byte, error) {
	return msg.(VTMarshaler).MarshalVT()
}

func (VTProtoSerializer) Deserialize(data []byte, mtype string) (any, error) {
	v, err := registryGetType(mtype)
	if err != nil {
		return nil, err
	}
	err = v.UnmarshalVT(data)
	return v, err
}
