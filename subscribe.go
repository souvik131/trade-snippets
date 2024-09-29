package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/souvik131/trade-snippets/storage"
	"golang.org/x/exp/rand"
)

var wg = &sync.WaitGroup{}
var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
func Subscribe() {

	nc, err := nats.Connect(os.Getenv("NATS_READ_URI"))
	if err != nil {
		log.Panic(err)
	}
	js, err := jetstream.New(nc)
	if err != nil {
		log.Panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	c, err := js.CreateOrUpdateConsumer(ctx, "FEED", jetstream.ConsumerConfig{
		Name:           RandStringRunes(32) + "_FEED",
		AckPolicy:      jetstream.AckNonePolicy,
		DeliverPolicy:  jetstream.DeliverLastPolicy,
		FilterSubjects: []string{},
	})
	if err != nil {
		log.Panic(err)
	}
	c.Consume(func(msg jetstream.Msg) {
		b := msg.Data()
		s := msg.Subject()

		t := &storage.Ticker{}
		err := json.Unmarshal(b, t)
		if err != nil {
			log.Print(err)
		}

		log.Printf("%s : %+v \n\n", s, t)
	})
	wg.Add(1)
	wg.Wait()

}
