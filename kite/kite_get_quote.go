package kite

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/souvik131/trade-snippets/requests"
)

func (kite *Kite) GetQuote(ctx *context.Context, exchange string, tradingSymbol string) (*Quote, error) {

	k := *(*kite).Creds
	if k["LoginType"] == "WEB" {
		return nil, fmt.Errorf("not_supported")
	}
	url := k["Url"] + "/quote?i=" + exchange + ":" + url.QueryEscape(tradingSymbol)
	headers := make(map[string]string)
	headers["authorization"] = k["Token"]
	headers["content-type"] = "application/x-www-form-urlencoded"

	response, _, err := requests.Get(ctx, url, headers)

	if err != nil {
		return nil, err
	}
	var respData *QuoteResponsePayload
	err = json.Unmarshal(response, &respData)
	if err != nil {
		return nil, err
	}

	if respData.Data == nil {
		return nil, errors.New(respData.Message)

	}

	return (respData.Data)[exchange+":"+tradingSymbol], nil
}

func (kite *Kite) GetLastPrice(ctx *context.Context, exchange string, tradingSymbol string) (float64, error) {

	k := *(*kite).Creds
	if k["LoginType"] == "WEB" {
		return 0.0, fmt.Errorf("not_supported")
	}
	url := k["Url"] + "/quote?i=" + exchange + ":" + url.QueryEscape(tradingSymbol)
	headers := make(map[string]string)
	headers["authorization"] = k["Token"]
	headers["content-type"] = "application/x-www-form-urlencoded"

	response, _, err := requests.Get(ctx, url, headers)

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
	price := (respData.Data)[exchange+":"+tradingSymbol].LastPrice

	return price, nil
}
