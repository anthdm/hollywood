package price

import "sync"

type priceStates struct {
	lock      sync.RWMutex
	lastPrice float64 // using float in example
	updatedAt int64
	lastCall  int64
	callCount int64
}

func NewPriceStates() *priceStates {
	return &priceStates{}
}

func (ps *priceStates) LastPrice() float64 {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	return ps.lastPrice
}

func (ps *priceStates) SetLastPrice(price float64) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	ps.lastPrice = price
}

func (ps *priceStates) UpdatedAt() int64 {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	return ps.updatedAt
}

func (ps *priceStates) SetUpdatedAt(updatedAt int64) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	ps.updatedAt = updatedAt
}

func (ps *priceStates) LastCall() int64 {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	return ps.lastCall
}

func (ps *priceStates) SetLastCall(lastCall int64) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	ps.lastCall = lastCall
}

func (ps *priceStates) CallCount() int64 {
	ps.lock.RLock()
	defer ps.lock.RUnlock()
	return ps.callCount
}

func (ps *priceStates) SetCallCount(callCount int64) {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	ps.callCount = callCount
}

func (ps *priceStates) IncCallCount() {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	ps.callCount++
}
