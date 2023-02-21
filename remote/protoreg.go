package remote

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

var registry = map[string]VTUnmarshaler{}

func RegisterType(v VTUnmarshaler) {
	tname := string(proto.MessageName(v))
	registry[tname] = v
}

func registryGetType(t string) (VTUnmarshaler, error) {
	if m, ok := registry[t]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("given type (%s) is not registered. Did you forget to register your type with remote.RegisterType(&instance{})?", t)
}
