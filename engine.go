package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/joho/godotenv"
	"github.com/klauspost/compress/zstd"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/souvik131/trade-snippets/kite"
	"github.com/souvik131/trade-snippets/notifications"
	"github.com/souvik131/trade-snippets/storage"
	"google.golang.org/protobuf/proto"
)

var rotationInterval = 3.0
var instrumentsPerRequest = 3000.0
var dateFormatConcise = "20060102"
var t = &notifications.Telegram{}

func Write() {

	t.Send("Started Writing Feed Data")
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

func readMap(dateStr string) (map[uint32]*storage.TickerMap, error) {

	tokenNameMap := map[uint32]*storage.TickerMap{}
	b, err := os.ReadFile("./binary/map_" + dateStr + ".proto.zstd")
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

func Read(dateStr string) {
	tokenMap, err := readMap(dateStr)
	if err != nil {
		log.Panicf("%s", err)
	}

	b, err := os.ReadFile("./binary/market_data_" + dateStr + ".bin.zstd")
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
	counter := 0
	start := time.Now()
	timeElapsed := time.Microsecond
	indices := map[string]bool{}
	for {
		select {
		case ticker := <-t.TickerChan:
			counter++
			if t, ok := tokenMap[ticker.Token]; ok {
				ticker.TradingSymbol = t.TradingSymbol
				// if counter%1000000 == 0 {
				fmt.Printf("%v: %+v\n", counter, ticker)
				// }
				indices[t.Name] = true
			}
			timeElapsed = time.Since(start)
		case <-time.After(time.Second):

			keys := make([]string, 0, len(indices))

			for key := range indices {
				keys = append(keys, key)
			}
			fmt.Println("Read", counter, "F&O records of ("+strings.Join(keys, ", ")+")", "in", timeElapsed)
			log.Panic("exiting")
		}
	}

}

func Host() {

	dir := "./web"
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
	bestLevel := zstd.WithEncoderLevel(zstd.SpeedBestCompression)
	encoder, err := zstd.NewWriter(&b, bestLevel)
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
	log.Println(os.Getenv("TA_NATS_WRITE_URI"))
	nc, err := nats.Connect(os.Getenv("TA_NATS_WRITE_URI"))
	if err != nil {
		log.Panic(err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		log.Panic(err)
	}

	_, err = js.CreateOrUpdateStream(*ctx, jetstream.StreamConfig{
		Name:              "FEED",
		Subjects:          []string{"FEED_EQ.*.*", "FEED_FUT.*.*.*", "FEED_OPT.*.*.*.*.*"},
		MaxMsgsPerSubject: 1,
		Storage:           jetstream.MemoryStorage,
	})
	if err != nil {
		log.Fatal(err)
	}

	tokenTradingsymbolMap := map[uint32]*storage.TickerMap{}

	iMap := &storage.Map{
		TickerMap: []*storage.TickerMap{},
	}

	for name, data := range *kite.BrokerInstrumentTokens {
		tokenTradingsymbolMap[data.Token] = &storage.TickerMap{
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
		}
		iMap.TickerMap = append(iMap.TickerMap, tokenTradingsymbolMap[data.Token])
	}

	bytes, err := proto.Marshal(iMap)
	if err != nil {
		log.Panicf("%s", err)
	}

	saveFile("./binary/map_"+time.Now().Format(dateFormatConcise)+".proto.zstd", bytes)

	log.Println("Instrument Map successfully written to file")

	k.TickerClients = []*kite.TickerClient{}
	// Initialize token tracking
	var processedTokens int64 = 0
	processedSymbols := make(map[string]bool)

	allTokens := []string{}

	for _, data := range *kite.BrokerInstrumentTokens {
		if data.Exchange == "NFO" || data.Exchange == "NFO-OPT" || data.Name == "SENSEX" {
			allTokens = append(allTokens, data.TradingSymbol)
		}
	}
	totalTokens := len(allTokens)
	log.Printf("Total unique tokens to process: %d", totalTokens)

	var symbolsMutex sync.Mutex
	ticker, err := k.GetWebSocketClient(ctx /*, false*/)
	if err != nil {
		log.Panicf("%v", err)
	}
	k.TickerClients = append(k.TickerClients, ticker)
	k.TickSymbolMap = map[string]kite.KiteTicker{}

	// Handle websocket connection
	go func(t *kite.TickerClient) {
		for range t.ConnectChan {
			log.Println("Websocket is connected")

			// Start rotation
			go func() {
				for {
					select {
					case <-(*ctx).Done():
						return
					default:
						// Rotate through all tokens in chunks
						for start := 0; start < len(allTokens); start += int(instrumentsPerRequest) {
							select {
							case <-(*ctx).Done():
								return
							default:
								end := start + int(instrumentsPerRequest)
								if end > len(allTokens) {
									end = len(allTokens)
								}

								// Unsubscribe from previous chunk
								prevStart := start - int(instrumentsPerRequest)
								if prevStart >= 0 {
									prevEnd := start
									t.Unsubscribe(ctx, allTokens[prevStart:prevEnd])
									log.Printf("Unsubscribed from tokens %d-%d", prevStart, prevEnd)
								}

								// Subscribe to new chunk
								t.SubscribeFull(ctx, allTokens[start:end])
								log.Printf("Subscribed to tokens %d-%d", start, end)

								// Sleep for rotation interval
								<-time.After(time.Duration(rotationInterval) * time.Second)
							}
						}
					}
				}
			}()
		}
	}(ticker)

	// Handle ticker data
	go func(t *kite.TickerClient) {
		for ticker := range t.TickerChan {
			symbolsMutex.Lock()
			if !processedSymbols[ticker.TradingSymbol] {
				processedSymbols[ticker.TradingSymbol] = true
				processed := atomic.AddInt64(&processedTokens, 1)
				if processed%1000 == 0 || float64(processed)/float64(totalTokens) == 1 {
					log.Printf("New token processed: %s (%d/%d - %.2f%%)",
						ticker.TradingSymbol,
						processed,
						totalTokens,
						float64(processed)/float64(totalTokens)*100)
				}
				if float64(processed)/float64(totalTokens) == 1 {
					processedTokens = 0
					processedSymbols = make(map[string]bool)
				}
			}
			symbolsMutex.Unlock()
		}
	}(ticker)

	// Handle binary data
	go func(t *kite.TickerClient) {
		for message := range t.BinaryTickerChan {

			appendToFile("./binary/market_data_"+time.Now().Format(dateFormatConcise)+".bin.zstd", message)

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
					bytes, err := json.Marshal(ticker)
					if err != nil {
						log.Panicf("%s", err)
					}
					tokenData := tokenTradingsymbolMap[ticker.Token]
					if tokenData.InstrumentType == "FUT" {
						if nc.Status() != nats.CONNECTED {
							continue
						} else {
							js.PublishAsync(fmt.Sprintf("FEED_FUT.%v.%v.%v", tokenData.Exchange, tokenData.Name, tokenData.Expiry), bytes)

						}

					} else if tokenData.InstrumentType == "CE" || tokenData.InstrumentType == "PE" {
						if nc.Status() != nats.CONNECTED {
							continue
						} else {
							js.PublishAsync(fmt.Sprintf("FEED_OPT.%v.%v.%v.%v.%v", tokenData.Exchange, tokenData.Name, tokenData.Expiry, tokenData.Strike, tokenData.InstrumentType), bytes)

						}
					} else if tokenData.InstrumentType == "EQ" {
						if nc.Status() != nats.CONNECTED {
							continue
						} else {
							js.PublishAsync(fmt.Sprintf("FEED_EQ.%v.%v", tokenData.Exchange, tokenData.Name), bytes)

						}
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
	}(ticker)

	// Start serving
	go ticker.Serve(ctx)
	log.Println("Websocket service started")

	// Wait for context cancellation
	<-(*ctx).Done()
	log.Println("Shutting down websocket service")

}

func Upload() error {
	t.Send("Uploading Feed Data File")
	key := os.Getenv("TA_DO_KEY")
	secret := os.Getenv("TA_DO_SECRET")
	bucket := os.Getenv("TA_DO_BUCKET")
	endpoint := os.Getenv("TA_DO_ENDPOINT")
	region := os.Getenv("TA_DO_REGION")
	log.Println(region)
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(key, secret, ""),
		Endpoint:         aws.String(endpoint),
		Region:           aws.String(region),
		S3ForcePathStyle: aws.Bool(false),
	}

	sess, err := session.NewSession(s3Config)
	if err != nil {
		return err
	}

	uploader := s3manager.NewUploader(sess)

	fileDir := "./binary"

	file, err := os.Open(fileDir)
	if err != nil {
		return err
	}
	defer file.Close()

	files, err := os.ReadDir(fileDir)
	if err != nil {
		return err
	}

	for _, f := range files {

		mapFile := fileDir + "/" + f.Name()

		if strings.Contains(f.Name(), "_"+time.Now().Format(dateFormatConcise)) {
			fileInfo, err := os.Stat(mapFile)
			if err != nil {
				return fmt.Errorf("failed to get file info %q, %v", mapFile, err)
			}
			fileSizeInMB := float64(fileInfo.Size()) / (1024 * 1024)
			t.Send(fmt.Sprintf("%s file of %.2f MB\n", mapFile, fileSizeInMB))
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
		}
	}

	return nil
}
