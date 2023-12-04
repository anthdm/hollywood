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
	subscribers []*actor.PID
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

		// add the subscriber to the slice
		pw.subscribers = append(pw.subscribers, msg.Sendto)

	case types.Unsubscribe:
		slog.Info("priceWatcher.Unsubscribe", "ticker", pw.ticker, "subscriber", msg.Sendto)

		// remove the subscriber from the slice
		for i, sub := range pw.subscribers {
			if sub == msg.Sendto {
				pw.subscribers = append(pw.subscribers[:i], pw.subscribers[i+1:]...)
				break
			}
		}

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
	for _, sub := range pw.subscribers {
		pw.ActorEngine.Send(sub, types.PriceUpdate{
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
			ticker: opts.Ticker,
			token0: opts.Token0,
			token1: opts.Token1,
			chain:  opts.Chain,
		}
	}
}
