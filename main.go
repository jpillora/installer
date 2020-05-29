package main

import (
	"log"
	"net/http"
	"strconv"

	"github.com/jpillora/installer/handler"
	"github.com/jpillora/opts"
)

var version = "0.0.0-src"

func main() {
	c := handler.DefaultConfig
	opts.New(&c).Repo("github.com/jpillora/installer").Version(version).Parse()
	log.Printf("Default user is '%s', GH token set: %v, listening on %d...", c.User, c.Token != "", c.Port)
	h := &handler.Handler{Config: c}
	if err := http.ListenAndServe(":"+strconv.Itoa(c.Port), h); err != nil {
		log.Fatal(err)
	}
}
