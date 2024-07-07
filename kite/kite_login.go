package kite

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"log"

	"github.com/gin-gonic/gin"
	"github.com/pquerna/otp/hotp"
	"github.com/souvik131/trade-snippets/requests"
)

func (kiteClient *Kite) oauth(c *gin.Context) {
	k := *kiteClient
	queries := c.Request.URL.Query()
	requestToken, idExists := queries["request_token"]
	if !idExists {
		c.Data(http.StatusFailedDependency, "text/plain; charset=utf-8", []byte("failed"))
		return
	}

	k["RequestToken"] = requestToken[0]
	ctx := context.WithoutCancel(c)
	headers := map[string]string{
		"Connection":      "keep-alive",
		"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
		"Accept-Encoding": "gzip, deflate",
		"Host":            "kite.zerodha.com",
		"Accept":          "*/*",
		"Content-Type":    "application/x-www-form-urlencoded",
		"x-kite-version":  "3",
	}

	payload := fmt.Sprintf("api_key=%v&request_token=%v&checksum=%v", k["ApiKey"], k["RequestToken"], GetSha256(k["ApiKey"]+k["RequestToken"]+k["ApiSecret"]))
	body, code, _, err := requests.PostWithCookies(&ctx, "https://api.kite.trade/session/token", payload, headers, "")
	if err != nil {
		log.Println(err)
		c.Data(http.StatusFailedDependency, "text/plain; charset=utf-8", []byte("failed"))
		return
	}
	if code != 200 {
		log.Printf("failed %v", code)
		c.Data(http.StatusFailedDependency, "text/plain; charset=utf-8", []byte("failed"))
		return
	}
	type LoginCompletePayload struct {
		Status    string `json:"error"`
		Message   string `json:"message"`
		ErrorType string `json:"error_type"`
		Data      *struct {
			AccessToken string `json:"access_token"`
		} `json:"data"`
	}
	var respLogin LoginCompletePayload
	err = json.Unmarshal(body, &respLogin)
	if err != nil {
		log.Println(err)
		c.Data(http.StatusFailedDependency, "text/plain; charset=utf-8", []byte("failed"))
		return
	}
	if respLogin.Data == nil || respLogin.Data.AccessToken == "" {
		c.Data(http.StatusFailedDependency, "text/plain; charset=utf-8", []byte("failed"))
		return
	}
	k["AccessToken"] = respLogin.Data.AccessToken
	k["Token"] = fmt.Sprintf("token %v:%v", k["ApiKey"], respLogin.Data.AccessToken)
	//log.Println("Stage 7: OAuth Complete ", k["Token"])
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte("ok"))

}

func (kiteClient *Kite) Login(ctx *context.Context, router *gin.Engine) error {

	k := *kiteClient
	present := false
	for _, route := range router.Routes() {
		if route.Path == k["Path"] {
			present = true
		}
	}
	if !present {
		//	log.Println("Stage 0: Router set to ", k["Path"])
		router.GET(k["Path"], k.oauth)
	}
	type LoginPayload struct {
		Status    string `json:"error"`
		Message   string `json:"message"`
		ErrorType string `json:"error_type"`
		Data      *struct {
			RequestId string `json:"request_id"`
		} `json:"data"`
	}

	type TFAPayload struct {
		Status string `json:"status"`
	}

	headers := map[string]string{
		"Connection":      "keep-alive",
		"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
		"Accept-Encoding": "gzip, deflate",
		"Host":            "kite.zerodha.com",
		"Accept":          "*/*",
	}

	//Session ID request
	body, code, cookie, err := requests.GetWithCookies(ctx, "https://kite.zerodha.com/connect/login?v=3&api_key="+k["ApiKey"], headers, "")
	if code != 302 {
		return fmt.Errorf("no_redirect %v", code)
	}
	if err != nil {
		return err
	}
	redirectUrl := string(body)
	sessionId := ""
	for _, pairString := range strings.Split(strings.Split(redirectUrl, "?")[1], "&") {
		pair := strings.Split(pairString, "=")
		if pair[0] == "sess_id" {
			sessionId = pair[1]
			break
		}
	}
	//log.Println("Stage 1: Got Session Id ")

	//Open Login URL
	_, code, cookie, err = requests.GetWithCookies(ctx, "https://kite.zerodha.com/connect/login?sess_id="+sessionId+"&api_key="+k["ApiKey"], headers, cookie)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("failed %v", code)
	}
	//log.Println("Stage 2: Opened Login URL")

	//Hit Session API
	_, code, cookie, err = requests.GetWithCookies(ctx, "https://kite.zerodha.com/api/connect/session?sess_id="+sessionId+"&api_key="+k["ApiKey"], headers, cookie)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("failed %v", code)
	}
	//log.Println("Stage 3: Hit Session API")

	//Hit Login API
	headers["Content-Type"] = "application/x-www-form-urlencoded"
	headers["x-kite-version"] = "3"
	payload := fmt.Sprintf("user_id=%v&password=%v", k["Id"], k["Password"])
	body, code, cookie, err = requests.PostWithCookies(ctx, "https://kite.zerodha.com/api/login", payload, headers, cookie)
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("failed %v", code)
	}
	var respLogin LoginPayload
	err = json.Unmarshal(body, &respLogin)
	if err != nil {
		return err
	}
	if respLogin.Data == nil || respLogin.Data.RequestId == "" {

		return fmt.Errorf("no_request_id")
	}
	//log.Println("Stage 4: Hit Login API ")

	//Hit TOTP API
	otp, err := hotp.GenerateCode(k["Totp"], uint64(time.Now().Unix()/30))
	if err != nil {
		return err
	}
	payload = fmt.Sprintf("user_id=%v&request_id=%v&twofa_value=%v&twofa_type=totp&skip_session=true", k["Id"], respLogin.Data.RequestId, otp)
	body, code, cookie, err = requests.PostWithCookies(ctx, "https://kite.zerodha.com/api/twofa", payload, headers, cookie)

	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("failed %v", code)
	}
	var respTFA TFAPayload
	err = json.Unmarshal(body, &respTFA)
	if err != nil {
		return err
	}
	//log.Println("Stage 5: Hit TOTP API ")
	headers = map[string]string{
		"Connection":      "keep-alive",
		"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
		"Accept-Encoding": "gzip, deflate",
		"Host":            "kite.zerodha.com",
		"Accept":          "*/*",
	}
	body, code, _, err = requests.GetWithCookies(ctx, "https://kite.zerodha.com/connect/finish?api_key="+k["ApiKey"]+"&sess_id="+sessionId, headers, cookie)
	if err != nil {
		return err
	}
	if code != 302 {
		return fmt.Errorf("no_redirect %v", code)
	}
	redirectUrl = string(body)

	//log.Println("Stage 6: Get Redirect URL ", redirectUrl)
	_, code, _, err = requests.GetWithCookies(ctx, redirectUrl, headers, "")
	if err != nil {
		return err
	}
	if code != 200 {
		return fmt.Errorf("failed %v", code)
	}
	//log.Println("Stage 8: Login Complete ")
	return nil
}
func GetOtp(totp string) (string, error) {
	return hotp.GenerateCode(totp, uint64(time.Now().Unix()/30))
}

func GetSha256(key string) string {
	h := sha256.New()
	h.Write([]byte(key))
	return fmt.Sprintf("%x", h.Sum(nil))
}
