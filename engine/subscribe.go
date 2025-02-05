package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"github.com/souvik131/trade-snippets/analytics"
	"github.com/souvik131/trade-snippets/greeks"
	"github.com/souvik131/trade-snippets/storage"
)

func Subscribe() {
	nc, err := nats.Connect(os.Getenv("TA_NATS_URI"))
	if err != nil {
		log.Panic(err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		log.Panic(err)
	}

	// Create cron scheduler
	c := cron.New()

	// Add cron job to run every minute
	_, err = c.AddFunc("15-59 9 * * 1-5", func() {
		fetchLatestFeeds(js, time.Now())
	})
	if err != nil {
		log.Warnf("Error adding cron job: %v", err)
		return
	}
	_, err = c.AddFunc("* 10-14 * * 1-5", func() {
		fetchLatestFeeds(js, time.Now())
	})
	if err != nil {
		log.Warnf("Error adding cron job: %v", err)
		return
	}
	_, err = c.AddFunc("0-30 15 * * 1-5", func() {
		fetchLatestFeeds(js, time.Now())
	})
	if err != nil {
		log.Warnf("Error adding cron job: %v", err)
		return
	}

	// Start cron scheduler
	c.Start()

}

func stringToTime(dateStr string) *time.Time {

	// Parse the date string to a time.Time object
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		fmt.Println("Error parsing date:", err)
		return nil
	}

	// Load Asia/Kolkata time zone
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		fmt.Println("Error loading time zone:", err)
		return nil
	}

	// Set time to 3:30 PM
	t = time.Date(t.Year(), t.Month(), t.Day(), 15, 30, 0, 0, loc)
	return &t
}

func getHoursToExpiry(expiry time.Time) float64 {
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.Panic(err)
	}

	expiry = time.Date(expiry.Year(), expiry.Month(), expiry.Day(), 15, 30, 0, 0, loc)
	now := time.Now()
	todayPreMarketHours := 0.0
	if now.Hour() < 9 || (now.Hour() == 9 && now.Minute() < 15) {
		preMarketTimestamp := time.Date(now.Year(), now.Month(), now.Day(), 9, 15, 0, 0, loc)
		todayPreMarketHours = preMarketTimestamp.Sub(now).Hours()
	}

	tradingDurationInHours := 6.25
	tradeEndHour := 15.0
	tradeEndMinute := 30.0

	postMarketOffset := 24 - tradeEndHour - tradeEndMinute/60

	duration := expiry.AddDate(0, 0, 1).Sub(now).Hours() - postMarketOffset

	days := math.Floor(duration / 24)

	remainingHours := duration - days*24
	// remainingHours := expiry.Sub(now).Hours()
	if remainingHours > tradingDurationInHours {
		remainingHours = tradingDurationInHours
	}

	todayTradeEndTime := time.Date(now.Year(), now.Month(), now.Day(), 15, 30, 0, 0, loc)
	remainingHoursToday := todayTradeEndTime.Sub(now).Hours()
	if remainingHoursToday > 0 {
		remainingHours = remainingHours - tradingDurationInHours + remainingHoursToday
	} else {
		remainingHours = remainingHours - tradingDurationInHours
	}
	indexDayOfWeek := float64(int(now.Weekday()))
	indexEndDayOfWeek := int(indexDayOfWeek+days) % 7
	weekPartials := (indexDayOfWeek + days) / 7

	days = days - math.Floor(weekPartials)*2

	if indexDayOfWeek == 6 {
		days = days + 1
	}

	if indexEndDayOfWeek == 6 {
		days = days - 1
	}

	days = days - getHolidaysCount(expiry)

	hours := days*tradingDurationInHours + remainingHours - todayPreMarketHours

	if hours < 0 {
		hours = 0
	}

	// m.Lock()
	// localMap[cacheKey] = hours
	// m.Unlock()

	return hours

}

func getHolidaysCount(expiry time.Time) float64 {
	loc, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		log.Panic(err)
	}
	holidaysCount := 0
	dateLayout := "02-Jan-2006"
	if strings.TrimSpace(os.Getenv("HOLIDAYS")) != "" {
		holidays := strings.Split(strings.TrimSpace(os.Getenv("HOLIDAYS")), ",")

		for i := range holidays {
			holiday, err := time.ParseInLocation(dateLayout, holidays[i], loc)
			if err != nil {
				log.Panic(err)
			}
			if holiday.After(time.Now()) && holiday.Before(expiry) && holiday.Weekday() != 0 && holiday.Weekday() != 6 {
				holidaysCount++
			}
		}
	}
	return float64(holidaysCount)
}

