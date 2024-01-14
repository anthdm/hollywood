package cluster

import "github.com/anthdm/hollywood/actor"

// KindConfig holds configuration for a registered kind.
type KindConfig struct{}

// NewKindConfig returns a default kind configuration.
func NewKindConfig() KindConfig {
	return KindConfig{}
}

// A kind is a type of actor that can be activated from any member of the cluster.
type kind struct {
	config   KindConfig
	name     string
	producer actor.Producer
}

// newKind returns a new kind.
func newKind(name string, p actor.Producer, config KindConfig) kind {
	return kind{
		name:     name,
		config:   config,
		producer: p,
	}
}
