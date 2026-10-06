package handler_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/jpillora/installer/handler"
)

type bunReleaseFixtures struct {
	ReleaseCount int                `json:"release_count"`
	Layouts      []bunReleaseLayout `json:"layouts"`
}

type bunReleaseLayout struct {
	Tags     []string          `json:"tags"`
	Assets   []bunReleaseAsset `json:"assets"`
	Expected map[string]string `json:"expected"`
}

type bunReleaseAsset struct {
	Name string `json:"name"`
	Size int    `json:"size"`
}

func (f *bunReleaseFixtures) load(t *testing.T) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("test", "fixtures", "bun-releases.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, f); err != nil {
		t.Fatal(err)
	}
	tags := map[string]bool{}
	for _, layout := range f.Layouts {
		for _, tag := range layout.Tags {
			if tags[tag] {
				t.Fatalf("duplicate release %s", tag)
			}
			tags[tag] = true
		}
	}
	if len(tags) != f.ReleaseCount || f.ReleaseCount != 217 {
		t.Fatalf("expected 217 releases, got %d (declared %d)", len(tags), f.ReleaseCount)
	}
}

type bunReleaseClient struct {
	Tag      string
	Assets   []bunReleaseAsset
	Requests []string
}

func (c *bunReleaseClient) assetURL(name string) string {
	return "https://github.com/oven-sh/bun/releases/download/" + c.Tag + "/" + name
}

func (c *bunReleaseClient) checksum(name string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(name)))
}

func (c *bunReleaseClient) RoundTrip(r *http.Request) (*http.Response, error) {
	c.Requests = append(c.Requests, r.URL.String())
	if r.Method != http.MethodGet {
		return nil, fmt.Errorf("unexpected method %s", r.Method)
	}
	body := ""
	switch r.URL.String() {
	case "https://api.github.com/repos/oven-sh/bun/releases/tags/" + c.Tag,
		"https://api.github.com/repos/oven-sh/bun/releases/latest":
		assets := make([]map[string]any, 0, len(c.Assets))
		for _, asset := range c.Assets {
			assets = append(assets, map[string]any{
				"name":                 asset.Name,
				"size":                 asset.Size,
				"browser_download_url": c.assetURL(asset.Name),
			})
		}
		b, err := json.Marshal(map[string]any{"tag_name": c.Tag, "assets": assets})
		if err != nil {
			return nil, err
		}
		body = string(b)
	case c.assetURL("SHASUMS256.txt"):
		for _, asset := range c.Assets {
			body += c.checksum(asset.Name) + "  " + asset.Name + "\n"
		}
	default:
		return nil, fmt.Errorf("unexpected URL %s", r.URL)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    r,
	}, nil
}

func (c *bunReleaseClient) handler() *handler.Handler {
	return &handler.Handler{Client: &http.Client{Transport: c}}
}

