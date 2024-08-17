package notifications

import (
	"context"
	"log"
	"os"
	"strconv"

	"github.com/bot-api/telegram"
)

type Telegram struct {
}

func (t *Telegram) Send(message string) {

	token := os.Getenv("TA_TELEGRAM_TOKEN")
	idString := os.Getenv("TA_TELEGRAM_ID")
	if token == "" {
		log.Panic("telegram token required")
	}

	api := telegram.New(token)
	api.Debug(false)

	id, err := strconv.ParseInt(idString, 10, 64)
	if err != nil {
		log.Panic(err)
	}

	msg := telegram.NewMessage(id, "FEED :> "+message)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err = api.Send(ctx, msg)
	if err != nil {
		log.Print(err)
	}
}
