package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/souvik131/trade-snippets/kite"
)

var k = &kite.Kite{
	Id:        "<ID>",
	Password:  "<PASSWORD>",
	Totp:      "<TOTP>",
	ApiKey:    "<API_KEY>",
	ApiSecret: "<API_SECRET>",
	Path:      "<PATH>", //save http://127.0.0.1<PATH> in kite.trade
}

func main() {

	wg := &sync.WaitGroup{}
	wg.Add(1)
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	go func() {
		router.Run("0.0.0.0:80")
		wg.Done()
	}()
	time.Sleep(1 * time.Second)

	ctx := context.Background()
	err := k.Login(&ctx, router)
	log.Println(err)

	pnl, err := k.GetPnl(&ctx)
	log.Println(pnl, err)

	charges, err := k.GetCharges(&ctx)
	log.Println(charges, err)

	margin, err := k.GetMargin(&ctx)
	log.Println(margin, err)

	positions, err := k.GetPositions(&ctx)
	for _, pos := range positions {
		log.Printf("%+v, %v", pos, err)
	}

	orders, err := k.GetOrders(&ctx)
	for _, order := range orders {
		log.Printf("%+v, %v", order, err)
	}

	price, err := k.GetLastPrice(&ctx, "NSE", "ZOMATO")
	log.Println(price, err)

	price, err = k.GetMidPrice(&ctx, "NSE", "ZOMATO")
	log.Println(price, err)

	resp, err := k.PlaceOrder(&ctx, &kite.Order{
		Exchange:                   "NSE",
		TradingSymbol:              "ZOMATO",
		Quantity:                   50,
		MarketProtectionPercentage: 5,
		TickSize:                   0.05,
		TransactionType:            "BUY",
		Product:                    "CNC",
		OrderType:                  "MARKET",
	})
	log.Println(resp, err)

	resp, err = k.PlaceOrder(&ctx, &kite.Order{
		Exchange:                   "NSE",
		TradingSymbol:              "ZOMATO",
		Price:                      160,
		Quantity:                   50,
		MarketProtectionPercentage: 5,
		TickSize:                   0.05,
		TransactionType:            "BUY",
		Product:                    "CNC",
		OrderType:                  "LIMIT",
	})
	log.Println(resp, err)

	resp, err = k.PlaceOrder(&ctx, &kite.Order{
		Exchange:                   "NSE",
		TradingSymbol:              "ZOMATO",
		Price:                      160,
		Quantity:                   50,
		MarketProtectionPercentage: 5,
		TickSize:                   0.05,
		TransactionType:            "BUY",
		Product:                    "CNC",
		OrderType:                  "SL",
	})
	log.Println(resp, err)
	wg.Wait()
}
