package main

import (
	"context"
	"log"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/souvik131/trade-snippets/kite"
	"github.com/souvik131/trade-snippets/ws"
)

var instrumentsPerSocket = 3000.0
var instrumentsPerRequest = 100.0 // Reduced batch size
var dateFormat = "2006-01-02"

type StockOptions struct {
	Future      string
	FuturePrice float64
	LotSize     float64
	Options     map[string]kite.KiteTicker
	mu          sync.RWMutex
}

type FutureStatus struct {
	tradingSymbol string
	received      bool
	price         float64
	subscribed    bool
}

var activeSubscriptions = struct {
	count int
	mu    sync.RWMutex
}{count: 0}

func updateSubscriptionCount(delta int) {
	activeSubscriptions.mu.Lock()
	defer activeSubscriptions.mu.Unlock()

	newCount := activeSubscriptions.count + delta
	if newCount > 3000 {
		// log.Printf("Warning: Would exceed 3000 limit (%d + %d = %d)", activeSubscriptions.count, delta, newCount)
		return
	}

	activeSubscriptions.count = newCount
	// log.Printf("Active subscriptions: %d", activeSubscriptions.count)
}

func Serve(ctx *context.Context, k *kite.Kite) {
	// log.Println("Starting server...")

	k.TickerClients = []*kite.TickerClient{}
	stockOptionsMap := make(map[string]*StockOptions)
	futuresStatus := make(map[string]*FutureStatus)
	var futuresLock sync.RWMutex

	marketDataServer := ws.NewMarketDataServer()
	go marketDataServer.Start()

	http.Handle("/", http.FileServer(http.Dir(".")))
	http.HandleFunc("/ws", marketDataServer.ServeWs)
	go func() {
		// log.Printf("Starting HTTP server on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal("HTTP server error:", err)
		}
	}()

	// Phase 1: Load all stock futures
	stocks := getStockFutures(*kite.BrokerInstrumentTokens)
	// log.Printf("Found %d stocks in F&O", len(stocks))

	// Initialize websocket client
	// log.Println("Initializing Kite WebSocket client...")
	ticker, err := k.GetWebSocketClient(ctx)
	if err != nil {
		log.Panicf("Failed to get websocket client: %v", err)
	}
	k.TickerClients = append(k.TickerClients, ticker)
	k.TickSymbolMap = map[string]kite.KiteTicker{}

	// Subscribe to futures in smaller batches
	go func(t *kite.TickerClient) {
		// log.Println("Waiting for connect channel...")
		for range t.ConnectChan {
			log.Printf("Websocket connected, preparing futures subscription...")

			// First, prepare all futures data
			futuresToSubscribe := make([]string, 0)
			for _, stock := range stocks {
				expiry := getLatestExpiry(*kite.BrokerInstrumentTokens, stock)
				if expiry == "" {
					continue
				}

				lotSize := getLotSize(*kite.BrokerInstrumentTokens, stock)

				for _, inst := range *kite.BrokerInstrumentTokens {
					if inst.Name == stock && inst.Exchange == "NFO" &&
						inst.InstrumentType == "FUT" && inst.Expiry == expiry {
						futuresToSubscribe = append(futuresToSubscribe, inst.TradingSymbol)

						stockOptionsMap[stock] = &StockOptions{
							Future:  inst.TradingSymbol,
							Options: make(map[string]kite.KiteTicker),
							LotSize: lotSize,
						}

						// Initialize future status
						futuresLock.Lock()
						futuresStatus[stock] = &FutureStatus{
							tradingSymbol: inst.TradingSymbol,
							received:      false,
							price:         0,
							subscribed:    false,
						}
						futuresLock.Unlock()

						// log.Printf("Added future for %s: %s (lot size: %.0f)", stock, inst.TradingSymbol, lotSize)
						break
					}
				}
			}

			// Subscribe to futures in smaller batches with delays
			// log.Printf("Starting subscription for %d futures in batches...", len(futuresToSubscribe))
			for i := 0; i < len(futuresToSubscribe); i += int(instrumentsPerRequest) {
				end := i + int(instrumentsPerRequest)
				if end > len(futuresToSubscribe) {
					end = len(futuresToSubscribe)
				}

				batch := futuresToSubscribe[i:end]
				// log.Printf("Subscribing to futures batch %d-%d of %d", i+1, end, len(futuresToSubscribe))

				err := t.SubscribeFull(ctx, batch)
				if err != nil {
					// log.Printf("Error subscribing to futures batch: %v", err)
					continue
				}

				// Mark these futures as subscribed
				futuresLock.Lock()
				for _, symbol := range batch {
					for _, status := range futuresStatus {
						if status.tradingSymbol == symbol {
							status.subscribed = true
							// log.Printf("Marked %s as subscribed", stock)
							break
						}
					}
				}
				futuresLock.Unlock()

				updateSubscriptionCount(len(batch))
				time.Sleep(2 * time.Second) // Increased delay between batches
			}

			// Start a goroutine to check when all futures are received
			go func() {
				checkInterval := time.NewTicker(2 * time.Second)
				timeout := time.After(60 * time.Second) // Increased timeout
				allFuturesReceived := false

				for !allFuturesReceived {
					select {
					case <-checkInterval.C:
						futuresLock.RLock()
						receivedCount := 0
						subscribedCount := 0
						pendingFutures := []string{}
						for stock, status := range futuresStatus {
							if status.received {
								receivedCount++
							}
							if status.subscribed {
								subscribedCount++
							}
							if !status.received && status.subscribed {
								pendingFutures = append(pendingFutures, stock)
							}
						}
						// totalFutures := len(futuresStatus)
						futuresLock.RUnlock()

						// log.Printf("Status: %d/%d subscribed, %d/%d received", subscribedCount, totalFutures, receivedCount, totalFutures)

						if receivedCount >= subscribedCount && subscribedCount > 0 {
							allFuturesReceived = true
							// log.Printf("All subscribed futures data received (%d/%d). Starting options subscription...", receivedCount, totalFutures)

							// Now subscribe to ATM options
							for stock, status := range futuresStatus {
								if !status.received || status.price <= 0 {
									// log.Printf("Skipping options for %s: no price data", stock)
									continue
								}

								expiry := getLatestExpiry(*kite.BrokerInstrumentTokens, stock)
								strikes := getStockStrikes(*kite.BrokerInstrumentTokens, stock, expiry)
								atmStrike := getNearestStrike(status.price, strikes)

								var optionsToSubscribe []string
								for _, inst := range *kite.BrokerInstrumentTokens {
									if inst.Name == stock && inst.Strike == atmStrike &&
										inst.Expiry == expiry && (inst.InstrumentType == "CE" || inst.InstrumentType == "PE") {
										optionsToSubscribe = append(optionsToSubscribe, inst.TradingSymbol)
									}
								}

								if len(optionsToSubscribe) > 0 {
									// log.Printf("Subscribing to ATM options for %s: %v", stock, optionsToSubscribe)
									err := t.SubscribeFull(ctx, optionsToSubscribe)
									if err != nil {
										// log.Printf("Error subscribing to options: %v", err)
									}
									updateSubscriptionCount(len(optionsToSubscribe))
									time.Sleep(500 * time.Millisecond)
								}
							}
						} else if len(pendingFutures) > 0 {
							// log.Printf("Waiting for futures data... (%d/%d received, %d/%d subscribed). Pending subscribed: %v", receivedCount, totalFutures, subscribedCount, totalFutures, pendingFutures[:min(5, len(pendingFutures))])
						}

					case <-timeout:
						// log.Printf("Timeout waiting for futures data. Proceeding with available data...")
						allFuturesReceived = true
					}
				}
			}()
		}
	}(ticker)

	count := 0
	// Handle ticker updates
	go func(t *kite.TickerClient) {
		for tick := range t.TickerChan {
			count++
			if strings.HasSuffix(tick.TradingSymbol, "FUT") {
				var stockName string
				for stock, opts := range stockOptionsMap {
					if opts.Future == tick.TradingSymbol {
						stockName = stock
						break
					}
				}

				if stockName == "" {
					continue
				}

				go func(tick kite.KiteTicker) {

					stockOpts := stockOptionsMap[stockName]
					stockOpts.mu.Lock()

					if tick.LastPrice > 0 {
						// Mark this future as received
						futuresLock.Lock()
						if status, exists := futuresStatus[stockName]; exists {
							if !status.received {
								status.received = true
								// log.Printf("Received first future price for %s: %v", stockName, tick.LastPrice)
							}
							status.price = tick.LastPrice
						}
						futuresLock.Unlock()

						// log.Printf("Broadcasting future data for %s: %v", stockName, tick.LastPrice)
						marketDataServer.BroadcastMarketData(ws.MarketData{
							StockName:      stockName,
							TradingSymbol:  tick.TradingSymbol,
							InstrumentType: "FUT",
							LastPrice:      tick.LastPrice,
							LotSize:        stockOpts.LotSize,
							Depth:          convertDepth(tick.Depth),
							LastUpdateTime: time.Now(),
						})

						// Update future price and trigger options subscription if needed
						if math.Abs(stockOpts.FuturePrice-tick.LastPrice) > 0.5 {
							stockOpts.FuturePrice = tick.LastPrice

							expiry := getLatestExpiry(*kite.BrokerInstrumentTokens, stockName)
							strikes := getStockStrikes(*kite.BrokerInstrumentTokens, stockName, expiry)
							atmStrike := getNearestStrike(tick.LastPrice, strikes)

							var toSubscribe []string
							for _, inst := range *kite.BrokerInstrumentTokens {
								if inst.Name == stockName && inst.Strike == atmStrike &&
									inst.Expiry == expiry && (inst.InstrumentType == "CE" || inst.InstrumentType == "PE") {
									toSubscribe = append(toSubscribe, inst.TradingSymbol)
								}
							}

							if len(toSubscribe) > 0 {
								// log.Printf("Updating ATM options for %s: %v", stockName, toSubscribe)
								err := t.SubscribeFull(ctx, toSubscribe)
								if err != nil {
									// log.Printf("Error subscribing to options: %v", err)
								}
								updateSubscriptionCount(len(toSubscribe))
							}
						}
					}

					stockOpts.mu.Unlock()
				}(tick)
			} else {
				// log.Println("Starting option handler...")
				for tick := range t.TickerChan {
					var inst *kite.Instrument
					for _, i := range *kite.BrokerInstrumentTokens {
						if i.TradingSymbol == tick.TradingSymbol {
							inst = i
							break
						}
					}

					if inst == nil || (inst.InstrumentType != "CE" && inst.InstrumentType != "PE") {
						continue
					}

					if tick.LastPrice > 0 {
						// log.Printf("Broadcasting option data for %s %s: %v", inst.Name, inst.InstrumentType, tick.LastPrice)
						go marketDataServer.BroadcastMarketData(ws.MarketData{
							StockName:      inst.Name,
							TradingSymbol:  tick.TradingSymbol,
							InstrumentType: inst.InstrumentType,
							StrikePrice:    inst.Strike,
							LastPrice:      tick.LastPrice,
							LotSize:        inst.LotSize,
							Depth:          convertDepth(tick.Depth),
							LastUpdateTime: time.Now(),
						})
					}
				}
			}
		}
	}(ticker)

	go ticker.Serve(ctx)
	// log.Println("Websocket service started")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getLatestExpiry(instruments map[string]*kite.Instrument, stockName string) string {
	var expiries []string
	for _, inst := range instruments {
		if inst.Name == stockName && inst.Exchange == "NFO" && inst.InstrumentType == "FUT" {
			expiries = append(expiries, inst.Expiry)
		}
	}
	if len(expiries) == 0 {
		return ""
	}
	sort.Strings(expiries)
	return expiries[0]
}

