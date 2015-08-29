package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Printf("Listening on %s...", port)
	http.ListenAndServe(":"+port, http.HandlerFunc(install))
}

func install(w http.ResponseWriter, r *http.Request) {
	repo := strings.TrimPrefix(r.URL.Path, "/")

	mv := false
	if strings.HasSuffix(repo, "!") {
		mv = true
		repo = strings.TrimSuffix(repo, "!")
	}

	b, err := ioutil.ReadFile("scripts/install.sh")
	if err != nil {
		return
	}
	script := []byte(fmt.Sprintf(string(b), repo, mv))

	w.Header().Set("Content-Type", "text/x-shellscript")
	w.Write(script)
}

// func load(script string) {
// 	if os.Getenv("DEV") != "" {
// 	}
// 	resp, err := http.Get("https://gist.githubusercontent.com/jpillora/529a89dd86529eb2a213/raw/install.sh")
// 	if err != nil {
// 		log.Printf("failed to load script: %s", err)
// 		return
// 	}
// 	defer resp.Body.Close()
// 	b, err := ioutil.ReadAll(resp.Body)
// 	if err != nil {
// 		log.Printf("failed to download script: %s", err)
// 		return
// 	}
// 	installsh = string(b)
// }
