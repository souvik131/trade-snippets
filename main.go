package main

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	"github.com/souvik131/trade-snippets/analytics"
	"github.com/souvik131/trade-snippets/engine"
)

var (
	cronJob = cron.New()
)

func main() {
	if os.Getenv("TA_KITE_ID") == "" {
		godotenv.Load()

		os.Setenv("TA_NATS_URI", "nats://127.0.0.1:4222")
		os.Setenv("DB_URI", "127.0.0.1:9000")
	}

	analytics.Init()
	cronJob.AddFunc(os.Getenv("45 17 * * 1-5"), func() {
		engine.Upload()
	})
	cronJob.Start()

	engine.Subscribe()
	engine.Write()
	// engine.Read(time.Now().Format(dateFormatConcise))

}
