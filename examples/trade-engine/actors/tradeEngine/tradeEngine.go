package tradeEngine

import (
	"fmt"
	"log/slog"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/trade-engine/actors/executor"
	"github.com/anthdm/hollywood/examples/trade-engine/actors/price"
)

type tradeEngine struct {
}

type TradeOrderRequest struct {
	TradeID    string
	Token0     string
	Token1     string
	Chain      string
	Wallet     string
	PrivateKey string
	Expires    int64
}

type CancelOrderRequest struct {
	ID string
}

func (t *tradeEngine) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Stopped:
		slog.Info("Stopped Trade Engine")

	case actor.Started:
		slog.Info("Started Trade Engine")

	case *TradeOrderRequest:
		// got new trade order, create the executor
		slog.Info("Got New Trade Order", "id", msg.TradeID, "wallet", msg.Wallet)
		t.spawnExecutor(msg, c)

	case CancelOrderRequest:
		// cancel the order
		slog.Info("Cancelling Trade Order", "id", msg.ID)
		t.cancelOrder(msg.ID, c)

	}
}

func (t *tradeEngine) spawnExecutor(msg *TradeOrderRequest, c *actor.Context) {
	// make sure is price stream for the pair
	pricePID := t.ensurePriceStream(msg, c)

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

func (t *tradeEngine) ensurePriceStream(order *TradeOrderRequest, c *actor.Context) *actor.PID {
	ticker := toTicker(order.Token0, order.Token1, order.Chain)

	pid := c.Child("trade-engine/" + ticker)
	if pid != nil {
		return pid
	}

	options := price.PriceOptions{
		Ticker: ticker,
		Token0: order.Token0,
		Token1: order.Token1,
		Chain:  order.Chain,
	}

	// spawn the actor
	pid = c.SpawnChild(price.NewPriceActor(options), ticker)
	return pid
}

func (t *tradeEngine) cancelOrder(id string, c *actor.Context) {
	// get the executor
	pid := c.Child("trade-engine/" + id)
	if pid == nil {
		slog.Error("Failed to cancel order", "err", "order not found", "id", id)
		return
	}

	// send cancel message
	// sending specific message to handle cancel
	c.Send(pid, executor.CancelOrderRequest{})
}

func NewTradeEngine() actor.Producer {
	return func() actor.Receiver {
		return &tradeEngine{}
	}
}

func toTicker(token0, token1, chain string) string {
	return fmt.Sprintf("%s-%s-%s", token0, token1, chain)
}
