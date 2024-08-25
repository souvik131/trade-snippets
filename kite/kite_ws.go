package kite

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/souvik131/trade-snippets/notifications"
	"github.com/souvik131/trade-snippets/ws"
)

var (
	t = &notifications.Telegram{}
)

const HeartBeatIntervalInSeconds float64 = 20
const BufferSize int = 1000

func GetWebsocketClientForWeb(ctx *context.Context, id string, token string, receiveBinaryTickers bool) (*TickerClient, error) {

	token = strings.Replace(token, "enctoken ", "", 1)
	return getWebsocketClient(ctx, fmt.Sprintf("user_id=%v&access_token=%v&api_key=kitefront", id, token), receiveBinaryTickers)
}

func GetWebsocketClientForAPI(ctx *context.Context, token string, receiveBinaryTickers bool) (*TickerClient, error) {
	token = strings.Replace(token, "token ", "", 1)
	apiKey := strings.Split(token, ":")[0]
	accessToken := strings.Replace(token, apiKey+":", "", 1)
	return getWebsocketClient(ctx, fmt.Sprintf("access_token=%v&api_key=%v", accessToken, apiKey), receiveBinaryTickers)
}

func getWebsocketClient(ctx *context.Context, rawQuery string, receiveBinaryTickers bool) (*TickerClient, error) {
	log.Printf("websocket : start")
	go t.Send("websocket : start")
	k := &TickerClient{
		Client: &ws.Client{
			URL: &url.URL{
				Scheme:   "wss",
				Host:     "ws.kite.trade",
				RawQuery: rawQuery,
			},
			Header: &http.Header{},
		},
		TickerChan:                 make(chan KiteTicker, BufferSize),
		BinaryTickerChan:           make(chan []byte, BufferSize),
		ConnectChan:                make(chan struct{}, 10),
		ErrorChan:                  make(chan interface{}, BufferSize),
		FullTokens:                 map[uint32]bool{},
		QuoteTokens:                map[uint32]bool{},
		LtpTokens:                  map[uint32]bool{},
		HeartBeatIntervalInSeconds: HeartBeatIntervalInSeconds,
		ReceiveBinaryTickers:       receiveBinaryTickers,
	}

	k.LastUpdatedTime.Store(time.Now().Unix())
	err := k.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return k, nil
}

func (k *TickerClient) Connect(ctx *context.Context) error {

	data, err := k.Client.Connect(ctx)
	if err != nil {
		return err
	}
	if len(data) > 0 {
		respBody := map[string]string{}
		err = json.Unmarshal(data, &respBody)
		if err != nil {
			return err
		}
		if err, ok := respBody["Error"]; ok {
			return fmt.Errorf("%v", err)
		}
		return fmt.Errorf("%v", respBody)
	}
	log.Printf("websocket : client ready")
	go t.Send("websocket : client ready")
	err = k.Client.Read(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (k *TickerClient) Serve(ctx *context.Context) {

	<-time.After(time.Millisecond)
	log.Println("websocket : serve")
	go t.Send("websocket : serve")

	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ticker.C:
			if !k.checkHeartBeat(ctx) {
				return
			}
		case reader := <-k.Client.ReaderChannel:
			k.LastUpdatedTime.Store(time.Now().Unix())
			if reader.Error != nil {
				log.Panic(reader.Error)
			}
			switch reader.MessageType {
			case ws.TEXT:
				go k.onTextMessage(reader)
			case ws.BINARY:
				go k.onBinaryMessage(reader)

			default:
				log.Printf("recv: %v %v", reader.MessageType, reader.Message)
			}
		}
	}
}

func (k *TickerClient) Close(ctx *context.Context) error {
	return k.Client.Close(ctx)
}

func (k *TickerClient) Reconnect(ctx *context.Context) error {
	err := k.Close(ctx)
	if err != nil {
		log.Printf("websocket : attempted to close, got response -> %v", err)
		go t.Send("websocket : reconnecting")
	}
	err = k.Connect(ctx)
	if err != nil {
		return err
	}
	go k.Serve(ctx)
	k.Resubscribe(ctx)
	return nil
}

