package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientCRUD(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /sync/{repo}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("repo") != "demo" {
			http.NotFound(w, r)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"url": "git@x:y.git", "registered": true, "ssh-reading-public-key": "ssh-ed25519 AAAA"})
	})
	mux.HandleFunc("PUT /sync/{repo}", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("POST /sync/{repo}", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("DELETE /sync/{repo}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := &Client{BaseURL: srv.URL, Token: "t"}
	st, err := c.GetRepo("demo")
	if err != nil || st == nil || st.URL != "git@x:y.git" || st.SSHReadingPublicKey != "ssh-ed25519 AAAA" {
		t.Fatalf("get: %#v %v", st, err)
	}
	if err := c.PutRepo("demo", RepoRequest{URL: "u", SSHReadingPrivateKey: "k"}); err != nil {
		t.Fatal(err)
	}
	if err := c.PostRepo("demo", RepoRequest{URL: "u2", SSHReadingPrivateKey: "k"}); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteRepo("demo"); err != nil {
		t.Fatal(err)
	}

	dry := &Client{BaseURL: "http://127.0.0.1:1", DryRun: true}
	if err := dry.PutRepo("x", RepoRequest{}); err != nil {
		t.Fatal("dry_run should skip network", err)
	}
}
