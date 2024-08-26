package main

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/robfig/cron"
)

var (
	cronJob = cron.New()
)

func main() {
	if os.Getenv("TA_KITE_ID") == "" {
		godotenv.Load()
	}
	cronJob.AddFunc(os.Getenv("TA_CRON_STRING"), func() {
		Upload()
	})
	cronJob.Start()
	Write()
	// Read(time.Now().Format(dateFormatConcise))
	// Host()
	// Subscribe()

}