func fetchLatestFeeds(js jetstream.JetStream, now time.Time) {
	log.Info("Fetching latest feeds...")

	if os.Getenv("NATS_ENABLED") == "1" {
		// Create a channel to signal completion
		done := make(chan struct{})
		closeOnce := sync.Once{}

		ctx := context.Background()
		jsc, err := js.CreateOrUpdateConsumer(ctx, "FEED", jetstream.ConsumerConfig{
			Name:           "GET_LATEST_FEED",
			AckPolicy:      jetstream.AckNonePolicy,
			DeliverPolicy:  jetstream.DeliverAllPolicy,
			FilterSubjects: []string{},
		})
		if err != nil {
			log.Panic(err)
		}
		scriptsCount := 0

		// Declare msgs before the goroutine

		msgs, err := jsc.Consume(func(msg jetstream.Msg) {
			b := msg.Data()
			s := msg.Subject()

			t := &storage.Ticker{}
			err := json.Unmarshal(b, t)
			if err != nil {
				log.Warn(err)
			}

			// log.Infof("%s : %+v \n\n", s, t)
			storage.DataMapMutex.Lock()
			for len(t.Depth.Buy) < 5 {
				dummyBuy := storage.Order{}
				t.Depth.Buy = append(t.Depth.Buy, &dummyBuy)
			}

			for len(t.Depth.Sell) < 5 {
				dummySell := storage.Order{}
				t.Depth.Sell = append(t.Depth.Sell, &dummySell)
			}
			storage.DataMap[s] = t
			storage.DataMapMutex.Unlock()

			if err := json.Unmarshal(b, t); err != nil {
				log.Warnf("Error unmarshaling message: %v", err)
				return
			}
			// If no more pending messages, signal completion
			meta, err := msg.Metadata()
			if err != nil {
				log.Warnf("Error getting metadata: %v", err)
				return
			}
			scriptsCount++
			if meta.NumPending == 0 {
				closeOnce.Do(func() {
					close(done)
				})
			}
			if scriptsCount%10000 == 0 {
				log.Info(scriptsCount, " messages processed")
			}
		})
		if err != nil {
			log.Warnf("Error in consume: %v", err)
			closeOnce.Do(func() {
				close(done)
			})
			return
		}

		// Wait for either completion or timeout
		<-done
		defer msgs.Stop()
		log.Info(scriptsCount, ", all messages processed")
	}
	//save the required data to clickhouse

	storage.DataMapMutex.Lock()
	allData := make([]*log.Fields, 0, len(storage.DataMap))
	allGreeks := []*log.Fields{}
	allDerivedOptions := []*log.Fields{}
	type DerivedOptions struct {
		atm             float64
		atmCeIv         float64
		atmPeIv         float64
		atmCeTime       uint32
		atmPeTime       uint32
		straddlePrice   float64
		underlyingPrice float32
	}

	derivedOptionsByScriptExpiry := map[string]*DerivedOptions{}

	for s, ticker := range storage.DataMap {

		values := strings.Split(s, ".")
		typeOfInstrument := values[0]
		exchange := values[1]
		script, _ := url.QueryUnescape(values[2])

		expiry := ""
		strike := ""
		instrumentType := ""
		switch typeOfInstrument {
		case "FEED_EQ":
			instrumentType = "EQ"
		case "FEED_FUT":
			expiry = values[3]
			instrumentType = "FUT"
		case "FEED_OPT":
			expiry = values[3]
			strike = values[4]
			instrumentType = values[5]

			strikeFloat, _ := strconv.ParseFloat(strike, 64)
			underlyingPrice := 0.0

			otherOptionType := "CE"
			if instrumentType == "CE" {
				otherOptionType = "PE"
			}
			straddlePrice := 0.0

			price := (float64(ticker.Depth.Buy[0].Price*ticker.Depth.Buy[0].Quantity) + float64(ticker.Depth.Sell[0].Price*ticker.Depth.Sell[0].Quantity)) / (100 * (float64(ticker.Depth.Buy[0].Quantity) + float64(ticker.Depth.Sell[0].Quantity)))

			if undelyingTicker, ok := storage.DataMap[fmt.Sprintf("FEED_FUT.%v.%v.%v", exchange, script, expiry)]; ok {
				underlyingPrice = (float64(undelyingTicker.Depth.Buy[0].Price*undelyingTicker.Depth.Buy[0].Quantity) + float64(undelyingTicker.Depth.Sell[0].Price*undelyingTicker.Depth.Sell[0].Quantity)) / (100 * (float64(undelyingTicker.Depth.Buy[0].Quantity) + float64(undelyingTicker.Depth.Sell[0].Quantity)))

				if otherOptionTicker, ok := storage.DataMap[fmt.Sprintf("FEED_OPT.%v.%v.%v.%v.%v", exchange, script, expiry, strike, otherOptionType)]; ok {
					otherOptionPrice := (float64(otherOptionTicker.Depth.Buy[0].Price*otherOptionTicker.Depth.Buy[0].Quantity) + float64(otherOptionTicker.Depth.Sell[0].Price*otherOptionTicker.Depth.Sell[0].Quantity)) / (100 * (float64(otherOptionTicker.Depth.Buy[0].Quantity) + float64(otherOptionTicker.Depth.Sell[0].Quantity)))
					if otherOptionPrice > 0 {
						straddlePrice = price + otherOptionPrice
					}
				}
			} else if otherOptionTicker, ok := storage.DataMap[fmt.Sprintf("FEED_OPT.%v.%v.%v.%v.%v", exchange, script, expiry, strike, otherOptionType)]; ok {
				otherOptionPrice := (float64(otherOptionTicker.Depth.Buy[0].Price*otherOptionTicker.Depth.Buy[0].Quantity) + float64(otherOptionTicker.Depth.Sell[0].Price*otherOptionTicker.Depth.Sell[0].Quantity)) / (100 * (float64(otherOptionTicker.Depth.Buy[0].Quantity) + float64(otherOptionTicker.Depth.Sell[0].Quantity)))
				if otherOptionPrice > 0 {
					if instrumentType == "CE" {
						underlyingPrice = strikeFloat + price - otherOptionPrice
					} else {
						underlyingPrice = strikeFloat - price + otherOptionPrice
					}
					straddlePrice = price + otherOptionPrice
				}
			}

			if underlyingPrice > 0 {
				hrsLeft := getHoursToExpiry(*stringToTime(expiry))
				g := greeks.RunGreek(underlyingPrice, hrsLeft, strikeFloat, price, instrumentType == "CE")
				// log.Info(g.Iv, underlyingPrice, hrsLeft, strikeFloat, price, instrumentType == "CE")
				allGreeks = append(allGreeks, &log.Fields{
					"exchange":           exchange,
					"script":             script,
					"expiry":             expiry,
					"strike":             strike,
					"instrument_type":    instrumentType,
					"delta":              int32(math.Round(g.Delta * 100)),
					"gamma":              int32(math.Round(g.Gamma * 100)),
					"theta":              int32(math.Round(g.Theta * 100)),
					"vega":               int32(math.Round(g.Vega * 100)),
					"rho":                int32(math.Round(g.Rho * 100)),
					"implied_volatility": int32(math.Round(g.Iv * 10000)),
					"timestamp":          time.Unix(int64(ticker.ExchangeTimestamp), 0),
				})
				scriptExpiry := fmt.Sprintf("%v.%v", script, expiry)
				if _, ok := derivedOptionsByScriptExpiry[scriptExpiry]; !ok {
					derivedOptionsByScriptExpiry[scriptExpiry] = &DerivedOptions{
						atm:           strikeFloat,
						straddlePrice: straddlePrice,
					}
				}
				if math.Abs(underlyingPrice-strikeFloat) <= math.Abs(underlyingPrice-derivedOptionsByScriptExpiry[scriptExpiry].atm) {
					derivedOptionsByScriptExpiry[scriptExpiry].atm = strikeFloat
					derivedOptionsByScriptExpiry[scriptExpiry].straddlePrice = straddlePrice
					derivedOptionsByScriptExpiry[scriptExpiry].underlyingPrice = float32(underlyingPrice)
					if instrumentType == "CE" {
						derivedOptionsByScriptExpiry[scriptExpiry].atmCeIv = g.Iv
						derivedOptionsByScriptExpiry[scriptExpiry].atmCeTime = ticker.ExchangeTimestamp
					} else {
						derivedOptionsByScriptExpiry[scriptExpiry].atmPeIv = g.Iv
						derivedOptionsByScriptExpiry[scriptExpiry].atmPeTime = ticker.ExchangeTimestamp
					}

				}
			}
		default:
			log.Error("unknown instrument type: ", typeOfInstrument)
			continue
		}
		marketData := &log.Fields{
			"exchange":              exchange,
			"script":                script,
			"expiry":                expiry,
			"strike":                strike,
			"instrument_type":       instrumentType,
			"last_price":            ticker.LastPrice,
			"lot_size":              ticker.LotSize,
			"timestamp":             time.Unix(int64(ticker.ExchangeTimestamp), 0),
			"open":                  ticker.Open,
			"high":                  ticker.High,
			"low":                   ticker.Low,
			"close":                 ticker.Close,
			"volume":                ticker.VolumeTraded,
			"oi":                    ticker.OI,
			"last_traded_timestamp": time.Unix(int64(ticker.LastTradedTimestamp), 0),

			"buy_price_1":     ticker.Depth.Buy[0].Price,
			"buy_quantity_1":  ticker.Depth.Buy[0].Quantity,
			"buy_order_1":     ticker.Depth.Buy[0].Orders,
			"sell_price_1":    ticker.Depth.Sell[0].Price,
			"sell_quantity_1": ticker.Depth.Sell[0].Quantity,
			"sell_order_1":    ticker.Depth.Sell[0].Quantity,

			"buy_price_2":     ticker.Depth.Buy[1].Price,
			"buy_quantity_2":  ticker.Depth.Buy[1].Quantity,
			"buy_order_2":     ticker.Depth.Buy[1].Orders,
			"sell_price_2":    ticker.Depth.Sell[1].Price,
			"sell_quantity_2": ticker.Depth.Sell[1].Quantity,
			"sell_order_2":    ticker.Depth.Sell[1].Quantity,

			"buy_price_3":     ticker.Depth.Buy[2].Price,
			"buy_quantity_3":  ticker.Depth.Buy[2].Quantity,
			"buy_order_3":     ticker.Depth.Buy[2].Orders,
			"sell_price_3":    ticker.Depth.Sell[2].Price,
			"sell_quantity_3": ticker.Depth.Sell[2].Quantity,
			"sell_order_3":    ticker.Depth.Sell[2].Quantity,

			"buy_price_4":     ticker.Depth.Buy[3].Price,
			"buy_quantity_4":  ticker.Depth.Buy[3].Quantity,
			"buy_order_4":     ticker.Depth.Buy[3].Orders,
			"sell_price_4":    ticker.Depth.Sell[3].Price,
			"sell_quantity_4": ticker.Depth.Sell[3].Quantity,
			"sell_order_4":    ticker.Depth.Sell[3].Quantity,

			"buy_price_5":     ticker.Depth.Buy[4].Price,
			"buy_quantity_5":  ticker.Depth.Buy[4].Quantity,
			"buy_order_5":     ticker.Depth.Buy[4].Orders,
			"sell_price_5":    ticker.Depth.Sell[4].Price,
			"sell_quantity_5": ticker.Depth.Sell[4].Quantity,
			"sell_order_5":    ticker.Depth.Sell[4].Quantity,
		}

		allData = append(allData, marketData)

	}

	storage.DataMapMutex.Unlock()
	log.Info("Saving to DB")
	analytics.LIVE_DATA.BatchStore(allData)
	analytics.GREEKS.BatchStore(allGreeks)
	for se, data := range derivedOptionsByScriptExpiry {
		vals := strings.Split(se, ".")
		script := vals[0]
		expiry := vals[1]
		t := math.Max(float64(data.atmCeTime), float64(data.atmPeTime))
		if data.straddlePrice > 0 {
			allDerivedOptions = append(allDerivedOptions, &log.Fields{
				"script":           script,
				"expiry":           expiry,
				"atm_strike":       int32(math.Round(data.atm * 100)),
				"atm_iv":           int32((data.atmCeIv + data.atmPeIv) * 10000 / 2),
				"timestamp":        time.Unix(int64(t), 0),
				"straddle_price":   data.straddlePrice * 100,
				"underlying_price": data.underlyingPrice * 100,
			})
		}
	}
	analytics.DERIVED_OPTIONS.BatchStore(allDerivedOptions)

}
