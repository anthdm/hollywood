package price

import (
	"fmt"
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

type priceWatcher struct {
	actorEngine *actor.Engine
	PID         *actor.PID
	ticker      string
	token0      string
	token1      string
	chain       string
	states      *priceStates
}

func (pw *priceWatcher) Receive(c *actor.Context) {

	switch msg := c.Message().(type) {
	case actor.Started:
		slog.Info("Started Price Actor", "ticker", pw.ticker)

		pw.actorEngine = c.Engine()
		pw.states.SetLastCall(time.Now().UnixMilli())
		pw.PID = c.PID()

		// start updating the price
		go pw.start()

	case actor.Stopped:
		slog.Info("Stopped Price Actor", "ticker", pw.ticker)

	case FetchPriceRequest:
		slog.Info("Fetching Price Request", "ticker", pw.ticker)

		// update last called time
		pw.states.SetLastCall(time.Now().UnixMilli())

		// increment call count
		pw.states.IncCallCount()

		// respond with the lastest price
		c.Respond(&FetchPriceResponse{
			Iat:   time.Now().UnixMilli(),
			Price: pw.states.LastPrice(),
		})

	default:
		_ = msg
	}
}

func (pw *priceWatcher) start() {
	fmt.Println("starting price watcher", pw.ticker)

	// mimic getting price every 2 seconds
	for {
		// check if the last call was more than 10 seconds ago
		if pw.states.LastCall() < time.Now().UnixMilli()-(time.Second.Milliseconds()*10) {
			slog.Warn("Inactivity: Killing Price Watcher", "ticker", pw.ticker, "callCount", pw.states.CallCount())

			// if no call in 10 seconds => kill itself
			pw.Kill()
			return // stops goroutine
		}

		// mimic fetching the price every 2s
		time.Sleep(time.Millisecond * 2)
		pw.states.SetLastPrice(10)
		pw.states.SetUpdatedAt(time.Now().UnixMilli())

	}
}

func (pw *priceWatcher) Kill() {
	// kill itself
	if pw.actorEngine == nil {
		slog.Error("priceWatcher.actorEngine is <nil>", "ticker", pw.ticker)
	}
	if pw.PID == nil {
		slog.Error("priceWatcher.PID is <nil>", "ticker", pw.ticker)
	}

	pw.actorEngine.Poison(pw.PID)
}

func NewPriceActor(opts PriceOptions) actor.Producer {
	return func() actor.Receiver {
		return &priceWatcher{
			ticker: opts.Ticker,
			token0: opts.Token0,
			token1: opts.Token1,
			chain:  opts.Chain,
			states: NewPriceStates(),
		}
	}
}
