package actor

import (
	"sync"
)

const LocalLookupAddr = "local"

type Registry struct {
	lookup sync.Map
	engine *Engine
}

func newRegistry(e *Engine) *Registry {
	return &Registry{
		engine: e,
	}
}

// GetPID returns the process id associated for the given kind and its id.
// GetPID returns nil if the process was not found.
func (r *Registry) GetPID(kind, id string) *PID {
	proc := r.getByID(kind + pidSeparator + id)
	if proc != nil {
		return proc.PID()
	}
	return nil
}

// Remove removes the given PID from the registry.
func (r *Registry) Remove(pid *PID) {
	if pid == nil {
		return
	}
	r.lookup.Delete(pid.ID)
}

// get returns the processer for the given PID, if it exists.
// If it doesn't exist, nil is returned so the caller must check for that
// and direct the message to the deadletter processer instead.
func (r *Registry) get(pid *PID) Processer {
	if pid == nil {
		return nil
	}
	if v, ok := r.lookup.Load(pid.ID); ok {
		return v.(Processer)
	}
	return nil // didn't find the processer
}

func (r *Registry) getByID(id string) Processer {
	if v, ok := r.lookup.Load(id); ok {
		return v.(Processer)
	}
	return nil
}

func (r *Registry) add(proc Processer) {
	id := proc.PID().ID
	_, loaded := r.lookup.LoadOrStore(id, proc)
	if loaded {
		r.engine.BroadcastEvent(ActorDuplicateIdEvent{PID: proc.PID()})
		return
	}
	proc.Start()
}
