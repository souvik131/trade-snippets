package kite

import (
	"net/http"
	"sync"
	"time"

	"github.com/gocarina/gocsv"
)

var StrikeDistance = map[string]float64{}
var LotSize = map[string]float64{}
var TickSize = map[string]float64{}
var ExpiryDatesByScript = map[string][]time.Time{}
var (
	BrokerInstrumentTokens = &InstrumentSymbolMap{}
	instrumentsLock        sync.RWMutex
	allMonthlyExpiries     = make(map[string]bool)
	DateMap                = map[string]string{
		"JAN": "01",
		"FEB": "02",
		"MAR": "03",
		"APR": "04",
		"MAY": "05",
		"JUN": "06",
		"JUL": "07",
		"AUG": "08",
		"SEP": "09",
		"OCT": "10",
		"NOV": "11",
		"DEC": "12",
	}
	DateReverseMap = map[string]string{
		"01": "JAN",
		"02": "FEB",
		"03": "MAR",
		"04": "APR",
		"05": "MAY",
		"06": "JUN",
		"07": "JUL",
		"08": "AUG",
		"09": "SEP",
		"10": "OCT",
		"11": "NOV",
		"12": "DEC",
	}
	DateReverseCharMap = map[string]string{
		"01": "1",
		"02": "2",
		"03": "3",
		"04": "4",
		"05": "5",
		"06": "6",
		"07": "7",
		"08": "8",
		"09": "9",
		"10": "OCT",
		"11": "NOV",
		"12": "DEC",
	}
)

var IndexMap = map[string]string{
	"BANKNIFTY":  "NIFTY BANK",
	"NIFTY":      "NIFTY 50",
	"FINNIFTY":   "NIFTY FIN SERVICE",
	"MIDCPNIFTY": "NIFTY MID SELECT",
	"SENSEX":     "SENSEX",
	"BANKEX":     "BANKEX",
}

const (
	YYYYMMDD = "2006-01-02"
)

var SymbolTokenMap = map[string]uint32{}
var TokenSymbolMap = map[uint32]string{}

func init() {
	*BrokerInstrumentTokens = make(InstrumentSymbolMap)
}

func (kite *Kite) FetchInstruments() (Instruments, error) {
	// log.Println("Fetching instruments...")

	instrumentsLock.Lock()
	defer instrumentsLock.Unlock()

	// Clear existing maps
	*BrokerInstrumentTokens = make(InstrumentSymbolMap)
	SymbolTokenMap = make(map[string]uint32)
	TokenSymbolMap = make(map[uint32]string)

	var insts Instruments
	resp, err := http.Get("https://api.kite.trade/instruments")
	if err != nil {
		// log.Printf("Error fetching instruments: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if err = gocsv.Unmarshal(resp.Body, &insts); err != nil {
		// log.Printf("Error unmarshaling instruments: %v", err)
		return nil, err
	}

	// log.Printf("Fetched %d instruments", len(insts))
	stockCount := 0
	futureCount := 0
	optionCount := 0

	for _, i := range insts {
		TokenSymbolMap[i.Token] = i.TradingSymbol
		SymbolTokenMap[i.TradingSymbol] = i.Token
		(*BrokerInstrumentTokens)[i.TradingSymbol] = i

		if i.Exchange == "NFO" {
			switch i.InstrumentType {
			case "FUT":
				futureCount++
			case "CE", "PE":
				optionCount++
			}
			stockCount++
		}
	}

	// log.Printf("Instrument breakdown - Total: %d, Stocks: %d, Futures: %d, Options: %d", len(insts), stockCount, futureCount, optionCount)

	// Verify initialization
	if len(*BrokerInstrumentTokens) == 0 {
		// log.Printf("Warning: BrokerInstrumentTokens is empty after initialization")
	}
	if len(SymbolTokenMap) == 0 {
		// log.Printf("Warning: SymbolTokenMap is empty after initialization")
	}
	if len(TokenSymbolMap) == 0 {
		// log.Printf("Warning: TokenSymbolMap is empty after initialization")
	}

	return insts, nil
}

func (kite *Kite) GetInstrument(tradingSymbol string) *Instrument {
	instrumentsLock.RLock()
	defer instrumentsLock.RUnlock()

	if inst, ok := (*BrokerInstrumentTokens)[tradingSymbol]; ok {
		return inst
	}
	return nil
}

func (kite *Kite) GetInstrumentByToken(token uint32) *Instrument {
	instrumentsLock.RLock()
	defer instrumentsLock.RUnlock()

	if symbol, ok := TokenSymbolMap[token]; ok {
		if inst, ok := (*BrokerInstrumentTokens)[symbol]; ok {
			return inst
		}
	}
	return nil
}
