package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"text/template"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}
	log.Printf("Listening on %s...", port)
	http.ListenAndServe(":"+port, http.HandlerFunc(install))
}

var (
	user    = `(\/([\w\-]+))?`
	repo    = `([\w\-\_]+)`
	release = `(@([\w\-\.\_]+?))?`
	ext     = `(\.(\w+))?(!)?`
	pathRe  = regexp.MustCompile(`^` + user + `\/` + repo + release + ext + `$`)
)

func install(w http.ResponseWriter, r *http.Request) {
	m := pathRe.FindStringSubmatch(r.URL.Path)
	if len(m) == 0 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	user := m[2]
	if user == "" {
		user = "jpillora"
	}
	repo := m[3]
	release := m[5]
	ext := m[7]
	if ext == "" {
		ext = "sh"
	}
	downloadOnly := m[8] != "!"

	switch ext {
	case "txt":
		w.Header().Set("Content-Type", "text/plain")
		//debug
		fmt.Fprintf(w, "user: %s\n", user)
		fmt.Fprintf(w, "repo: %s\n", repo)
		fmt.Fprintf(w, "release: %s\n", release)
		fmt.Fprintf(w, "ext: %s\n", ext)
		fmt.Fprintf(w, "download-only: %v\n", downloadOnly)
		return
	case "sh":
		w.Header().Set("Content-Type", "text/x-shellscript")
	case "rb":
		w.Header().Set("Content-Type", "text/ruby")
	default:
		http.Error(w, "Unsupported extension", http.StatusBadRequest)
		return
	}

	b, err := ioutil.ReadFile("scripts/install." + ext)
	if err != nil {
		http.Error(w, "Installer script not found", http.StatusInternalServerError)
		return
	}
	t, err := template.New("installer").Parse(string(b))
	if err != nil {
		http.Error(w, "Installer script invalid", http.StatusInternalServerError)
		return
	}

	assets, release, err := getAssets(user, repo, release)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	buff := bytes.Buffer{}
	if err := t.Execute(&buff, &struct {
		User         string
		Program      string
		Release      string
		DownloadOnly bool
		Assets       []asset
	}{
		User:         user,
		Program:      repo,
		Release:      release,
		DownloadOnly: downloadOnly,
		Assets:       assets,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Write(buff.Bytes())
}

type asset struct {
	Name, OS, Arch, URL string
	Is32bit, IsMac      bool
}

func getAssets(user, repo, release string) ([]asset, string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", user, repo)
	ghr := ghRelease{}
	if release == "" {
		url += "/latest"
		if err := get(url, &ghr); err != nil {
			return nil, release, err
		}
		release = ghr.TagName
	} else {
		ghrs := []ghRelease{}
		if err := get(url, &ghrs); err != nil {
			return nil, release, err
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
			return nil, release, fmt.Errorf("Release tag '%s' not found", release)
		}
	}

	if len(ghr.Assets) == 0 {
		return nil, release, errors.New("No assets found")
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
		assets = append(assets, asset{
			Name:    m[1],
			OS:      m[2],
			IsMac:   m[2] == "darwin",
			Arch:    m[3],
			Is32bit: m[3] == "386",
			URL:     ga.BrowserDownloadURL,
		})
	}
	if len(assets) == 0 {
		return nil, release, errors.New("No assets produces")
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

	if resp.StatusCode != 200 {
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
