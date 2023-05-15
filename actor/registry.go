package actor

import (
	"github.com/anthdm/hollywood/log"
	"github.com/anthdm/hollywood/safemap"
)

const LocalLookupAddr = "local"

type Registry struct {
	lookup     *safemap.SafeMap[string, Processer]
	deadletter Processer
}

func newRegistry(dl Processer) *Registry {
	return &Registry{
		lookup:     safemap.New[string, Processer](),
		deadletter: dl,
	}
}

func (r *Registry) Remove(pid *PID) {
	r.lookup.Delete(pid.ID)
}

func (r *Registry) get(pid *PID) Processer {
	if proc, ok := r.lookup.Get(pid.ID); ok {
		return proc
	}
	return r.deadletter
}

func (r *Registry) getByID(id string) Processer {
	processer, _ := r.lookup.Get(id)
	return processer
}

// TODO: When a process is already registered, we "should" create
// a random tag for it? Or are we going to prevent that, and let the user
// decide?
func (r *Registry) add(proc Processer) {
	id := proc.PID().ID
	if proc, ok := r.lookup.Get(id); ok {
		log.Warnw("[REGISTRY] process already registered", log.M{
			"pid": proc.PID(),
		})
		return
	}
	r.lookup.Set(id, proc)
}
