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

// GetPID returns the process id associated for the given kind and its id.
// GetPID returns nil if the process was not found.
func (r *Registry) GetPID(kind, id string) *PID {
	proc := r.getByID(kind + pidSeparator + id)
	if proc != nil {
		return proc.PID()
	}
	return nil
}

// GetAllPIDs returns a slice of all currently registered PIDs.
func (r *Registry) GetAllPIDs() []*actor.PID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pids := make([]*actor.PID, 0, len(r.lookup))
	for _, proc := range r.lookup {
		pids = append(pids, proc.PID())
	}
	return pids
}

// GetPIDsByKind returns all PIDs that match the given kind.
func (r *Registry) GetPIDsByKind(kind string) []*actor.PID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var pids []*actor.PID
	prefix := kind + pidSeparator
	for id, proc := range r.lookup {
		if strings.HasPrefix(id, prefix) {
			pids = append(pids, proc.PID())
		}
	}
	return pids
}

// GetPIDMap returns a map of actor IDs to their PIDs.
func (r *Registry) GetPIDMap() map[string]*actor.PID {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pidMap := make(map[string]*actor.PID, len(r.lookup))
	for id, proc := range r.lookup {
		pidMap[id] = proc.PID()
	}
	return pidMap
}

// Remove removes the given PID from the registry.
func (r *Registry) Remove(pid *PID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.lookup, pid.ID)
}

// get returns the processer for the given PID, if it exists.
// If it doesn't exist, nil is returned so the caller must check for that
// and direct the message to the deadletter processer instead.
func (r *Registry) get(pid *PID) Processer {
	if pid == nil {
		return nil
	}
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
	proc.Start()
}
