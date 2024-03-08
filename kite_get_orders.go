package kite

import (
	"context"
	"encoding/json"
	"errors"
)

func (kite *Kite) GetOrders(ctx *context.Context) ([]*OrderStatus, error) {

	k := *kite
	url := k.BaseUrl + "/orders"

	// log.Println(url, k["token"])

	headers := make(map[string]string)
	headers["authorization"] = k.Token
	headers["content-type"] = "application/x-www-form-urlencoded"

	res, code, err := Get(ctx, url, headers)
	if err != nil {
		return nil, err
	}

	var respData *OrdersResponsePayload
	err = json.Unmarshal(res, &respData)
	if err != nil {
		return nil, err
	}
	if code == 200 && respData.Data != nil {

		return respData.Data, nil
	}
	return nil, errors.New(respData.Status + ":" + respData.Message)
}
