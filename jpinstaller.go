package main

import (
	"log"
	"os"

	"github.com/labstack/echo"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	e := echo.New()
	e.Get("/install", install)
	log.Printf("Listening on %s...", port)
	e.Run(":" + port)
}

func install(c *echo.Context) {
	c.String(200, "installingggg")
}
