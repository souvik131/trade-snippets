package kite

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/souvik131/trade-snippets/ws"
)

const HeartBeatIntervalInSeconds float64 = 30

func GetWebsocketClientForWeb(ctx *context.Context, id string, token string) (*TickerClient, error) {
	token = strings.Replace(token, "enctoken ", "", 1)
	return getWebsocketClient(ctx, fmt.Sprintf("user_id=%v&access_token=%v&api_key=kitefront", id, token))
}

func GetWebsocketClientForAPI(ctx *context.Context, token string) (*TickerClient, error) {
	token = strings.Replace(token, "token ", "", 1)
	apiKey := strings.Split(token, ":")[0]
	accessToken := strings.Replace(token, apiKey+":", "", 1)
	return getWebsocketClient(ctx, fmt.Sprintf("access_token=%v&api_key=%v", accessToken, apiKey))
}

func getWebsocketClient(ctx *context.Context, rawQuery string) (*TickerClient, error) {
	// log.Printf("websocket : start")

	k := &TickerClient{
		Client: &ws.Client{
			URL: &url.URL{
				Scheme:   "wss",
				Host:     "ws.kite.trade",
				RawQuery: rawQuery,
			},
			Header: &http.Header{},
		},
		TickerChan:                 make(chan KiteTicker, 1000),
		BinaryTickerChan:           make(chan []byte, 1000),
		ConnectChan:                make(chan struct{}, 100),
		ErrorChan:                  make(chan interface{}, 100),
		FullTokens:                 map[uint32]bool{},
		QuoteTokens:                map[uint32]bool{},
		LtpTokens:                  map[uint32]bool{},
		HeartBeatIntervalInSeconds: HeartBeatIntervalInSeconds,
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
	// log.Printf("websocket : client ready")
	err = k.Client.Read(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (k *TickerClient) Serve(ctx *context.Context) {
	<-time.After(time.Millisecond)
	// log.Println("websocket : serve")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !k.checkHeartBeat(ctx) {
				// log.Println("no heartbeat")
				break
			}
		case reader := <-k.Client.ReaderChannel:
			k.LastUpdatedTime.Store(time.Now().Unix())
			if reader.Error != nil {
				// log.Printf("Reader error: %v", reader.Error)
				continue
			}
			switch reader.MessageType {
			case ws.TEXT:
				go k.onTextMessage(reader)
			case ws.BINARY:
				go k.onBinaryMessage(reader)
			default:
				// log.Printf("Unknown message type: %v", reader.MessageType)
			}
		case <-(*ctx).Done():
			// log.Println("Context cancelled, stopping ticker client")
			return
		}
	}
}

func (k *TickerClient) Close(ctx *context.Context) error {
	return k.Client.Close(ctx)
}

