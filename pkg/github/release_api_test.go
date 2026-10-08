package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v60/github"
)

func TestApiRepo_FetchReleases(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/octocat/Hello-World/releases", func(w http.ResponseWriter, r *http.Request) {
		body := `[{
				"tag_name": "v1.0.0",
				"name": "first",
				"body": "hello",
				"html_url": "https://github.com/octocat/Hello-World/releases/tag/v1.0.0",
				"published_at": "2024-01-02T03:04:05Z"
			}]`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := github.NewClient(nil)
	baseURL := srv.URL + "/"
	client.BaseURL, _ = client.BaseURL.Parse(baseURL)

	repo2 := &apiRepo{client: client}
	got, err := repo2.FetchReleases(context.Background(), "https://github.com/octocat/Hello-World")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d, want 1", len(got))
	}
	want := Release{
		TagName: "v1.0.0", Name: "first", Body: "hello",
		URL: "https://github.com/octocat/Hello-World/releases/tag/v1.0.0",
	}
	if got[0].TagName != want.TagName || got[0].Name != want.Name || got[0].Body != want.Body || got[0].URL != want.URL {
		t.Fatalf("got %+v", got[0])
	}
	if got[0].PublishedAt.IsZero() {
		t.Fatalf("PublishedAt should not be zero")
	}
}

func TestApiRepo_FetchReleases_InvalidURL(t *testing.T) {
	repo2 := &apiRepo{client: github.NewClient(nil)}
	_, err := repo2.FetchReleases(context.Background(), "https://gitlab.com/octocat/Hello-World")
	if err == nil {
		t.Fatal("expected error for non-github URL")
	}
	if !strings.Contains(err.Error(), "unsupported host") {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = time.Now
}

func TestNewClient(t *testing.T) {
	t.Run("empty token returns unauthenticated client", func(t *testing.T) {
		r := NewClient("")
		if r == nil {
			t.Fatal("NewClient(\"\") returned nil")
		}
		repo, ok := r.(*apiRepo)
		if !ok {
			t.Fatalf("NewClient(\"\") returned %T, want *apiRepo", r)
		}
		if repo.client == nil {
			t.Fatal("apiRepo.client is nil for empty token")
		}
	})

	t.Run("non-empty token returns authenticated client", func(t *testing.T) {
		r := NewClient("ghp_x")
		if r == nil {
			t.Fatal("NewClient(\"ghp_x\") returned nil")
		}
		repo, ok := r.(*apiRepo)
		if !ok {
			t.Fatalf("NewClient(\"ghp_x\") returned %T, want *apiRepo", r)
		}
		if repo.client == nil {
			t.Fatal("apiRepo.client is nil for non-empty token")
		}
	})
}

func TestApiRepo_FetchReleases_HTTPError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/octocat/Hello-World/releases", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"internal server error"}`, http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := github.NewClient(nil)
	baseURL := srv.URL + "/"
	client.BaseURL, _ = client.BaseURL.Parse(baseURL)

	repo2 := &apiRepo{client: client}
	_, err := repo2.FetchReleases(context.Background(), "https://github.com/octocat/Hello-World")
	if err == nil {
		t.Fatal("expected error from list releases, got nil")
	}
	if !strings.Contains(err.Error(), "list releases") {
		t.Fatalf("expected error wrapping 'list releases', got: %v", err)
	}
}
