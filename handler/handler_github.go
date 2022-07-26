package handler

import (
	"errors"
	"fmt"
	"log"
	"time"
)

func (h *Handler) execute(q *Query) (Result, error) {
	//cached?
	key := q.cacheKey()
	h.cacheMut.Lock()
	if h.cache == nil {
		h.cache = map[string]Result{}
	}
	result, ok := h.cache[key]
	h.cacheMut.Unlock()
	if ok && time.Since(result.Timestamp) < cacheTTL {
		//cache hit
		return result, nil
	}
	//do real operation
	result.Timestamp = time.Now()
	assets, err := h.getAssetsNoCache(q)
	if err == nil {
		//didn't need google
		q.Google = false
	} else if errors.Is(err, errNotFound) && q.Google {
		//use google to auto-detect user...
		user, program, gerr := searchGoogle(q.Program)
		if gerr != nil {
			log.Printf("google search failed: %s", gerr)
		} else {
			log.Printf("google search found: %s/%s", user, program)
			if program != q.Program {
				log.Printf("program mismatch: got %s: expected %s", q.Program, program)
			}
			q.Program = program
			q.User = user
			//retry assets...
			assets, err = h.getAssetsNoCache(q)
		}
	}
	//detect if we have a native m1 asset
	for _, a := range assets {
		if a.OS == "darwin" && a.Arch == "arm64" {
			result.M1Asset = true
			break
		}
	}
	result.Assets = assets
	//asset fetch failed, dont cache
	if err != nil {
		return result, err
	}
	//success store results
	h.cacheMut.Lock()
	h.cache[key] = result
	h.cacheMut.Unlock()
	return result, nil
}

func (h *Handler) getAssetsNoCache(q *Query) (Assets, error) {
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
		if err := h.get(url, &ghr); err != nil {
			return nil, err
		}
		q.Release = ghr.TagName //discovered
		ghas = ghr.Assets
	} else {
		ghrs := []ghRelease{}
		if err := h.get(url, &ghrs); err != nil {
			return nil, err
		}
		found := false
		for _, ghr := range ghrs {
			if ghr.TagName == release {
				found = true
				if err := h.get(ghr.AssetsURL, &ghas); err != nil {
					return nil, err
				}
				ghas = ghr.Assets
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("release tag '%s' not found", release)
		}
	}
	if len(ghas) == 0 {
		return nil, errors.New("no assets found")
	}
	assets := Assets{}

	index := map[string]bool{}

	//TODO: handle duplicate asset.targets
	for _, ga := range ghas {
		url := ga.BrowserDownloadURL
		//only binary containers are supported
		//TODO deb,rpm etc
		fext := getFileExt(url)
		if fext != ".zip" && fext != ".gz" && fext != ".tar.gz" && fext != ".tgz" {
			continue
		}
		//match
		os := getOS(ga.Name)
		arch := getArch(ga.Name)
		//windows not supported yet
		if os == "windows" {
			//TODO: powershell
			//  EG: iwr https://deno.land/x/install/install.ps1 -useb | iex
			continue
		}
		//unknown os, cant use
		if os == "" {
			continue
		}
		//there can only be 1 file for each OS/Arch
		key := os + "/" + arch
		if index[key] {
			continue
		}
		index[key] = true
		//include!
		assets = append(assets, Asset{
			//target
			OS:   os,
			Arch: arch,
			//
			Name: ga.Name,
			URL:  url,
			Type: fext,
			//computed
			IsMac:   os == "darwin",
			Is32bit: arch == "386",
		})
	}
	if len(assets) == 0 {
		return nil, errors.New("no downloads found for this release")
	}
	return assets, nil
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
