package handler_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jpillora/installer/handler"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

func setupRecorder(t *testing.T) (*recorder.Recorder, *http.Client) {
	t.Helper()
	recordMode := os.Getenv("RECORD") == "1"
	cassettePath := filepath.Join("test", "fixtures", strings.ReplaceAll(t.Name(), "/", "_"))
	var opts []recorder.Option
	if recordMode {
		opts = append(opts, recorder.WithMode(recorder.ModeRecordOnly))
	} else {
		opts = append(opts, recorder.WithMode(recorder.ModeReplayOnly))
	}
	r, err := recorder.New(cassettePath, opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Stop(); err != nil {
			t.Error(err)
		}
	})
	client := r.GetDefaultClient()
	return r, client
}

func checkAsset(t *testing.T, w *httptest.ResponseRecorder, osArch, expectedName string) {
	t.Helper()

	// Decode JSON response
	var result handler.QueryResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}

	// Parse OS and arch from osArch parameter
	parts := strings.Split(osArch, "/")
	if len(parts) != 2 {
		t.Fatalf("invalid osArch format %q, expected 'os/arch'", osArch)
	}
	expectedOS, expectedArch := parts[0], parts[1]

	// Find the specified asset
	var targetAsset *handler.Asset
	for _, asset := range result.Assets {
		if asset.OS == expectedOS && asset.Arch == expectedArch {
			targetAsset = &asset
			break
		}
	}

	if targetAsset == nil {
		t.Fatalf("%s asset not found in response", osArch)
	}

	if targetAsset.Name != expectedName {
		t.Fatalf("expected %s asset name %q, got %q", osArch, expectedName, targetAsset.Name)
	}
}

func batchCheckAssets(t *testing.T, w *httptest.ResponseRecorder, assets map[string]string) {
	for osArch, expectedName := range assets {
		checkAsset(t, w, osArch, expectedName)
	}
}

