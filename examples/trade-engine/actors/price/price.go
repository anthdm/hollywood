package price

import (
	"log/slog"
	"time"

	"github.com/anthdm/hollywood/actor"
)

type PriceOptions struct {
	Ticker string
	Token0 string
	Token1 string
	Chain  string
}

type FetchPriceRequest struct{}

type FetchPriceResponse struct {
	Iat   int64
	Price float64 // using float in example
}

type updatePrice struct{}

type priceWatcherActor struct {
	ActorEngine *actor.Engine
	PID         *actor.PID
	repeater    actor.SendRepeater
	ticker      string
	token0      string
	token1      string
	chain       string
	lastCall    int64
	lastPrice   float64
	updatedAt   int64
	callCount   int64
}

func (pw *priceWatcherActor) Receive(c *actor.Context) {

	switch msg := c.Message().(type) {
	case actor.Started:
		slog.Info("Started Price Actor", "ticker", pw.ticker)

		pw.ActorEngine = c.Engine()
		pw.lastCall = time.Now().UnixMilli()
		pw.PID = c.PID()

		pw.repeater = pw.ActorEngine.SendRepeat(pw.PID, updatePrice{}, time.Millisecond*20)

	case updatePrice:
		pw.refresh()

	case actor.Stopped:
		slog.Info("Stopped Price Actor", "ticker", pw.ticker)

	case FetchPriceRequest:
		slog.Info("Fetching Price Request", "ticker", pw.ticker)

		// update last called time
		pw.lastCall = time.Now().UnixMilli()

		// increment call count
		pw.callCount++

		// respond with the lastest price
		c.Respond(FetchPriceResponse{
			Iat:   time.Now().UnixMilli(),
			Price: pw.lastPrice,
		})

	default:
		_ = msg
	}
}

func (pw *priceWatcherActor) refresh() {

	// check if the last call was more than 10 seconds ago
	if pw.lastCall < time.Now().UnixMilli()-(time.Second.Milliseconds()*10) {
		slog.Warn("Inactivity: Killing Price Watcher", "ticker", pw.ticker, "callCount", pw.callCount)

		// if no call in 10 seconds => kill itself
		pw.Kill()
	}

	// mimic fetching the price every 2s
	time.Sleep(time.Millisecond * 2)
	pw.lastPrice += 0.0001
	pw.updatedAt = time.Now().UnixMilli()

}

func (pw *priceWatcherActor) Kill() {
	// kill itself
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

func NewPriceActor(opts PriceOptions) actor.Producer {
	return func() actor.Receiver {
		return &priceWatcherActor{
			ticker: opts.Ticker,
			token0: opts.Token0,
			token1: opts.Token1,
			chain:  opts.Chain,
		}
	}
}
