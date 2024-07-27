package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/fatih/color"
	"github.com/souvik131/trade-snippets/kite"
)

func main() {

	var k = &kite.Kite{}
	ctx := context.Background()
	err := k.Login(&ctx)
	if err != nil {
		color.Red(fmt.Sprintf("%s", err))
		return
	}

	k.TickerClients = []*kite.TickerClient{}

	Serve(&ctx, k)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()

}
