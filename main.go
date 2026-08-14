package main

import (
	"fmt"
	"io"
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

type command struct {
	version string
	stdout  io.Writer
}

func (c command) handleVersion(args []string) bool {
	if len(args) != 2 || (args[1] != "--version" && args[1] != "-v") {
		return false
	}
	fmt.Fprintln(c.stdout, c.version)
	return true
}

func main() {
	cmd := command{version: version, stdout: os.Stdout}
	if cmd.handleVersion(os.Args) {
		return
	}
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
