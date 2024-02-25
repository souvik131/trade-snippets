Golang Snippets

- Kite order placement

  Test by running

  ```
  go test -v
  ```

  Example order

  ```
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
  ```
