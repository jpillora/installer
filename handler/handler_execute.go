package handler

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

func (h *Handler) execute(q Query) (Result, error) {
	//load from cache
	key := q.cacheKey()
	h.cacheMut.Lock()
	if h.cache == nil {
		h.cache = map[string]Result{}
	}
	cached, ok := h.cache[key]
	h.cacheMut.Unlock()
	//cache hit
	if ok && time.Since(cached.Timestamp) < cacheTTL {
		return cached, nil
	}
	//do real operation
	ts := time.Now()
	release, assets, err := h.getAssetsNoCache(q)
	if err == nil {
		//didn't need search
		q.Search = false
	} else if errors.Is(err, errNotFound) && q.Search {
		//use ddg/google to auto-detect user...
		user, program, gerr := imFeelingLuck(q.Program)
		if gerr != nil {
			log.Printf("web search failed: %s", gerr)
		} else {
			log.Printf("web search found: %s/%s", user, program)
			if program != q.Program {
				log.Printf("program mismatch: got %s: expected %s", q.Program, program)
			}
			q.Program = program
			q.User = user
			//retry assets...
			release, assets, err = h.getAssetsNoCache(q)
		}
	}
	//asset fetch failed, dont cache
	if err != nil {
		return Result{}, err
	}
	//success
	if q.Release == "" && release != "" {
		log.Printf("detected release: %s", release)
		q.Release = release
	}
	result := Result{
		Timestamp:       ts,
		Query:           q,
		ResolvedRelease: release,
		Assets:          assets,
		M1Asset:         assets.HasM1(),
	}
	//success store results
	h.cacheMut.Lock()
	h.cache[key] = result
	h.cacheMut.Unlock()
	return result, nil
}

func (h *Handler) getAssetsNoCache(q Query) (string, Assets, error) {
	user := q.User
	repo := q.Program
	release := q.Release
	//not cached - ask github
	log.Printf("fetching asset info for %s/%s@%s", user, repo, release)
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", user, repo)
	ghas := ghAssets{}
	if release == "" || release == "latest" {
		url += "/latest"
		ghr := ghRelease{}
		if err := h.get(url, &ghr); err != nil {
			return release, nil, err
		}
		release = ghr.TagName //discovered
		ghas = ghr.Assets
	} else {
		ghrs := []ghRelease{}
		if err := h.get(url, &ghrs); err != nil {
			return release, nil, err
		}
		found := false
		for _, ghr := range ghrs {
			if ghr.TagName == release {
				found = true
				if err := h.get(ghr.AssetsURL, &ghas); err != nil {
					return release, nil, err
				}
				ghas = ghr.Assets
				break
			}
		}
		if !found {
			return release, nil, fmt.Errorf("release tag '%s' not found", release)
		}
	}
	if len(ghas) == 0 {
		return release, nil, errors.New("no assets found")
	}
	sumIndex, _ := ghas.getSumIndex()
	if l := len(sumIndex); l > 0 {
		log.Printf("fetched %d asset shasums", l)
	}
	index := map[string]Asset{}
	for _, ga := range ghas {
		url := ga.BrowserDownloadURL
		//only binary containers are supported
		//TODO deb,rpm etc
		fext := getFileExt(url)
		if fext == "" && ga.Size > 1024*1024 {
			fext = ".bin" // +1MB binary
		}
		switch fext {
		case ".bin", ".zip", ".tar.bz", ".tar.bz2", ".bz2", ".gz", ".tar.gz", ".tgz":
			// valid
		default:
			log.Printf("fetched asset has unsupported file type: %s (ext '%s')", ga.Name, fext)
			continue
		}
		//match
		os := getOS(ga.Name)
		arch := getArch(ga.Name)
		//windows not supported yet
		if os == "windows" {
			log.Printf("fetched asset is for windows: %s", ga.Name)
			//TODO: powershell
			// EG: iwr https://deno.land/x/install/install.ps1 -useb | iex
			continue
		}
		//unknown os, cant use
		if os == "" {
			log.Printf("fetched asset has unknown os: %s", ga.Name)
			continue
		}
		// user selecting a particular asset?
		if q.Select != "" && !strings.Contains(ga.Name, q.Select) {
			log.Printf("select excludes asset: %s", ga.Name)
			continue
		}
		asset := Asset{
			OS:     os,
			Arch:   arch,
			Name:   ga.Name,
			URL:    url,
			Type:   fext,
			SHA256: sumIndex[ga.Name],
		}
		//there can only be 1 file for each OS/Arch
		key := asset.Key()
		other, exists := index[key]
		if exists {
			gnu := func(s string) bool { return strings.Contains(s, "gnu") }
			musl := func(s string) bool { return strings.Contains(s, "musl") }
			g2m := gnu(other.Name) && !musl(other.Name) && !gnu(asset.Name) && musl(asset.Name)
			// prefer musl over glib for portability, override with select=gnu
			if !g2m {
				continue
			}
		}
		index[key] = asset
	}
	if len(index) == 0 {
		return release, nil, errors.New("no downloads found for this release")
	}
	assets := Assets{}
	for _, a := range index {
		log.Printf("including asset: %s (%s)", a.Name, a.Key())
		assets = append(assets, a)
	}
	sort.Slice(assets, func(i, j int) bool {
		return assets[i].Key() < assets[j].Key()
	})
	return release, assets, nil
}

type ghAssets []ghAsset

func (as ghAssets) getSumIndex() (map[string]string, error) {
	url := ""
	for _, ga := range as {
		//is checksum file?
		if ga.IsChecksumFile() {
			url = ga.BrowserDownloadURL
			break
		}
	}
	if url == "" {
		return nil, errors.New("no sum file found")
	}
	resp, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// take each line and insert into the index
	index := map[string]string{}
	s := bufio.NewScanner(resp.Body)
	for s.Scan() {
		fs := strings.Fields(s.Text())
		if len(fs) != 2 {
			continue
		}
		index[fs[1]] = fs[0]
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return index, nil
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
		ID    int    `json:"id"`
		Login string `json:"login"`
	} `json:"uploader"`
	URL string `json:"url"`
}

func (g ghAsset) IsChecksumFile() bool {
	return checksumRe.MatchString(strings.ToLower(g.Name)) && g.Size < 64*1024 //maximum file size 64KB
}

type ghRelease struct {
	Assets    []ghAsset `json:"assets"`
	AssetsURL string    `json:"assets_url"`
	Author    struct {
		ID    int    `json:"id"`
		Login string `json:"login"`
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
