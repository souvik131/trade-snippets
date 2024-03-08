package kite

type Kite struct {
	Token       string
	PublicToken string
	BaseUrl     string
	Id          string
	Password    string
	Totp        string
}

type Margin struct {
	MarginUsed  float64
	MarginTotal float64
}

type Order struct {
	Exchange                   string
	TradingSymbol              string
	Quantity                   float64
	Price                      float64
	MarketProtectionPercentage float64
	TickSize                   float64
	TransactionType            string
	Product                    string
	OrderType                  string
}

type OrderPayload struct {
	Exchange          string `query:"exchange"`
	TradingSymbol     string `query:"tradingsymbol"`
	TransactionType   string `query:"transaction_type"`
	Product           string `query:"product"`
	Quantity          string `query:"quantity"`
	Price             string `query:"price"`
	Variety           string `query:"variety"`
	OrderType         string `query:"order_type"`
	Validity          string `query:"validity"`
	DisclosedQuantity string `query:"disclosed_quantity"`
	TriggerPrice      string `query:"trigger_price"`
	SquareOff         string `query:"squareoff"`
	StopLoss          string `query:"stoploss"`
	TrailingStopLoss  string `query:"trailing_stoploss"`
}

type Equity struct {
	Net       float64 `json:"net"`
	Available *struct {
		Cash       float64 `json:"cash"`
		Collateral float64 `json:"collateral"`
	} `json:"available"`
	Utilised *struct {
		Debits float64 `json:"debits"`
	} `json:"utilised"`
}

type ChargesOrderRequest struct {
	OrderId         string  `json:"order_id"`
	Variety         string  `json:"variety"`
	Exchange        string  `json:"exchange"`
	TradingSymbol   string  `json:"tradingsymbol"`
	Product         string  `json:"product"`
	Quantity        uint32  `json:"quantity"`
	AveragePrice    float64 `json:"average_price"`
	OrderType       string  `json:"order_type"`
	TransactionType string  `json:"transaction_type"`
}

type Position struct {
	TradingSymbol     string  `json:"tradingsymbol"`
	Exchange          string  `json:"exchange"`
	InstrumentToken   uint32  `json:"instrument_token"`
	Product           string  `json:"product"`
	Quantity          int64   `json:"quantity"`
	OvernightQuantity int64   `json:"overnight_quantity"`
	Multiplier        int64   `json:"multiplier"`
	AveragePrice      float64 `json:"average_price"`
	ClosePrice        float64 `json:"close_price"`
	LastPrice         float64 `json:"last_price"`
	Value             float64 `json:"value"`
	Pnl               float64 `json:"pnl"`
	M2m               float64 `json:"m2m"`
	Unrealized        float64 `json:"unrealised"`
	Realised          float64 `json:"realised"`
	BuyQuantity       int64   `json:"buy_quantity"`
	BuyPrice          float64 `json:"buy_price"`
	BuyValue          float64 `json:"buy_value"`
	BuyM2m            float64 `json:"buy_m2m"`
	SellQuantity      int64   `json:"sell_quantity"`
	SellPrice         float64 `json:"sell_price"`
	SellValue         float64 `json:"sell_value"`
	SellM2m           float64 `json:"sell_m2m"`
	DayBuyQuantity    int64   `json:"day_buy_quantity"`
	DayBuyPrice       float64 `json:"day_buy_price"`
	DayBuyValue       float64 `json:"day_buy_value"`
	DaySellQuantity   int64   `json:"day_sell_quantity"`
	DaySellPrice      float64 `json:"day_sell_price"`
	DaySellValue      float64 `json:"day_sell_value"`
}

type QuoteResponsePayload struct {
	Status    string `json:"error"`
	Message   string `json:"message"`
	ErrorType string `json:"error_type"`
	Data      *map[string]struct {
		LastPrice float64 `json:"last_price"`
		Depth     struct {
			Buy []struct {
				Price float64 `json:"price"`
			} `json:"buy"`
			Sell []struct {
				Price float64 `json:"price"`
			} `json:"sell"`
		} `json:"depth"`
	} `json:"data"`
}
type OrderResponsePayload struct {
	Status    string `json:"error"`
	Message   string `json:"message"`
	ErrorType string `json:"error_type"`
	Data      *struct {
		OrderId string `json:"order_id"`
	} `json:"data"`
}
type OrderStatus struct {
	PlacedBy                string  `json:"placed_by"`
	OrderId                 string  `json:"order_id"`
	ExchangeOrderId         string  `json:"exchange_order_id"`
	OrderState              string  `json:"status"`
	Remarks                 string  `json:"status_message"`
	OrderTimestamp          string  `json:"order_timestamp"`
	ExchangeUpdateTimestamp string  `json:"exchange_update_timestamp"`
	ExchangeTimestamp       string  `json:"exchange_timestamp"`
	Variety                 string  `json:"variety"`
	Modified                bool    `json:"modified"`
	Exchange                string  `json:"exchange"`
	TradingSymbol           string  `json:"tradingsymbol"`
	InstrumentToken         uint32  `json:"instrument_token"`
	OrderType               string  `json:"order_type"`
	TransactionType         string  `json:"transaction_type"`
	Validity                string  `json:"validity"`
	ValidityTTL             uint64  `json:"validity_ttl"`
	Product                 string  `json:"product"`
	Quantity                uint32  `json:"quantity"`
	DisclosedQuantity       uint32  `json:"disclosed_quantity"`
	Price                   float64 `json:"price"`
	TriggerPrice            float64 `json:"trigger_price"`
	AveragePrice            float64 `json:"average_price"`
	FilledQuantity          uint32  `json:"filled_quantity"`
	PendingQuantity         uint32  `json:"pending_quantity"`
	CancelledQuantity       uint32  `json:"cancelled_quantity"`
	MarketProtection        float64 `json:"market_protection"`
	Guid                    string  `json:"guid"`
}

type BrokerCharges struct {
	Charges *struct {
		Total float64 `json:"total"`
	} `json:"charges"`
}

type BrokerChargesPayload struct {
	Status    string           `json:"error"`
	Message   string           `json:"message"`
	ErrorType string           `json:"error_type"`
	Data      []*BrokerCharges `json:"data"`
}

type OrdersResponsePayload struct {
	Status    string         `json:"error"`
	Message   string         `json:"message"`
	ErrorType string         `json:"error_type"`
	Data      []*OrderStatus `json:"data"`
}

type PositionResponsePayload struct {
	Status    string `json:"error"`
	Message   string `json:"message"`
	ErrorType string `json:"error_type"`
	Data      *struct {
		Net []*Position `json:"net"`
		Day []*Position `json:"day"`
	} `json:"data"`
}

type MarginResponsePayload struct {
	Status    string `json:"error"`
	Message   string `json:"message"`
	ErrorType string `json:"error_type"`
	Data      *struct {
		Equity *Equity `json:"equity"`
	} `json:"data"`
}

type LoginPayload struct {
	Status    string `json:"error"`
	Message   string `json:"message"`
	ErrorType string `json:"error_type"`
	Data      *struct {
		RequestId string `json:"request_id"`
	} `json:"data"`
}

type TFAPayload struct {
	Status string `json:"status"`
}
