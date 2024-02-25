package kite

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pquerna/otp/hotp"
)

func (kite *Kite) Login(ctx *context.Context) error {
	k := *kite

	urlLogin := k.BaseUrl + "/api/login"
	urlTFA := k.BaseUrl + "/api/twofa"
	id := k.Id
	password := k.Password
	totp := k.Totp

	payload := fmt.Sprintf("user_id=%v&password=%v", id, password)

	headers := map[string]string{
		"Connection":     "keep-alive",
		"Content-Type":   "application/x-www-form-urlencoded",
		"User-Agent":     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.128 Safari/537.36",
		"x-kite-version": "3",
	}

	body, _, cookie, err := PostWithCookies(ctx, urlLogin, payload, headers)
	if err != nil {
		return err
	}
	headers["Cookie"] = cookie
	var respLogin LoginPayload
	err = json.Unmarshal(body, &respLogin)
	if err != nil {
		return err
	}

	if respLogin.Data == nil || respLogin.Data.RequestId == "" {
		return fmt.Errorf("no_request_id")
	}

	otp, err := hotp.GenerateCode(totp, uint64(time.Now().Unix()/30))
	if err != nil {
		return err
	}
	k.RequestId = respLogin.Data.RequestId
	payload = fmt.Sprintf("user_id=%v&request_id=%v&twofa_value=%v", id, respLogin.Data.RequestId, otp)

	body, _, cookie, err = PostWithCookies(ctx, urlTFA, payload, headers)
	if err != nil {
		return err
	}
	var respTFA TFAPayload
	err = json.Unmarshal(body, &respTFA)
	if err != nil {
		return err
	}
	if respTFA.Status == "success" {
		encToken := ""
		allCookies := strings.Split(cookie, ";")
		for _, c := range allCookies {
			c = strings.TrimSpace(c)
			if strings.HasPrefix(c, "enctoken=") {
				encToken = fmt.Sprintf("enctoken %v", strings.ReplaceAll(c, "enctoken=", ""))
			}
		}
		k.Token = encToken
		*kite = k
		return nil
	}
	return fmt.Errorf("no_enc_token")

}
