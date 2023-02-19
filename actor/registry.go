package actor

import (
	"sync"

	"github.com/anthdm/hollywood/log"
)

const localLookupAddr = "local"

type registry struct {
	mu     sync.RWMutex
	lookup map[uint64]processer
	engine *Engine
}

func newRegistry(e *Engine) *registry {
	return &registry{
		lookup: make(map[uint64]processer, 1024),
		engine: e,
	}
}

func (r *registry) remove(pid *PID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.lookup, pid.lookupKey())
}

func (r *registry) get(pid *PID) processer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if proc, ok := r.lookup[pid.lookupKey()]; ok {
		return proc
	}
	return r.engine.deadLetter
}

func (r *registry) getByName(name string) processer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	panic("help")
	// for key, proc := range r.lookup {
	// 	parts := strings.SplitN(key, PIDSeparator, 2)
	// 	if len(parts) < 2 {
	// 		return nil
	// 	}
	// 	if parts[1] == name {
	// 		return proc
	// 	}
	// }
	return nil
}

func (r *registry) add(proc processer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pidKey := proc.PID().lookupKey()
	if _, ok := r.lookup[pidKey]; ok {
		log.Warnw("[REGISTRY] process already registered", log.M{
			"pid": proc.PID(),
		})
		return
	}
	r.lookup[pidKey] = proc
}
