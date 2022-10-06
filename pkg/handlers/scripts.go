package handlers

import (
	"bytes"
	_ "embed"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"text/template"

	"github.com/google/go-github/v39/github"

	"github.com/gorilla/mux"
	"github.com/replicatedhq/installer/pkg/api"
	"github.com/replicatedhq/installer/pkg/helpers"
)

//go:embed scripts/install.sh
var installTemplate string

var (
	fileExtRe = regexp.MustCompile(`(\.[a-z][a-z0-9]+)+$`)
	posixOSRe = regexp.MustCompile(`(darwin|linux|(net|free|open)bsd|mac|osx)`)
	archRe    = regexp.MustCompile(`(arm|386|amd64|32|64)`)
)

type Asset struct {
	Name, OS, Arch, URL, Type string
	Is32bit, IsMac            bool
}

type TemplateStruct struct {
	User, Program, Release string
	MoveToPath, Insecure   bool
	Assets                 []Asset
}

func parseRelease(release *github.RepositoryRelease, owner, project string) (TemplateStruct, error) {
	if release == nil {
		return TemplateStruct{}, fmt.Errorf("no release provided")
	}

	toReturn := TemplateStruct{
		User:       owner,
		Program:    project,
		Release:    release.GetName(),
		MoveToPath: true,
		Assets:     []Asset{},
	}

	for _, asset := range release.Assets {
		url := asset.GetBrowserDownloadURL()
		os := posixOSRe.FindString(asset.GetName())
		arch := archRe.FindString(asset.GetName())
		if os == "" {
			continue //unknown os
		}
		if os == "mac" || os == "osx" {
			os = "darwin"
		}
		if arch == "64" || arch == "" {
			arch = "amd64" //default
		} else if arch == "32" {
			arch = "386"
		}
		if strings.Contains(asset.GetName(), "kots.so") {
			continue
		}
		toReturn.Assets = append(toReturn.Assets, Asset{
			Name:    asset.GetName(),
			OS:      os,
			IsMac:   os == "darwin",
			Arch:    arch,
			Is32bit: arch == "386",
			URL:     url,
			Type:    fileExtRe.FindString(url),
		})
	}

	return toReturn, nil
}

func InstallScript(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	owner := vars["owner"]
	project := vars["project"]
	if owner == "" {
		owner = "replicatedhq"
	}

	specifiedVersion := r.URL.Query().Get("version")     // if this is set, get the release with this tag
	includePrerelease := r.URL.Query().Get("prerelease") // set to 'true' to include prerelease releases
	filter := r.URL.Query().Get("filter")                // if this is set, get the latest release with a name containing this substring

	var desiredRelease *github.RepositoryRelease
	var err error

	if specifiedVersion != "" { // get the specified release
		desiredRelease, err = api.FetchSpecificRelease(r.Context(), owner, project, specifiedVersion)
		if err != nil {
			fmt.Printf("Got error %q", err.Error())
			http.Error(w, fmt.Sprintf("Unable to get release %q: %s", specifiedVersion, err.Error()), http.StatusInternalServerError)
			return
		}
	} else if filter == "" && includePrerelease != "true" { // get the latest release
		desiredRelease, err = api.FetchLatestRelease(r.Context(), owner, project)
		if err != nil {
			fmt.Printf("Got error %q", err.Error())
			http.Error(w, fmt.Sprintf("Unable to get latest release: %s", err.Error()), http.StatusInternalServerError)
			return
		}
	} else { // get the list of releases, and find the best one from that list
		releases, err := api.FetchReleases(r.Context(), owner, project)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			fmt.Printf("Got error %q", err.Error())
			return
		}

		includePrerelease := includePrerelease == "true"

		// figure out which release is the latest matching the criteria
		desiredRelease, err = helpers.LatestRelease(releases, includePrerelease, filter)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Got error %q for owner %q project %q", err.Error(), owner, project)
			return
		}
	}

	// parse the release object to get the info we need (asset URLs for various architectures, mainly)
	templateStruct, err := parseRelease(desiredRelease, owner, project)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Printf("Got error %q for owner %q project %q", err.Error(), owner, project)
		return
	}

	t, err := template.New("installer").Parse(installTemplate)
	if err != nil {
		http.Error(w, "Installer script invalid", http.StatusInternalServerError)
		return
	}

	//create script
	buff := bytes.Buffer{}
	if err := t.Execute(&buff, templateStruct); err != nil {
		http.Error(w, "template invalid", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/x-shellscript")
	w.WriteHeader(http.StatusOK)
	w.Write(buff.Bytes())
}

// GetLatestVersion returns the name of the latest version of the specified project
func GetLatestVersion(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	owner := vars["owner"]
	project := vars["project"]
	if owner == "" {
		owner = "replicatedhq"
	}

	includePrerelease := r.URL.Query().Get("prerelease") // set to 'true' to include prerelease releases
	filter := r.URL.Query().Get("filter")                // if this is set, get the latest release with a name containing this substring

	var desiredRelease *github.RepositoryRelease
	var err error

	if filter == "" && includePrerelease != "true" { // get the latest release
		desiredRelease, err = api.FetchLatestRelease(r.Context(), owner, project)
		if err != nil {
			fmt.Printf("Got error %q", err.Error())
			http.Error(w, fmt.Sprintf("Unable to get latest release: %s", err.Error()), http.StatusInternalServerError)
			return
		}
	} else { // get the list of releases, and find the best one from that list
		releases, err := api.FetchReleases(r.Context(), owner, project)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			fmt.Printf("Got error %q", err.Error())
			return
		}

		includePrerelease := includePrerelease == "true"

		// figure out which release is the latest matching the criteria
		desiredRelease, err = helpers.LatestRelease(releases, includePrerelease, filter)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Printf("Got error %q for owner %q project %q", err.Error(), owner, project)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(desiredRelease.GetName()))
}
