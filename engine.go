package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math"
	"os"
	"time"

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

	k.TickerClients = []*kite.TickerClient{}

	expiryByName := map[string]string{}
	for _, data := range *kite.BrokerInstrumentTokens {

		isIndex := data.Name == "NIFTY" ||
			data.Name == "BANKNIFTY" ||
			data.Name == "FINNIFTY" ||
			data.Name == "MIDCPNIFTY" ||
			data.Name == "BANKEX" ||
			data.Name == "SENSEX"

		//isNotIndex := data.Name != "NIFTY" &&
		//	data.Name == "BANKNIFTY" &&
		//	data.Name == "FINNIFTY" &&
		//	data.Name == "MIDCPNIFTY" &&
		//	data.Name == "BANKEX" &&
		//	data.Name == "SENSEX" &&
		//	data.Name == "SENSEX50" &&
		//	data.Name == "NIFTYNXT50"

		if data.Expiry != "" && (data.Exchange == "NFO" || data.Exchange == "BFO") && isIndex {
			name := data.Exchange + ":" + data.Name
			if date, ok := expiryByName[name]; ok && date != "" {
				dateSaved, err := time.Parse(dateFormat, date)
				if err != nil {
					log.Panicf("%v", err)
				}

				dateExpiry, err := time.Parse(dateFormat, data.Expiry)
				if err != nil {
					log.Panicf("%v", err)
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
	log.Println("Subscribed tokens: ", len(allTokens))

	i := 0
	for len(allTokens) > 0 {
		minLen := int(math.Min(instrumentsPerSocket, float64(len(allTokens))))
		tokens := allTokens[0:minLen]
		allTokens = allTokens[minLen:]
		ticker, err := k.GetWebSocketClient(ctx)
		if err != nil {
			log.Panicf("%v", err)
		}
		k.TickerClients = append(k.TickerClients, ticker)
		k.TickSymbolMap = map[string]kite.KiteTicker{}
		go func(t *kite.TickerClient) {
			for range t.ConnectChan {
				log.Printf("%v : Websocket is connected", i)
				for len(tokens) > 0 {
					minLen := int(math.Min(instrumentsPerRequest, float64(len(tokens))))
					t.SubscribeFull(ctx, tokens[0:minLen])
					tokens = tokens[minLen:]
					log.Println("subscribed", minLen, i)
					<-time.After(time.Millisecond * 500)
				}
			}
		}(k.TickerClients[i])
		go func(t *kite.TickerClient) {
			for range t.TickerChan {

			}
		}(k.TickerClients[i])
		go func(t *kite.TickerClient) {
			for b := range t.BinaryTickerChan {
				err := appendBinaryDataToFile(fmt.Sprintf("./binary/data_%v.bin", time.Now().Format(dateFormat)), b, []byte{})
				if err != nil {
					log.Panicf("%v", err)
					return
				}
			}
		}(k.TickerClients[i])
		go k.TickerClients[i].Serve(ctx)
		<-time.After(time.Millisecond * time.Duration(500*instrumentsPerSocket/instrumentsPerRequest))
		i++
	}
	log.Println("All subscribed")

}
