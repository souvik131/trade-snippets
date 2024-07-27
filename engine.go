package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/souvik131/trade-snippets/kite"
)

var instrumentsPerSocket = 3000.0
var instrumentsPerRequest = 100.0
var dateFormat = "2006-01-02"

func appendBinaryDataToFile(filePath string, binaryData []byte, delimiter []byte) error {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	var buffer bytes.Buffer
	buffer.Write(binaryData)
	buffer.Write(delimiter)
	_, err = file.Write(buffer.Bytes())
	if err != nil {
		return err
	}
	return nil
}

func Serve(ctx *context.Context, k *kite.Kite) {

	expiryByName := map[string]string{}
	for _, data := range *kite.BrokerInstrumentTokens {

		isIndex := data.Name == "NIFTY" ||
			data.Name == "BANKNIFTY" ||
			data.Name == "FINNIFTY" ||
			data.Name == "MIDCPNIFTY" ||
			data.Name == "BANKEX" ||
			data.Name == "SENSEX" ||
			data.Name == "SENSEX50" ||
			data.Name == "NIFTYNXT50"
		if data.Expiry != "" && (data.Exchange == "NFO" || data.Exchange == "BFO") && !isIndex {
			name := data.Exchange + ":" + data.Name
			if date, ok := expiryByName[name]; ok && date != "" {
				dateSaved, err := time.Parse(dateFormat, date)
				if err != nil {
					color.Red(fmt.Sprintf("%v", err))
					return
				}

				dateExpiry, err := time.Parse(dateFormat, data.Expiry)
				if err != nil {
					color.Red(fmt.Sprintf("%v", err))
					return
				}
				if dateSaved.Sub(dateExpiry) > 0 {
					expiryByName[name] = data.Expiry
				}
			} else {
				expiryByName[name] = data.Expiry
			}
		}

	}

	allTokens := []string{}

	for _, data := range *kite.BrokerInstrumentTokens {
		if exp, ok := expiryByName[data.Exchange+":"+data.Name]; ok && exp == data.Expiry && data.Expiry != "" {
			allTokens = append(allTokens, data.TradingSymbol)
		}
	}
	log.Println(len(allTokens))

	i := 0
	for len(allTokens) > 0 {
		minLen := int(math.Min(instrumentsPerSocket, float64(len(allTokens))))
		tokens := allTokens[0:minLen]
		allTokens = allTokens[minLen:]
		ticker, err := k.GetWebSocketClient(ctx)
		if err != nil {
			color.Red(fmt.Sprintf("%v", err))
			return
		}
		k.TickerClients = append(k.TickerClients, ticker)
		k.TickSymbolMap = map[string]kite.KiteTicker{}
		go func(t *kite.TickerClient) {
			for range t.ConnectChan {
				color.HiBlue(fmt.Sprintf("%v : Websocket is connected", i))
				color.HiCyan("Subscribing Ticks")
				for len(tokens) > 0 {
					minLen := int(math.Min(instrumentsPerRequest, float64(len(tokens))))
					t.SubscribeFull(ctx, tokens[0:minLen])
					tokens = tokens[minLen:]
					log.Println("subscribed", minLen, i)
					<-time.After(time.Second)
				}
			}
		}(k.TickerClients[i])
		go func(t *kite.TickerClient) {
			for range t.TickerChan {
				// color.HiWhite(fmt.Sprintf("\nTick %v %v: %+v\n", i, tick.TradingSymbol, tick))
			}
		}(k.TickerClients[i])
		go func(t *kite.TickerClient) {
			for b := range t.BinaryTickerChan {
				err := appendBinaryDataToFile(fmt.Sprintf("binary/data_%v.bin", time.Now().Format(dateFormat)), b, []byte{0x00})
				if err != nil {
					color.Red(fmt.Sprintf("%v", err))
					return
				}
			}
		}(k.TickerClients[i])
		go k.TickerClients[i].Serve(ctx)
		<-time.After(time.Second * time.Duration(instrumentsPerSocket/instrumentsPerRequest))
		i++
	}
	log.Println("All subscribed")

}
