package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/jpillora/installer/handler"
	"github.com/jpillora/opts"
	"github.com/jpillora/requestlog/v2"
)

var version = "0.0.0-src"

func main() {
	c := handler.DefaultConfig
	opts.New(&c).Repo("github.com/jpillora/installer").Version(version).Parse()
	log.Printf("default user is '%s'", c.User)
	if c.Token == "" && os.Getenv("GH_TOKEN") != "" {
		c.Token = os.Getenv("GH_TOKEN") // GH_TOKEN was renamed
	}
	if c.Token != "" {
		log.Printf("github token will be used for requests to api.github.com")
	}
	if c.ForceUser != "" {
		log.Printf("locked user to '%s'", c.ForceUser)
	}
	if c.ForceRepo != "" {
		log.Printf("locked repo to '%s'", c.ForceRepo)
	}
	addr := fmt.Sprintf("%s:%d", c.Host, c.Port)
	l, err := net.Listen("tcp4", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listening on %s...", addr)
	h := &handler.Handler{Config: c}
	lh := requestlog.New(h, requestlog.Options{
		TrustProxy: true, // assume will be run in paas
		Filter: func(r *http.Request, code int, duration time.Duration, size int64) bool {
			return r.URL.Path != "/healthz"
		},
	})
	if err := http.Serve(l, lh); err != nil {
		log.Fatal(err)
	}
	log.Print("exiting")
}
