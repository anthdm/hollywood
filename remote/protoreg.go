package remote

import (
	"fmt"

	"google.golang.org/protobuf/proto"
)

// registry is a global variable that maps type names to VTUnmarshaler instances.
var registry = map[string]VTUnmarshaler{}

// RegisterType is a function that registers a VTUnmarshaler with its type name in the registry.
// It takes a VTUnmarshaler as a parameter.
func RegisterType(v VTUnmarshaler) {
	tname := string(proto.MessageName(v))
	registry[tname] = v
}

// registryGetType is a function that gets a VTUnmarshaler from the registry given its type name.
// It takes a type name as a parameter and returns a VTUnmarshaler or an error.
func registryGetType(t string) (VTUnmarshaler, error) {

	if m, ok := registry[t]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("given type (%s) is not registered. Did you forget to register your type with remote.RegisterType(&instance{})?", t)
}
