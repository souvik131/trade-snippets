package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/davecgh/go-spew/spew"
	"github.com/joho/godotenv"
	"github.com/klauspost/compress/zstd"
	"github.com/souvik131/trade-snippets/kite"
	"github.com/souvik131/trade-snippets/storage"
	"google.golang.org/protobuf/proto"
)

var instrumentsPerSocket = 3000.0
var instrumentsPerRequest = 3000.0
var dateFormat = "2006-01-02"
var dateFormatConcise = "20060102"

func Write() {
	ctx := context.Background()

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	var k = &kite.Kite{}
	err = k.Login(&ctx)
	if err != nil {
		log.Panicf("%s", err)
		return
	}

	Serve(&ctx, k)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}

func readMap() (map[uint32]*storage.TickerMap, error) {

	tokenNameMap := map[uint32]*storage.TickerMap{}
	b, err := os.ReadFile("./binary/map_" + time.Now().Format(dateFormatConcise) + ".proto.zstd")
	if err != nil {
		return nil, err
	}
	for len(b) > 8 {
		sizeOfPacket := binary.BigEndian.Uint64(b[0:8])
		packet, err := decompress(b[8 : sizeOfPacket+8])
		if err != nil {
			return nil, err
		}
		b = b[sizeOfPacket+8:]

		data := &storage.Map{}
		err = proto.Unmarshal(packet, data)
		if err != nil {
			return nil, err
		}
		for _, ts := range data.TickerMap {
			tokenNameMap[ts.Token] = ts
		}
	}
	return tokenNameMap, nil
}

func Read(date time.Time) {
	tokenMap, err := readMap()
	if err != nil {
		log.Panicf("%s", err)
	}

	b, err := os.ReadFile("./binary/index_" + date.Format(dateFormatConcise) + ".bin.zstd")
	if err != nil {
		log.Panicf("%s", err)
	}

	t := &kite.TickerClient{
		TickerChan: make(chan kite.KiteTicker),
	}

	go func() {
		for len(b) > 8 {
			sizeOfPacket := binary.BigEndian.Uint64(b[0:8])
			packet, err := decompress(b[8 : sizeOfPacket+8])
			if err != nil {
				log.Panicf("%s", err)
			}
			t.ParseBinary(packet)
			b = b[sizeOfPacket+8:]
		}
	}()
	if err != nil {
		log.Panicf("%s", err)
	}
	for ticker := range t.TickerChan {
		ticker.TradingSymbol = tokenMap[ticker.Token].TradingSymbol
		spew.Dump(ticker, tokenMap[ticker.Token])
	}

}

func Host() {

	dir := "./binary"
	fileServer := http.FileServer(http.Dir(dir))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			indexPath := filepath.Join(dir, "index.html")
			if _, err := os.Stat(indexPath); os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.ServeFile(w, r, indexPath)
		} else {
			fileServer.ServeHTTP(w, r)
		}
	})
	absDir, err := filepath.Abs(dir)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Serving files from %s on http://localhost:8080", absDir)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func compress(input []byte) ([]byte, error) {
	var b bytes.Buffer
	encoder, err := zstd.NewWriter(&b)
	if err != nil {
		return nil, err
	}

	_, err = encoder.Write(input)
	if err != nil {
		encoder.Close()
		return nil, err
	}

	err = encoder.Close()
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