func (k *TickerClient) Resubscribe(ctx *context.Context) error {

	keys := make([]string, 0, len(k.LtpTokens))
	for k2 := range k.LtpTokens {
		keys = append(keys, TokenSymbolMap[k2])
	}
	for len(keys) > 0 {
		minLen := int(math.Min(float64(BufferSize), float64(len(keys))))
		keys = keys[0:minLen]
		err := k.SubscribeLTP(ctx, keys[minLen:])
		if err != nil {
			return err
		}
	}

	keys = make([]string, 0, len(k.QuoteTokens))
	for k2 := range k.QuoteTokens {
		keys = append(keys, TokenSymbolMap[k2])
	}

	for len(keys) > 0 {
		minLen := int(math.Min(float64(BufferSize), float64(len(keys))))
		keys = keys[0:minLen]
		err := k.SubscribeQuote(ctx, keys[minLen:])
		if err != nil {
			return err
		}
	}

	keys = make([]string, 0, len(k.FullTokens))
	for k2 := range k.FullTokens {
		keys = append(keys, TokenSymbolMap[k2])
	}

	for len(keys) > 0 {
		minLen := int(math.Min(float64(BufferSize), float64(len(keys))))
		keys = keys[0:minLen]
		err := k.SubscribeFull(ctx, keys[minLen:])
		if err != nil {
			return err
		}
	}

	return nil

}
func (k *TickerClient) SubscribeLTP(ctx *context.Context, tokens []string) error {
	r := &Request{
		Message: "mode",
		Tokens: []interface{}{
			"ltp",
		},
	}
	iTokens := []uint32{}
	for _, t := range tokens {
		iTokens = append(iTokens, SymbolTokenMap[t])
	}
	r.Tokens = append(r.Tokens, iTokens)
	for _, t := range tokens {
		tInt := SymbolTokenMap[t]
		k.FullTokens[tInt] = true
	}

	return k.writeTextRequest(ctx, r)
}

func (k *TickerClient) SubscribeFull(ctx *context.Context, tokens []string) error {
	r := &Request{
		Message: "mode",
		Tokens: []interface{}{
			"full",
		},
	}

	iTokens := []uint32{}
	for _, t := range tokens {
		iTokens = append(iTokens, SymbolTokenMap[t])
	}
	r.Tokens = append(r.Tokens, iTokens)
	for _, t := range tokens {
		tInt := SymbolTokenMap[t]
		k.FullTokens[tInt] = true
	}
	return k.writeTextRequest(ctx, r)
}

func (k *TickerClient) SubscribeQuote(ctx *context.Context, tokens []string) error {
	r := &Request{
		Message: "subscribe",
		Tokens:  []interface{}{},
	}
	for _, t := range tokens {
		tInt := SymbolTokenMap[t]
		r.Tokens = append(r.Tokens, tInt)
		k.QuoteTokens[tInt] = true
	}
	return k.writeTextRequest(ctx, r)
}

func (k *TickerClient) Unsubscribe(ctx *context.Context, tokens []string) error {
	r := &Request{
		Message: "unsubscribe",
		Tokens:  []interface{}{},
	}
	for _, t := range tokens {
		r.Tokens = append(r.Tokens, SymbolTokenMap[t])
		delete(k.QuoteTokens, SymbolTokenMap[t])
		delete(k.LtpTokens, SymbolTokenMap[t])
		delete(k.QuoteTokens, SymbolTokenMap[t])
	}

	return k.writeTextRequest(ctx, r)
}

func (k *TickerClient) checkHeartBeat(ctx *context.Context) bool {
	if time.Since(time.Unix(k.LastUpdatedTime.Load(), 0)).Seconds() > float64(k.HeartBeatIntervalInSeconds) {
		k.HeartBeatIntervalInSeconds *= 2
		go k.Reconnect(ctx)
		return false
	} else {
		k.HeartBeatIntervalInSeconds = HeartBeatIntervalInSeconds
	}
	return true

}

func (k *TickerClient) writeTextRequest(ctx *context.Context, req *Request) error {
	message, err := json.Marshal(req)

	if err != nil {
		return err
	}
	err = k.Client.Write(ctx, &ws.Writer{
		MessageType: ws.TEXT,
		Message:     message,
	})
	if err != nil {
		return err
	}
	return nil
}