func makeTestRequest(t *testing.T, method, target string) (*httptest.ResponseRecorder, error) {
	_, client := setupRecorder(t)
	h := &handler.Handler{Client: client}

	r := httptest.NewRequest(method, target, nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Result().StatusCode != 200 {
		return nil, fmt.Errorf("failed to get assets status of %s", target)
	}

	return w, nil
}

// musl over GNU
// almost every arches that linux supported
// i686
// arch: long names
func TestUV(t *testing.T) {
	w, err := makeTestRequest(t, "GET", "/astral-sh/uv@0.8.17?type=json")
	if err != nil {
		t.Fatal(err)
	}

	testCases := map[string]string{
		"linux/amd64":   "uv-x86_64-unknown-linux-musl.tar.gz",
		"linux/loong64": "uv-loongarch64-unknown-linux-gnu.tar.gz",
		"linux/ppc64":   "uv-powerpc64-unknown-linux-gnu.tar.gz",
		"linux/ppc64le": "uv-powerpc64le-unknown-linux-gnu.tar.gz",
		"linux/riscv64": "uv-riscv64gc-unknown-linux-gnu.tar.gz",
		"linux/s390x":   "uv-s390x-unknown-linux-gnu.tar.gz",
		"linux/arm":     "uv-arm-unknown-linux-musleabihf.tar.gz",
		"linux/386":     "uv-i686-unknown-linux-musl.tar.gz",
		"linux/arm64":   "uv-aarch64-unknown-linux-musl.tar.gz",
		"darwin/amd64":  "uv-x86_64-apple-darwin.tar.gz",
		"darwin/arm64":  "uv-aarch64-apple-darwin.tar.gz",
		// "windows/amd64": "uv-x86_64-pc-windows-msvc.zip",
		// "windows/arm64": "uv-aarch64-pc-windows-msvc.zip",
		// "windows/386":   "uv-i686-pc-windows-msvc.zip",
	}
	batchCheckAssets(t, w, testCases)
}

// mac
// x86
func TestGitui(t *testing.T) {
	w, err := makeTestRequest(t, "GET", "/gitui-org/gitui@v0.27.0?type=json")
	if err != nil {
		t.Fatal(err)
	}

	testCases := map[string]string{
		"darwin/386":  "gitui-mac-x86.tar.gz",
		"linux/amd64": "gitui-linux-x86_64.tar.gz",
		"linux/arm":   "gitui-linux-arm.tar.gz",
		"linux/arm64": "gitui-linux-aarch64.tar.gz",
	}
	batchCheckAssets(t, w, testCases)
}

// x32, x64, armv6
func TestGitleaks(t *testing.T) {
	w, err := makeTestRequest(t, "GET", "/gitleaks/gitleaks@v8.28.0?type=json")
	if err != nil {
		t.Fatal(err)
	}

	testCases := map[string]string{
		"darwin/amd64": "gitleaks_8.28.0_darwin_x64.tar.gz",
		"darwin/arm64": "gitleaks_8.28.0_darwin_arm64.tar.gz",
		"linux/386":    "gitleaks_8.28.0_linux_x32.tar.gz",
		"linux/amd64":  "gitleaks_8.28.0_linux_x64.tar.gz",
		"linux/arm":    "gitleaks_8.28.0_linux_armv6.tar.gz",
		"linux/arm64":  "gitleaks_8.28.0_linux_arm64.tar.gz",
	}
	batchCheckAssets(t, w, testCases)
}

// macos, dragonflybsd, freebsd, netbsd, openbsd
// no arch, i386, arm
func TestPiknik(t *testing.T) {
	w, err := makeTestRequest(t, "GET", "/jedisct1/piknik@0.10.2?type=json")
	if err != nil {
		t.Fatal(err)
	}

	testCases := map[string]string{
		// no linux/arm64 asset
		"darwin/amd64":    "piknik-macos-0.10.2.tar.gz",
		"dragonfly/amd64": "piknik-dragonflybsd_amd64-0.10.2.tar.gz",
		"freebsd/386":     "piknik-freebsd_i386-0.10.2.tar.gz",
		"freebsd/amd64":   "piknik-freebsd_amd64-0.10.2.tar.gz",
		"linux/386":       "piknik-linux_i386-0.10.2.tar.gz",
		"linux/amd64":     "piknik-linux_x86_64-0.10.2.tar.gz",
		"linux/arm":       "piknik-linux_arm-0.10.2.tar.gz",
		"netbsd/386":      "piknik-netbsd_i386-0.10.2.tar.gz",
		"netbsd/amd64":    "piknik-netbsd_amd64-0.10.2.tar.gz",
		"openbsd/386":     "piknik-openbsd_i386-0.10.2.tar.gz",
		"openbsd/amd64":   "piknik-openbsd_amd64-0.10.2.tar.gz",
	}
	batchCheckAssets(t, w, testCases)
}

// 32-bit
func TestLazygit(t *testing.T) {
	w, err := makeTestRequest(t, "GET", "/jesseduffield/lazygit@v0.55.0?type=json")
	if err != nil {
		t.Fatal(err)
	}

	testCases := map[string]string{
		"darwin/amd64":  "lazygit_0.55.0_darwin_x86_64.tar.gz",
		"darwin/arm64":  "lazygit_0.55.0_darwin_arm64.tar.gz",
		"freebsd/386":   "lazygit_0.55.0_freebsd_32-bit.tar.gz",
		"freebsd/amd64": "lazygit_0.55.0_freebsd_x86_64.tar.gz",
		"freebsd/arm":   "lazygit_0.55.0_freebsd_armv6.tar.gz",
		"freebsd/arm64": "lazygit_0.55.0_freebsd_arm64.tar.gz",
		"linux/386":     "lazygit_0.55.0_linux_32-bit.tar.gz",
		"linux/amd64":   "lazygit_0.55.0_linux_x86_64.tar.gz",
		"linux/arm":     "lazygit_0.55.0_linux_armv6.tar.gz",
		"linux/arm64":   "lazygit_0.55.0_linux_arm64.tar.gz",
	}
	batchCheckAssets(t, w, testCases)
}

// 32bit, 64bit
func TestCroc(t *testing.T) {
	w, err := makeTestRequest(t, "GET", "/schollz/croc@v10.2.4?type=json")
	if err != nil {
		t.Fatal(err)
	}

	testCases := map[string]string{
		"darwin/amd64":    "croc_v10.2.4_macOS-64bit.tar.gz",
		"darwin/arm64":    "croc_v10.2.4_macOS-ARM64.tar.gz",
		"dragonfly/amd64": "croc_v10.2.4_DragonFlyBSD-64bit.tar.gz",
		"freebsd/amd64":   "croc_v10.2.4_FreeBSD-64bit.tar.gz",
		"freebsd/arm64":   "croc_v10.2.4_FreeBSD-ARM64.tar.gz",
		"linux/386":       "croc_v10.2.4_Linux-32bit.tar.gz",
		"linux/amd64":     "croc_v10.2.4_Linux-64bit.tar.gz",
		"linux/arm":       "croc_v10.2.4_Linux-ARM.tar.gz",
		"linux/arm64":     "croc_v10.2.4_Linux-ARM64.tar.gz",
		"netbsd/386":      "croc_v10.2.4_NetBSD-32bit.tar.gz",
		"netbsd/amd64":    "croc_v10.2.4_NetBSD-64bit.tar.gz",
		"netbsd/arm64":    "croc_v10.2.4_NetBSD-ARM64.tar.gz",
		"openbsd/amd64":   "croc_v10.2.4_OpenBSD-64bit.tar.gz",
		"openbsd/arm64":   "croc_v10.2.4_OpenBSD-ARM64.tar.gz",
	}
	batchCheckAssets(t, w, testCases)
}

// 386
func TestJid(t *testing.T) {
	w, err := makeTestRequest(t, "GET", "/simeji/jid@v0.7.6?type=json")
	if err != nil {
		t.Fatal(err)
	}

	testCases := map[string]string{
		// no darwin/arm64
		"darwin/386":    "jid_darwin_386.zip",
		"darwin/amd64":  "jid_darwin_amd64.zip",
		"freebsd/386":   "jid_freebsd_386.zip",
		"freebsd/amd64": "jid_freebsd_amd64.zip",
		"linux/386":     "jid_linux_386.zip",
		"linux/amd64":   "jid_linux_amd64.zip",
		"linux/arm64":   "jid_linux_arm64.zip",
		"netbsd/386":    "jid_netbsd_386.zip",
		"netbsd/amd64":  "jid_netbsd_amd64.zip",
		"openbsd/386":   "jid_openbsd_386.zip",
		"openbsd/amd64": "jid_openbsd_amd64.zip",
	}
	batchCheckAssets(t, w, testCases)
}

// no os, no arch
// armv7
func TestYtDlp(t *testing.T) {
	w, err := makeTestRequest(t, "GET", "/yt-dlp/yt-dlp@2025.09.05?type=json")
	if err != nil {
		t.Fatal(err)
	}

	testCases := map[string]string{
		// not ideal, but good enough
		"darwin/amd64": "yt-dlp_macos",
		"linux/amd64":  "yt-dlp_musllinux.zip",
		"linux/arm":    "yt-dlp_linux_armv7l.zip",
		"linux/arm64":  "yt-dlp_linux_aarch64",
	}
	batchCheckAssets(t, w, testCases)
}

func TestJPilloraServe(t *testing.T) {
	w, err := makeTestRequest(t, "GET", "/jpillora/serve?type=json")
	if err != nil {
		t.Fatal(err)
	}

	checkAsset(t, w, "linux/amd64", "serve_1.9.8_linux_amd64.gz")
}

func TestMicro(t *testing.T) {
	_, err := makeTestRequest(t, "GET", "/micro")
	if err != nil {
		t.Fatal(err)
	}

	// TestMicroDoubleBang
	_, err = makeTestRequest(t, "GET", "/micro!!")
	if err != nil {
		t.Fatal(err)
	}

	var (
		w   *httptest.ResponseRecorder
		out []byte
	)
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("Skipping integration test - set INTEGRATION=1 to run")
	}
	// TestMicroInstall
	if w, err = makeTestRequest(t, "GET", "/micro?type=script"); err != nil {
		t.Fatal(err)
	}
	bash := exec.Command("bash") // pipe into bash
	bash.Stdin = w.Body
	bash.Dir = os.TempDir()
	if out, err = bash.CombinedOutput(); err != nil {
		t.Fatalf("failed to install micro: %s %s", err, out)
	}
	t.Log(string(out))

	// TestMicroInstallAs
	if w, err = makeTestRequest(t, "GET", "/micro?type=script&as=mymicro"); err != nil {
		t.Fatal(err)
	}
	// pipe into bash
	bash = exec.Command("bash")
	bash.Stdin = w.Body
	bash.Dir = os.TempDir()
	if out, err = bash.CombinedOutput(); err != nil {
		t.Fatalf("failed to install micro as mymicro: %s %s", err, out)
	}
	t.Log(string(out))
}

