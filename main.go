package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/souvik131/trade-snippets/kite"
)

func main() {

	var k = &kite.Kite{}
	ctx := context.Background()
	err := k.Login(&ctx)
	if err != nil {
		log.Panicf("%s", err)
		return
	}

	Serve(&ctx, k)
	Host()

}
func Host() {
	// Define the directory to serve
	dir := "./binary"

	// Create a file server handler
	fileServer := http.FileServer(http.Dir(dir))

	// Custom handler for the root path
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			// Serve the index.html file
			indexPath := filepath.Join(dir, "index.html")
			if _, err := os.Stat(indexPath); os.IsNotExist(err) {
				http.NotFound(w, r)
				return
			}
			http.ServeFile(w, r, indexPath)
		} else {
			// Serve other files
			fileServer.ServeHTTP(w, r)
		}
	})

	// Get the absolute path of the directory
	absDir, err := filepath.Abs(dir)
	if err != nil {
		log.Fatal(err)
	}

	// Output the directory being served
	log.Printf("Serving files from %s on http://localhost:8080\n", absDir)

	// Start the HTTP server
	log.Fatal(http.ListenAndServe(":8080", nil))
}
