package remote

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// Serializer is an interface that defines methods for serializing messages.
// It has two methods: Serialize, which serializes a message into a byte slice, and TypeName, which returns the type name of a message.
type Serializer interface {
	Serialize(msg any) ([]byte, error)
	TypeName(any) string
}

// Deserializer is an interface that defines a method for deserializing messages.
// It has one method: Deserialize, which deserializes a byte slice into a message given the type name.
type Deserializer interface {
	Deserialize([]byte, string) (any, error)
}

// VTMarshaler is an interface that extends the proto.Message interface and defines a method for marshaling messages.
// It has one method: MarshalVT, which marshals a message into a byte slice.
type VTMarshaler interface {
	proto.Message
	MarshalVT() ([]byte, error)
}

// VTUnmarshaler is an interface that extends the proto.Message interface and defines a method for unmarshaling messages.
// It has one method: UnmarshalVT, which unmarshals a byte slice into a message.
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

// ProtoSerializer is a struct that implements the Serializer and Deserializer interfaces using the protobuf library.
type ProtoSerializer struct{}

// Serialize is a method that serializes a protobuf message into a byte slice.
// It takes a message of any type and returns a byte slice and an error.
func (ProtoSerializer) Serialize(msg any) ([]byte, error) {
	// Use the protobuf library's Marshal function to serialize the message.
	return proto.Marshal(msg.(proto.Message))
}

// Deserialize is a method that deserializes a byte slice into a protobuf message given the type name.
// It takes a byte slice and a type name as parameters and returns a message of any type and an error.
func (ProtoSerializer) Deserialize(data []byte, tname string) (any, error) {
	// Convert the type name into a protobuf full name.
	pname := protoreflect.FullName(tname)
	// Find the message descriptor by name in the global types registry.
	n, err := protoregistry.GlobalTypes.FindMessageByName(pname)
	if err != nil {
		return nil, err
	}
	// Create a new message of the found type.
	pm := n.New().Interface()
	// Use the protobuf library's Unmarshal function to deserialize the data into the message.
	err = proto.Unmarshal(data, pm)
	return pm, err
}

// TypeName is a method that returns the type name of a protobuf message.
// It takes a message of any type and returns a string.
func (ProtoSerializer) TypeName(msg any) string {
	// Use the protobuf library's MessageName function to get the type name of the message.
	return string(proto.MessageName(msg.(proto.Message)))
}

// VTProtoSerializer is a struct that implements the Serializer and Deserializer interfaces using the VTMarshaler and VTUnmarshaler interfaces.
type VTProtoSerializer struct{}

// TypeName is a method that returns the type name of a protobuf message.
// It takes a message of any type and returns a string.
func (VTProtoSerializer) TypeName(msg any) string {
	// Use the protobuf library's MessageName function to get the type name of the message.
	return string(proto.MessageName(msg.(proto.Message)))
}

// Serialize is a method that serializes a message into a byte slice using the VTMarshaler interface.
// It takes a message of any type and returns a byte slice and an error.
func (VTProtoSerializer) Serialize(msg any) ([]byte, error) {
	// Use the VTMarshaler's MarshalVT function to serialize the message.
	return msg.(VTMarshaler).MarshalVT()
}

// Deserialize is a method that deserializes a byte slice into a message using the VTUnmarshaler interface.
// It takes a byte slice and a type name as parameters and returns a message of any type and an error.
func (VTProtoSerializer) Deserialize(data []byte, mtype string) (any, error) {
	// Get the type from the registry.
	v, err := registryGetType(mtype)
	if err != nil {
		return nil, err
	}
	// Use the VTUnmarshaler's UnmarshalVT function to deserialize the data into the message.
	err = v.UnmarshalVT(data)
	return v, err
}
