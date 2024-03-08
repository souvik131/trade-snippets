package kite

import (
	"context"
	"encoding/json"
	"fmt"
)

func (kiteClient *Kite) GetCharges(ctx *context.Context) (float64, error) {

	k := *kiteClient
	url := k.BaseUrl + "/orders"

	headers := make(map[string]string)
	headers["authorization"] = k.Token
	headers["content-type"] = "application/x-www-form-urlencoded"

	res, code, err := Get(ctx, url, headers)
	if err != nil {
		return 0.0, err
	}

	var respData *OrdersResponsePayload
	err = json.Unmarshal(res, &respData)
	if err != nil {
		return 0.0, err
	}
	if code == 200 && respData.Data != nil {
		requestOrders := make([]*ChargesOrderRequest, 0)

		for _, order := range respData.Data {
			if order.OrderState == "COMPLETE" {
				requestOrders = append(requestOrders, &ChargesOrderRequest{
					AveragePrice:    order.AveragePrice,
					Exchange:        order.Exchange,
					OrderId:         order.OrderId,
					Product:         order.Product,
					Quantity:        order.Quantity,
					TradingSymbol:   order.TradingSymbol,
					Variety:         order.Variety,
					OrderType:       order.OrderType,
					TransactionType: order.TransactionType,
				})
			}
		}

		url := k.BaseUrl + "/charges/orders"

		bytes, err := json.Marshal(requestOrders)
		if err != nil {
			return 0.0, err
		}
		payload := string(bytes)
		headers := make(map[string]string)
		headers["authorization"] = k.Token
		headers["content-type"] = "application/json"

		res, code, err := Post(ctx, url, payload, headers)
		if err != nil {
			return 0.0, err
		}

		var respData *BrokerChargesPayload
		err = json.Unmarshal(res, &respData)
		if err != nil {
			return 0.0, err
		}

		if code == 200 && respData.Data != nil {
			charges := 0.0
			for _, c := range respData.Data {
				charges += c.Charges.Total
			}
			return charges, nil
		}
		return 0.0, nil

	}
	return 0.0, fmt.Errorf("%v", respData.Data)

}
