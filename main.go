//go:generate statik -dest=. -f -p=scripts -src=scripts -include=*.sh

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/jpillora/opts"
	"github.com/rakyll/statik/fs"

	_ "github.com/jpillora/installer/scripts"
)

var c = struct {
	Port  int    `opts:"help=port, env"`
	User  string `opts:"help=default user when not provided in URL, env"`
	Token string `opts:"help=github api token, env=GH_TOKEN"`
}{
	Port: 3000,
	User: "jpillora",
}

var (
	version       = "0.0.0-src"
	installScript = []byte{}
)

func main() {
	//load static file
	hfs, err := fs.New()
	if err != nil {
		log.Fatalf("bad static file system: %s, fix statik", err)
	}
	installScript, err = fs.ReadFile(hfs, "/install.sh")
	if err != nil {
		log.Fatalf("read script file: %s, fix statik", err)
	}
	//run program
	opts.New(&c).Repo("github.com/jpillora/installer").Version(version).Parse()
	log.Printf("Default user is '%s', GH token set: %v, listening on %d...", c.User, c.Token != "", c.Port)
	if err := http.ListenAndServe(":"+strconv.Itoa(c.Port), http.HandlerFunc(install)); err != nil {
		log.Fatal(err)
	}
}

const (
	cacheTTL = time.Hour
)

var (
	userRe       = `(\/([\w\-]{1,128}))?`
	repoRe       = `([\w\-\_]{1,128})`
	releaseRe    = `(@([\w\-\.\_]{1,128}?))?`
	moveRe       = `(!*)`
	pathRe       = regexp.MustCompile(`^` + userRe + `\/` + repoRe + releaseRe + moveRe + `$`)
	fileExtRe    = regexp.MustCompile(`(\.[a-z][a-z0-9]+)+$`)
	isTermRe     = regexp.MustCompile(`(?i)^(curl|wget)\/`)
	isHomebrewRe = regexp.MustCompile(`(?i)^homebrew`)
	posixOSRe    = regexp.MustCompile(`(?i)(darwin|linux|(net|free|open)bsd|mac|osx)`)
	archRe       = regexp.MustCompile(`(arm|386|amd64|32|64)`)
	cache        = map[string]*query{}
	cacheMut     = sync.Mutex{}
	errNotFound  = errors.New("not found")
)

type query struct {
	Timestamp                              time.Time
	User, Program, Release                 string
	MoveToPath, SudoMove, Google, Insecure bool
	Assets                                 []asset
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
			q.User = c.User
			q.Google = true
		}
	}
	//fetch assets
	if err := getAssets(q); err != nil {
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

func getAssets(q *query) error {
	//cached?
	key := strings.Join([]string{q.User, q.Program, q.Release}, "|")
	cacheMut.Lock()
	cq, ok := cache[key]
	cacheMut.Unlock()
	if ok && time.Now().Sub(cq.Timestamp) < cacheTTL {
		//cache hit
		*q = *cq
		return nil
	}
	//do real operation
	err := getAssetsNoCache(q)
	if err == nil {
		//didn't need google
		q.Google = false
	} else if err == errNotFound && q.Google {
		//use google to auto-detect user...
		user, program, gerr := searchGoogle(q.Program)
		if gerr != nil {
			log.Printf("google search failed: %s", gerr)
		} else if program == q.Program {
			q.User = user
			//retry assets...
			err = getAssetsNoCache(q)
		}
	}
	//asset fetch failed, dont cache
	if err != nil {
		return err
	}
	//success store results
	cacheMut.Lock()
	cache[key] = q
	cacheMut.Unlock()
	return nil
}

func getAssetsNoCache(q *query) error {
	user := q.User
	repo := q.Program
	release := q.Release
	if release == "" {
		release = "latest"
	}
	//not cached - ask github
	log.Printf("fetching asset info for %s/%s@%s", user, repo, release)
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", user, repo)
	ghas := []ghAsset{}
	if q.Release == "" {
		url += "/latest"
		ghr := ghRelease{}
		if err := get(url, &ghr); err != nil {
			return err
		}
		q.Release = ghr.TagName //discovered
		ghas = ghr.Assets
	} else {
		ghrs := []ghRelease{}
		if err := get(url, &ghrs); err != nil {
			return err
		}
		found := false
		for _, ghr := range ghrs {
			if ghr.TagName == release {
				found = true
				if err := get(ghr.AssetsURL, &ghas); err != nil {
					return err
				}
				ghas = ghr.Assets
				break
			}
		}
		if !found {
			return fmt.Errorf("Release tag '%s' not found", release)
		}
	}
	if len(ghas) == 0 {
		return errors.New("No assets found")
	}
	assets := []asset{}
	for _, ga := range ghas {
		url := ga.BrowserDownloadURL
		//match
		os := posixOSRe.FindString(ga.Name)
		arch := archRe.FindString(ga.Name)
		//os modifications
		if os == "" {
			continue //unknown os
		}
		if os == "mac" || os == "osx" {
			os = "darwin"
		}
		//arch modifications
		if arch == "64" || arch == "" {
			arch = "amd64" //default
		} else if arch == "32" {
			arch = "386"
		}
		assets = append(assets, asset{
			//target
			OS:   os,
			Arch: arch,
			//
			Name: ga.Name,
			URL:  url,
			Type: fileExtRe.FindString(url),
			//computed
			IsMac:   os == "darwin",
			Is32bit: arch == "386",
		})
	}
	if len(assets) == 0 {
		return errors.New("No downloads found for this release")
	}
	//TODO: handle duplicate asset.targets
	q.Assets = assets
	return nil
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

var searchGithubRe = regexp.MustCompile(`https:\/\/github\.com\/(\w+)\/(\w+)`)

//uses im feeling lucky and grabs the "Location"
//header from the 302, which contains the IMDB ID
func searchGoogle(phrase string) (user, project string, err error) {
	phrase += " site:github.com"
	log.Printf("google search for '%s'", phrase)
	v := url.Values{}
	v.Set("btnI", "") //I'm feeling lucky
	v.Set("q", phrase)
	urlstr := "https://www.google.com.au/search?" + v.Encode()
	req, err := http.NewRequest("HEAD", urlstr, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Accept", "*/*")
	//I'm a browser... :)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_2) "+
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/41.0.2272.118 Safari/537.36")
	//roundtripper doesn't follow redirects
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	//assume redirection
	if resp.StatusCode != 302 {
		return "", "", fmt.Errorf("non-redirect response: %d", resp.StatusCode)
	}
	//extract Location header URL
	loc := resp.Header.Get("Location")
	m := searchGithubRe.FindStringSubmatch(loc)
	if len(m) == 0 {
		return "", "", fmt.Errorf("github url not found in redirect: %s", loc)
	}
	user = m[1]
	project = m[2]
	return user, project, nil
}
