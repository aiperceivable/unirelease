package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	gh "github.com/google/go-github/v60/github"
)

// newTestClient creates a Client that talks to the given httptest.Server.
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	ghClient := gh.NewClient(nil)
	url := server.URL + "/"
	ghClient.BaseURL, _ = ghClient.BaseURL.Parse(url)
	ghClient.UploadURL, _ = ghClient.UploadURL.Parse(url)
	return NewClientWithGH(ghClient, "owner/repo")
}

// --- ReleaseExists ---

func TestReleaseExists_True(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/releases/tags/v1.0.0" {
			json.NewEncoder(w).Encode(gh.RepositoryRelease{
				ID:      gh.Int64(123),
				TagName: gh.String("v1.0.0"),
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := newTestClient(t, server)
	exists, err := client.ReleaseExists("v1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected release to exist")
	}
}

func TestReleaseExists_False(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	exists, err := client.ReleaseExists("v9.9.9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected release to not exist")
	}
}

// --- CreateRelease ---

func TestCreateRelease_Success(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/repos/owner/repo/releases" {
			data, _ := io.ReadAll(r.Body)
			json.Unmarshal(data, &receivedBody)

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(gh.RepositoryRelease{
				ID:      gh.Int64(456),
				TagName: gh.String("v2.0.0"),
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := newTestClient(t, server)
	id, err := client.CreateRelease("v2.0.0", "Release 2.0.0", "## Changes\n- Feature X")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != 456 {
		t.Errorf("expected release ID 456, got %d", id)
	}
	if receivedBody["tag_name"] != "v2.0.0" {
		t.Errorf("expected tag_name v2.0.0, got %v", receivedBody["tag_name"])
	}
	if receivedBody["body"] != "## Changes\n- Feature X" {
		t.Errorf("expected body with changelog, got %v", receivedBody["body"])
	}
}

func TestCreateRelease_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"message":"Internal Server Error"}`)
	}))
	defer server.Close()

	client := newTestClient(t, server)
	_, err := client.CreateRelease("v1.0.0", "Release", "body")
	if err == nil {
		t.Fatal("expected error for server error")
	}
}

// --- UploadAsset ---

func TestUploadAsset_Success(t *testing.T) {
	var receivedName string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/owner/repo/releases/123/assets" {
			receivedName = r.URL.Query().Get("name")
			receivedBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(gh.ReleaseAsset{
				ID:   gh.Int64(789),
				Name: gh.String(receivedName),
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	// Create a temp file to upload
	dir := t.TempDir()
	assetPath := filepath.Join(dir, "myapp-linux-amd64")
	os.WriteFile(assetPath, []byte("binary-content"), 0755)

	client := newTestClient(t, server)
	err := client.UploadAsset(123, assetPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedName != "myapp-linux-amd64" {
		t.Errorf("expected asset name 'myapp-linux-amd64', got %q", receivedName)
	}
	if string(receivedBody) != "binary-content" {
		t.Errorf("expected body 'binary-content', got %q", string(receivedBody))
	}
}

func TestUploadAsset_FileNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	client := newTestClient(t, server)
	err := client.UploadAsset(123, "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// --- ResolveToken ---

func TestResolveToken_FromGITHUB_TOKEN(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "test-gh-token-123")
	t.Setenv("GH_TOKEN", "") // ensure lower priority doesn't interfere

	token, source, err := ResolveToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "test-gh-token-123" {
		t.Errorf("expected token 'test-gh-token-123', got %q", token)
	}
	if source != "GITHUB_TOKEN environment variable" {
		t.Errorf("expected source 'GITHUB_TOKEN environment variable', got %q", source)
	}
}

func TestResolveToken_FromGH_TOKEN(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "test-gh-alt-token")

	token, source, err := ResolveToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "test-gh-alt-token" {
		t.Errorf("expected token 'test-gh-alt-token', got %q", token)
	}
	if source != "GH_TOKEN environment variable" {
		t.Errorf("expected source 'GH_TOKEN environment variable', got %q", source)
	}
}

func TestResolveToken_Priority(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "primary")
	t.Setenv("GH_TOKEN", "secondary")

	token, _, err := ResolveToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "primary" {
		t.Errorf("expected GITHUB_TOKEN to take priority, got %q", token)
	}
}

// --- owner / repoName ---

func TestOwnerAndRepoName(t *testing.T) {
	c := &Client{Repo: "myorg/myrepo"}
	if c.owner() != "myorg" {
		t.Errorf("expected 'myorg', got %q", c.owner())
	}
	if c.repoName() != "myrepo" {
		t.Errorf("expected 'myrepo', got %q", c.repoName())
	}
}

func TestRepoName_NoSlash(t *testing.T) {
	c := &Client{Repo: "noslash"}
	if c.repoName() != "" {
		t.Errorf("expected empty string, got %q", c.repoName())
	}
}
