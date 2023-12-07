package cluster

import "github.com/anthdm/hollywood/actor"

type KindOpts struct {
	activateOnClusterStart bool
	id                     string
	local                  bool
}

type Kind struct {
	opts     KindOpts
	name     string
	producer actor.Producer
}

func NewKind(name string, p actor.Producer, opts KindOpts) *Kind {
	return &Kind{
		name:     name,
		opts:     opts,
		producer: p,
	}
}
