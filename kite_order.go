package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"reflect"
	"strings"

	"github.com/valyala/fasthttp"
)

type Kite struct {
	Token   string
	BaseUrl string
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
type OrderResponsePayload struct {
	Status    string `json:"error"`
	Message   string `json:"message"`
	ErrorType string `json:"error_type"`
	Data      *struct {
		OrderId string `json:"order_id"`
	} `json:"data"`
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

func (kite *Kite) PlaceOrder(ctx *context.Context, order *Order) (string, error) {
	k := *kite

	kOrder := &OrderPayload{
		Exchange:          order.Exchange,
		TradingSymbol:     order.TradingSymbol,
		TransactionType:   string(order.TransactionType),
		Product:           string(order.Product),
		Quantity:          fmt.Sprintf("%v", order.Quantity),
		OrderType:         string(order.OrderType),
		Price:             "0",
		Variety:           "regular",
		Validity:          "DAY",
		DisclosedQuantity: "0",
		TriggerPrice:      "0",
		SquareOff:         "0",
		StopLoss:          "0",
		TrailingStopLoss:  "0",
	}
	tickSize := order.TickSize
	mpp := order.MarketProtectionPercentage

	switch kOrder.OrderType {
	case "LIMIT":
		kOrder.Price = fmt.Sprintf("%v", order.Price)
	case "MARKET":
		lastPrice, err := k.GetMidPrice(ctx, kOrder.Exchange, kOrder.TradingSymbol)
		if err != nil {
			return "", err
		}
		if kOrder.TransactionType == "BUY" {
			kOrder.Price = fmt.Sprintf("%v", math.Floor((lastPrice*(1+mpp/100))/tickSize)*tickSize)
		}
		if kOrder.TransactionType == "SELL" {
			kOrder.Price = fmt.Sprintf("%v", math.Ceil((lastPrice*(1-mpp/100))/tickSize)*tickSize)
		}
		kOrder.OrderType = "LIMIT"
	case "SL":
		if kOrder.TransactionType == "BUY" {
			kOrder.Price = fmt.Sprintf("%v", math.Floor((order.Price*(1+mpp/100))/tickSize)*tickSize)
			kOrder.TriggerPrice = fmt.Sprintf("%v", order.Price)
		}
		if kOrder.TransactionType == "SELL" {
			kOrder.Price = fmt.Sprintf("%v", math.Floor((order.Price*(1-mpp/100))/tickSize)*tickSize)
			kOrder.TriggerPrice = fmt.Sprintf("%v", order.Price)
		}
	default:
		return "", errors.New("order_type_not_allowed")
	}

	log.Printf("Placing the following order : %+v", kOrder)

	url := k.BaseUrl + "/orders/" + kOrder.Variety
	queries := make([]string, 0)
	typ := reflect.TypeOf(*kOrder)
	val := reflect.ValueOf(kOrder).Elem()
	for i := 0; i < val.NumField(); i++ {
		fieldName := val.Type().Field(i).Name
		ft, _ := typ.FieldByName(fieldName)
		fv := val.FieldByName(fieldName)
		queries = append(queries, fmt.Sprintf("%v=%v", ft.Tag.Get("query"), fv))
	}
	payload := strings.Join(queries, "&")
	headers := make(map[string]string)
	headers["authorization"] = k.Token
	headers["content-type"] = "application/x-www-form-urlencoded"

	response, code, err := Post(ctx, url, payload, headers)

	if err != nil {
		return "", err
	}

	var kiteResponse *OrderResponsePayload
	err = json.Unmarshal(response, &kiteResponse)
	if err != nil {
		return "", err
	}
	if code == 200 && kiteResponse.Data != nil && kiteResponse.Data.OrderId != "" {
		return kiteResponse.Data.OrderId, nil
	}
	return "", errors.New(kiteResponse.Message)
}

func (kite *Kite) GetMidPrice(ctx *context.Context, exchange string, tradingSymbol string) (float64, error) {
	k := *kite
	url := k.BaseUrl + "/quote?i=" + exchange + ":" + tradingSymbol
	headers := make(map[string]string)
	headers["authorization"] = k.Token
	headers["content-type"] = "application/x-www-form-urlencoded"

	response, _, err := Get(ctx, url, headers)

	if err != nil {
		return 0.0, err
	}
	var respData *QuoteResponsePayload
	err = json.Unmarshal(response, &respData)
	if err != nil {
		return 0, err
	}

	if respData.Data == nil {
		return 0, errors.New(respData.Message)

	}
	price := 0.0
	depth := (*respData.Data)[exchange+":"+tradingSymbol].Depth
	if len(depth.Buy) > 0 && len(depth.Sell) > 0 {
		price = (depth.Buy[0].Price + depth.Sell[0].Price) / 2
	}

	if price == 0 {
		return 0, errors.New("offer_bid_price_zero")
	}
	return price, nil
}
func Get(ctx *context.Context, urlLink string, headers map[string]string) ([]byte, int, error) {
	req := fasthttp.AcquireRequest()
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	req.SetRequestURI(urlLink)
	resp := fasthttp.AcquireResponse()
	if err := fasthttp.Do(req, resp); err != nil {
		return nil, 0, err
	}
	fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()
	code := resp.StatusCode()
	return body, code, nil
}

func Post(ctx *context.Context, urlLink string, payload string, headers map[string]string) ([]byte, int, error) {
	req := fasthttp.AcquireRequest()
	req.SetBody([]byte(payload))
	req.Header.SetMethod("POST")
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	req.SetRequestURI(urlLink)
	resp := fasthttp.AcquireResponse()
	if err := fasthttp.Do(req, resp); err != nil {
		return nil, 0, err
	}
	fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()
	code := resp.StatusCode()
	return body, code, nil
}
