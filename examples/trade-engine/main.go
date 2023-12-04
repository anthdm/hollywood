package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/trade-engine/actors/tradeEngine"
	"github.com/anthdm/hollywood/examples/trade-engine/types"
	"github.com/anthdm/hollywood/log"
)

func main() {
	done := make(chan struct{}, 1)

	logHandler := log.NewHandler(os.Stdout, log.TextFormat, slog.LevelInfo)
	e, err := actor.NewEngine(actor.EngineOptLogger(log.NewLogger("[engine]", logHandler)))
	if err != nil {
		panic(err)
	}

	tradeEnginePID := e.Spawn(tradeEngine.NewTradeEngine(), "trade-engine")

	// create 5 trade orders
	// Expiry of 10s so after 10s the orders will be cancelled
	// the price watcher will be stopped due to inactivity
	fmt.Println("\n\ncreating 5 trade orders")
	for i := 1; i < 6; i++ {
		o := types.TradeOrderRequest{
			TradeID:    GenID(),
			Token0:     "token0",
			Token1:     "token1",
			Chain:      "ETH",
			Wallet:     fmt.Sprintf("wallet_%d", i),     // for example
			PrivateKey: fmt.Sprintf("privateKey_%d", i), // for example
			// expire after 10 seconds
			Expires: time.Now().Add(time.Second * 10),
		}

		e.Send(tradeEnginePID, o)
	}

	time.Sleep(time.Second * 20)
	fmt.Println("\n\ncreating 1 trade order to test cancellation")
	tradeOrder := types.TradeOrderRequest{
		TradeID:    GenID(),
		Token0:     "token0",
		Token1:     "token1",
		Chain:      "ETH",
		Wallet:     "wallet6",
		PrivateKey: "privateKey",
		// Expires:    (the zero time indicate no expiry)
	}

	e.Send(tradeEnginePID, tradeOrder)
	time.Sleep(time.Second * 5)

	// get trade info
	fmt.Println("\n\ngetting trade info")
	resp := e.Request(tradeEnginePID, types.TradeInfoRequest{TradeID: tradeOrder.TradeID}, 5*time.Second)
	res, _ := resp.Result()
	switch msg := res.(type) {
	case types.TradeInfoResponse:
		fmt.Println("trade info", msg)
	}

	// cancel trade order
	fmt.Println("\n\ncancelling trade order")
	e.Send(tradeEnginePID, types.CancelOrderRequest{TradeID: tradeOrder.TradeID})

	// wait for signal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		call := <-sigs
		slog.Info("received signal", "signal", call)

		wg := &sync.WaitGroup{}
		e.Poison(tradeEnginePID, wg)
		wg.Wait()

		slog.Info("shutdown completed")

		done <- struct{}{}
	}()

	// wait until done
	<-done
}

// The GenID function generates a random ID string of length 16 using a cryptographic random number
// generator.
func GenID() string {
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	return hex.EncodeToString(id)
}