func TestGotty(t *testing.T) {
	_, err := makeTestRequest(t, "GET", "/yudai/gotty@v0.0.12")
	if err != nil {
		t.Fatal(err)
	}
}

// arm32 should be detected as arm (32-bit ARM architecture)
func TestTmuxArm32(t *testing.T) {
	w, err := makeTestRequest(t, "GET", "/jpillora/tmux-static-builds@v3.5i?type=json")
	if err != nil {
		t.Fatal(err)
	}

	checkAsset(t, w, "linux/arm", "tmux.linux-arm32.gz")
}

func TestBuf(t *testing.T) {
	w, err := makeTestRequest(t, "GET", "/bufbuild/buf@v1.60.0?type=json")
	if err != nil {
		t.Fatal(err)
	}

	testCases := map[string]string{
		"darwin/amd64":  "buf-Darwin-x86_64",
		"darwin/arm64":  "buf-Darwin-arm64",
		"linux/amd64":   "buf-Linux-x86_64",
		"linux/arm64":   "buf-Linux-aarch64",
		"linux/arm":     "buf-Linux-armv7",
		"linux/ppc64le": "buf-Linux-ppc64le",
		"linux/riscv64": "buf-Linux-riscv64",
		"linux/s390x":   "buf-Linux-s390x",
	}
	batchCheckAssets(t, w, testCases)
}

func TestProtoc(t *testing.T) {
	w, err := makeTestRequest(t, "GET", "/protocolbuffers/protobuf@v33.1?type=json")
	if err != nil {
		t.Fatal(err)
	}

	testCases := map[string]string{
		"darwin/amd64":  "protoc-33.1-osx-universal_binary.zip",
		"darwin/arm64":  "protoc-33.1-osx-aarch_64.zip",
		"linux/amd64":   "protoc-33.1-linux-x86_64.zip",
		"linux/386":     "protoc-33.1-linux-x86_32.zip",
		"linux/arm64":   "protoc-33.1-linux-aarch_64.zip",
		"linux/ppc64le": "protoc-33.1-linux-ppcle_64.zip",
		"linux/s390x":   "protoc-33.1-linux-s390_64.zip",
	}
	batchCheckAssets(t, w, testCases)
}
