package requests

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
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
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()
	code := resp.StatusCode()
	return []byte(string(body)), code, nil
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
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()
	code := resp.StatusCode()
	return []byte(string(body)), code, nil
}
func Put(ctx *context.Context, urlLink string, payload string, headers map[string]string) ([]byte, int, error) {
	req := fasthttp.AcquireRequest()
	req.SetBody([]byte(payload))
	req.Header.SetMethod("PUT")
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	req.SetRequestURI(urlLink)
	resp := fasthttp.AcquireResponse()
	if err := fasthttp.Do(req, resp); err != nil {
		return nil, 0, err
	}
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()
	code := resp.StatusCode()
	return []byte(string(body)), code, nil
}

func GetWithCookies(ctx *context.Context, urlLink string, headers map[string]string, cookie string) ([]byte, int, string, error) {

	headers["Cookie"] = cookie
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod("GET")
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	req.SetRequestURI(urlLink)
	resp := fasthttp.AcquireResponse()
	if err := fasthttp.Do(req, resp); err != nil {
		return nil, 0, "", err
	}
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()
	code := resp.StatusCode()

	cookieStrings := []string{}
	resp.Header.VisitAllCookie(func(key, value []byte) {
		cookieStrings = append(cookieStrings, fmt.Sprintf("%v=%v", string(key), string(value)))
	})
	if code == 302 {
		return []byte(string(resp.Header.Peek("Location"))), code, strings.Join(cookieStrings, "; "), nil
	}
	switch string(resp.Header.Peek("Content-Encoding")) {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, 0, "", err
		}
		defer reader.Close()
		body, err = io.ReadAll(reader)
		if err != nil {
			return nil, 0, "", err
		}
	default:
		body = []byte(string(body))
	}
	return body, code, strings.Join(cookieStrings, "; "), nil
}

func DeleteWithCookies(ctx *context.Context, urlLink string, headers map[string]string, cookie string) ([]byte, int, string, error) {

	headers["Cookie"] = cookie
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod("DELETE")
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	req.SetRequestURI(urlLink)
	resp := fasthttp.AcquireResponse()
	if err := fasthttp.Do(req, resp); err != nil {
		return nil, 0, "", err
	}
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()
	code := resp.StatusCode()

	cookieStrings := []string{}
	resp.Header.VisitAllCookie(func(key, value []byte) {
		cookieStrings = append(cookieStrings, fmt.Sprintf("%v=%v", string(key), string(value)))
	})
	if code == 302 {
		return []byte(string(resp.Header.Peek("Location"))), code, strings.Join(cookieStrings, "; "), nil
	}
	switch string(resp.Header.Peek("Content-Encoding")) {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, 0, "", err
		}
		defer reader.Close()
		body, err = io.ReadAll(reader)
		if err != nil {
			return nil, 0, "", err
		}
	default:
		body = []byte(string(body))
	}
	return body, code, strings.Join(cookieStrings, "; "), nil
}

func PostWithCookies(ctx *context.Context, urlLink string, payload string, headers map[string]string, cookie string) ([]byte, int, string, error) {
	headers["Cookie"] = cookie
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
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)
	body := resp.Body()
	code := resp.StatusCode()
	cookieStrings := []string{}
	resp.Header.VisitAllCookie(func(key, value []byte) {
		cookieStrings = append(cookieStrings, fmt.Sprintf("%v=%v", string(key), string(value)))
	})

	switch string(resp.Header.Peek("Content-Encoding")) {
	case "gzip":
		reader, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			return nil, 0, "", err
		}
		defer reader.Close()
		body, err = io.ReadAll(reader)
		if err != nil {
			return nil, 0, "", err
		}
	default:
		body = []byte(string(body))
	}
	return body, code, strings.Join(cookieStrings, "; "), nil
}
