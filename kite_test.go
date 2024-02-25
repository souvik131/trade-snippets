package kite

import (
	"context"
	"log"
	"testing"
)

var kite = &Kite{
	BaseUrl:  "https://api.kite.trade",
	Id:       "<USER_ID>",
	Password: "<PASSWORD>",
	Totp:     "<TOTP>",
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	err := kite.Login(&ctx)
	log.Println(err)

	price, err := kite.GetMidPrice(&ctx, "NSE", "ZOMATO")
	log.Println(price, err)

	resp, err := kite.PlaceOrder(&ctx, &Order{
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

	resp, err = kite.PlaceOrder(&ctx, &Order{
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

	resp, err = kite.PlaceOrder(&ctx, &Order{
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
}
