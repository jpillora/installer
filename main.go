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
	"text/template"

	"github.com/jpillora/opts"
)

func main() {
	c := &struct {
		Port int `help:"port" env:"PORT"`
	}{
		Port: 3000,
	}
	opts.Parse(&c)
	port := strconv.Itoa(c.Port)
	log.Printf("Listening on %s...", port)
	http.ListenAndServe(":"+port, http.HandlerFunc(install))
}

var (
	userRe      = `(\/([\w\-]+))?`
	repoRe      = `([\w\-\_]+)`
	releaseRe   = `(@([\w\-\.\_]+?))?`
	extRe       = `(\.(\w+))?(!)?`
	pathRe      = regexp.MustCompile(`^` + userRe + `\/` + repoRe + releaseRe + extRe + `$`)
	fileExtRe   = `(\.[a-z][a-z0-9]+)+$`
	fileExt     = regexp.MustCompile(fileExtRe)
	isTermReStr = `(?i)^(curl|wget)\/`
	isTermRe    = regexp.MustCompile(isTermReStr)
)

func install(w http.ResponseWriter, r *http.Request) {
	//terminal client?
	isTerm := isTermRe.MatchString(r.Header.Get("User-Agent"))
	//extension specific error
	var ext string
	showError := func(msg string, code int) {
		if ext == "txt" {
			//noop
		} else if ext == "rb" {
			//TODO write ruby
		} else if ext == "sh" || isTerm {
			msg = fmt.Sprintf("echo '%s'\n", msg)
		}
		http.Error(w, msg, http.StatusInternalServerError)
	}

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
		MoveToPath: m[8] == "!",
	}
	if data.User == "" {
		data.User = "jpillora"
	}
	ext = m[7]
	if isTerm && ext == "" {
		ext = "sh"
	}
	switch ext {
	case "txt":
		w.Header().Set("Content-Type", "text/plain")
		//debug
		fmt.Fprintf(w, "user: %s\n", data.User)
		fmt.Fprintf(w, "program: %s\n", data.Program)
		fmt.Fprintf(w, "release: %s\n", data.Release)
		fmt.Fprintf(w, "ext: %s\n", ext)
		fmt.Fprintf(w, "move-to-path: %v\n", data.MoveToPath)
		return
	case "sh":
		w.Header().Set("Content-Type", "text/x-shellscript")
	case "rb":
		w.Header().Set("Content-Type", "text/ruby")
	default:
		showError("Unsupported extension", http.StatusBadRequest)
		return
	}
	b, err := ioutil.ReadFile("scripts/install." + ext)
	if err != nil {
		showError("Installer script not found", http.StatusInternalServerError)
		return
	}
	t, err := template.New("installer").Parse(string(b))
	if err != nil {
		http.Error(w, "Installer script invalid", http.StatusInternalServerError)
		return
	}
	//fetch assets
	assets, release, err := getAssets(data.User, data.Program, data.Release)
	if err != nil {
		showError(err.Error(), http.StatusBadGateway)
		return
	}
	data.Release = release //update release
	data.Assets = assets
	//execute script
	buff := bytes.Buffer{}
	if err := t.Execute(&buff, data); err != nil {
		showError("Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	//ready
	w.Write(buff.Bytes())
}

type asset struct {
	Name, OS, Arch, URL, Type string
	Is32bit, IsMac            bool
}

func getAssets(user, repo, release string) ([]asset, string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", user, repo)
	ghr := ghRelease{}
	if release == "" {
		url += "/latest"
		if err := get(url, &ghr); err != nil {
			return nil, "", err
		}
		release = ghr.TagName
	} else {
		ghrs := []ghRelease{}
		if err := get(url, &ghrs); err != nil {
			return nil, "", err
		}
		found := false
		for _, r := range ghrs {
			if r.TagName == release {
				found = true
				ghr = r
				break
			}
		}
		if !found {
			return nil, "", fmt.Errorf("Release tag '%s' not found", release)
		}
	}

	if len(ghr.Assets) == 0 {
		return nil, "", errors.New("No assets found")
	}

	assets := []asset{}
	for _, ga := range ghr.Assets {
		m := assetRe.FindStringSubmatch(ga.Name)
		if len(m) == 0 {
			continue
		}
		if m[2] != "linux" && m[2] != "darwin" {
			continue
		}
		url := ga.BrowserDownloadURL
		assets = append(assets, asset{
			Name:    m[1],
			OS:      m[2],
			IsMac:   m[2] == "darwin",
			Arch:    m[3],
			Is32bit: m[3] == "386",
			URL:     url,
			Type:    fileExt.FindString(url),
		})
	}
	if len(assets) == 0 {
		return nil, "", errors.New("No assets produces")
	}
	return assets, release, nil
}

var assetRe = regexp.MustCompile(`^(.+)_(darwin|linux)_(arm|386|amd64)(\.[\w\.]+)$`)

func get(url string, v interface{}) error {
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

type ghRelease struct {
	Assets []struct {
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
	} `json:"assets"`
	AssetsURL string `json:"assets_url"`
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
