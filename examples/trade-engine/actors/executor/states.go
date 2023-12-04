package executor

import "sync"

type executorStates struct {
	lock    sync.RWMutex
	price   float64 // using float in example
	status  string
	active  bool
	expires int64
}

func NewExecutorStates() *executorStates {
	return &executorStates{}
}

func (es *executorStates) Price() float64 {
	es.lock.RLock()
	defer es.lock.RUnlock()
	return es.price
}

func (es *executorStates) SetPrice(price float64) {
	es.lock.Lock()
	defer es.lock.Unlock()
	es.price = price
}

func (es *executorStates) Status() string {
	es.lock.RLock()
	defer es.lock.RUnlock()
	return es.status
}

func (es *executorStates) SetStatus(status string) {
	es.lock.Lock()
	defer es.lock.Unlock()
	es.status = status
}

func (es *executorStates) Active() bool {
	es.lock.RLock()
	defer es.lock.RUnlock()
	return es.active
}

func (es *executorStates) SetActive(actice bool) {
	es.lock.Lock()
	defer es.lock.Unlock()
	es.active = actice
}

func (es *executorStates) Expires() int64 {
	es.lock.RLock()
	defer es.lock.RUnlock()
	return es.expires
}

func (es *executorStates) SetExpires(expires int64) {
	es.lock.Lock()
	defer es.lock.Unlock()
	es.expires = expires
}
