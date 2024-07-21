package main

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/souvik131/trade-snippets/kite"
)

func main() {

	start := time.Now()
	var k = &kite.Kite{}
	ctx := context.Background()
	err := k.Login(&ctx)
	if err != nil {
		color.Red(fmt.Sprintf("%s", err))
		return
	}

	color.Green(fmt.Sprintf("Account login took %v seconds", time.Since(start).Seconds()))

	k.TickSymbolMap = map[string]kite.KiteTicker{}
	go func() {
		for range k.TickerClient.ConnectChan {

			color.HiBlue(fmt.Sprintf("Websocket is connected in %v seconds", time.Since(start).Seconds()))
			color.HiCyan("Subscribing Ticks")
			start = time.Now()
			k.TickerClient.SubscribeLTP(&ctx, []string{"NIFTY 50", "TATAMOTORS"})
			k.TickerClient.SubscribeQuote(&ctx, []string{"INFY"})
			k.TickerClient.SubscribeFull(&ctx, []string{"ACC"})
		}
	}()
	go func() {
		for tick := range k.TickerClient.TickerChan {
			color.HiWhite(fmt.Sprintf("\nTick %v: %+v\n", tick.TradingSymbol, tick))
			k.TickerClient.Unsubscribe(&ctx, []string{"NIFTY 50", "TATAMOTORS", "INFY"})
			color.HiCyan(fmt.Sprintf("Received in %v secs", time.Since(start).Seconds()))
		}
	}()
	k.TickerClient.Serve(&ctx)

}
