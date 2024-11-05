package tradeEngine

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/fancom/hollywood/actor"
	"github.com/fancom/hollywood/examples/trade-engine/actors/executor"
	"github.com/fancom/hollywood/examples/trade-engine/actors/price"
	"github.com/fancom/hollywood/examples/trade-engine/types"
)

// tradeEngineActor is the main actor for the trade engine
type tradeEngineActor struct {
}

// TradeOrderRequest is the message sent to the trade engine to create a new trade order

func (t *tradeEngineActor) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		slog.Info("tradeEngine.Started")
	case actor.Stopped:
		slog.Info("tradeEngine.Stopped")
	case types.TradeOrderRequest:
		// got new trade order, create the executor
		slog.Info("tradeEngine.TradeOrderRequest", "id", msg.TradeID, "wallet", msg.Wallet)
		// spawn the executor
		t.spawnExecutor(msg, c)
	case types.CancelOrderRequest:
		// cancel the order
		slog.Info("tradeEngine.CancelOrderRequest", "id", msg.TradeID)

		// cancel the order
		t.cancelOrder(msg.TradeID, c)

	case types.TradeInfoRequest:
		// get trade info
		slog.Info("tradeEngine.TradeInfoRequest", "id", msg.TradeID)

		t.handleTradeInfoRequest(msg, c)
	}
}

func (t *tradeEngineActor) spawnExecutor(msg types.TradeOrderRequest, c *actor.Context) {
	// make sure there is a price Watcher for this token pair
	pricePID := t.ensurePriceWatcher(msg, c)

	// spawn the executor
	options := &executor.ExecutorOptions{
		PriceWatcherPID: pricePID,
		TradeID:         msg.TradeID,
		Ticker:          toTicker(msg.Token0, msg.Token1, msg.Chain),
		Token0:          msg.Token0,
		Token1:          msg.Token1,
		Chain:           msg.Chain,
		Wallet:          msg.Wallet,
		Pk:              msg.PrivateKey,
		Expires:         msg.Expires,
	}

	// spawn the actor
	c.SpawnChild(executor.NewExecutorActor(options), msg.TradeID)
}

func (t *tradeEngineActor) ensurePriceWatcher(order types.TradeOrderRequest, c *actor.Context) *actor.PID {
	// create the ticker string
	ticker := toTicker(order.Token0, order.Token1, order.Chain)

	// look for existing price watcher in trade-engine child actors
	pid := c.Child("trade-engine/" + ticker)
	if pid != nil {
		// if we found a price watcher, return it
		return pid
	}

	// no price watcher found, spawn a new one
	options := types.PriceOptions{
		Ticker: ticker,
		Token0: order.Token0,
		Token1: order.Token1,
		Chain:  order.Chain,
	}

	// spawn the actor
	pid = c.SpawnChild(price.NewPriceActor(options), ticker)
	return pid
}

func (t *tradeEngineActor) cancelOrder(id string, c *actor.Context) {
	// get the executor
	pid := c.Child("trade-engine/" + id)
	if pid == nil {
		// no executor found
		slog.Error("Failed to cancel order", "err", "tradeExecutor PID not found", "id", id)
		return
	}

	// send cancel message
	c.Send(pid, types.CancelOrderRequest{})
}

func (t *tradeEngineActor) handleTradeInfoRequest(msg types.TradeInfoRequest, c *actor.Context) {
	// get the executor
	pid := c.Child("trade-engine/" + msg.TradeID)
	if pid == nil {
		// no executor found
		slog.Error("Failed to get trade info", "err", "tradeExecutor PID not found", "id", msg.TradeID)
		return
	}

	// send tradeInfo Request
	resp := c.Request(pid, types.TradeInfoRequest{}, time.Second*5)
	res, err := resp.Result()
	if err != nil {
		slog.Error("Failed to get trade info", "err", err)
		return
	}

	switch msg := res.(type) {

	case types.TradeInfoResponse:
		c.Respond(msg)

	default:
		slog.Error("Failed to get trade info", "err", "unknown response type")
	}

}

func NewTradeEngine() actor.Producer {
	return func() actor.Receiver {
		return &tradeEngineActor{}
	}
}

// utility function to format the ticker
func toTicker(token0, token1, chain string) string {
	return fmt.Sprintf("%s-%s-%s", token0, token1, chain)
}
