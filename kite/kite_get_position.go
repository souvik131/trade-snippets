package kite

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/souvik131/trade-snippets/requests"
)

func (kite *Kite) GetPositions(ctx *context.Context) ([]*Position, error) {

	k := *kite
	url := "https://api.kite.trade/portfolio/positions"

	headers := make(map[string]string)
	headers["authorization"] = k.Token
	headers["content-type"] = "application/x-www-form-urlencoded"

	res, code, err := requests.Get(ctx, url, headers)

	if err != nil {
		return nil, err
	}
	var respData *PositionResponsePayload
	err = json.Unmarshal(res, &respData)
	if err != nil {
		return nil, err
	}

	if respData == nil {
		return nil, errors.New("kite_broker_api_issue")
	}

	if code == 200 && respData.Data != nil {
		positions := []*Position{}

		for _, net := range respData.Data.Net {

			if net.Quantity != 0 {

				lastPrice, err := k.GetLastPrice(ctx, net.Exchange, net.TradingSymbol)
				if err != nil || lastPrice == 0 {
					log.Println(err)
				}
				positions = append(positions, net)
			}
		}
		return positions, nil
	}
	return nil, errors.New(respData.Status + ":" + respData.Message)

}
