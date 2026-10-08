package main

import (
	"log"
	"net/http"
	"os"

	"teamflow/internal/platform"
)

func main() {
	address := os.Getenv("API_ADDR")
	if address == "" {
		log.Fatal("API_ADDR must be set")
	}

	if err := http.ListenAndServe(address, platform.NewRouter()); err != nil {
		log.Fatal(err)
	}
}
