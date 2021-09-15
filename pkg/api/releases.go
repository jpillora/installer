package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
)

const cacheDuration = time.Minute * 5

var client *github.Client
var clientMut sync.Mutex
var cacheMut sync.Mutex

var resultsCache = map[string]*cacheEntry{}

type cacheEntry struct {
	created  time.Time
	releases []*github.RepositoryRelease
	release  *github.RepositoryRelease
}

func InitClient(token string) {
	clientMut.Lock()
	defer clientMut.Unlock()
	if client == nil {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)

		go backgroundCacheCleanupLoop()
	}
	return
}

func FetchReleases(ctx context.Context, owner, repo string) ([]*github.RepositoryRelease, error) {
	if client == nil {
		return nil, fmt.Errorf("initialize the client before fetching releases")
	}

	cacheIndex := fmt.Sprintf("allreleases:%s:%s", owner, repo)
	cached := getCachedIfPresent(cacheIndex)
	if cached != nil {
		return cached.releases, nil
	}

	releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{PerPage: 20})
	if err != nil {
		return nil, fmt.Errorf("unable to list releases for %s/%s: %w", owner, repo, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unable to list releases for %s/%s, got error code %d", owner, repo, resp.StatusCode)
	}

	go addToCache(cacheIndex, cacheEntry{
		created:  time.Now(),
		releases: releases,
	})

	return releases, nil
}

func FetchLatestRelease(ctx context.Context, owner, repo string) (*github.RepositoryRelease, error) {
	if client == nil {
		return nil, fmt.Errorf("initialize the client before fetching releases")
	}

	cacheIndex := fmt.Sprintf("latestrelease:%s:%s", owner, repo)
	cached := getCachedIfPresent(cacheIndex)
	if cached != nil {
		return cached.release, nil
	}

	release, resp, err := client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("unable to get latest release for %s/%s: %w", owner, repo, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unable to get latest release for %s/%s, got error code %d", owner, repo, resp.StatusCode)
	}

	go addToCache(cacheIndex, cacheEntry{
		created: time.Now(),
		release: release,
	})

	return release, nil
}

func FetchSpecificRelease(ctx context.Context, owner, repo, version string) (*github.RepositoryRelease, error) {
	if client == nil {
		return nil, fmt.Errorf("initialize the client before fetching releases")
	}

	cacheIndex := fmt.Sprintf("versionrelease:%s:%s:%s", owner, repo, version)
	cached := getCachedIfPresent(cacheIndex)
	if cached != nil {
		return cached.release, nil
	}

	release, resp, err := client.Repositories.GetReleaseByTag(ctx, owner, repo, version)
	if err != nil {
		return nil, fmt.Errorf("unable to get release %q for %s/%s: %w", version, owner, repo, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unable to get release %q for %s/%s, got error code %d", version, owner, repo, resp.StatusCode)
	}

	go addToCache(cacheIndex, cacheEntry{
		created: time.Now(),
		release: release,
	})

	return release, nil
}

func getCachedIfPresent(index string) *cacheEntry {
	cacheMut.Lock()
	defer cacheMut.Unlock()
	cached, ok := resultsCache[index]
	if !ok {
		return nil
	}
	if cached == nil {
		return nil
	}

	if cached.created.After(time.Now().Add(-cacheDuration)) {
		log.Printf("using item %s created at %s from cache\n", index, cached.created.Format(time.RFC3339))
		return cached
	}

	// cache has expired, remove it
	delete(resultsCache, index)
	log.Printf("removing item %s created at %s from cache\n", index, cached.created.Format(time.RFC3339))
	return nil
}

func addToCache(index string, entry cacheEntry) {
	cacheMut.Lock()
	defer cacheMut.Unlock()

	resultsCache[index] = &entry
}

// once an hour, look at the list of things we have cached and remove things that do not need to be there
func backgroundCacheCleanupLoop() {
	for {
		time.Sleep(time.Hour)
		log.Printf("running cache cleanup\n")
		cacheCleanup()
	}
}

func cacheCleanup() {
	cacheMut.Lock()
	defer cacheMut.Unlock()

	for idx, entry := range resultsCache {
		if !entry.created.After(time.Now().Add(-cacheDuration)) {
			log.Printf("removing item %s created at %s from cache\n", idx, entry.created.Format(time.RFC3339))
			delete(resultsCache, idx)
		}
	}
}