func (c *bunReleaseClient) check(t *testing.T, w *httptest.ResponseRecorder, expected map[string]string) {
	t.Helper()
	if len(expected) == 0 {
		if w.Code != http.StatusInternalServerError || !strings.Contains(w.Body.String(), "no assets found") && !strings.Contains(w.Body.String(), "no downloads found") {
			t.Fatalf("expected release without binaries to fail, got %d: %s", w.Code, w.Body.String())
		}
		return
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result handler.QueryResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.ResolvedRelease != c.Tag || result.User != "oven-sh" || result.Program != "bun" {
		t.Fatalf("unexpected release identity: %#v", result.Query)
	}
	if len(result.Assets) != len(expected) {
		t.Fatalf("expected %d assets, got %d: %#v", len(expected), len(result.Assets), result.Assets)
	}
	_, hasM1 := expected["darwin/arm64"]
	if result.M1Asset != hasM1 {
		t.Fatalf("expected M1Asset=%t, got %t", hasM1, result.M1Asset)
	}
	seen := map[string]bool{}
	hasChecksums := false
	for _, asset := range c.Assets {
		hasChecksums = hasChecksums || asset.Name == "SHASUMS256.txt"
	}
	for i, asset := range result.Assets {
		key := asset.Key()
		name, ok := expected[key]
		if !ok || seen[key] || asset.Name != name {
			t.Fatalf("unexpected asset for %s: %q, want %q", key, asset.Name, name)
		}
		seen[key] = true
		if i > 0 && result.Assets[i-1].Key() >= key {
			t.Fatal("assets are not sorted by platform")
		}
		wantType := ".zip"
		if strings.HasSuffix(name, ".tgz") {
			wantType = ".tgz"
		}
		if asset.URL != c.assetURL(name) || asset.Type != wantType {
			t.Fatalf("unexpected download metadata: %#v", asset)
		}
		wantChecksum := ""
		if hasChecksums {
			wantChecksum = c.checksum(name)
		}
		if asset.SHA256 != wantChecksum {
			t.Fatalf("expected checksum %q for %s, got %q", wantChecksum, name, asset.SHA256)
		}
	}
}

func TestBunAllReleases(t *testing.T) {
	fixtures := bunReleaseFixtures{}
	fixtures.load(t)
	for _, layout := range fixtures.Layouts {
		for _, tag := range layout.Tags {
			t.Run(tag, func(t *testing.T) {
				for _, order := range []string{"published", "reversed", "shuffled"} {
					t.Run(order, func(t *testing.T) {
						assets := slices.Clone(layout.Assets)
						if order == "reversed" {
							slices.Reverse(assets)
						}
						if order == "shuffled" {
							rand.New(rand.NewSource(1)).Shuffle(len(assets), func(i, j int) {
								assets[i], assets[j] = assets[j], assets[i]
							})
						}
						client := &bunReleaseClient{Tag: tag, Assets: assets}
						h := client.handler()
						w := httptest.NewRecorder()
						r := httptest.NewRequest(http.MethodGet, "/oven-sh/bun@"+tag+"?type=json", nil)
						h.ServeHTTP(w, r)
						client.check(t, w, layout.Expected)
						requestCount := len(client.Requests)
						if len(layout.Expected) == 0 {
							return
						}
						w = httptest.NewRecorder()
						h.ServeHTTP(w, r)
						client.check(t, w, layout.Expected)
						if len(client.Requests) != requestCount {
							t.Fatal("cached request fetched the release again")
						}
					})
				}
			})
		}
	}
}

func TestBunSelection(t *testing.T) {
	fixtures := bunReleaseFixtures{}
	fixtures.load(t)
	layout := fixtures.Layouts[0]
	for _, tc := range []struct {
		Name     string
		Target   string
		Expected map[string]string
	}{
		{Name: "latest", Target: "/oven-sh/bun?type=json", Expected: layout.Expected},
		{Name: "explicit_latest", Target: "/oven-sh/bun@latest?type=json", Expected: layout.Expected},
		{Name: "musl", Target: "/oven-sh/bun?type=json&select=musl", Expected: map[string]string{
			"linux/amd64": "bun-linux-x64-musl-baseline.zip", "linux/arm64": "bun-linux-aarch64-musl.zip",
		}},
		{Name: "glibc_baseline", Target: "/oven-sh/bun?type=json&select=linux-x64-baseline.zip", Expected: map[string]string{
			"linux/amd64": "bun-linux-x64-baseline.zip",
		}},
		{Name: "modern_cpu", Target: "/oven-sh/bun?type=json&select=linux-x64.zip", Expected: map[string]string{
			"linux/amd64": "bun-linux-x64.zip",
		}},
		{Name: "profile", Target: "/oven-sh/bun?type=json&select=linux-x64-profile.zip", Expected: map[string]string{
			"linux/amd64": "bun-linux-x64-profile.zip",
		}},
		{Name: "android", Target: "/oven-sh/bun?type=json&select=android", Expected: map[string]string{
			"android/amd64": "bun-linux-x64-android-baseline.zip", "android/arm64": "bun-linux-aarch64-android.zip",
		}},
		{Name: "windows_unsupported", Target: "/oven-sh/bun?type=json&select=windows", Expected: map[string]string{}},
		{Name: "missing_asset", Target: "/oven-sh/bun?type=json&select=does-not-exist", Expected: map[string]string{}},
	} {
		t.Run(tc.Name, func(t *testing.T) {
			client := &bunReleaseClient{Tag: layout.Tags[0], Assets: layout.Assets}
			w := httptest.NewRecorder()
			client.handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.Target, nil))
			client.check(t, w, tc.Expected)
			if client.Requests[0] != "https://api.github.com/repos/oven-sh/bun/releases/latest" {
				t.Fatalf("unexpected latest release URL: %s", client.Requests[0])
			}
		})
	}
}

