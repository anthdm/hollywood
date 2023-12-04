package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/examples/trade-engine/actors/tradeEngine"
	"github.com/anthdm/hollywood/examples/trade-engine/types"
	"github.com/anthdm/hollywood/log"
)

func main() {
	// example script showing the trade engine in action
	// this script creates a trade order, gets the trade info and then cancels the order

	// set up log handler
	logHandler := log.NewHandler(os.Stdout, log.TextFormat, slog.LevelInfo)

	// create the actor engine
	e, err := actor.NewEngine(actor.EngineOptLogger(log.NewLogger("[engine]", logHandler)))
	if err != nil {
		fmt.Printf("failed to create actor engine: %v", err)
		os.Exit(1)
	}

	// spawn the trade engine
	tradeEnginePID := e.Spawn(tradeEngine.NewTradeEngine(), "trade-engine")

	// create the trade order
	fmt.Println("Creating 1 trade order to test getting trade info and cancelling")
	tradeOrder := types.TradeOrderRequest{
		TradeID:    GenID(),
		Token0:     "token0",
		Token1:     "token1",
		Chain:      "ETH",
		Wallet:     "wallet6",
		PrivateKey: "privateKey",
		// Expires:    (the zero time indicate no expiry)
	}

	// send the trade order
	e.Send(tradeEnginePID, tradeOrder)

	// get trade info
	fmt.Println("\nGetting trade info")

	// send a TradeInfoRequest to the tradeEngine
	resp := e.Request(tradeEnginePID, types.TradeInfoRequest{TradeID: tradeOrder.TradeID}, 5*time.Second)

	// wait for response
	res, err := resp.Result()
	if err != nil {
		fmt.Printf("Failed to get trade info: %v", err)
		os.Exit(1)
	}

	tradeInfo, ok := res.(types.TradeInfoResponse)
	if !ok {
		fmt.Printf("Failed to cast response to TradeInfoResponse")
		os.Exit(1)
	}
	fmt.Printf("Trade info: %+v\n", tradeInfo)

	// small delay before cancelling
	time.Sleep(2 * time.Second)

	fmt.Println("\nCancelling trade order")
	e.Send(tradeEnginePID, types.CancelOrderRequest{TradeID: tradeOrder.TradeID})

	time.Sleep(2 * time.Second)

}

// The GenID function generates a random ID string of length 16 using a cryptographic random number
// generator.
func GenID() string {
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	return hex.EncodeToString(id)
}
