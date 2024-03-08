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
	Totp:     "<TOTP_KEY>",
}
```

- Kite login

  ```
  err := kite.Login(&ctx)
  ```

- Kite Get Pnl

  ```
  pnl, err := kite.GetPnl(&ctx)
  ```

- Kite Get Charges

  ```
  charges, err := kite.GetCharges(&ctx)
  ```

- Kite Get Margin

  ```
  margin, err := kite.GetMargin(&ctx)
  ```

- Kite Get Positions

  ```
  positions, err := kite.GetPositions(&ctx)
  ```

- Kite Get Orders

  ```
  orders, err := kite.GetOrders(&ctx)
  ```

- Kite Get Last Price

  ```
  price, err := kite.GetLastPrice(&ctx, "NSE", "ZOMATO")
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
