package kite

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/souvik131/trade-snippets/requests"
)

func (kite *Kite) GetCharges(ctx *context.Context) (float64, error) {

	k := *(*kite).Creds
	url := k["Url"] + "/orders"

	headers := map[string]string{
		"Connection":      "keep-alive",
		"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
		"Accept-Encoding": "gzip, deflate",
		"Host":            "kite.zerodha.com",
		"Accept":          "*/*",
	}
	headers["authorization"] = k["Token"]
	headers["content-type"] = "application/x-www-form-urlencoded"

	res, code, cookie, err := requests.GetWithCookies(ctx, url, headers, k["Cookie"])
	k["Cookie"] = cookie
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

		url := k["Url"] + "/charges/orders"

		bytes, err := json.Marshal(requestOrders)
		if err != nil {
			return 0.0, err
		}
		payload := string(bytes)
		headers := make(map[string]string)
		headers["authorization"] = k["Token"]
		headers["content-type"] = "application/json"

		res, code, cookie, err := requests.PostWithCookies(ctx, url, payload, headers, k["Cookie"])
		k["Cookie"] = cookie
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
