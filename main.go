package main

import (
	"context"
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
	"github.com/souvik131/trade-snippets/kite"
)

func main() {
	log.Println("Starting application...")

	// Load environment variables]
	envpath := os.Getenv("ENVPATH")
	if envpath == "" {
		err := godotenv.Load()
		if err != nil {
			log.Panicf("Error loading .env file: %s", err)
			return
		}

	} else {
		err := godotenv.Load(envpath)
		if err != nil {
			log.Panicf("Error loading .env file: %s", err)
			return
		}
	}

	// Initialize Kite client
	var k = &kite.Kite{}
	ctx := context.Background()

	// Login to Kite
	log.Println("Logging in to Kite...")
	err := k.Login(&ctx)
	if err != nil {
		log.Panicf("Login failed: %s", err)
		return
	}
	log.Println("Login successful")

	// Verify instruments are loaded
	if len(*kite.BrokerInstrumentTokens) == 0 {
		log.Println("Warning: No instruments loaded after login")
	} else {
		log.Printf("Loaded %d instruments", len(*kite.BrokerInstrumentTokens))
	}

	// Start the server
	log.Println("Starting server...")
	Serve(&ctx, k)

	// Keep main thread alive
	wg := &sync.WaitGroup{}
	wg.Add(1)
	wg.Wait()
}
