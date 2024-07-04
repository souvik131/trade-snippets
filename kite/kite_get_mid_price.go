package kite

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"

	"github.com/souvik131/trade-snippets/requests"
)

func (kite *Kite) GetMidPrice(ctx *context.Context, exchange string, tradingSymbol string) (float64, error) {
	k := *kite
	url := "https://api.kite.trade/quote?i=" + exchange + ":" + url.QueryEscape(tradingSymbol)
	headers := make(map[string]string)
	headers["authorization"] = k.Token
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
