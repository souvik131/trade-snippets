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
	cronJob.AddFunc("0 20 * * *", Upload)
	Write()
	// Read(time.Now())
	// Host()

}
