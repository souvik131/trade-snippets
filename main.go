package main

import (
	"context"
	"encoding/binary"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/joho/godotenv"
	"github.com/souvik131/trade-snippets/kite"
	"github.com/souvik131/trade-snippets/storage"
	"google.golang.org/protobuf/proto"
)

func main() {
	// read()

	go Host()
	write()
}

func write() {
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

func read() {

	b, err := os.ReadFile("./binary/data_proto_" + time.Now().Format("20060102") + ".zstd")
	if err != nil {
		log.Panicf("%s", err)
		return
	}
	log.Println(len(b))
	for len(b) > 8 {
		sizeOfPacket := binary.BigEndian.Uint64(b[0:8])
		log.Println(sizeOfPacket)
		packet, err := decompress(b[8 : sizeOfPacket+8])
		if err != nil {
			log.Panicf("%s", err)
			return
		}
		b = b[sizeOfPacket+8:]

		data := &storage.Data{}
		err = proto.Unmarshal(packet, data)
		if err != nil {
			log.Panicf("%s", err)
			return
		}
		spew.Dump(data)
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
