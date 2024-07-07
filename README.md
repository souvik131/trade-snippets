Golang Kite CLI

Download and Unzip build.zip

Set Kite Creds

```
TA_ID=              //Kite Username
TA_PASSWORD=        //Kite Password
TA_TOTP=            //Kite TOTP Secret ( not  OTP )
TA_APIKEY=          //API key shared in kite.trade
TA_APISECRET=       //API secret shared in kite.trade
TA_PATH=            //API path you want to run on. eg /kite, For this path URL in kite.trade set should be http://127.0.0.1/kite
TA_PORT=80          //Port you want the application to host

```

For Mac

```
./mac
```

For linux

```
./linux
```

For windows

Run the win.exe file

Additional Examples in Golang

- Kite login

  ```
  err := kite.Login(&ctx,router)
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
