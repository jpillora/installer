package main

import (
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/jpillora/installer/handler"
	"github.com/jpillora/opts"
)

var version = "0.0.0-src"

func main() {
	c := handler.DefaultConfig
	opts.New(&c).Repo("github.com/jpillora/installer").Version(version).Parse()
	log.Printf("default user is '%s', github token set: %v, listening on %d...", c.User, c.Token != "", c.Port)
	l, err := net.Listen("tcp4", "0.0.0.0:"+strconv.Itoa(c.Port))
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listening on port %d...", c.Port)
	h := &handler.Handler{Config: c}
	if err := http.Serve(l, h); err != nil {
		log.Fatal(err)
	}
	log.Print("exiting")
}
