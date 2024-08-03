package main

import (
	"time"

	"github.com/robfig/cron"
)

var (
	cronJob = cron.New()
)

func main() {
	cronJob.AddFunc("0 0 8 * *", func() {
		Upload(time.Now())
	})
	Write()

	// Read(time.Now())
	// Host()

}
