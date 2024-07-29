Lightweight Golang Kite Library for Web and API

Set Kite Creds (rename .env_example to .env and set following creds)

```
TA_ID=              //Kite Username
TA_PASSWORD=        //Kite Password
TA_TOTP=            //Kite TOTP Secret ( not  OTP )
TA_APIKEY=          //API key shared in kite.trade
TA_APISECRET=       //API secret shared in kite.trade
TA_PATH=            //API path you want to run on. eg /kite, For this path URL in kite.trade set should be http://127.0.0.1/kite
TA_PORT=            //Port you want the application to host
TA_LOGINTYPE=       //Mention Login Type ( WEB / API)

```

Run Docker

```
docker-compose up -d
```

Web & API Support

```
  //http
  Login(ctx *context.Context) error
  PlaceOrder(ctx *context.Context, order *Order) (string, error)
  ModifyOrder(ctx *context.Context, orderId string, order *Order) error
  CancelOrder(ctx *context.Context, orderId string) error
  GetPositions(ctx *context.Context) error
  GetOrders(ctx *context.Context) ([]*OrderStatus, error)
  GetOrderHistory(ctx *context.Context, orderId string) ([]*OrderStatus, error)
  GetMargin(ctx *context.Context) (*Margin, error)
  GetCharges(ctx *context.Context) (float64, error)

  //websocket
  SubscribeLTP(ctx *context.Context, tokens []string) error
  SubscribeFull(ctx *context.Context, tokens []string) error
  SubscribeQuote(ctx *context.Context, tokens []string) error
  Unsubscribe(ctx *context.Context, tokens []string) error
```

Only API Support

```
  //http
  GetQuote(ctx *context.Context, exchange string, tradingSymbol string) (*Quote, error)
  GetLastPrice(ctx *context.Context, exchange string, tradingSymbol string) (float64, error)
```
