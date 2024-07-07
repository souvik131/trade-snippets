package main

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/souvik131/trade-snippets/kite"
)

var k = &kite.Kite{}
var inputs = []string{"Id", "Password", "Totp", "ApiKey", "ApiSecret", "Path"}

func main() {

	err := godotenv.Load("creds.txt")
	if err != nil {
		log.Fatalf("Error loading creds.txt file: %s", err)
	}
	for _, input := range inputs {
		val := strings.TrimSpace(os.Getenv("TA_" + strings.ToUpper(input)))
		if val == "" {
			log.Fatalln("Please ensure creds.txt file has all the creds including ", "TA_"+strings.ToUpper(input))
		}
		(*k)[input] = val
	}

	port := strings.TrimSpace(os.Getenv("TA_PORT"))

	portString := ""
	if port != "80" {
		portString = ":" + port
	}
	color.Yellow("Ensure that the URL set in kite.trade is http://127.0.0.1" + portString + (*k)["Path"])
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	go func() {
		router.Run("0.0.0.0:" + port)
	}()
	time.Sleep(1 * time.Second)
	ctx := context.Background()
	err = k.Login(&ctx, router)
	if err != nil {
		log.Fatalln(err)
	}
	color.Green("Request Token : %s", (*k)["RequestToken"])
	color.Green("Access Token : %s", (*k)["AccessToken"])

	// pnl, err := k.GetPnl(&ctx)
	// log.Println(pnl, err)

	// charges, err := k.GetCharges(&ctx)
	// log.Println(charges, err)

	// margin, err := k.GetMargin(&ctx)
	// log.Println(margin, err)

	// positions, err := k.GetPositions(&ctx)
	// for _, pos := range positions {
	// 	log.Printf("%+v, %v", pos, err)
	// }

	// orders, err := k.GetOrders(&ctx)
	// for _, order := range orders {
	// 	log.Printf("%+v, %v", order, err)
	// }

	// price, err := k.GetLastPrice(&ctx, "NSE", "ZOMATO")
	// log.Println(price, err)

	// price, err = k.GetMidPrice(&ctx, "NSE", "ZOMATO")
	// log.Println(price, err)

	// resp, err := k.PlaceOrder(&ctx, &kite.Order{
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

	// resp, err = k.PlaceOrder(&ctx, &kite.Order{
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

	// resp, err = k.PlaceOrder(&ctx, &kite.Order{
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
