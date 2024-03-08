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
	Totp:     "<TOTP_KEY>",
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	err := kite.Login(&ctx)
	log.Println(err)

	pnl, err := kite.GetPnl(&ctx)
	log.Println(pnl, err)

	charges, err := kite.GetCharges(&ctx)
	log.Println(charges, err)

	margin, err := kite.GetMargin(&ctx)
	log.Println(margin, err)

	positions, err := kite.GetPositions(&ctx)
	for _, pos := range positions {
		log.Printf("%+v, %v", pos, err)
	}

	orders, err := kite.GetOrders(&ctx)
	for _, order := range orders {
		log.Printf("%+v, %v", order, err)
	}

	price, err := kite.GetLastPrice(&ctx, "NSE", "ZOMATO")
	log.Println(price, err)

	// price, err := kite.GetMidPrice(&ctx, "NSE", "ZOMATO")
	// log.Println(price, err)

	// resp, err := kite.PlaceOrder(&ctx, &Order{
	// 	Exchange:                   "NSE",
	// 	TradingSymbol:              "ZOMATO",
	// 	Quantity:                   50,
	// 	MarketProtectionPercentage: 5,
	// 	TickSize:                   0.05,
	// 	TransactionType:            "BUY",
	// 	Product:                    "CNC",
	// 	OrderType:                  "MARKET",
	// })
	// log.Println(resp, err)

	// resp, err = kite.PlaceOrder(&ctx, &Order{
	// 	Exchange:                   "NSE",
	// 	TradingSymbol:              "ZOMATO",
	// 	Price:                      160,
	// 	Quantity:                   50,
	// 	MarketProtectionPercentage: 5,
	// 	TickSize:                   0.05,
	// 	TransactionType:            "BUY",
	// 	Product:                    "CNC",
	// 	OrderType:                  "LIMIT",
	// })
	// log.Println(resp, err)

	// resp, err = kite.PlaceOrder(&ctx, &Order{
	// 	Exchange:                   "NSE",
	// 	TradingSymbol:              "ZOMATO",
	// 	Price:                      160,
	// 	Quantity:                   50,
	// 	MarketProtectionPercentage: 5,
	// 	TickSize:                   0.05,
	// 	TransactionType:            "BUY",
	// 	Product:                    "CNC",
	// 	OrderType:                  "SL",
	// })
	// log.Println(resp, err)
}
