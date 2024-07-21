package kite

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"

	"github.com/souvik131/trade-snippets/requests"
)

func (kiteClient *Kite) GetPositions(ctx *context.Context) error {

	k := *(*kiteClient).Creds
	url := k["Url"] + "/portfolio/positions"

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
		return err
	}
	var respData *PositionResponsePayload
	err = json.Unmarshal(res, &respData)
	if err != nil {
		return err
	}

	if respData == nil {
		return errors.New("kite_broker_api_issue")
	}

	if code == 200 && respData.Data != nil {
		positions := []*Position{}

		priceMap := map[string]float64{}

		for _, net := range respData.Data.Net {
			lp, err := kiteClient.GetLastPrice(ctx, net.Exchange, net.TradingSymbol)
			if err != nil {
				log.Panic(net.TradingSymbol, err)
			} else {

				net.LastPrice = lp
				priceMap[net.TradingSymbol] = lp
			}

			positions = append(positions, net)

		}

		kiteClient.Positions = positions
		pnl := 0.0
		for _, net := range positions {
			if lastPrice, ok := priceMap[net.TradingSymbol]; ok {
				if net.Quantity == 0 {
					pnl += net.SellValue - net.BuyValue
				} else {
					pnl += net.SellValue + float64(net.Quantity)*math.Abs(lastPrice*float64(net.Multiplier)) - net.BuyValue
				}
			} else {
				log.Fatal("price not present", net.TradingSymbol)
			}
		}
		kiteClient.Pnl = pnl

		return nil
	}
	return errors.New(respData.Status + ":" + respData.Message)

}
