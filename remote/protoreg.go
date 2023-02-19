package remote

import (
	fmt "fmt"

	proto "google.golang.org/protobuf/proto"
)

var registry = map[string]Unmarshaler{}

func RegisterType(v Unmarshaler) {
	tname := string(proto.MessageName(v))
	registry[tname] = v
}

func registryGetType(t string) (Unmarshaler, error) {
	if m, ok := registry[t]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("given type (%s) is not registered. Did you forget to register your type with remote.RegisterType(&instance{})?", t)
}
