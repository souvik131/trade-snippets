package kite

import (
	"context"
	"fmt"
	"strings"

	"github.com/valyala/fasthttp"
)

func Get(ctx *context.Context, urlLink string, headers map[string]string) ([]byte, int, error) {
	req := fasthttp.AcquireRequest()
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	req.SetRequestURI(urlLink)
	resp := fasthttp.AcquireResponse()
	if err := fasthttp.Do(req, resp); err != nil {
		return nil, 0, err
	}
	fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()
	code := resp.StatusCode()
	return body, code, nil
}

func Post(ctx *context.Context, urlLink string, payload string, headers map[string]string) ([]byte, int, error) {
	req := fasthttp.AcquireRequest()
	req.SetBody([]byte(payload))
	req.Header.SetMethod("POST")
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	req.SetRequestURI(urlLink)
	resp := fasthttp.AcquireResponse()
	if err := fasthttp.Do(req, resp); err != nil {
		return nil, 0, err
	}
	fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()
	code := resp.StatusCode()
	return body, code, nil
}

func PostWithCookies(ctx *context.Context, urlLink string, payload string, headers map[string]string) ([]byte, int, string, error) {
	req := fasthttp.AcquireRequest()
	req.SetBody([]byte(payload))
	req.Header.SetMethod("POST")
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	req.SetRequestURI(urlLink)
	resp := fasthttp.AcquireResponse()
	if err := fasthttp.Do(req, resp); err != nil {
		return nil, 0, "", err
	}
	fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()
	code := resp.StatusCode()
	cookieStrings := []string{}
	resp.Header.VisitAllCookie(func(key, value []byte) {
		cookieStrings = append(cookieStrings, fmt.Sprintf("%v=%v", string(key), string(value)))
	})
	return body, code, strings.Join(cookieStrings, "; "), nil
}
