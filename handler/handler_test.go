package handler_test

import (
	"net/http/httptest"
	"os/exec"
	"testing"

	"github.com/jpillora/installer/handler"
)

func TestJPilloraServe(t *testing.T) {
	h := &handler.Handler{}
	r := httptest.NewRequest("GET", "/jpillora/serve", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Result().StatusCode != 200 {
		t.Fatalf("failed to get jpillora/serve asset status")
	}
	t.Log(w.Body.String())
}

func TestMicro(t *testing.T) {
	h := &handler.Handler{}
	r := httptest.NewRequest("GET", "/micro", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Result().StatusCode != 200 {
		t.Fatalf("failed to get micro asset status")
	}
	t.Log(w.Body.String())
}

func TestMicroInstall(t *testing.T) {
	h := &handler.Handler{}
	r := httptest.NewRequest("GET", "/micro?type=script", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Result().StatusCode != 200 {
		t.Fatalf("failed to get micro asset status")
	}
	// pipe into bash
	bash := exec.Command("bash")
	bash.Stdin = w.Body
	out, err := bash.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to install micro: %s %s", err, out)
	}
	t.Log(string(out))
}

func TestMicroInstallAs(t *testing.T) {
	h := &handler.Handler{}
	r := httptest.NewRequest("GET", "/micro?type=script&as=mymicro", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Result().StatusCode != 200 {
		t.Fatalf("failed to get micro asset status")
	}
	// pipe into bash
	bash := exec.Command("bash")
	bash.Stdin = w.Body
	out, err := bash.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to install micro as mymicro: %s %s", err, out)
	}
	t.Log(string(out))
}
