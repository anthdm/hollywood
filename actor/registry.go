package actor

import (
	"sync"
)

const LocalLookupAddr = "local"

type Registry struct {
	mu     sync.RWMutex
	lookup map[string]Processer
	engine *Engine
}

func newRegistry(e *Engine) *Registry {
	return &Registry{
		lookup: make(map[string]Processer, 1024),
		engine: e,
	}
}

// Remove removes the given PID from the registry.
func (r *Registry) Remove(pid *PID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.lookup, pid.ID)
}

// GetReceiver returns the underlying receiver that is associated with
// the given ID.
func (r *Registry) GetReceiver(id string) Receiver {
	proc := r.getByID(id)
	if _, ok := proc.(*process); ok {
		return proc.(*process).context.Receiver()
	}
	return nil
}

// get returns the processer for the given PID, if it exists.
// If it doesn't exist, nil is returned so the caller must check for that
// and direct the message to the deadletter processer instead.
// Todo: consider returning a bool in addition to the processer so the semantics are clear.
func (r *Registry) get(pid *PID) Processer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if proc, ok := r.lookup[pid.ID]; ok {
		return proc
	}
	return nil // didn't find the processer
}

func (r *Registry) getByID(id string) Processer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lookup[id]
}

// TODO: When a process is already registered, we "should" create
// a random tag for it? Or are we going to prevent that, and let the user
// decide?
func (r *Registry) add(proc Processer) {
	r.mu.Lock()

	id := proc.PID().ID
	if _, ok := r.lookup[id]; ok {
		r.mu.Unlock()
		r.engine.BroadcastEvent(ActorDuplicateIdEvent{PID: proc.PID()})
		return
	}
	r.lookup[id] = proc
	r.mu.Unlock()
}
