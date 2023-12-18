package cluster

import "github.com/anthdm/hollywood/actor"

// KindConfig holds the Kind configuration
type KindConfig struct{}

// A kind is a type of actor that can be activate on a node.
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
