package kite

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
)

func (kite *Kite) GetLastPrice(ctx *context.Context, exchange string, tradingSymbol string) (float64, error) {
	k := *kite
	url := k.BaseUrl + "/quote?i=" + exchange + ":" + url.QueryEscape(tradingSymbol)
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
	price := (*respData.Data)[exchange+":"+tradingSymbol].LastPrice

	return price, nil
}
