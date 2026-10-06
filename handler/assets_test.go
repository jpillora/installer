package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestBinaryCandidates(t *testing.T) {
	for _, tc := range []struct {
		Name     string
		Size     int
		Select   string
		Expected bool
	}{
		{Name: "tool-linux-amd64.zip", Size: 900 * 1024, Expected: true},
		{Name: "tool-linux-arm64.tgz", Size: 800 * 1024, Expected: true},
		{Name: "tool-linux-amd64", Size: 2 * 1024 * 1024, Expected: true},
		{Name: "libtree_x86_64", Size: 2 * 1024 * 1024, Expected: true},
		{Name: "tool_x86_64.tgz", Size: 900 * 1024, Expected: true},
		{Name: "tool-linux-amd64.zip", Expected: true},
		{Name: "tool.zip", Expected: true},
		{Name: "tool-cli-1.0.tgz", Size: 3200},
		{Name: "tool-cli-0.0.32.tgz", Size: 3200},
		{Name: "tool-cli-0.0.64.tgz", Size: 3200},
		{Name: "tool-cli-darwin-amd64-1.0.tgz", Size: 210},
		{Name: "tool-darwin-arm64.dSYM.tar.gz", Size: 20 * 1024 * 1024},
		{Name: "tool-linux-amd64-profile.zip", Size: 20 * 1024 * 1024},
		{Name: "tool-linux-amd64_profile.zip", Size: 20 * 1024 * 1024},
		{Name: "tool-linux-amd64-profile.zip", Size: 20 * 1024 * 1024, Select: "musl"},
		{Name: "tool-linux-amd64-profile.zip", Size: 20 * 1024 * 1024, Select: "profile", Expected: true},
		{Name: "profile-linux-amd64.zip", Size: 20 * 1024 * 1024, Expected: true},
	} {
		t.Run(tc.Name+"/"+tc.Select, func(t *testing.T) {
			asset := ghAsset{Name: tc.Name, Size: tc.Size}
			if asset.isBinaryCandidate(tc.Select) != tc.Expected {
				t.Fatalf("expected candidate=%t for %#v with select=%q", tc.Expected, asset, tc.Select)
			}
		})
	}
}

func TestAssetPreference(t *testing.T) {
	for _, tc := range []struct {
		Name      string
		Preferred string
		Other     string
	}{
		{Name: "musl", Preferred: "tool-linux-amd64-musl.zip", Other: "tool-linux-amd64.zip"},
		{Name: "baseline", Preferred: "tool-linux-amd64-baseline.zip", Other: "tool-linux-amd64.zip"},
		{Name: "musl_baseline", Preferred: "tool-linux-amd64-musl-baseline.zip", Other: "tool-linux-amd64-musl.zip"},
		{Name: "gnu", Preferred: "tool-linux-amd64.zip", Other: "tool-linux-amd64-gnu.zip"},
		{Name: "lexical", Preferred: "tool-a-linux-amd64.zip", Other: "tool-z-linux-amd64.zip"},
		{Name: "standalone", Preferred: "tool-linux-amd64", Other: "tool-linux-amd64.zip"},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			preferred := Asset{Name: tc.Preferred}
			other := Asset{Name: tc.Other}
			if !preferred.preferredOver(other) || other.preferredOver(preferred) || preferred.preferredOver(preferred) {
				t.Fatalf("expected %s to win over %s regardless of order", preferred.Name, other.Name)
			}
		})
	}
}

func TestGenericAssetSelection(t *testing.T) {
	for _, tc := range []struct {
		Name     string
		Names    []string
		Select   string
		Expected map[string]string
	}{
		{
			Name: "native_variants",
			Names: []string{
				"tool-linux-amd64.zip", "tool-linux-amd64-musl.zip", "tool-linux-amd64-baseline.zip",
				"tool-linux-amd64-musl-baseline.zip", "tool-linux-amd64-profile.zip",
				"tool-linux-amd64-musl-baseline-profile.zip", "tool-darwin-arm64.zip",
				"tool-darwin-arm64.dSYM.tar.gz", "tool-windows-amd64.zip",
			},
			Expected: map[string]string{"linux/amd64": "tool-linux-amd64-musl-baseline.zip", "darwin/arm64": "tool-darwin-arm64.zip"},
		},
		{
			Name:     "inferred_variants",
			Names:    []string{"tool_linux.zip", "tool_linux_gnu.zip", "tool_linux_musl.zip", "tool_linux_musl-baseline.zip"},
			Expected: map[string]string{"linux/amd64": "tool_linux_musl-baseline.zip"},
		},
		{
			Name:     "explicit_profile",
			Names:    []string{"tool-linux-amd64.zip", "tool-linux-amd64-profile.zip", "tool-linux-amd64-baseline-profile.zip"},
			Select:   "profile",
			Expected: map[string]string{"linux/amd64": "tool-linux-amd64-baseline-profile.zip"},
		},
		{
			Name:     "explicit_modern_cpu",
			Names:    []string{"tool-linux-amd64.zip", "tool-linux-amd64-baseline.zip", "tool-linux-amd64-musl-baseline.zip"},
			Select:   "amd64.zip",
			Expected: map[string]string{"linux/amd64": "tool-linux-amd64.zip"},
		},
		{
			Name:     "equal_variants",
			Names:    []string{"tool-z-linux-amd64.zip", "tool-a-linux-amd64.zip"},
			Expected: map[string]string{"linux/amd64": "tool-a-linux-amd64.zip"},
		},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			for _, reverse := range []bool{false, true} {
				names := slices.Clone(tc.Names)
				if reverse {
					slices.Reverse(names)
				}
				assets := ghAssets{}
				for _, name := range names {
					assets = append(assets, ghAsset{Name: name, Size: 900 * 1024, BrowserDownloadURL: "https://example.com/" + name})
				}
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path != "/repos/acme/tool/releases/tags/v1" {
						t.Errorf("unexpected release path %s", r.URL.Path)
						http.NotFound(w, r)
						return
					}
					json.NewEncoder(w).Encode(ghRelease{TagName: "v1", Assets: assets})
				}))
				h := &Handler{Client: srv.Client(), GHAPI: srv.URL}
				tag, selected, err := h.getAssetsNoCache(Query{User: "acme", Program: "tool", Release: "v1", Select: tc.Select})
				srv.Close()
				if err != nil {
					t.Fatal(err)
				}
				if tag != "v1" || len(selected) != len(tc.Expected) {
					t.Fatalf("unexpected release %s: %#v", tag, selected)
				}
				for _, asset := range selected {
					if expected := tc.Expected[asset.Key()]; asset.Name != expected {
						t.Fatalf("expected %s for %s, got %s", expected, asset.Key(), asset.Name)
					}
				}
			}
		})
	}
}