func getLotSize(instruments map[string]*kite.Instrument, stockName string) float64 {
	for _, inst := range instruments {
		if inst.Name == stockName && inst.Exchange == "NFO" {
			return inst.LotSize
		}
	}
	return 0
}

func getNearestStrike(futurePrice float64, strikes []float64) float64 {
	sort.Float64s(strikes)
	atmIndex := sort.SearchFloat64s(strikes, futurePrice)

	if atmIndex >= len(strikes) {
		return strikes[len(strikes)-1]
	}
	if atmIndex == 0 {
		return strikes[0]
	}

	if atmIndex < len(strikes) {
		if math.Abs(strikes[atmIndex]-futurePrice) < math.Abs(strikes[atmIndex-1]-futurePrice) {
			return strikes[atmIndex]
		}
		return strikes[atmIndex-1]
	}
	return strikes[atmIndex-1]
}

func getStockStrikes(instruments map[string]*kite.Instrument, stockName, expiry string) []float64 {
	var strikes []float64
	strikesMap := make(map[float64]bool)

	for _, inst := range instruments {
		if inst.Name == stockName && inst.Exchange == "NFO" && inst.Expiry == expiry &&
			(inst.InstrumentType == "CE" || inst.InstrumentType == "PE") {
			if !strikesMap[inst.Strike] {
				strikes = append(strikes, inst.Strike)
				strikesMap[inst.Strike] = true
			}
		}
	}
	return strikes
}

func getStockFutures(instruments map[string]*kite.Instrument) []string {
	stocksMap := make(map[string]bool)
	var stocks []string

	for _, inst := range instruments {
		if inst.Exchange == "NFO" && inst.InstrumentType == "FUT" {
			if !stocksMap[inst.Name] {
				stocks = append(stocks, inst.Name)
				stocksMap[inst.Name] = true
			}
		}
	}
	sort.Strings(stocks)
	return stocks
}

func convertDepth(kiteDepth kite.Depth) ws.MarketDepth {
	wsDepth := ws.MarketDepth{
		Buy:  make([]ws.LimitOrder, len(kiteDepth.Buy)),
		Sell: make([]ws.LimitOrder, len(kiteDepth.Sell)),
	}

	for i, order := range kiteDepth.Buy {
		wsDepth.Buy[i] = ws.LimitOrder{
			Price:    order.Price,
			Quantity: order.Quantity,
			Orders:   order.Orders,
		}
	}

	for i, order := range kiteDepth.Sell {
		wsDepth.Sell[i] = ws.LimitOrder{
			Price:    order.Price,
			Quantity: order.Quantity,
			Orders:   order.Orders,
		}
	}

	return wsDepth
}
