package store

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGhNextLink(t *testing.T) {
	cases := []struct{ in, want string }{
		{``, ``},
		{`<https://api.github.com/x?page=3>; rel="next"`, `https://api.github.com/x?page=3`},
		{`<https://api.github.com/x?page=2>; rel="next", <https://api.github.com/x?page=5>; rel="last"`, `https://api.github.com/x?page=2`},
		{`<https://api.github.com/x?page=1>; rel="prev", <https://api.github.com/x?page=1>; rel="first"`, ``},
		{`garbage`, ``},
	}
	for _, c := range cases {
		if got := ghNextLink(c.in); got != c.want {
			t.Errorf("ghNextLink(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// ghGetAll must follow rel="next" links and concatenate every page.
func TestGhGetAllPaginates(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("page") {
		case "", "1":
			w.Header().Set("Link", fmt.Sprintf(`<%s/items?page=2>; rel="next"`, srv.URL))
			_, _ = w.Write([]byte(`[1,2]`))
		case "2":
			w.Header().Set("Link", fmt.Sprintf(`<%s/items?page=3>; rel="next"`, srv.URL))
			_, _ = w.Write([]byte(`[3,4]`))
		default:
			_, _ = w.Write([]byte(`[5,6]`)) // no Link header ⇒ last page
		}
	}))
	defer srv.Close()

	cfg := ghConfig{provider: "github", apiBase: srv.URL}
	got, err := ghGetAll[int](context.Background(), cfg, "/items?per_page=2")
	if err != nil {
		t.Fatalf("ghGetAll: %v", err)
	}
	want := []int{1, 2, 3, 4, 5, 6}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// ghProbe must send If-None-Match and treat 304 as unchanged, 200 as changed.
func TestGhProbe(t *testing.T) {
	const etag = `"abc123"`
	var lastINM string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastINM = r.Header.Get("If-None-Match")
		if lastINM == etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", etag)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()
	cfg := ghConfig{provider: "github", apiBase: srv.URL}

	// No stored etag ⇒ changed, no If-None-Match sent, fresh etag returned.
	if changed, et := ghProbe(context.Background(), cfg, "/x", ""); !changed || et != etag {
		t.Fatalf("first probe: changed=%v etag=%q", changed, et)
	}
	if lastINM != "" {
		t.Fatalf("expected no If-None-Match on first probe, got %q", lastINM)
	}
	// With the etag ⇒ 304 ⇒ unchanged, etag preserved.
	if changed, et := ghProbe(context.Background(), cfg, "/x", etag); changed || et != etag {
		t.Fatalf("second probe: changed=%v etag=%q", changed, et)
	}
	if lastINM != etag {
		t.Fatalf("expected If-None-Match %q, got %q", etag, lastINM)
	}
}

// A server that always advertises a next page must be bounded by ghPageCap.
func TestGhGetAllPageCap(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", fmt.Sprintf(`<%s/x?p=next>; rel="next"`, srv.URL))
		_, _ = w.Write([]byte(`[0]`))
	}))
	defer srv.Close()

	cfg := ghConfig{provider: "github", apiBase: srv.URL}
	got, err := ghGetAll[int](context.Background(), cfg, "/x")
	if err != nil {
		t.Fatalf("ghGetAll: %v", err)
	}
	if len(got) != ghPageCap {
		t.Fatalf("expected page cap %d items, got %d", ghPageCap, len(got))
	}
}
