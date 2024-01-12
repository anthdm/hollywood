package cluster

import "github.com/anthdm/hollywood/actor"

// KindConfig holds configuration for a registered kind.
type KindConfig struct {
	activateOnMember ActivateOnMemberFunc
}

// NewKindConfig returns a default kind configuration.
func NewKindConfig() KindConfig {
	return KindConfig{
		activateOnMember: ActivateOnRandomMember,
	}
}

// WithActivateOnMemberFunc set the function that will be used to select the member
// where this kind will be spawned/activated on.
func (config KindConfig) WithActivateOnMemberFunc(fun ActivateOnMemberFunc) KindConfig {
	config.activateOnMember = fun
	return config
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
