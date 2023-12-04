package executor

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/trade-engine/actors/price"
)

// message to get trade info
type TradeInfoRequest struct{}

// message to cancel order
type CancelOrderRequest struct{}

// response message for trade info
type TradeInfoResponse struct {
	// info regarding the current position
	// eg price, pnl, etc
	foo   int
	bar   int
	price float64 // using float in example
}

type ExecutorOptions struct {
	PriceWatcherPID *actor.PID
	TradeID         string
	Ticker          string
	Token0          string
	Token1          string
	Chain           string
	Wallet          string
	Pk              string
	Expires         int64
}

type tradeExecutorActor struct {
	id              string
	ActorEngine     *actor.Engine
	PID             *actor.PID
	priceWatcherPID *actor.PID
	ticker          string
	token0          string
	token1          string
	chain           string
	wallet          string
	pk              string
	status          string
	lastPrice       float64
	expires         int64
}

func (te *tradeExecutorActor) Receive(c *actor.Context) {
	switch msg := c.Message().(type) {
	case actor.Started:
		slog.Info("Started Trade Executor Actor", "id", te.id, "wallet", te.wallet)

		te.ActorEngine = c.Engine()
		te.PID = c.PID()

		// subscribe to eventstream
		te.ActorEngine.Send(te.priceWatcherPID, price.Subscribe{Sendto: te.PID})

	case actor.Stopped:
		slog.Info("Stopped Trade Executor Actor", "id", te.id, "wallet", te.wallet)

	case price.PriceUpdate:
		// ensure correct ticker
		if msg.Ticker != te.ticker {
			return
		}
		// update the price
		te.processUpdate(msg)

	case TradeInfoRequest:
		slog.Info("Got TradeInfoRequest", "id", te.id, "wallet", te.wallet)
		te.replyWithTradeInfo(c)

	case CancelOrderRequest:
		slog.Info("Got CancelOrderRequest", "id", te.id, "wallet", te.wallet)
		// update status
		te.status = "cancelled"

		// stop the executor
		te.Finished()

	default:
		_ = msg

	}
}

func (te *tradeExecutorActor) processUpdate(update price.PriceUpdate) {

	// if expires is set and is less than current time, cancel the order
	if te.expires != 0 && time.Now().UnixMilli() > te.expires {
		slog.Warn("Trade Expired", "id", te.id, "wallet", te.wallet)
		te.Finished()
		return
	}

	// update the price
	te.lastPrice = update.Price

	// do something with the price
	// eg update pnl, etc

	fmt.Println("price update", update.Price)

}

func (te *tradeExecutorActor) replyWithTradeInfo(c *actor.Context) {
	c.Respond(&TradeInfoResponse{
		foo:   100,
		bar:   100,
		price: te.lastPrice,
	})
}

func (te *tradeExecutorActor) Finished() {
	// make sure actorEngine is safe
	if te.ActorEngine == nil {
		slog.Error("tradeExecutor.actorEngine is <nil>")
	}

	if te.PID == nil {
		slog.Error("tradeExecutor.PID is <nil>")
	}

	// unsubscribe from price updates
	te.ActorEngine.Send(te.priceWatcherPID, price.Unsubscribe{Sendto: te.PID})

	// poision itself
	te.ActorEngine.Poison(te.PID)
}

func NewExecutorActor(opts *ExecutorOptions) actor.Producer {
	return func() actor.Receiver {
		return &tradeExecutorActor{
			id:              opts.TradeID,
			ticker:          opts.Ticker,
			token0:          opts.Token0,
			token1:          opts.Token1,
			chain:           opts.Chain,
			wallet:          opts.Wallet,
			pk:              opts.Pk,
			priceWatcherPID: opts.PriceWatcherPID,
			expires:         opts.Expires,
			status:          "active",
		}
	}
}
