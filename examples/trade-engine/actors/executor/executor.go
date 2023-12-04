package executor

import (
	"log/slog"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/trade-engine/types"
)

// Options for creating a new executor
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
		slog.Info("tradeExecutor.Started", "id", te.id, "wallet", te.wallet)

		// set actorEngine and PID
		te.ActorEngine = c.Engine()
		te.PID = c.PID()

		// subscribe to price updates
		te.ActorEngine.Send(te.priceWatcherPID, types.Subscribe{Sendto: te.PID})

	case actor.Stopped:
		slog.Info("tradeExecutor.Stopped", "id", te.id, "wallet", te.wallet)

	case types.PriceUpdate:
		// update the price
		te.processUpdate(msg)

	case types.TradeInfoRequest:
		slog.Info("tradeExecutor.TradeInfoRequest", "id", te.id, "wallet", te.wallet)

		// handle the request
		te.handleTradeInfoRequest(c)

	case types.CancelOrderRequest:
		slog.Info("tradeExecutor.CancelOrderRequest", "id", te.id, "wallet", te.wallet)

		// update status
		te.status = "cancelled"

		// stop the executor
		te.Finished()
	}
}

func (te *tradeExecutorActor) processUpdate(update types.PriceUpdate) {

	// if expires is set and is less than current time, cancel the order
	if te.expires != 0 && time.Now().UnixMilli() > te.expires {
		slog.Info("Trade Expired", "id", te.id, "wallet", te.wallet)
		te.Finished()
		return
	}

	// update the price
	te.lastPrice = update.Price

	// do something with the price
	// eg update pnl, etc

	// for example just print price
	slog.Info("tradeExecutor.PriceUpdate", "ticker", update.Ticker, "price", update.Price)
}

func (te *tradeExecutorActor) handleTradeInfoRequest(c *actor.Context) {
	c.Respond(types.TradeInfoResponse{
		// for example
		Foo:   100,
		Bar:   100,
		Price: te.lastPrice,
	})
}

func (te *tradeExecutorActor) Finished() {
	// make sure ActorEngine and PID are set
	if te.ActorEngine == nil {
		slog.Error("tradeExecutor.actorEngine is <nil>")
	}

	if te.PID == nil {
		slog.Error("tradeExecutor.PID is <nil>")
	}

	// unsubscribe from price updates
	te.ActorEngine.Send(te.priceWatcherPID, types.Unsubscribe{Sendto: te.PID})

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