func (k *TickerClient) Reconnect(ctx *context.Context) error {
	err := k.Close(ctx)
	if err != nil {
		// log.Printf("websocket : attempted to close, got response -> %v", err)
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
	// Resubscribe to LTP
	keys := make([]string, 0, len(k.LtpTokens))
	for k2 := range k.LtpTokens {
		if symbol, ok := TokenSymbolMap[k2]; ok {
			keys = append(keys, symbol)
		}
	}

	for i := 0; i < len(keys); i += 100 {
		end := i + 100
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]
		if err := k.SubscribeLTP(ctx, batch); err != nil {
			// log.Printf("Error resubscribing LTP batch: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Resubscribe to Quote
	keys = make([]string, 0, len(k.QuoteTokens))
	for k2 := range k.QuoteTokens {
		if symbol, ok := TokenSymbolMap[k2]; ok {
			keys = append(keys, symbol)
		}
	}

	for i := 0; i < len(keys); i += 100 {
		end := i + 100
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]
		if err := k.SubscribeQuote(ctx, batch); err != nil {
			// log.Printf("Error resubscribing Quote batch: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Resubscribe to Full
	keys = make([]string, 0, len(k.FullTokens))
	for k2 := range k.FullTokens {
		if symbol, ok := TokenSymbolMap[k2]; ok {
			keys = append(keys, symbol)
		}
	}

	for i := 0; i < len(keys); i += 50 {
		end := i + 50
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[i:end]
		if err := k.SubscribeFull(ctx, batch); err != nil {
			// log.Printf("Error resubscribing Full batch: %v", err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	return nil
}

func (k *TickerClient) SubscribeLTP(ctx *context.Context, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}

	r := &Request{
		Message: "mode",
		Tokens: []interface{}{
			"ltp",
		},
	}

	iTokens := make([]uint32, 0, len(tokens))
	for _, t := range tokens {
		if token, ok := SymbolTokenMap[t]; ok {
			iTokens = append(iTokens, token)
			k.LtpTokens[token] = true
		}
	}
	r.Tokens = append(r.Tokens, iTokens)

	return k.writeTextRequest(ctx, r)
}

func (k *TickerClient) SubscribeFull(ctx *context.Context, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}

	r := &Request{
		Message: "mode",
		Tokens: []interface{}{
			"full",
		},
	}

	iTokens := make([]uint32, 0, len(tokens))
	for _, t := range tokens {
		if token, ok := SymbolTokenMap[t]; ok {
			iTokens = append(iTokens, token)
			k.FullTokens[token] = true
		}
	}
	r.Tokens = append(r.Tokens, iTokens)

	return k.writeTextRequest(ctx, r)
}

func (k *TickerClient) SubscribeQuote(ctx *context.Context, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}

	r := &Request{
		Message: "subscribe",
		Tokens:  make([]interface{}, 0, len(tokens)),
	}

	for _, t := range tokens {
		if token, ok := SymbolTokenMap[t]; ok {
			r.Tokens = append(r.Tokens, token)
			k.QuoteTokens[token] = true
		}
	}

	return k.writeTextRequest(ctx, r)
}

func (k *TickerClient) Unsubscribe(ctx *context.Context, tokens []string) error {
	if len(tokens) == 0 {
		return nil
	}

	r := &Request{
		Message: "unsubscribe",
		Tokens:  make([]interface{}, 0, len(tokens)),
	}

	for _, t := range tokens {
		if token, ok := SymbolTokenMap[t]; ok {
			r.Tokens = append(r.Tokens, token)
			delete(k.QuoteTokens, token)
			delete(k.LtpTokens, token)
			delete(k.FullTokens, token)
		}
	}

	return k.writeTextRequest(ctx, r)
}

func (k *TickerClient) checkHeartBeat(ctx *context.Context) bool {
	if time.Since(time.Unix(k.LastUpdatedTime.Load(), 0)).Seconds() > k.HeartBeatIntervalInSeconds {
		k.HeartBeatIntervalInSeconds *= 2
		go k.Reconnect(ctx)
		return false
	}
	k.HeartBeatIntervalInSeconds = HeartBeatIntervalInSeconds
	return true
}

func (k *TickerClient) writeTextRequest(ctx *context.Context, req *Request) error {
	message, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	err = k.Client.Write(ctx, &ws.Writer{
		MessageType: ws.TEXT,
		Message:     message,
	})
	if err != nil {
		return fmt.Errorf("error writing to websocket: %v", err)
	}

	return nil
}

func (k *TickerClient) onBinaryMessage(reader *ws.Reader) {
	message := reader.Message
	messageSize := len(message)
	// log.Printf("Binary message received: %d bytes", messageSize)

	if messageSize < 2 {
		// log.Printf("Binary message too short: %d bytes (need at least 2 bytes for packet count)", messageSize)
		return
	}

	// First 2 bytes contain number of packets
	numPackets := binary.BigEndian.Uint16(message[0:2])
	// log.Printf("Number of packets in message: %d", numPackets)

	if numPackets == 0 {
		// log.Printf("No packets in message")
		return
	}

	// Forward raw message to binary channel
	select {
	case k.BinaryTickerChan <- message:
	default:
		// log.Printf("Binary ticker channel full, dropping message")
	}

	// Process each packet
	offset := uint32(2) // Skip initial 2 bytes
	for i := uint16(0); i < numPackets; i++ {
		if offset+2 > uint32(messageSize) {
			// log.Printf("Message truncated at packet %d (offset %d, size %d)", i, offset, messageSize)
			return
		}

		// Get packet size (next 2 bytes)
		packetSize := binary.BigEndian.Uint16(message[offset : offset+2])
		// log.Printf("Packet %d: size = %d bytes", i, packetSize)

		// Validate packet size
		if offset+2+uint32(packetSize) > uint32(messageSize) {
			// log.Printf("Packet %d truncated (need %d bytes, have %d)",i, offset+2+uint32(packetSize), messageSize)
			return
		}

		// Extract packet data (skip size bytes)
		packet := Packet(message[offset+2 : offset+2+uint32(packetSize)])
		// log.Printf("Packet %d: extracted %d bytes", i, len(packet))

		// Parse based on mode/size
		var values []uint32
		switch packetSize {
		case 8: // LTP mode
			// log.Printf("Packet %d: LTP mode", i)
			values = packet.parseBinary(8)
		case 44: // Quote mode
			// log.Printf("Packet %d: Quote mode", i)
			values = packet.parseBinary(44)
		case 184: // Full mode
			// log.Printf("Packet %d: Full mode", i)
			values = packet.parseBinary(64) // First parse main data
		default:
			// log.Printf("Packet %d: Unknown size %d, first bytes: % x",i, packetSize, packet[:min(16, len(packet))])
			offset += 2 + uint32(packetSize) // Skip unknown packet
			continue
		}

		if len(values) < 2 {
			// log.Printf("Packet %d: Insufficient values (got %d, need at least 2)", i, len(values))
			offset += 2 + uint32(packetSize)
			continue
		}

		// Create ticker
		ticker := KiteTicker{}
		ticker.Token = values[0]
		ticker.LastPrice = float64(values[1]) / 100

		// Get symbol from token map
		var ok bool
		ticker.TradingSymbol, ok = TokenSymbolMap[ticker.Token]
		if !ok {
			// log.Printf("Unknown token: %d", ticker.Token)
			offset += 2 + uint32(packetSize)
			continue
		}

		// log.Printf(ticker.TradingSymbol)

		// Parse mode-specific data
		switch len(values) {
		case 2: // LTP packet
			// log.Printf("LTP data processed for %s", ticker.TradingSymbol)

		case 8: // Quote packet
			ticker.High = float64(values[2]) / 100
			ticker.Low = float64(values[3]) / 100
			ticker.Open = float64(values[4]) / 100
			ticker.Close = float64(values[5]) / 100
			ticker.PriceChange = float64(values[6]) / 100
			ticker.ExchangeTimestamp = time.Unix(int64(values[7]), 0)
			// log.Printf("Quote data processed for %s: OHLC %.2f/%.2f/%.2f/%.2f", ticker.TradingSymbol, ticker.Open, ticker.High, ticker.Low, ticker.Close)

		case 16: // Full packet
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

			// Parse market depth for full mode
			if packetSize == 184 {
				depthData := packet[64:] // Market depth starts after main data
				depthValues := depthData.parseMarketDepth()
				if len(depthValues) > 0 {
					numDepthEntries := len(depthValues) / 3
					midPoint := numDepthEntries / 2
					// log.Printf("Market depth entries for %s: %d", ticker.TradingSymbol, numDepthEntries)

					// First half is buy orders
					for j := 0; j < midPoint; j++ {
						idx := j * 3
						ticker.Depth.Buy = append(ticker.Depth.Buy, LimitOrder{
							Quantity: depthValues[idx],
							Price:    float64(depthValues[idx+1]) / 100,
							Orders:   depthValues[idx+2],
						})
					}

					// Second half is sell orders
					for j := midPoint; j < numDepthEntries; j++ {
						idx := j * 3
						ticker.Depth.Sell = append(ticker.Depth.Sell, LimitOrder{
							Quantity: depthValues[idx],
							Price:    float64(depthValues[idx+1]) / 100,
							Orders:   depthValues[idx+2],
						})
					}

					// log.Printf("Market depth processed for %s: %d buy, %d sell", ticker.TradingSymbol, len(ticker.Depth.Buy), len(ticker.Depth.Sell))
				}
			}
		}
		// Send ticker to channel
		select {
		case k.TickerChan <- ticker:

			// count++
			// log.Printf("Ticker sent to channel: %s", ticker.TradingSymbol)
		default:
			// log.Printf("Ticker channel full, dropping update for %s", ticker.TradingSymbol)
		}

		// Move to next packet
		offset += 2 + uint32(packetSize)
	}
}

func (k *TickerClient) onTextMessage(reader *ws.Reader) {
	if len(reader.Message) == 0 {
		return
	}

	m := &Message{}
	err := json.Unmarshal(reader.Message, m)
	if err != nil {
		// log.Printf("Error unmarshaling text message: %v", err)
		return
	}

	switch m.Type {
	case "instruments_meta":
		k.ConnectChan <- struct{}{}
	case "error":
		k.ErrorChan <- m.Data
		// log.Printf("Received error message: %v", m.Data)
	default:
		// log.Printf("Unknown text message type: %s", m.Type)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
