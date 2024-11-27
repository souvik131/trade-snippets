package ws

import (
	"bytes"
	"context"
	"encoding/binary"
	"log"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"
)

type MarketDataServer struct {
	clients    map[*Client]bool
	broadcast  chan []MarketData
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex

	// Latest valid market data storage
	latestData     map[string]*MarketData // key: stockName-instrumentType
	latestDataLock sync.RWMutex
}

type LimitOrder struct {
	Price    float64 `json:"price"`
	Quantity uint32  `json:"quantity"`
	Orders   uint32  `json:"orders"`
}

type MarketDepth struct {
	Buy  []LimitOrder `json:"buy"`
	Sell []LimitOrder `json:"sell"`
}

type MarketData struct {
	StockName           string      `json:"stockName"`
	TradingSymbol       string      `json:"tradingSymbol"`
	InstrumentType      string      `json:"instrumentType"`
	StrikePrice         float64     `json:"strikePrice,omitempty"`
	LastPrice           float64     `json:"lastPrice"`
	LastTradedQuantity  uint32      `json:"lastTradedQuantity"`
	AverageTradedPrice  float64     `json:"averageTradedPrice"`
	VolumeTraded        uint32      `json:"volumeTraded"`
	TotalBuy            uint32      `json:"totalBuy"`
	TotalSell           uint32      `json:"totalSell"`
	High                float64     `json:"high"`
	Low                 float64     `json:"low"`
	Open                float64     `json:"open"`
	Close               float64     `json:"close"`
	OI                  uint32      `json:"oi"`
	OIHigh              uint32      `json:"oiHigh"`
	OILow               uint32      `json:"oiLow"`
	LastTradedTimestamp time.Time   `json:"lastTradedTimestamp"`
	ExchangeTimestamp   time.Time   `json:"exchangeTimestamp"`
	LotSize             float64     `json:"lotSize"`
	Expiry              string      `json:"expiry"`
	Depth               MarketDepth `json:"depth"`
	LastUpdateTime      time.Time   `json:"lastUpdateTime"`
}

func NewMarketDataServer() *MarketDataServer {
	s := &MarketDataServer{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []MarketData, 100),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		latestData: make(map[string]*MarketData),
	}

	go s.processBatches()

	return s
}

func (s *MarketDataServer) getDataKey(data MarketData) string {
	if data.InstrumentType == "FUT" {
		return data.StockName + "-FUT-" + data.Expiry
	}
	return data.StockName + "-" + data.InstrumentType + "-" + data.Expiry + "-" + data.TradingSymbol
}

// RemoveOptionData removes data for unsubscribed options
func (s *MarketDataServer) RemoveOptionData(stockName string, expiry string, symbols []string) {
	s.latestDataLock.Lock()
	defer s.latestDataLock.Unlock()

	// Create a map of symbols for quick lookup
	symbolMap := make(map[string]bool)
	for _, symbol := range symbols {
		symbolMap[symbol] = true
	}

	// Remove data for these symbols
	for key, data := range s.latestData {
		if data.StockName == stockName && data.Expiry == expiry {
			if symbolMap[data.TradingSymbol] {
				delete(s.latestData, key)
			}
		}
	}
}

func (s *MarketDataServer) updateLatestData(data MarketData) {
	// Only store if we have valid data
	if data.LastPrice <= 0 && len(data.Depth.Buy) == 0 && len(data.Depth.Sell) == 0 {
		return
	}

	s.latestDataLock.Lock()
	defer s.latestDataLock.Unlock()

	key := s.getDataKey(data)
	data.LastUpdateTime = time.Now()
	s.latestData[key] = &data
}

func (s *MarketDataServer) getLatestValidData() []MarketData {
	s.latestDataLock.RLock()
	defer s.latestDataLock.RUnlock()

	var validData []MarketData
	for _, data := range s.latestData {
		if data.LastPrice > 0 || len(data.Depth.Buy) > 0 || len(data.Depth.Sell) > 0 {
			validData = append(validData, *data)
		}
	}
	return validData
}

