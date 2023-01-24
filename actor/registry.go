package actor

import (
	"sync"

	"github.com/anthdm/hollywood/log"
)

type Registry struct {
	mu    sync.RWMutex
	procs map[string]*process
}

func NewRegistry() *Registry {
	return &Registry{
		procs: make(map[string]*process),
	}
}

func (r *Registry) remove(pid *PID) {
	r.mu.Lock()
	delete(r.procs, pid.String())
	r.mu.Unlock()
}

func (r *Registry) add(pid *PID, proc *process) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.procs[pid.String()]; ok {
		log.Warnw("[ACTOR] pid already registered", log.M{
			"pid": pid,
		})
		return
	}
	r.procs[pid.String()] = proc
}

func (r *Registry) get(pid *PID) *process {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.procs[pid.String()]
}
