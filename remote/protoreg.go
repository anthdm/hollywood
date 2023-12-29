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
	// Get the type name of the VTUnmarshaler.
	tname := string(proto.MessageName(v))
	// Add the VTUnmarshaler to the registry with its type name as the key.
	registry[tname] = v
}

// registryGetType is a function that gets a VTUnmarshaler from the registry given its type name.
// It takes a type name as a parameter and returns a VTUnmarshaler and an error.
func registryGetType(t string) (VTUnmarshaler, error) {
	// Check if the type name is in the registry.
	if m, ok := registry[t]; ok {
		// If it is, return the VTUnmarshaler and nil for the error.
		return m, nil
	}
	// If it's not, return nil for the VTUnmarshaler and an error indicating that the type is not registered.
	return nil, fmt.Errorf("given type (%s) is not registered. Did you forget to register your type with remote.RegisterType(&instance{})?", t)
}
