package price

import (
	"log/slog"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/trade-engine/types"
)

type priceWatcherActor struct {
	ActorEngine *actor.Engine
	PID         *actor.PID
	repeater    actor.SendRepeater
	ticker      string
	token0      string
	token1      string
	chain       string
	lastPrice   float64
	updatedAt   int64
	subscribers map[*actor.PID]bool
}

func (pw *priceWatcherActor) Receive(c *actor.Context) {

	switch msg := c.Message().(type) {
	case actor.Started:
		slog.Info("priceWatcher.Started", "ticker", pw.ticker)

		// set actorEngine and PID
		pw.ActorEngine = c.Engine()
		pw.PID = c.PID()

		// create a repeater to trigger price updates every 200ms
		pw.repeater = pw.ActorEngine.SendRepeat(pw.PID, types.TriggerPriceUpdate{}, time.Millisecond*200)

	case actor.Stopped:
		slog.Info("priceWatcher.Stopped", "ticker", pw.ticker)

	case types.Subscribe:
		slog.Info("priceWatcher.Subscribe", "ticker", pw.ticker, "subscriber", msg.Sendto)

		// add the subscriber to the map
		pw.subscribers[msg.Sendto] = true

	case types.Unsubscribe:
		slog.Info("priceWatcher.Unsubscribe", "ticker", pw.ticker, "subscriber", msg.Sendto)

		// remove the subscriber from the map
		delete(pw.subscribers, msg.Sendto)

	case types.TriggerPriceUpdate:
		pw.refresh()
	}
}

func (pw *priceWatcherActor) refresh() {

	// check if there are any subscribers
	if len(pw.subscribers) == 0 {
		slog.Info("No Subscribers: Killing Price Watcher", "ticker", pw.ticker)

		// if no subscribers, kill itself
		pw.Kill()
	}

	// for example, just increment the price by 2
	pw.lastPrice += 2
	pw.updatedAt = time.Now().UnixMilli()

	// send the price update to all executors
	for pid := range pw.subscribers {
		pw.ActorEngine.Send(pid, types.PriceUpdate{
			Ticker:    pw.ticker,
			UpdatedAt: pw.updatedAt,
			Price:     pw.lastPrice,
		})
	}
}

func (pw *priceWatcherActor) Kill() {
	if pw.ActorEngine == nil {
		slog.Error("priceWatcher.actorEngine is <nil>", "ticker", pw.ticker)
	}
	if pw.PID == nil {
		slog.Error("priceWatcher.PID is <nil>", "ticker", pw.ticker)
	}

	// stop the repeater
	pw.repeater.Stop()

	// poision itself
	pw.ActorEngine.Poison(pw.PID)
}

func NewPriceActor(opts types.PriceOptions) actor.Producer {
	return func() actor.Receiver {
		return &priceWatcherActor{
			ticker:      opts.Ticker,
			token0:      opts.Token0,
			token1:      opts.Token1,
			chain:       opts.Chain,
			subscribers: make(map[*actor.PID]bool),
		}
	}
}
