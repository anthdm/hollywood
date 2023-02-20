package actor

import (
	"sync"

	"github.com/anthdm/hollywood/log"
)

const LocalLookupAddr = "local"

type registry struct {
	mu     sync.RWMutex
	lookup map[string]Processer
	engine *Engine
}

func newRegistry(e *Engine) *registry {
	return &registry{
		lookup: make(map[string]Processer, 1024),
		engine: e,
	}
}

func (r *registry) remove(pid *PID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.lookup, pid.ID)
}

func (r *registry) get(pid *PID) Processer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if proc, ok := r.lookup[pid.ID]; ok {
		return proc
	}
	return r.engine.deadLetter
}

func (r *registry) getByID(id string) Processer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.lookup[id]
}

// TODO: When a process is already registered, we "should" create
// a random tag for it? Or are we going to prevent that, and let the user
// decide?
func (r *registry) add(proc Processer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := proc.PID().ID
	if _, ok := r.lookup[id]; ok {
		log.Warnw("[REGISTRY] process already registered", log.M{
			"pid": proc.PID(),
		})
		return
	}
	r.lookup[id] = proc
}
