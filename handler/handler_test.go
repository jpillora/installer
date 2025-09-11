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

func TestUV(t *testing.T) {
	w, err := makeTestRequest(t, "GET", "/astral-sh/uv?type=json")
	if err != nil {
		t.Fatal(err)
	}

	checkAsset(t, w, "linux/amd64", "uv-x86_64-unknown-linux-musl.tar.gz")
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