func (s *MarketDataServer) processBatches() {
	batch := make([]MarketData, 0, 100)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case data := <-s.broadcast:
			for _, d := range data {
				s.updateLatestData(d)
			}
			batch = append(batch, data...)

		case <-ticker.C:
			if len(batch) > 0 {
				s.sendBatch(batch)
				batch = make([]MarketData, 0, 100)
			}
		}
	}
}

func (s *MarketDataServer) sendBatch(batch []MarketData) {
	buf := new(bytes.Buffer)
	binary.Write(buf, binary.LittleEndian, uint16(len(batch)))

	for _, data := range batch {
		binary.Write(buf, binary.LittleEndian, uint8(len(data.StockName)))
		buf.WriteString(data.StockName)

		var instType uint8
		switch data.InstrumentType {
		case "CE":
			instType = 1
		case "PE":
			instType = 2
		default:
			instType = 0
		}
		binary.Write(buf, binary.LittleEndian, instType)

		binary.Write(buf, binary.LittleEndian, float32(data.LastPrice))
		binary.Write(buf, binary.LittleEndian, float32(data.StrikePrice))
		binary.Write(buf, binary.LittleEndian, float32(data.LotSize))

		// Write expiry date length and string
		binary.Write(buf, binary.LittleEndian, uint8(len(data.Expiry)))
		buf.WriteString(data.Expiry)

		// Write best bid/ask from depth
		if len(data.Depth.Buy) > 0 {
			binary.Write(buf, binary.LittleEndian, float32(data.Depth.Buy[0].Price))
		} else {
			binary.Write(buf, binary.LittleEndian, float32(0))
		}
		if len(data.Depth.Sell) > 0 {
			binary.Write(buf, binary.LittleEndian, float32(data.Depth.Sell[0].Price))
		} else {
			binary.Write(buf, binary.LittleEndian, float32(0))
		}

		// Write additional market data fields
		binary.Write(buf, binary.LittleEndian, data.LastTradedQuantity)
		binary.Write(buf, binary.LittleEndian, float32(data.AverageTradedPrice))
		binary.Write(buf, binary.LittleEndian, data.VolumeTraded)
		binary.Write(buf, binary.LittleEndian, data.TotalBuy)
		binary.Write(buf, binary.LittleEndian, data.TotalSell)
		binary.Write(buf, binary.LittleEndian, float32(data.High))
		binary.Write(buf, binary.LittleEndian, float32(data.Low))
		binary.Write(buf, binary.LittleEndian, float32(data.Open))
		binary.Write(buf, binary.LittleEndian, float32(data.Close))
		binary.Write(buf, binary.LittleEndian, data.OI)
		binary.Write(buf, binary.LittleEndian, data.OIHigh)
		binary.Write(buf, binary.LittleEndian, data.OILow)
		binary.Write(buf, binary.LittleEndian, uint32(data.LastTradedTimestamp.Unix()))
		binary.Write(buf, binary.LittleEndian, uint32(data.ExchangeTimestamp.Unix()))
	}

	message := buf.Bytes()

	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx := context.Background()
	for client := range s.clients {
		err := client.Write(&ctx, &Writer{
			MessageType: BINARY,
			Message:     message,
		})
		if err != nil {
			log.Printf("Error broadcasting to client: %v", err)
			client.Close(&ctx)
			delete(s.clients, client)
		}
	}
}

