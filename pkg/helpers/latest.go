package helpers

import (
	"fmt"
	"strings"

	"github.com/google/go-github/v39/github"
)

// LatestRelease returns the latest matching entry in the provided list, or an error if none is present.
// by default, it will not include prerelease verisons, which can be changed by setting includePrerelease.
// if filter is provided, only releases with names not containing the filter will be returned.
func LatestRelease(releases []*github.RepositoryRelease, includePrerelease bool, filter string) (*github.RepositoryRelease, error) {
	for _, release := range releases {
		if filter != "" && strings.Contains(release.GetName(), filter) {
			continue
		}

		if includePrerelease || !release.GetPrerelease() {
			return release, nil
		}
	}
	return nil, fmt.Errorf("no matching entry in list")
}
