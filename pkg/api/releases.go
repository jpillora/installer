package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/google/go-github/v39/github"
	"golang.org/x/oauth2"
)

var client *github.Client
var clientMut sync.Mutex

func InitClient(token string) {
	clientMut.Lock()
	if client == nil {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	}
	return
}

func FetchReleases(ctx context.Context, repo, owner string) ([]*github.RepositoryRelease, error) {
	if client == nil {
		return nil, fmt.Errorf("initialize the client before fetching releases")
	}

	releases, resp, err := client.Repositories.ListReleases(ctx, owner, repo, &github.ListOptions{PerPage: 20})
	if err != nil {
		return nil, fmt.Errorf("unable to list releases for %s/%s: %w", owner, repo, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unable to list releases for %s/%s, got error code %d", owner, repo, resp.StatusCode)
	}

	return releases, nil
}

func FetchLatestRelease(ctx context.Context, repo, owner string) (*github.RepositoryRelease, error) {
	if client == nil {
		return nil, fmt.Errorf("initialize the client before fetching releases")
	}

	release, resp, err := client.Repositories.GetLatestRelease(ctx, repo, owner)
	if err != nil {
		return nil, fmt.Errorf("unable to get latest release for %s/%s: %w", owner, repo, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unable to get latest release for %s/%s, got error code %d", owner, repo, resp.StatusCode)
	}

	return release, nil
}

func FetchSpecificRelease(ctx context.Context, repo, owner, version string) (*github.RepositoryRelease, error) {
	if client == nil {
		return nil, fmt.Errorf("initialize the client before fetching releases")
	}

	release, resp, err := client.Repositories.GetReleaseByTag(ctx, repo, owner, version)
	if err != nil {
		return nil, fmt.Errorf("unable to get release %q for %s/%s: %w", version, owner, repo, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unable to get release %q for %s/%s, got error code %d", version, owner, repo, resp.StatusCode)
	}

	return release, nil
}
