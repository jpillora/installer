package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/jpillora/opts"
)

var c = &struct {
	Port  int    `help:"port" env:"PORT"`
	User  string `help:"default user when not provided in URL" env:"USER"`
	Token string `help:"github api token" env:"GH_TOKEN"`
}{
	Port: 3000,
	User: "jpillora",
}

var VERSION = "0.0.0-src"

func main() {
	opts.New(&c).Repo("github.com/jpillora/installer").Version(VERSION).Parse()
	log.Printf("Default user is '%s', GH token set: %v, listening on %d...", c.User, c.Token != "", c.Port)
	if err := http.ListenAndServe(":"+strconv.Itoa(c.Port), http.HandlerFunc(install)); err != nil {
		log.Fatal(err)
	}
}

const (
	cacheTTL = time.Hour
)

var (
	userRe       = `(\/([\w\-]+))?`
	repoRe       = `([\w\-\_]+)`
	releaseRe    = `(@([\w\-\.\_]+?))?`
	moveRe       = `(!)?`
	pathRe       = regexp.MustCompile(`^` + userRe + `\/` + repoRe + releaseRe + moveRe + `$`)
	fileExtRe    = regexp.MustCompile(`(\.[a-z][a-z0-9]+)+$`)
	isTermRe     = regexp.MustCompile(`(?i)^(curl|wget)\/`)
	isHomebrewRe = regexp.MustCompile(`(?i)^homebrew`)
	posixOSRe    = regexp.MustCompile(`(darwin|linux|(net|free|open)bsd)`)
	archRe       = regexp.MustCompile(`(arm|386|amd64)`)
	cache        = map[string]cacheItem{}
	cacheMut     = sync.Mutex{}
)

type cacheItem struct {
	added   time.Time
	assets  []asset
	release string
}

func install(w http.ResponseWriter, r *http.Request) {
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
	data := &struct {
		User, Program, Release string
		MoveToPath, Insecure   bool
		Assets                 []asset
	}{
		User:       m[2],
		Program:    m[3],
		Release:    m[5],
		MoveToPath: m[6] == "!",
		Insecure:   r.URL.Query().Get("insecure") == "1",
	}
	if data.User == "" {
		data.User = c.User
	}
	//fetch assets
	assets, release, err := getAssets(data.User, data.Program, data.Release)
	if err != nil {
		showError(err.Error(), http.StatusBadGateway)
		return
	}
	data.Release = release //update release
	data.Assets = assets

	ext := ""
	if isTerm {
		w.Header().Set("Content-Type", "text/x-shellscript")
		ext = "sh"
	} else if isHomebrew {
		w.Header().Set("Content-Type", "text/ruby")
		ext = "rb"
	} else if isText {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "repository: https://github.com/%s/%s\n", data.User, data.Program)
		fmt.Fprintf(w, "user: %s\n", data.User)
		fmt.Fprintf(w, "program: %s\n", data.Program)
		fmt.Fprintf(w, "release: %s\n", data.Release)
		fmt.Fprintf(w, "release assets:\n")
		for i, a := range data.Assets {
			fmt.Fprintf(w, "  [#%02d] %s\n", i+1, a.URL)
		}
		fmt.Fprintf(w, "move-into-path: %v\n", data.MoveToPath)
		fmt.Fprintf(w, "\nto see shell script, visit:\n  %s%s?type=script\n", r.Host, r.URL.String())
		fmt.Fprintf(w, "\nfor more information on this server, visit:\n  github.com/jpillora/installer\n")
		return
	} else {
		showError("Unknown type", http.StatusInternalServerError)
		return
	}
	script := "scripts/install." + ext
	b, err := ioutil.ReadFile(script)
	if err != nil {
		showError("Installer script not found", http.StatusInternalServerError)
		return
	}
	t, err := template.New("installer").Parse(string(b))
	if err != nil {
		http.Error(w, "Installer script invalid", http.StatusInternalServerError)
		return
	}
	//execute script
	buff := bytes.Buffer{}
	if err := t.Execute(&buff, data); err != nil {
		showError("Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	log.Printf("serving script %s/%s@%s (%s)", data.User, data.Program, data.Release, ext)
	//ready
	w.Write(buff.Bytes())
}

type asset struct {
	Name, OS, Arch, URL, Type string
	Is32bit, IsMac            bool
}

func getAssets(user, repo, release string) ([]asset, string, error) {
	//cached?
	key := strings.Join([]string{user, repo, release}, "|")
	cacheMut.Lock()
	ci, ok := cache[key]
	cacheMut.Unlock()
	if ok && time.Now().Sub(ci.added) < cacheTTL {
		return ci.assets, ci.release, nil
	}
	//not cached - ask github
	log.Printf("fetching asset info for %s/%s@%s", user, repo, release)
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", user, repo)
	ghas := []ghAsset{}
	if release == "" {
		url += "/latest"
		ghr := ghRelease{}
		if err := get(url, &ghr); err != nil {
			return nil, "", err
		}
		release = ghr.TagName
		ghas = ghr.Assets
	} else {
		ghrs := []ghRelease{}
		if err := get(url, &ghrs); err != nil {
			return nil, "", err
		}
		found := false
		for _, ghr := range ghrs {
			if ghr.TagName == release {
				found = true
				if err := get(ghr.AssetsURL, &ghas); err != nil {
					return nil, "", err
				}
				ghas = ghr.Assets
				break
			}
		}
		if !found {
			return nil, "", fmt.Errorf("Release tag '%s' not found", release)
		}
	}
	if len(ghas) == 0 {
		return nil, "", errors.New("No assets found")
	}
	assets := []asset{}
	for _, ga := range ghas {
		url := ga.BrowserDownloadURL
		os := posixOSRe.FindString(ga.Name)
		arch := archRe.FindString(ga.Name)
		if os == "" || arch == "" {
			continue
		}
		assets = append(assets, asset{
			Name:    ga.Name,
			OS:      os,
			IsMac:   os == "darwin",
			Arch:    arch,
			Is32bit: arch == "386",
			URL:     url,
			Type:    fileExtRe.FindString(url),
		})
	}
	if len(assets) == 0 {
		return nil, "", errors.New("No downloads found for this release")
	}
	//success store results
	cacheMut.Lock()
	cache[key] = cacheItem{time.Now(), assets, release}
	cacheMut.Unlock()
	return assets, release, nil
}

func get(url string, v interface{}) error {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if c.Token != "" {
		req.Header.Set("Authorization", "token "+c.Token)
	}
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("Request failed: %s: %s", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return fmt.Errorf("Download not found (%s)", url)
	} else if resp.StatusCode != 200 {
		b, _ := ioutil.ReadAll(resp.Body)
		return errors.New(http.StatusText(resp.StatusCode) + " " + string(b))
	}

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("Download failed: %s: %s", url, err)
	}

	return nil
}

