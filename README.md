Golang Snippets

Test command

```
go test -v
```

Creds

```
var kite = &Kite{
	BaseUrl:  "https://api.kite.trade",
	Id:       "<USER_ID>",
	Password: "<PASSWORD>",
	Totp:     "<TOTP>",
}
```

- Kite login

  ```
  err := kite.Login(&ctx)
  ```

- Kite Get Mid Price ( Between Offer an Bid )

  ```
  price, err := kite.GetMidPrice(&ctx, "NSE", "ZOMATO")
  ```

- Kite Order Placement

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
