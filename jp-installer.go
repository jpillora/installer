package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Listening on %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