type ghAsset struct {
	BrowserDownloadURL string `json:"browser_download_url"`
	ContentType        string `json:"content_type"`
	CreatedAt          string `json:"created_at"`
	DownloadCount      int    `json:"download_count"`
	ID                 int    `json:"id"`
	Label              string `json:"label"`
	Name               string `json:"name"`
	Size               int    `json:"size"`
	State              string `json:"state"`
	UpdatedAt          string `json:"updated_at"`
	Uploader           struct {
		AvatarURL         string `json:"avatar_url"`
		EventsURL         string `json:"events_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		GravatarID        string `json:"gravatar_id"`
		HTMLURL           string `json:"html_url"`
		ID                int    `json:"id"`
		Login             string `json:"login"`
		OrganizationsURL  string `json:"organizations_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		ReposURL          string `json:"repos_url"`
		SiteAdmin         bool   `json:"site_admin"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		Type              string `json:"type"`
		URL               string `json:"url"`
	} `json:"uploader"`
	URL string `json:"url"`
}
type ghRelease struct {
	Assets    []ghAsset `json:"assets"`
	AssetsURL string    `json:"assets_url"`
	Author    struct {
		AvatarURL         string `json:"avatar_url"`
		EventsURL         string `json:"events_url"`
		FollowersURL      string `json:"followers_url"`
		FollowingURL      string `json:"following_url"`
		GistsURL          string `json:"gists_url"`
		GravatarID        string `json:"gravatar_id"`
		HTMLURL           string `json:"html_url"`
		ID                int    `json:"id"`
		Login             string `json:"login"`
		OrganizationsURL  string `json:"organizations_url"`
		ReceivedEventsURL string `json:"received_events_url"`
		ReposURL          string `json:"repos_url"`
		SiteAdmin         bool   `json:"site_admin"`
		StarredURL        string `json:"starred_url"`
		SubscriptionsURL  string `json:"subscriptions_url"`
		Type              string `json:"type"`
		URL               string `json:"url"`
	} `json:"author"`
	Body            string      `json:"body"`
	CreatedAt       string      `json:"created_at"`
	Draft           bool        `json:"draft"`
	HTMLURL         string      `json:"html_url"`
	ID              int         `json:"id"`
	Name            interface{} `json:"name"`
	Prerelease      bool        `json:"prerelease"`
	PublishedAt     string      `json:"published_at"`
	TagName         string      `json:"tag_name"`
	TarballURL      string      `json:"tarball_url"`
	TargetCommitish string      `json:"target_commitish"`
	UploadURL       string      `json:"upload_url"`
	URL             string      `json:"url"`
	ZipballURL      string      `json:"zipball_url"`
}
