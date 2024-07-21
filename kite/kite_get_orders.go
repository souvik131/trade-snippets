package kite

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/souvik131/trade-snippets/requests"
)

func (kite *Kite) GetOrders(ctx *context.Context) ([]*OrderStatus, error) {

	k := *(*kite).Creds
	url := k["Url"] + "/orders"

	// log.Println(url, k["token"])

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

func (kite *Kite) GetOrderHistory(ctx *context.Context, orderId string) ([]*OrderStatus, error) {

	k := *(*kite).Creds
	url := k["Url"] + "/orders/" + orderId

	// log.Println(url, k["token"])

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
