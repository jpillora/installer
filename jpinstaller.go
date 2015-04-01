package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/labstack/echo"
)

var installsh string

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	installsh = load("install")

	e := echo.New()
	e.Get("/install/:repo", install)
	log.Printf("Listening on %s...", port)
	e.Run(":" + port)
}

func install(c *echo.Context) {
	repo := c.P(0)
	c.String(200, fmt.Sprintf(installsh, repo))
}

func load(script string) string {
	b, err := ioutil.ReadFile("scripts/" + script + ".sh")
	if err != nil {
		log.Fatal(err)
	}
	return string(b)
}