func decompress(input []byte) ([]byte, error) {
	b := bytes.NewReader(input)
	decoder, err := zstd.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer decoder.Close()

	var out bytes.Buffer
	_, err = out.ReadFrom(decoder)
	if err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func appendToFile(filename string, data []byte) error {

	compressedData, err := compress(data)
	if err != nil {
		log.Panicf("%s", err)
	}

	bytesToSave := make([]byte, 8)
	binary.BigEndian.PutUint64(bytesToSave, uint64(len(compressedData)))
	bytesToSave = append(bytesToSave, compressedData...)

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(bytesToSave)
	if err != nil {
		return err
	}
	return nil
}

func saveFile(filePath string, data []byte) error {
	compressedData, err := compress(data)
	if err != nil {
		log.Panicf("%s", err)
	}

	bytesToSave := make([]byte, 8)
	binary.BigEndian.PutUint64(bytesToSave, uint64(len(compressedData)))
	log.Println(binary.BigEndian.Uint16(bytesToSave), uint64(len(compressedData)))
	bytesToSave = append(bytesToSave, compressedData...)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(bytesToSave)
	return err
}

func Serve(ctx *context.Context, k *kite.Kite) {
	iMap := &storage.Map{
		TickerMap: []*storage.TickerMap{},
	}

	for name, data := range *kite.BrokerInstrumentTokens {
		iMap.TickerMap = append(iMap.TickerMap, &storage.TickerMap{
			Token:          data.Token,
			TradingSymbol:  name,
			Exchange:       data.Exchange,
			Name:           data.Name,
			Expiry:         data.Expiry,
			Strike:         float32(data.Strike),
			TickSize:       float32(data.TickSize),
			LotSize:        uint32(data.LotSize),
			InstrumentType: data.InstrumentType,
			Segment:        data.Segment,
		})
	}

	bytes, err := proto.Marshal(iMap)
	if err != nil {
		log.Panicf("%s", err)
	}

	saveFile("./binary/map_"+time.Now().Format(dateFormatConcise)+".proto.zstd", bytes)

	log.Println("Instrument Map successfully written to file")

	k.TickerClients = []*kite.TickerClient{}

	expiryByName := map[string]string{}
	for _, data := range *kite.BrokerInstrumentTokens {

		isIndex := data.Name == "NIFTY" ||
			data.Name == "BANKNIFTY" ||
			data.Name == "FINNIFTY" ||
			data.Name == "MIDCPNIFTY" ||
			data.Name == "BANKEX" ||
			data.Name == "SENSEX50" ||
			data.Name == "NIFTYNXT50"

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
		ticker, err := k.GetWebSocketClient(ctx, true)
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
					<-time.After(5 * time.Second)
				}
			}
		}(k.TickerClients[i])
		go func(t *kite.TickerClient) {
			for message := range t.BinaryTickerChan {

				appendToFile("./binary/index_"+time.Now().Format(dateFormatConcise)+".bin.zstd", message)

				data := &storage.Data{
					Tickers: []*storage.Ticker{},
				}
				numOfPackets := binary.BigEndian.Uint16(message[0:2])
				if numOfPackets > 0 {

					message = message[2:]
					for {
						if numOfPackets == 0 {
							break
						}

						numOfPackets--
						packetSize := binary.BigEndian.Uint16(message[0:2])
						packet := kite.Packet(message[2 : packetSize+2])
						values := packet.ParseBinary(int(math.Min(64, float64(len(packet)))))
						ticker := &storage.Ticker{
							Depth: &storage.Depth{
								Buy:  []*storage.Order{},
								Sell: []*storage.Order{},
							},
						}
						if len(values) >= 2 {
							ticker.Token = values[0]
							ticker.LastPrice = values[1]
						}
						switch len(values) {
						case 2:
						case 7:
							ticker.High = values[2]
							ticker.Low = values[3]
							ticker.Open = values[4]
							ticker.Close = values[5]
							ticker.ExchangeTimestamp = values[6]
						case 8:
							ticker.High = values[2]
							ticker.Low = values[3]
							ticker.Open = values[4]
							ticker.Close = values[5]
							ticker.PriceChange = values[6]
							ticker.ExchangeTimestamp = values[7]
						case 11:
							ticker.LastTradedQuantity = values[2]
							ticker.AverageTradedPrice = values[3]
							ticker.VolumeTraded = values[4]
							ticker.TotalBuy = values[5]
							ticker.TotalSell = values[6]
							ticker.High = values[7]
							ticker.Low = values[8]
							ticker.Open = values[9]
							ticker.Close = values[10]
						case 16:
							ticker.LastTradedQuantity = values[2]
							ticker.AverageTradedPrice = values[3]
							ticker.VolumeTraded = values[4]
							ticker.TotalBuy = values[5]
							ticker.TotalSell = values[6]
							ticker.High = values[7]
							ticker.Low = values[8]
							ticker.Open = values[9]
							ticker.Close = values[10]
							ticker.LastTradedTimestamp = values[11]
							ticker.OI = values[12]
							ticker.OIHigh = values[13]
							ticker.OILow = values[14]
							ticker.ExchangeTimestamp = values[15]
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
								price := values[1]
								orders := values[2]
								if len(ticker.Depth.Buy) < lobDepth {
									ticker.Depth.Buy = append(ticker.Depth.Buy, &storage.Order{Price: price, Quantity: qty, Orders: orders})
								} else {

									ticker.Depth.Sell = append(ticker.Depth.Sell, &storage.Order{Price: price, Quantity: qty, Orders: orders})
								}
								values = values[3:]

							}
						}
						if len(message) > int(packetSize+2) {
							message = message[packetSize+2:]
						}
						data.Tickers = append(data.Tickers, ticker)

					}
				}
				// bytes, err := proto.Marshal(data)
				// if err != nil {
				// 	log.Panicf("%s", err)
				// }
				// appendToFile("./binary/index_proto_"+time.Now().Format(dateFormatConcise)+".zstd", bytes)

			}
		}(k.TickerClients[i])
		go k.TickerClients[i].Serve(ctx)
		<-time.After(5 * time.Second * time.Duration(instrumentsPerSocket/instrumentsPerRequest))
		i++
	}
	log.Println("All subscribed")

}

func Upload(t time.Time) error {

	key := os.Getenv("TA_DO_KEY")
	secret := os.Getenv("TA_DO_SECRET")
	bucket := os.Getenv("TA_DO_BUCKET")
	endpoint := os.Getenv("TA_DO_ENDPOINT")
	region := os.Getenv("TA_DO_REGION")
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(key, secret, ""),
		Endpoint:         aws.String(endpoint),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(false), // // Configures to use subdomain/virtual calling format. Depending on your version, alternatively use o.UsePathStyle = false
	}

	sess, err := session.NewSession(s3Config)
	if err != nil {
		return err
	}

	uploader := s3manager.NewUploader(sess)

	mapFile := "./binary/map_" + t.Format(dateFormatConcise) + ".proto.zstd"
	f, err := os.Open(mapFile)
	if err != nil {
		return fmt.Errorf("failed to open file %q, %v", mapFile, err)
	}
	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(mapFile),
		Body:   f,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}
	fmt.Printf("file uploaded to, %s\n", aws.StringValue(&result.Location))
	os.Remove(mapFile)

	dataFile := "./binary/index_" + t.Format(dateFormatConcise) + ".bin.zstd"
	f, err = os.Open(dataFile)
	if err != nil {
		return fmt.Errorf("failed to open file %q, %v", mapFile, err)
	}
	result, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(dataFile),
		Body:   f,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file, %v", err)
	}
	fmt.Printf("file uploaded to, %s\n", aws.StringValue(&result.Location))
	os.Remove(dataFile)

	return nil
}
