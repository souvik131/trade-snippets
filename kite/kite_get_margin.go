package kite

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/souvik131/trade-snippets/requests"
)

func (kiteClient *Kite) GetMargin(ctx *context.Context) (*Margin, error) {
	k := *kiteClient
	url := "https://api.kite.trade/user/margins"

	headers := make(map[string]string)
	headers["authorization"] = k.Token
	headers["content-type"] = "application/x-www-form-urlencoded"

	res, code, err := requests.Get(ctx, url, headers)

	if err != nil {
		return nil, err
	}
	var respData *MarginResponsePayload
	err = json.Unmarshal(res, &respData)
	if err != nil {
		return nil, err
	}
	if code == 200 && respData.Data != nil {
		return &Margin{
			MarginUsed:  respData.Data.Equity.Utilised.Debits,
			MarginTotal: respData.Data.Equity.Net + respData.Data.Equity.Utilised.Debits,
		}, nil
	}
	return nil, errors.New(respData.Status + ":" + respData.Message)
}
