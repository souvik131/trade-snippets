package main

import (
	"context"
	"log"
	"sync"

	"github.com/souvik131/trade-snippets/kite"
)

func main() {

	var k = &kite.Kite{}
	ctx := context.Background()
	err := k.Login(&ctx)
	if err != nil {
		log.Panicf("%s", err)
		return
	}

	Serve(&ctx, k)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()

}
