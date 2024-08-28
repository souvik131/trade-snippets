package kite

import (
	"net/http"
	"time"

	"github.com/gocarina/gocsv"
)

var StrikeDistance = map[string]float64{}
var LotSize = map[string]float64{}
var TickSize = map[string]float64{}
var ExpiryDatesByScript = map[string][]time.Time{}
var (
	BrokerInstrumentTokens = &InstrumentSymbolMap{}
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

const (
	YYYYMMDD = "2006-01-02"
)

var SymbolTokenMap = map[string]uint32{}
var TokenSymbolMap = map[uint32]string{}

func (kite *Kite) FetchInstruments() (Instruments, error) {

	var insts Instruments

	resp, err := http.Get("https://api.kite.trade/instruments")

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err = gocsv.Unmarshal(resp.Body, &insts); err != nil {
		return nil, err
	}
	for _, i := range insts {

		TokenSymbolMap[i.Token] = i.TradingSymbol
		SymbolTokenMap[i.TradingSymbol] = i.Token
		(*BrokerInstrumentTokens)[i.TradingSymbol] = i
	}
	return insts, nil
}
