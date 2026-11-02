package actor

import (
	"sync"
)

const LocalLookupAddr = "local"

type Registry struct {
	// lookup maps a PID's ID to its Processer. It is read on every local send,
	// and a sync.Map serves those reads without a shared lock.
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
	r.lookup.Delete(pid.ID)
}

// get returns the processer for the given PID, if it exists.
// If it doesn't exist, nil is returned so the caller must check for that
// and direct the message to the deadletter processer instead.
func (r *Registry) get(pid *PID) Processer {
	if pid == nil {
		return nil
	}
	return r.getByID(pid.ID)
}

func (r *Registry) getByID(id string) Processer {
	if proc, ok := r.lookup.Load(id); ok {
		return proc.(Processer)
	}
	return nil // didn't find the processer
}

func (r *Registry) add(proc Processer) {
	if _, loaded := r.lookup.LoadOrStore(proc.PID().ID, proc); loaded {
		r.engine.BroadcastEvent(ActorDuplicateIdEvent{PID: proc.PID()})
		return
	}
	proc.Start()
}
