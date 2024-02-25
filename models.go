package kite

type Kite struct {
	Token       string
	PublicToken string
	BaseUrl     string
	Id          string
	Password    string
	Totp        string
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

type QuoteResponsePayload struct {
	Status    string `json:"error"`
	Message   string `json:"message"`
	ErrorType string `json:"error_type"`
	Data      *map[string]struct {
		Depth struct {
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
