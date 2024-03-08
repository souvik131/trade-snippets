package kite

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
)

func (kite *Kite) GetPnl(ctx *context.Context) (float64, error) {

	k := *kite
	url := k.BaseUrl + "/portfolio/positions"

	headers := make(map[string]string)
	headers["authorization"] = k.Token
	headers["content-type"] = "application/x-www-form-urlencoded"

	res, code, err := Get(ctx, url, headers)

	if err != nil {
		return 0, err
	}
	var respData *PositionResponsePayload
	err = json.Unmarshal(res, &respData)
	if err != nil {
		return 0, err
	}

	if respData == nil {
		return 0, errors.New("kite_broker_api_issue")
	}

	if code == 200 && respData.Data != nil {

		pnl := float64(0)
		for _, net := range respData.Data.Net {

			if net.Quantity == 0 {
				pnl += net.SellValue - net.BuyValue
			} else {
				lastPrice, err := k.GetLastPrice(ctx, net.Exchange, net.TradingSymbol)
				if err != nil {
					log.Println(err)
				}
				pnl += net.SellValue + float64(net.Quantity)*math.Abs(lastPrice*float64(net.Multiplier)) - net.BuyValue
			}
		}
		return pnl, nil
	}
	return 0.0, errors.New(respData.Status + ":" + respData.Message)

}
