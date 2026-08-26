package maccy

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSearchBuildsAuthenticatedEntriesRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/prefix/v1/entries" {
			t.Errorf("request = %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("authorization = %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("accept = %q", got)
		}
		want := url.Values{"limit": {"20"}, "mode": {"fuzzy"}, "q": {"hello world"}, "cursor": {"next"}}
		if got := r.URL.Query().Encode(); got != want.Encode() {
			t.Errorf("query = %q, want %q", got, want.Encode())
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"entries":[{"id":"entry-1","plain_text":"hello","last_copied_at":"2026-08-25T12:00:00Z","occurrence_count":3}],"next_cursor":"next-2"}`))
	}))
	defer server.Close()

	client, err := New(server.URL+"/prefix/", "test-token")
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.Search(context.Background(), SearchParams{
		Query:  "hello world",
		Mode:   SearchFuzzy,
		Limit:  20,
		Cursor: "next",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 1 || result.Entries[0].PlainText != "hello" || result.Entries[0].OccurrenceCount != 3 {
		t.Fatalf("result = %#v", result)
	}
	if result.NextCursor != "next-2" {
		t.Fatalf("next cursor = %q", result.NextCursor)
	}
}

func TestSearchReportsAPIErrorWithoutLeakingToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
	}))
	defer server.Close()

	client, err := New(server.URL, "super-secret")
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Search(context.Background(), SearchParams{})
	if err == nil || !strings.Contains(err.Error(), "unauthorized") || strings.Contains(err.Error(), "super-secret") {
		t.Fatalf("error = %v", err)
	}
}

func TestNewValidatesClipboardConfiguration(t *testing.T) {
	tests := []struct {
		name  string
		url   string
		token string
		want  string
	}{
		{name: "missing url", token: "secret", want: "base_url"},
		{name: "missing token", url: "http://localhost:8080", want: "token"},
		{name: "unsupported scheme", url: "ftp://localhost:8080", token: "secret", want: "http or https"},
		{name: "query in base url", url: "http://localhost:8080?x=1", token: "secret", want: "query"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.url, tt.token)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestSearchValidatesParametersBeforeRequest(t *testing.T) {
	client, err := New("http://localhost:8080", "secret")
	if err != nil {
		t.Fatal(err)
	}
	for _, params := range []SearchParams{
		{Limit: MaxLimit + 1},
		{Mode: "other"},
		{Mode: SearchFuzzy, Query: "ab"},
	} {
		_, err := client.Search(context.Background(), params)
		if err == nil {
			t.Fatalf("Search(%#v) error = nil", params)
		}
	}
}

func TestSearchPropagatesContextCancellation(t *testing.T) {
	client, err := New("http://localhost:1", "secret")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.Search(ctx, SearchParams{})
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context cancellation", err)
	}
}