func TestBunPlatformClients(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires bash")
	}
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Fatal(err)
	}
	fixtures := bunReleaseFixtures{}
	fixtures.load(t)
	bin := t.TempDir()
	for name, contents := range map[string]string{
		"uname":  "#!/bin/sh\ncase \"$1\" in\n-s) printf '%s\\n' \"$BUN_TEST_OS\";;\n-m) printf '%s\\n' \"$BUN_TEST_ARCH\";;\nesac\n",
		"sysctl": "#!/bin/sh\nprintf 'hw.optional.arm64: %s\\n' \"$BUN_TEST_APPLE_SILICON\"\n",
	} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(contents), 0755); err != nil {
			t.Fatal(err)
		}
	}
	for _, layout := range fixtures.Layouts {
		if len(layout.Expected) == 0 {
			continue
		}
		t.Run(layout.Tags[0], func(t *testing.T) {
			client := &bunReleaseClient{Tag: layout.Tags[0], Assets: layout.Assets}
			h := client.handler()
			for _, platform := range []struct {
				Name, OS, Arch, Key, Query, AppleSilicon string
			}{
				{Name: "linux_x64", OS: "Linux", Arch: "x86_64", Key: "linux/amd64"},
				{Name: "linux_arm64", OS: "Linux", Arch: "aarch64", Key: "linux/arm64"},
				{Name: "linux_arm64_alias", OS: "Linux", Arch: "arm64", Key: "linux/arm64"},
				{Name: "linux_arm", OS: "Linux", Arch: "armv7l", Key: "linux/arm"},
				{Name: "linux_386", OS: "Linux", Arch: "i386", Key: "linux/386"},
				{Name: "intel_mac", OS: "Darwin", Arch: "x86_64", Key: "darwin/amd64"},
				{Name: "apple_silicon", OS: "Darwin", Arch: "arm64", Key: "darwin/arm64", AppleSilicon: "1"},
				{Name: "rosetta", OS: "Darwin", Arch: "x86_64", Key: "darwin/arm64", AppleSilicon: "1"},
				{Name: "mac_arm64_uname", OS: "Darwin", Arch: "arm64", Key: "darwin/arm64"},
				{Name: "override_linux_arm64", OS: "Darwin", Arch: "x86_64", Key: "linux/arm64", Query: "os=linux&arch=arm64"},
				{Name: "override_mac_arm64", OS: "Darwin", Arch: "x86_64", Key: "darwin/arm64", Query: "os=darwin&arch=arm64"},
				{Name: "override_freebsd_x64", OS: "FreeBSD", Arch: "amd64", Key: "freebsd/amd64", Query: "os=freebsd&arch=amd64"},
				{Name: "override_freebsd_arm64", OS: "FreeBSD", Arch: "aarch64", Key: "freebsd/arm64", Query: "os=freebsd&arch=arm64"},
				{Name: "override_android_x64", OS: "Linux", Arch: "x86_64", Key: "android/amd64", Query: "os=android&arch=amd64"},
				{Name: "override_android_arm64", OS: "Linux", Arch: "aarch64", Key: "android/arm64", Query: "os=android&arch=arm64"},
				{Name: "override_windows", OS: "Windows", Arch: "x86_64", Key: "windows/amd64", Query: "os=windows&arch=amd64"},
			} {
				t.Run(platform.Name, func(t *testing.T) {
					w := httptest.NewRecorder()
					r := httptest.NewRequest(http.MethodGet, "/oven-sh/bun@"+client.Tag+"?type=script&"+platform.Query, nil)
					r.Header.Set("User-Agent", "curl/8.0")
					h.ServeHTTP(w, r)
					if w.Code != http.StatusOK {
						t.Fatalf("expected shell script, got %d: %s", w.Code, w.Body.String())
					}
					prefix, _, ok := strings.Cut(w.Body.String(), "\t#got URL! download it...")
					if !ok {
						t.Fatal("download boundary missing from shell script")
					}
					cmd := exec.Command(bash)
					cmd.Stdin = strings.NewReader(prefix + "\nprintf 'ASSET=%s TYPE=%s\\n' \"$URL\" \"$FTYPE\"\ncleanup\n}\ninstall\n")
					cmd.Dir = t.TempDir()
					cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "GITHUB_TOKEN=", "DEBUG=", "BUN_TEST_OS="+platform.OS, "BUN_TEST_ARCH="+platform.Arch, "BUN_TEST_APPLE_SILICON="+platform.AppleSilicon)
					out, err := cmd.CombinedOutput()
					key := platform.Key
					if (platform.AppleSilicon == "1" || platform.Name == "mac_arm64_uname") && layout.Expected[key] == "" {
						key = "darwin/amd64"
					}
					name := layout.Expected[key]
					if name == "" {
						if err == nil || !strings.Contains(string(out), "No asset for platform "+strings.ReplaceAll(key, "/", "-")) {
							t.Fatalf("expected unsupported platform %s, got %v: %s", key, err, out)
						}
						return
					}
					if err != nil {
						t.Fatalf("client failed: %v: %s", err, out)
					}
					wantType := ".zip"
					if strings.HasSuffix(name, ".tgz") {
						wantType = ".tgz"
					}
					want := "ASSET=" + client.assetURL(name) + " TYPE=" + wantType + "\n"
					if !strings.Contains(string(out), want) {
						t.Fatalf("expected %s, got %s", want, out)
					}
				})
			}
		})
	}
}
