package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type searchTestTransport struct {
	respond func(*http.Request) (*http.Response, error)
}

func (s searchTestTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	return s.respond(r)
}

func TestCaptureRepoLocation(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	for _, tc := range []struct {
		location string
		user     string
		program  string
	}{
		{"https://github.com/jpillora/installer", "jpillora", "installer"},
		{"https://github.com/astral-sh/uv", "astral-sh", "uv"},
		{"https://github.com/yt-dlp/yt-dlp/releases", "yt-dlp", "yt-dlp"},
		{"https://github.com/owner.name/repo.name?tab=readme", "owner.name", "repo.name"},
		{"https://github.com/owner_name/repo-name.v2#readme", "owner_name", "repo-name.v2"},
		{"https://github.com/owner", "", ""},
		{"https://example.com/owner/repo", "", ""},
		{"https://example.com/https://github.com/owner/repo", "", ""},
	} {
		t.Run(tc.location, func(t *testing.T) {
			http.DefaultTransport = searchTestTransport{respond: func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusFound,
					Header:     http.Header{"Location": []string{tc.location}},
					Body:       io.NopCloser(strings.NewReader("")),
				}, nil
			}}
			user, program, err := captureRepoLocation("https://search.example.com")
			if tc.user == "" {
				if err == nil {
					t.Fatal("expected invalid redirect to fail")
				}
				return
			}
			if err != nil || user != tc.user || program != tc.program {
				t.Fatalf("got %q/%q, %v; want %q/%q", user, program, err, tc.user, tc.program)
			}
		})
	}
}

func TestHTTPErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	for _, tc := range []struct {
		path string
		code int
		body string
	}{
		{"//?type=text", http.StatusBadRequest, "Invalid path\n"},
		{"//?type=script", http.StatusBadRequest, "echo 'Invalid path'\n"},
		{"/owner/repo?type=text", http.StatusBadGateway, "Service Unavailable upstream unavailable\n"},
		{"/owner/repo?type=script", http.StatusBadGateway, "echo 'Service Unavailable upstream unavailable'\n"},
		{"/owner/repo?type=unknown", http.StatusInternalServerError, "Unknown type\n"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			h := &Handler{Client: srv.Client(), GHAPI: srv.URL}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if w.Code != tc.code || w.Body.String() != tc.body {
				t.Fatalf("got %d %q; want %d %q", w.Code, w.Body.String(), tc.code, tc.body)
			}
		})
	}
}

func TestFailedSearchError(t *testing.T) {
	original := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = original })
	var searches []string
	http.DefaultTransport = searchTestTransport{respond: func(r *http.Request) (*http.Response, error) {
		searches = append(searches, r.URL.Host)
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("no results")),
		}, nil
	}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	h := &Handler{Client: srv.Client(), GHAPI: srv.URL}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/sharkbat?type=text", nil))
	if w.Code != http.StatusBadGateway || w.Body.String() != "web search for sharkbat failed: not found\n" {
		t.Fatalf("got %d %q", w.Code, w.Body.String())
	}
	if len(searches) != 2 || searches[0] != "html.duckduckgo.com" || searches[1] != "www.google.com" {
		t.Fatalf("unexpected search attempts: %v", searches)
	}
	if len(h.cache) != 0 {
		t.Fatal("failed search was cached")
	}
}
