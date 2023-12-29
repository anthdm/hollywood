package actor

import (
	"sync"
)

const LocalLookupAddr = "local"

// Registry is a struct that holds a mapping of process identifiers to their respective Processers.
// It's used for keeping track of all active processes in the system.
type Registry struct {
	mu     sync.RWMutex         // mu is a read-write mutex for synchronizing access to the lookup map.
	lookup map[string]Processer // lookup is a map that associates string identifiers with Processer instances.
	engine *Engine              // engine is a reference to the Engine associated with this Registry.
}

// newRegistry creates and returns a new instance of Registry.
// It initializes the lookup map and sets the engine.
func newRegistry(e *Engine) *Registry {
	return &Registry{
		lookup: make(map[string]Processer, 1024), // Initializes the lookup map with an initial capacity of 1024.
		engine: e,                                // Sets the engine reference.
	}
}

// Remove deletes a Processer from the registry using its PID.
func (r *Registry) Remove(pid *PID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.lookup, pid.ID)
}

// get returns the processer for the given PID, if it exists.
// If it doesn't exist, nil is returned so the caller must check for that
// and direct the message to the deadletter processer instead.
func (r *Registry) get(pid *PID) Processer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if proc, ok := r.lookup[pid.ID]; ok {
		return proc
	}
	return nil // Return nil if no Processer is found.
}

// getByID retrieves a Processer from the registry using an identifier string.
func (r *Registry) getByID(id string) Processer {
	r.mu.RLock()         // Lock the mutex for reading.
	defer r.mu.RUnlock() // Ensure the mutex is unlocked after this function.
	return r.lookup[id]  // Return the Processer associated with the given ID.
}

// add adds a new Processer to the registry.
// If a duplicate ID is detected, it broadcasts an ActorDuplicateIdEvent and does not add the Processer.
func (r *Registry) add(proc Processer) {
	r.mu.Lock()
	id := proc.PID().ID
	if _, ok := r.lookup[id]; ok {
		r.mu.Unlock()                                                   // Unlock the mutex if a duplicate is found.
		r.engine.BroadcastEvent(ActorDuplicateIdEvent{PID: proc.PID()}) // Broadcast a duplicate ID event.
		return
	}
	r.lookup[id] = proc // Add the Processer to the lookup map.
	r.mu.Unlock()       // Unlock the mutex after adding.
}