func (s *MarketDataServer) sendLatestToClient(client *Client) {
	validData := s.getLatestValidData()
	if len(validData) > 0 {
		log.Printf("Sending %d latest records to new client", len(validData))
		ctx := context.Background()
		buf := new(bytes.Buffer)
		binary.Write(buf, binary.LittleEndian, uint16(len(validData)))

		for _, data := range validData {
			binary.Write(buf, binary.LittleEndian, uint8(len(data.StockName)))
			buf.WriteString(data.StockName)

			var instType uint8
			switch data.InstrumentType {
			case "CE":
				instType = 1
			case "PE":
				instType = 2
			default:
				instType = 0
			}
			binary.Write(buf, binary.LittleEndian, instType)

			binary.Write(buf, binary.LittleEndian, float32(data.LastPrice))
			binary.Write(buf, binary.LittleEndian, float32(data.StrikePrice))
			binary.Write(buf, binary.LittleEndian, float32(data.LotSize))

			// Write expiry date length and string
			binary.Write(buf, binary.LittleEndian, uint8(len(data.Expiry)))
			buf.WriteString(data.Expiry)

			// Write best bid/ask from depth
			if len(data.Depth.Buy) > 0 {
				binary.Write(buf, binary.LittleEndian, float32(data.Depth.Buy[0].Price))
			} else {
				binary.Write(buf, binary.LittleEndian, float32(0))
			}
			if len(data.Depth.Sell) > 0 {
				binary.Write(buf, binary.LittleEndian, float32(data.Depth.Sell[0].Price))
			} else {
				binary.Write(buf, binary.LittleEndian, float32(0))
			}

			// Write additional market data fields
			binary.Write(buf, binary.LittleEndian, data.LastTradedQuantity)
			binary.Write(buf, binary.LittleEndian, float32(data.AverageTradedPrice))
			binary.Write(buf, binary.LittleEndian, data.VolumeTraded)
			binary.Write(buf, binary.LittleEndian, data.TotalBuy)
			binary.Write(buf, binary.LittleEndian, data.TotalSell)
			binary.Write(buf, binary.LittleEndian, float32(data.High))
			binary.Write(buf, binary.LittleEndian, float32(data.Low))
			binary.Write(buf, binary.LittleEndian, float32(data.Open))
			binary.Write(buf, binary.LittleEndian, float32(data.Close))
			binary.Write(buf, binary.LittleEndian, data.OI)
			binary.Write(buf, binary.LittleEndian, data.OIHigh)
			binary.Write(buf, binary.LittleEndian, data.OILow)
			binary.Write(buf, binary.LittleEndian, uint32(data.LastTradedTimestamp.Unix()))
			binary.Write(buf, binary.LittleEndian, uint32(data.ExchangeTimestamp.Unix()))
		}

		err := client.Write(&ctx, &Writer{
			MessageType: BINARY,
			Message:     buf.Bytes(),
		})
		if err != nil {
			log.Printf("Error sending initial data to client: %v", err)
		}
	}
}

func (s *MarketDataServer) Start() {
	log.Println("Market data server started")
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()
			log.Printf("New client connected. Total clients: %d", len(s.clients))

			// Send latest valid data immediately
			go s.sendLatestToClient(client)

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				ctx := context.Background()
				client.Close(&ctx)
			}
			s.mu.Unlock()
			log.Printf("Client disconnected. Total clients: %d", len(s.clients))
		}
	}
}

func (s *MarketDataServer) ServeWs(w http.ResponseWriter, r *http.Request) {
	log.Printf("New WebSocket connection request from %s", r.RemoteAddr)
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		log.Printf("Error accepting websocket: %v", err)
		return
	}

	client := &Client{
		Conn:          conn,
		ReaderChannel: make(chan *Reader, 100),
		IsInitialized: true,
	}

	s.register <- client
	log.Printf("WebSocket client registered: %s", r.RemoteAddr)

	ctx := r.Context()
	for {
		messageType, message, err := conn.Read(ctx)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			s.unregister <- client
			break
		}

		client.ReaderChannel <- &Reader{
			MessageType: MessageType(messageType),
			Message:     message,
			Error:       nil,
		}
	}
}

func (s *MarketDataServer) BroadcastMarketData(data MarketData) {
	s.broadcast <- []MarketData{data}
}
