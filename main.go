package main

import (
	"context"
	"fmt"
	"sync"
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

	k.TickerClients = []*kite.TickerClient{}
	i := 0
	// for i := 0; i < 2; i++ {
	ticker, err := k.GetWebSocketClient(&ctx)
	if err != nil {
		color.Red(fmt.Sprintf("%v", err))
		return
	}
	ticker.Id = i
	k.TickerClients = append(k.TickerClients, ticker)
	k.TickSymbolMap = map[string]kite.KiteTicker{}
	go func(t *kite.TickerClient) {
		for range t.ConnectChan {
			color.HiBlue(fmt.Sprintf("Websocket is connected in %v seconds %v", time.Since(start).Seconds(), i))
			color.HiCyan("Subscribing Ticks")
			start = time.Now()
			t.SubscribeLTP(&ctx, []string{"NIFTY 50", "TATAMOTORS"})
			t.SubscribeQuote(&ctx, []string{"INFY"})
			t.SubscribeFull(&ctx, []string{"ACC"})
		}
	}(k.TickerClients[i])
	go func(t *kite.TickerClient) {
		for tick := range t.TickerChan {
			color.HiWhite(fmt.Sprintf("\nTick %v: %+v %v\n", tick.TradingSymbol, tick, i))
			t.Unsubscribe(&ctx, []string{"NIFTY 50", "TATAMOTORS", "INFY"})
			color.HiCyan(fmt.Sprintf("Received in %v secs", time.Since(start).Seconds()))
		}
	}(k.TickerClients[i])
	go k.TickerClients[i].Serve(&ctx)
	<-time.After(time.Second)
	// }
	wg := &sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()

}
