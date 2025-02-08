package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	"github.com/souvik131/trade-snippets/analytics"
	"github.com/souvik131/trade-snippets/engine"
	"github.com/souvik131/trade-snippets/queries"
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
	cronJob.AddFunc("45 15 * * *", func() {
		err := engine.Upload()
		if err != nil {
			log.Panicln(err)
		}
	})
	cronJob.Start()

	engine.Subscribe()
	go engine.Write()
	// engine.Read(time.Now().Format(dateFormatConcise))

	// Start query server on port 8080
	if err := queries.StartServer(8080); err != nil {
		log.Panicf("Failed to start query server: %v", err)
	}
}