func (k *TickerClient) onBinaryMessage(reader *ws.Reader) {

	message := reader.Message
	numOfPackets := binary.BigEndian.Uint16(message[0:2])
	if numOfPackets > 0 {
		if k.ReceiveBinaryTickers {
			k.BinaryTickerChan <- reader.Message
		} else {
			k.ParseBinary(message)
		}
	}

}

func (k *TickerClient) ParseBinary(message []byte) {
	numOfPackets := binary.BigEndian.Uint16(message[0:2])
	message = message[2:]
	for {
		if numOfPackets == 0 {
			break
		}

		numOfPackets--
		packetSize := binary.BigEndian.Uint16(message[0:2])
		packet := Packet(message[2 : packetSize+2])
		values := packet.ParseBinary(int(math.Min(64, float64(len(packet)))))
		ticker := KiteTicker{}
		if len(values) >= 2 {
			ticker.Token = values[0]
			ticker.TradingSymbol = TokenSymbolMap[ticker.Token]
			ticker.LastPrice = float64(values[1]) / 100
		}
		switch len(values) {
		case 2:
		case 7:
			ticker.High = float64(values[2]) / 100
			ticker.Low = float64(values[3]) / 100
			ticker.Open = float64(values[4]) / 100
			ticker.Close = float64(values[5]) / 100
			ticker.ExchangeTimestamp = time.Unix(int64(values[6]), 0)
		case 8:
			ticker.High = float64(values[2]) / 100
			ticker.Low = float64(values[3]) / 100
			ticker.Open = float64(values[4]) / 100
			ticker.Close = float64(values[5]) / 100
			ticker.PriceChange = float64(values[6]) / 100
			ticker.ExchangeTimestamp = time.Unix(int64(values[7]), 0)
		case 11:
			ticker.LastTradedQuantity = values[2]
			ticker.AverageTradedPrice = float64(values[3]) / 100
			ticker.VolumeTraded = values[4]
			ticker.TotalBuy = values[5]
			ticker.TotalSell = values[6]
			ticker.High = float64(values[7]) / 100
			ticker.Low = float64(values[8]) / 100
			ticker.Open = float64(values[9]) / 100
			ticker.Close = float64(values[10]) / 100
		case 16:
			ticker.LastTradedQuantity = values[2]
			ticker.AverageTradedPrice = float64(values[3]) / 100
			ticker.VolumeTraded = values[4]
			ticker.TotalBuy = values[5]
			ticker.TotalSell = values[6]
			ticker.High = float64(values[7]) / 100
			ticker.Low = float64(values[8]) / 100
			ticker.Open = float64(values[9]) / 100
			ticker.Close = float64(values[10]) / 100
			ticker.LastTradedTimestamp = time.Unix(int64(values[11]), 0)
			ticker.OI = values[12]
			ticker.OIHigh = values[13]
			ticker.OILow = values[14]
			ticker.ExchangeTimestamp = time.Unix(int64(values[15]), 0)
		default:
			log.Println("unkown length of packet", len(values), values)
		}

		if len(packet) > 64 {

			packet = packet[64:]

			values := packet.ParseMarketDepth()
			lobDepth := len(values) / 6

			for {
				if len(values) == 0 {

					break
				}
				qty := values[0]
				price := float64(values[1]) / 100
				orders := values[2]
				if len(ticker.Depth.Buy) < lobDepth {
					ticker.Depth.Buy = append(ticker.Depth.Buy, LimitOrder{Price: price, Quantity: qty, Orders: orders})
				} else {

					ticker.Depth.Sell = append(ticker.Depth.Sell, LimitOrder{Price: price, Quantity: qty, Orders: orders})
				}
				values = values[3:]

			}
		}
		k.TickerChan <- ticker
		if len(message) > int(packetSize+2) {
			message = message[packetSize+2:]
		}
	}
}

func (k *TickerClient) onTextMessage(reader *ws.Reader) {
	if len(reader.Message) > 0 {
		m := &Message{}
		err := json.Unmarshal(reader.Message, m)
		if err != nil {
			log.Panic(err)
		}
		switch m.Type {
		case "instruments_meta":
			log.Printf("websocket : connected")
			t.Send("websocket : connected")
			k.ConnectChan <- struct{}{}
		case "error":
			k.ErrorChan <- m.Data
		}
	}
}
