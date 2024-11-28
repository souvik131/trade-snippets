package main

import (
	"context"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/souvik131/trade-snippets/kite"
	"github.com/souvik131/trade-snippets/ws"
)

var instrumentsPerSocket = 3000.0
var instrumentsPerRequest = 500.0 // Reduced batch size
var dateFormat = "2006-01-02"

// Change stockOptionsMap to use expiry in the key
var stockOptionsMap = make(map[string]*StockOptions) // key: stockName-expiry

type StockOptions struct {
	Future      string
	FuturePrice float64
	LotSize     float64
	Options     map[string]kite.KiteTicker
	mu          sync.RWMutex
	CurrentATM  map[string][]string
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

func getStockOptionsKey(stockName, expiry string) string {
	return stockName + "-" + expiry
}

func updateSubscriptionCount(delta int) {
	activeSubscriptions.mu.Lock()
	defer activeSubscriptions.mu.Unlock()

	newCount := activeSubscriptions.count + delta
	if newCount > 3000 {
		log.Printf("Warning: Would exceed 3000 limit (%d + %d = %d)", activeSubscriptions.count, delta, newCount)
		return
	}

	activeSubscriptions.count = newCount
	log.Printf("Active subscriptions: %d", activeSubscriptions.count)
}

func getLatestExpiry(instruments map[string]*kite.Instrument, stockName string) []string {
	var expiries []string
	expiryMap := make(map[string]bool)

	// Collect unique expiries
	for _, inst := range instruments {
		if inst.Name == stockName && inst.Exchange == "NFO" && inst.InstrumentType == "FUT" {
			if !expiryMap[inst.Expiry] {
				expiries = append(expiries, inst.Expiry)
				expiryMap[inst.Expiry] = true
			}
		}
	}

	if len(expiries) == 0 {
		return nil
	}

	// Sort expiries
	sort.Strings(expiries)

	// Return up to 2 expiries
	if len(expiries) > 2 {
		return expiries[:2]
	}
	return expiries
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

func Serve(ctx *context.Context, k *kite.Kite) {
	k.TickerClients = []*kite.TickerClient{}
	futuresStatus := make(map[string]*FutureStatus) // key: tradingSymbol
	var futuresLock sync.RWMutex

	marketDataServer := ws.NewMarketDataServer()
	go marketDataServer.Start()

	http.Handle("/", http.FileServer(http.Dir(os.Getenv("WEBPATH"))))
	http.HandleFunc("/ws", marketDataServer.ServeWs)
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatal("HTTP server error:", err)
		}
	}()

	// Phase 1: Load all stock futures
	stocks := getStockFutures(*kite.BrokerInstrumentTokens)

	// Initialize websocket client
	ticker, err := k.GetWebSocketClient(ctx)
	if err != nil {
		log.Panicf("Failed to get websocket client: %v", err)
	}
	k.TickerClients = append(k.TickerClients, ticker)
	k.TickSymbolMap = map[string]kite.KiteTicker{}

	// Subscribe to futures in smaller batches
	go func(t *kite.TickerClient) {
		for range t.ConnectChan {
			log.Printf("Websocket connected, preparing futures subscription...")

			// First, prepare all futures data
			futuresToSubscribe := make([]string, 0)
			for _, stock := range stocks {
				expiries := getLatestExpiry(*kite.BrokerInstrumentTokens, stock)
				if len(expiries) == 0 {
					continue
				}

				lotSize := getLotSize(*kite.BrokerInstrumentTokens, stock)

				// Subscribe to futures for each expiry
				for _, expiry := range expiries {
					for _, inst := range *kite.BrokerInstrumentTokens {
						if inst.Name == stock && inst.Exchange == "NFO" &&
							inst.InstrumentType == "FUT" && inst.Expiry == expiry {
							futuresToSubscribe = append(futuresToSubscribe, inst.TradingSymbol)

							// Use composite key for stockOptionsMap
							key := getStockOptionsKey(stock, expiry)
							stockOptionsMap[key] = &StockOptions{
								Future:     inst.TradingSymbol,
								Options:    make(map[string]kite.KiteTicker),
								LotSize:    lotSize,
								CurrentATM: make(map[string][]string),
							}

							// Initialize future status using trading symbol as key
							futuresLock.Lock()
							futuresStatus[inst.TradingSymbol] = &FutureStatus{
								tradingSymbol: inst.TradingSymbol,
								received:      false,
								price:         0,
								subscribed:    false,
							}
							futuresLock.Unlock()
							break
						}
					}
				}
			}

			// Subscribe to futures in smaller batches with delays
			for i := 0; i < len(futuresToSubscribe); i += int(instrumentsPerRequest) {
				end := i + int(instrumentsPerRequest)
				if end > len(futuresToSubscribe) {
					end = len(futuresToSubscribe)
				}

				batch := futuresToSubscribe[i:end]

				err := t.SubscribeFull(ctx, batch)
				if err != nil {
					continue
				}

				// Mark these futures as subscribed
				futuresLock.Lock()
				for _, symbol := range batch {
					if status, exists := futuresStatus[symbol]; exists {
						status.subscribed = true
					}
				}
				futuresLock.Unlock()

				updateSubscriptionCount(len(batch))
				time.Sleep(2 * time.Second)
			}

			// Start a goroutine to check when all futures are received
			go func() {
				checkInterval := time.NewTicker(2 * time.Second)
				timeout := time.After(60 * time.Second)
				allFuturesReceived := false

				for !allFuturesReceived {
					select {
					case <-checkInterval.C:
						futuresLock.RLock()
						receivedCount := 0
						subscribedCount := 0
						pendingFutures := []string{}
						for symbol, status := range futuresStatus {
							if status.received {
								receivedCount++
							}
							if status.subscribed {
								subscribedCount++
							}
							if !status.received && status.subscribed {
								pendingFutures = append(pendingFutures, symbol)
							}
						}
						futuresLock.RUnlock()

						if receivedCount >= subscribedCount && subscribedCount > 0 {
							allFuturesReceived = true

							// Now subscribe to ATM options
							for symbol, status := range futuresStatus {
								if !status.received || status.price <= 0 {
									continue
								}

								// Find stock and expiry from trading symbol
								var stockName, expiry string
								for _, inst := range *kite.BrokerInstrumentTokens {
									if inst.TradingSymbol == symbol {
										stockName = inst.Name
										expiry = inst.Expiry
										break
									}
								}

								if stockName == "" || expiry == "" {
									continue
								}

								key := getStockOptionsKey(stockName, expiry)
								stockOpts := stockOptionsMap[key]
								if stockOpts == nil {
									continue
								}

								strikes := getStockStrikes(*kite.BrokerInstrumentTokens, stockName, expiry)
								atmStrike := getNearestStrike(status.price, strikes)

								// Unsubscribe from previous ATM options
								if prevATM, ok := stockOpts.CurrentATM[expiry]; ok && len(prevATM) > 0 {
									marketDataServer.RemoveOptionData(stockName, expiry, prevATM)
									err := t.Unsubscribe(ctx, prevATM)
									if err != nil {
										log.Printf("Error unsubscribing from previous ATM options: %v", err)
									} else {
										updateSubscriptionCount(-len(prevATM))
									}
								}

								// Get new ATM options
								var optionsToSubscribe []string
								for _, inst := range *kite.BrokerInstrumentTokens {
									if inst.Name == stockName && inst.Strike == atmStrike &&
										inst.Expiry == expiry && (inst.InstrumentType == "CE" || inst.InstrumentType == "PE") {
										optionsToSubscribe = append(optionsToSubscribe, inst.TradingSymbol)
									}
								}

								if len(optionsToSubscribe) > 0 {
									err := t.SubscribeFull(ctx, optionsToSubscribe)
									if err != nil {
										log.Printf("Error subscribing to new ATM options: %v", err)
									} else {
										stockOpts.CurrentATM[expiry] = optionsToSubscribe
										updateSubscriptionCount(len(optionsToSubscribe))
									}
									time.Sleep(500 * time.Millisecond)
								}
							}
						}

					case <-timeout:
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
				// Find stock and expiry from trading symbol
				var stockName, expiry string
				for _, inst := range *kite.BrokerInstrumentTokens {
					if inst.TradingSymbol == tick.TradingSymbol {
						stockName = inst.Name
						expiry = inst.Expiry
						break
					}
				}

				if stockName == "" || expiry == "" {
					continue
				}

				key := getStockOptionsKey(stockName, expiry)
				stockOpts := stockOptionsMap[key]
				if stockOpts == nil {
					continue
				}

				go func(tick kite.KiteTicker) {
					stockOpts.mu.Lock()

					if tick.LastPrice > 0 {
						// Mark this future as received
						futuresLock.Lock()
						if status, exists := futuresStatus[tick.TradingSymbol]; exists {
							if !status.received {
								status.received = true
							}
							status.price = tick.LastPrice
						}
						futuresLock.Unlock()

						marketDataServer.BroadcastMarketData(ws.MarketData{
							StockName:           stockName,
							TradingSymbol:       tick.TradingSymbol,
							InstrumentType:      "FUT",
							LastPrice:           tick.LastPrice,
							LastTradedQuantity:  tick.LastTradedQuantity,
							AverageTradedPrice:  tick.AverageTradedPrice,
							VolumeTraded:        tick.VolumeTraded,
							TotalBuy:            tick.TotalBuy,
							TotalSell:           tick.TotalSell,
							High:                tick.High,
							Low:                 tick.Low,
							Open:                tick.Open,
							Close:               tick.Close,
							OI:                  tick.OI,
							OIHigh:              tick.OIHigh,
							OILow:               tick.OILow,
							LastTradedTimestamp: tick.LastTradedTimestamp,
							ExchangeTimestamp:   tick.ExchangeTimestamp,
							LotSize:             stockOpts.LotSize,
							Expiry:              expiry,
							Depth:               convertDepth(tick.Depth),
							LastUpdateTime:      time.Now(),
						})

						// Update future price and trigger options subscription if needed
						if math.Abs(stockOpts.FuturePrice-tick.LastPrice) > 0.5 {
							stockOpts.FuturePrice = tick.LastPrice

							strikes := getStockStrikes(*kite.BrokerInstrumentTokens, stockName, expiry)
							atmStrike := getNearestStrike(tick.LastPrice, strikes)

							// Unsubscribe from previous ATM options
							if prevATM, ok := stockOpts.CurrentATM[expiry]; ok && len(prevATM) > 0 {
								marketDataServer.RemoveOptionData(stockName, expiry, prevATM)
								err := t.Unsubscribe(ctx, prevATM)
								if err != nil {
									log.Printf("Error unsubscribing from previous ATM options: %v", err)
								} else {
									updateSubscriptionCount(-len(prevATM))
								}
							}

							// Subscribe to new ATM options
							var toSubscribe []string
							for _, inst := range *kite.BrokerInstrumentTokens {
								if inst.Name == stockName && inst.Strike == atmStrike &&
									inst.Expiry == expiry && (inst.InstrumentType == "CE" || inst.InstrumentType == "PE") {
									toSubscribe = append(toSubscribe, inst.TradingSymbol)
								}
							}

							if len(toSubscribe) > 0 {
								err := t.SubscribeFull(ctx, toSubscribe)
								if err != nil {
									log.Printf("Error subscribing to new ATM options: %v", err)
								} else {
									stockOpts.CurrentATM[expiry] = toSubscribe
									updateSubscriptionCount(len(toSubscribe))
								}
							}
						}
					}

					stockOpts.mu.Unlock()
				}(tick)
			} else {
				// Handle options data...
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
					go marketDataServer.BroadcastMarketData(ws.MarketData{
						StockName:           inst.Name,
						TradingSymbol:       tick.TradingSymbol,
						InstrumentType:      inst.InstrumentType,
						StrikePrice:         inst.Strike,
						LastPrice:           tick.LastPrice,
						LastTradedQuantity:  tick.LastTradedQuantity,
						AverageTradedPrice:  tick.AverageTradedPrice,
						VolumeTraded:        tick.VolumeTraded,
						TotalBuy:            tick.TotalBuy,
						TotalSell:           tick.TotalSell,
						High:                tick.High,
						Low:                 tick.Low,
						Open:                tick.Open,
						Close:               tick.Close,
						OI:                  tick.OI,
						OIHigh:              tick.OIHigh,
						OILow:               tick.OILow,
						LastTradedTimestamp: tick.LastTradedTimestamp,
						ExchangeTimestamp:   tick.ExchangeTimestamp,
						LotSize:             inst.LotSize,
						Expiry:              inst.Expiry,
						Depth:               convertDepth(tick.Depth),
						LastUpdateTime:      time.Now(),
					})
				}
			}
		}
	}(ticker)

	go ticker.Serve(ctx)
}
