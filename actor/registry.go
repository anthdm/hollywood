package actor

import (
	"bytes"
	"sync"

	"github.com/anthdm/hollywood/log"
)

const localLookupAddr = "local"

type registry struct {
	mu        sync.RWMutex
	lookup    map[string]processer
	keyWriter *keyWriter
	engine    *Engine
}

func newRegistry(e *Engine) *registry {
	return &registry{
		lookup:    make(map[string]processer, 1024),
		keyWriter: newKeyWriter(),
		engine:    e,
	}
}

func (r *registry) remove(pid *PID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := r.keyWriter.writePIDKey(pid)
	delete(r.lookup, key)
}

func (r *registry) get(pid *PID) processer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := r.keyWriter.writePIDKey(pid)
	if proc, ok := r.lookup[key]; ok {
		return proc
	}
	return r.engine.deadLetter
}

func (r *registry) add(proc processer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	pidKey := r.keyWriter.writePIDKey(proc.PID())
	if _, ok := r.lookup[pidKey]; ok {
		log.Warnw("[REGISTRY] process already registered", log.M{
			"pid": proc.PID(),
		})
		return
	}
	r.lookup[pidKey] = proc
}

type keyWriter struct {
	buf *bytes.Buffer
}

func newKeyWriter() *keyWriter {
	return &keyWriter{
		buf: new(bytes.Buffer),
	}
}

func (w *keyWriter) writePIDKey(pid *PID) string {
	w.buf.WriteString(pid.Address)
	w.buf.WriteString("/")
	w.buf.WriteString(pid.ID)
	for _, tag := range pid.Tags {
		w.buf.WriteString("/")
		w.buf.WriteString(tag)
	}
	key := w.buf.String()
	w.buf.Reset()
	return key
}
