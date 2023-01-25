package actor

import (
	"github.com/anthdm/hollywood/log"
	cmap "github.com/orcaman/concurrent-map/v2"
)

type registry struct {
	lookup cmap.ConcurrentMap[string, cmap.ConcurrentMap[string, *process]]
}

func newRegistry() *registry {
	return &registry{
		lookup: cmap.New[cmap.ConcurrentMap[string, *process]](),
	}
}

func (r *registry) remove(pid *PID) {
	addrs, ok := r.lookup.Get(pid.Address)
	if !ok {
		return
	}
	addrs.Remove(pid.ID)
	if addrs.Count() == 0 {
		r.lookup.Remove(pid.Address)
	}
}

func (r *registry) add(proc *process) {
	pid := proc.pid
	if addrs, ok := r.lookup.Get(pid.Address); ok {
		if !addrs.Has(pid.ID) {
			addrs.Set(pid.ID, proc)
		} else {
			log.Warnw("[REGISTRY] process already registered", log.M{
				"pid": pid,
			})
		}
		return
	}

	addrs := cmap.New[*process]()
	addrs.Set(pid.ID, proc)
	r.lookup.Set(pid.Address, addrs)
}

func (r *registry) get(pid *PID) *process {
	maddr, ok := r.lookup.Get(pid.Address)
	if !ok {
		panic("deadletter")
	}
	proc, ok := maddr.Get(pid.ID)
	if !ok {
		panic("deadletter")
	}
	return proc
}
