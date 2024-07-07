package kite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"reflect"
	"strings"

	"github.com/souvik131/trade-snippets/requests"
)

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

	url := "https://api.kite.trade/orders/" + kOrder.Variety
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
	headers["authorization"] = k["Token"]
	headers["content-type"] = "application/x-www-form-urlencoded"

	response, code, err := requests.Post(ctx, url, payload, headers)

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
