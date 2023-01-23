package actor

import (
	"sync"
)

type Registry struct {
	mu    sync.RWMutex
	procs map[string]*Process
}

func NewRegistry() *Registry {
	return &Registry{
		procs: make(map[string]*Process),
	}
}

func (r *Registry) remove(pid *PID) {
	r.mu.Lock()
	delete(r.procs, pid.String())
	r.mu.Unlock()
}

func (r *Registry) add(pid *PID, proc *Process) {
	r.mu.Lock()
	r.procs[pid.String()] = proc
	r.mu.Unlock()
}

func (r *Registry) get(pid *PID) *Process {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.procs[pid.String()]
}
