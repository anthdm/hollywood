package types

import "github.com/anthdm/hollywood/actor"

// message to cancel a trade
type CancelOrderRequest struct {
	TradeID string
}

// message sent to get trade info
type TradeInfoRequest struct {
	TradeID string
}

// response message for trade info
type TradeInfoResponse struct {
	// info regarding the current position
	// eg price, pnl, etc
	Foo   int
	Bar   int
	Price float64 // using float in example
}

// message sent to create a new trade order
type TradeOrderRequest struct {
	TradeID    string
	Token0     string
	Token1     string
	Chain      string
	Wallet     string
	PrivateKey string
	Expires    int64
}

// options when creating new price watcher
type PriceOptions struct {
	Ticker string
	Token0 string
	Token1 string
	Chain  string
}

// price update from price watcher
type PriceUpdate struct {
	Ticker    string
	UpdatedAt int64
	Price     float64
}

// subscribe to price watcher
type Subscribe struct {
	Sendto *actor.PID
}

// unsubscribe from price watcher
type Unsubscribe struct {
	Sendto *actor.PID
}

// used with SendRepeat to trigger price update
type TriggerPriceUpdate struct{}
