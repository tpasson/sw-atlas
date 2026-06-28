package store

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParseSCMURL(t *testing.T) {
	cases := []struct {
		raw                       string
		wantOK                    bool
		owner, repo, kind, ref, p string
	}{
		{"https://github.com/acme/app/pull/42", true, "acme", "app", "pull", "42", "github"},
		{"https://github.com/acme/app/issues/7", true, "acme", "app", "issue", "7", "github"},
		{"https://github.com/acme/app/releases/tag/v1.2.0", true, "acme", "app", "release", "v1.2.0", "github"},
		{"https://gitea.example.com/acme/app/pulls/9", true, "acme", "app", "pull", "9", "gitea"},
		{"https://github.com/acme/app", false, "", "", "", "", ""},
		{"not a url at all", false, "", "", "", "", ""},
	}
	for _, c := range cases {
		cfg, kind, ref, ok := parseSCMURL(c.raw)
		if ok != c.wantOK {
			t.Errorf("%s: ok=%v want %v", c.raw, ok, c.wantOK)
			continue
		}
		if !ok {
			continue
		}
		if cfg.owner != c.owner || cfg.repo != c.repo || kind != c.kind || ref != c.ref || cfg.provider != c.p {
			t.Errorf("%s: got owner=%s repo=%s kind=%s ref=%s provider=%s", c.raw, cfg.owner, cfg.repo, kind, ref, cfg.provider)
		}
	}
}

func TestFetchSCMState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/o/r/pulls/1":
			_, _ = w.Write([]byte(`{"state":"open","merged":true,"merged_at":"2026-01-01T00:00:00Z"}`))
		case "/repos/o/r/pulls/2":
			_, _ = w.Write([]byte(`{"state":"closed","merged":false}`))
		case "/repos/o/r/pulls/3":
			_, _ = w.Write([]byte(`{"state":"open","merged":false}`))
		case "/repos/o/r/issues/4":
			_, _ = w.Write([]byte(`{"state":"closed"}`))
		case "/repos/o/r/releases/tags/v1":
			_, _ = w.Write([]byte(`{"prerelease":false}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	cfg := ghConfig{provider: "github", apiBase: srv.URL, owner: "o", repo: "r"}

	cases := []struct {
		kind, ref string
		want      int
		status    string
	}{
		{"pull", "1", 100, "merged"},
		{"pull", "2", 0, "closed"},
		{"pull", "3", 50, "open"},
		{"issue", "4", 100, "closed"},
		{"release", "v1", 100, "released"},
	}
	for _, c := range cases {
		p, _, status, err := fetchSCMState(context.Background(), cfg, c.kind, c.ref)
		if err != nil {
			t.Fatalf("%s/%s: %v", c.kind, c.ref, err)
		}
		if p == nil || *p != c.want || status != c.status {
			t.Errorf("%s/%s: progress=%v status=%q want %d/%q", c.kind, c.ref, p, status, c.want, c.status)
		}
	}
}
