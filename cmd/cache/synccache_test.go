package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRefGetRepo(t *testing.T) {
	root := t.TempDir()
	keys := t.TempDir()
	s := synccache{fileroot: root, keypath: keys}

	keyPath := filepath.Join(t.TempDir(), "k")
	if out, err := exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", keyPath, "-q").CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen: %v %s", err, out)
	}
	priv, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	wantPubBytes, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	wantPub := strings.TrimSpace(string(wantPubBytes))

	body, _ := json.Marshal(RepoRequest{URL: "git@example.com:o/r.git", SSHReadingPrivateKey: string(priv)})
	req := httptest.NewRequest(http.MethodPut, "/sync/demo", strings.NewReader(string(body)))
	req.SetPathValue("repo", "demo")
	rr := httptest.NewRecorder()
	s.NewRef(rr, req)
	if rr.Code != 200 {
		t.Fatalf("put %d %s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/sync/demo", nil)
	req.SetPathValue("repo", "demo")
	rr = httptest.NewRecorder()
	s.GetRepo(rr, req)
	if rr.Code != 200 {
		t.Fatalf("get %d %s", rr.Code, rr.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["url"] != "git@example.com:o/r.git" || got["ssh-reading-public-key"] != wantPub {
		t.Fatalf("got %#v want url+pub %q", got, wantPub)
	}
	repoPath := filepath.Join(root, "demo")
	if gitConfigGet(repoPath, "vaelci.sshKeyPath") == "" || gitConfigGet(repoPath, "vaelci.sshKeyPublic") != wantPub {
		t.Fatalf("vaelci configs missing: path=%q pub=%q", gitConfigGet(repoPath, "vaelci.sshKeyPath"), gitConfigGet(repoPath, "vaelci.sshKeyPublic"))
	}

	// adjust URL + same key
	body, _ = json.Marshal(RepoRequest{URL: "git@example.com:o/other.git", SSHReadingPrivateKey: string(priv)})
	req = httptest.NewRequest(http.MethodPost, "/sync/demo", strings.NewReader(string(body)))
	req.SetPathValue("repo", "demo")
	rr = httptest.NewRecorder()
	s.AdjustRef(rr, req)
	if rr.Code != 200 {
		t.Fatalf("post %d %s", rr.Code, rr.Body.String())
	}
	req = httptest.NewRequest(http.MethodGet, "/sync/demo", nil)
	req.SetPathValue("repo", "demo")
	rr = httptest.NewRecorder()
	s.GetRepo(rr, req)
	_ = json.Unmarshal(rr.Body.Bytes(), &got)
	if got["url"] != "git@example.com:o/other.git" {
		t.Fatalf("url not adjusted: %#v", got)
	}
}

func TestGetDeleteRepo(t *testing.T) {
	root := t.TempDir()
	keys := t.TempDir()
	s := synccache{fileroot: root, keypath: keys}

	name := "demo"
	repoPath := filepath.Join(root, name)
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("git", "init", "--bare")
	cmd.Env = append(os.Environ(), "GIT_DIR="+repoPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("init: %v %s", err, out)
	}
	cmd = exec.Command("git", "remote", "add", "origin", "git@example.com:o/r.git")
	cmd.Env = append(os.Environ(), "GIT_DIR="+repoPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("remote: %v %s", err, out)
	}

	keyPath := filepath.Join(keys, "testkey")
	cmd = exec.Command("ssh-keygen", "-t", "ed25519", "-N", "", "-f", keyPath, "-q")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen: %v %s", err, out)
	}
	wantPubBytes, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	wantPub := strings.TrimSpace(string(wantPubBytes))
	if err := configureRepoSSH(repoPath, keyPath, wantPub); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/sync/"+name, nil)
	req.SetPathValue("repo", name)
	rr := httptest.NewRecorder()
	s.GetRepo(rr, req)
	if rr.Code != 200 {
		t.Fatalf("get status %d %s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["url"] != "git@example.com:o/r.git" {
		t.Fatalf("unexpected body %#v", body)
	}
	if body["ssh-reading-public-key"] != wantPub {
		t.Fatalf("public key mismatch got=%q want=%q", body["ssh-reading-public-key"], wantPub)
	}

	req = httptest.NewRequest(http.MethodDelete, "/sync/"+name, nil)
	req.SetPathValue("repo", name)
	rr = httptest.NewRecorder()
	s.DeleteRepo(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete status %d", rr.Code)
	}
	if isBareRepository(repoPath) {
		t.Fatal("repo still present")
	}

	// idempotent delete + 404 get
	rr = httptest.NewRecorder()
	s.DeleteRepo(rr, req)
	if rr.Code != http.StatusNoContent {
		t.Fatalf("re-delete status %d", rr.Code)
	}
	rr = httptest.NewRecorder()
	s.GetRepo(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("get after delete status %d", rr.Code)
	}
}
