package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"
)

const (
	cacheTTL = time.Hour
)

var (
	userRe    = `(\/([\w\-]{1,128}))?`
	repoRe    = `([\w\-\_]{1,128})`
	releaseRe = `(@([\w\-\.\_]{1,128}?))?`
	moveRe    = `(!*)`
	pathRe    = regexp.MustCompile(`^` + userRe + `\/` + repoRe + releaseRe + moveRe + `$`)

	isTermRe     = regexp.MustCompile(`(?i)^(curl|wget)\/`)
	isHomebrewRe = regexp.MustCompile(`(?i)^homebrew`)
	errNotFound  = errors.New("not found")
)

type query struct {
	Timestamp                              time.Time
	User, Program, Release                 string
	MoveToPath, SudoMove, Google, Insecure bool
	Assets                                 []asset
}

func (q query) cacheKey() string {
	h := sha256.New()
	q.Timestamp = time.Time{}
	q.Assets = nil
	if err := json.NewEncoder(h).Encode(q); err != nil {
		panic(err)
	}
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

//Handler serves install scripts using Github releases
type Handler struct {
	Config
	cacheMut sync.Mutex
	cache    map[string]*query
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" {
		http.Redirect(w, r, "https://github.com/jpillora/installer", http.StatusMovedPermanently)
		return
	}
	//calculate reponse type
	var isTerm, isHomebrew, isText bool
	switch r.URL.Query().Get("type") {
	case "script":
		isTerm = true
	case "homebrew":
		isHomebrew = true
	case "text":
		isText = true
	default:
		ua := r.Header.Get("User-Agent")
		switch {
		case isTermRe.MatchString(ua):
			isTerm = true
		case isHomebrewRe.MatchString(ua):
			isHomebrew = true
		default:
			isText = true
		}
	}
	//type specific error response
	showError := func(msg string, code int) {
		if isTerm {
			msg = fmt.Sprintf("echo '%s'\n", msg)
		}
		http.Error(w, msg, http.StatusInternalServerError)
	}
	//"route"
	m := pathRe.FindStringSubmatch(r.URL.Path)
	if len(m) == 0 {
		showError("Invalid path", http.StatusBadRequest)
		return
	}
	q := &query{
		Timestamp:  time.Now(),
		User:       m[2],
		Program:    m[3],
		Release:    m[5],
		MoveToPath: strings.HasPrefix(m[6], "!"),
		SudoMove:   strings.HasPrefix(m[6], "!!"),
		Google:     false,
		Insecure:   r.URL.Query().Get("insecure") == "1",
	}
	//pick a user
	if q.User == "" {
		if q.Program == "micro" {
			//micro > nano!
			q.User = "zyedidia"
		} else {
			//use default user, but fallback to google
			q.User = h.Config.User
			q.Google = true
		}
	}
	//fetch assets
	if err := h.getAssets(q); err != nil {
		showError(err.Error(), http.StatusBadGateway)
		return
	}
	//ready!
	ext := ""
	if isTerm {
		w.Header().Set("Content-Type", "text/x-shellscript")
		ext = "sh"
	} else if isHomebrew {
		w.Header().Set("Content-Type", "text/ruby")
		ext = "rb"
	} else {
		if isText {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "repository: https://github.com/%s/%s\n", q.User, q.Program)
			fmt.Fprintf(w, "user: %s\n", q.User)
			fmt.Fprintf(w, "program: %s\n", q.Program)
			fmt.Fprintf(w, "release: %s\n", q.Release)
			fmt.Fprintf(w, "release assets:\n")
			for i, a := range q.Assets {
				fmt.Fprintf(w, "  [#%02d] %s\n", i+1, a.URL)
			}
			fmt.Fprintf(w, "move-into-path: %v\n", q.MoveToPath)
			fmt.Fprintf(w, "\nto see shell script, visit:\n  %s%s?type=script\n", r.Host, r.URL.String())
			fmt.Fprintf(w, "\nfor more information on this server, visit:\n  github.com/jpillora/installer\n")
			return
		}
		showError("Unknown type", http.StatusInternalServerError)
		return
	}
	t, err := template.New("installer").Parse(string(installScript))
	if err != nil {
		http.Error(w, "Installer script invalid", http.StatusInternalServerError)
		return
	}
	//execute script
	buff := bytes.Buffer{}
	if err := t.Execute(&buff, q); err != nil {
		showError("Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("serving script %s/%s@%s (%s)", q.User, q.Program, q.Release, ext)
	//ready
	w.Write(buff.Bytes())
}

type asset struct {
	Name, OS, Arch, URL, Type string
	Is32bit, IsMac            bool
}

func (a asset) target() string {
	return fmt.Sprintf("%s_%s", a.OS, a.Arch)
}

func (h *Handler) get(url string, v interface{}) error {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if h.Config.Token != "" {
		req.Header.Set("Authorization", "token "+h.Config.Token)
	}
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Request failed: %s: %s", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return errNotFound
	} else if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		return errors.New(http.StatusText(resp.StatusCode) + " " + string(b))
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("Download failed: %s: %s", url, err)
	}

	return nil
}
