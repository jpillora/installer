package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/replicatedhq/installer/pkg/api"
	"github.com/replicatedhq/installer/pkg/handlers"
)

func main() {
	var port int
	var user string
	var token string

	flag.IntVar(&port, "port", 3000, "port")
	flag.StringVar(&user, "user", "", "default user when not provided in URL")
	flag.StringVar(&token, "token", "", "github api token")
	flag.Parse()

	// if username is not provided as a flag, check the environment
	// if it's not provided there, use the default of 'jpillora'
	if user == "" {
		user, _ = os.LookupEnv("USER")
		if user == "" {
			user = "jpillora"
		}
	}
	if token == "" {
		token, _ = os.LookupEnv("GH_TOKEN")
	}
	if envPort, ok := os.LookupEnv("PORT"); ok {
		var err error
		port, err = strconv.Atoi(envPort)
		if err != nil {
			log.Fatal(err)
		}
	}

	api.InitClient(token)
	log.Printf("Default user is '%s', GH token set: %v, listening on %d...", user, token != "", port)

	r := mux.NewRouter()
	r.HandleFunc("/healthz", handlers.Healthz)
	r.HandleFunc("/{project}", handlers.InstallScript)
	r.HandleFunc("/{owner}/{project}", handlers.InstallScript)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), r); err != nil {
		log.Fatal(err)
	}
}
