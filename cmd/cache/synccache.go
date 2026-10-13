package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type synccache struct {
	fileroot string
	keypath  string
}

type RepoRequest struct {
	URL                  string `json:"url"`
	SSHReadingPrivateKey string `json:"ssh-reading-private-key"`
}

func (s synccache) SyncRef(w http.ResponseWriter, r *http.Request) {
	repo := r.PathValue("repo")
	ref := r.PathValue("ref")
	repoPath := filepath.Join(s.fileroot, repo)
	cmd := exec.Command("git", "fetch", "origin", ref)
	cmd.Dir = s.fileroot
	cmd.Env = append(cmd.Env, "GIT_DIR="+repoPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, string(out), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("ok"))
}

func isBareRepository(path string) bool {
	cmd := exec.Command("git", "rev-parse", "--is-bare-repository")
	cmd.Env = append(cmd.Env, "GIT_DIR="+path)
	if out, err := cmd.CombinedOutput(); err != nil || strings.TrimSpace(string(out)) != "true" {
		return false
	}
	return true
}

func repoWithOriginURL(path string, url string) bool {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Env = append(cmd.Env, "GIT_DIR="+path)
	if out, err := cmd.CombinedOutput(); err != nil || strings.TrimSpace(string(out)) != strings.TrimSpace(url) {
		return false
	}
	return true
}

func gitConfigSet(repoPath, key, value string) error {
	cmd := exec.Command("git", "config", key, value)
	cmd.Env = append(cmd.Env, "GIT_DIR="+repoPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, out)
	}
	return nil
}

func gitConfigGet(repoPath, key string) string {
	cmd := exec.Command("git", "config", "--get", key)
	cmd.Env = append(cmd.Env, "GIT_DIR="+repoPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func originURL(repoPath string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Env = append(cmd.Env, "GIT_DIR="+repoPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

// storeSSHKey content-addresses the private key; returns path + derived public key.
func (s synccache) storeSSHKey(privateKey string) (path, pub string, err error) {
	hasher := sha256.New()
	hasher.Write([]byte(privateKey))
	path = filepath.Join(s.keypath, hex.EncodeToString(hasher.Sum(nil)))
	if _, err := os.Stat(path); os.IsNotExist(err) {
		keyContent := privateKey
		if !strings.HasSuffix(keyContent, "\n") {
			keyContent += "\n"
		}
		if err := ioutil.WriteFile(path, []byte(keyContent), 0600); err != nil {
			return "", "", err
		}
	}
	cmd := exec.Command("ssh-keygen", "-y", "-f", path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return path, "", fmt.Errorf("ssh-keygen -y: %s: %s", err, out)
	}
	return path, strings.TrimSpace(string(out)), nil
}

func configureRepoSSH(repoPath, sshKeyPath, pub string) error {
	if err := gitConfigSet(repoPath, "core.sshCommand", "ssh -F /dev/null -i "+sshKeyPath+" -o StrictHostKeyChecking=accept-new"); err != nil {
		return err
	}
	if err := gitConfigSet(repoPath, "vaelci.sshKeyPath", sshKeyPath); err != nil {
		return err
	}
	return gitConfigSet(repoPath, "vaelci.sshKeyPublic", pub)
}

func (s synccache) NewRef(w http.ResponseWriter, r *http.Request) {
	repo := r.PathValue("repo")

	var req RepoRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, "failed to parse JSON", http.StatusBadRequest)
		return
	}

	sshKeyPath, pub, err := s.storeSSHKey(req.SSHReadingPrivateKey)
	if err != nil {
		http.Error(w, "failed to store ssh key: "+err.Error(), 500)
		return
	}

	repoPath := filepath.Join(s.fileroot, repo)
	if !isBareRepository(repoPath) {
		os.MkdirAll(repoPath, 0755)
		cmd := exec.Command("git", "init", "--bare")
		cmd.Dir = s.fileroot
		cmd.Env = append(cmd.Env, "GIT_DIR="+repoPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			http.Error(w, string(out), 500)
			return
		}
		cmd = exec.Command("git", "remote", "add", "origin", req.URL)
		cmd.Dir = s.fileroot
		cmd.Env = append(cmd.Env, "GIT_DIR="+repoPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			http.Error(w, string(out), 500)
			return
		}
		if err := configureRepoSSH(repoPath, sshKeyPath, pub); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		w.Write([]byte("repo registered"))
	} else {
		w.Write([]byte("repo already registered"))
	}
}

func (s synccache) AdjustRef(w http.ResponseWriter, r *http.Request) {
	repo := r.PathValue("repo")

	var req RepoRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, "failed to parse JSON", http.StatusBadRequest)
		return
	}

	sshKeyPath, pub, err := s.storeSSHKey(req.SSHReadingPrivateKey)
	if err != nil {
		http.Error(w, "failed to store ssh key: "+err.Error(), 500)
		return
	}

	repoPath := filepath.Join(s.fileroot, repo)
	if !repoWithOriginURL(repoPath, req.URL) {
		cmd := exec.Command("git", "remote", "set-url", "origin", req.URL)
		cmd.Dir = s.fileroot
		cmd.Env = append(cmd.Env, "GIT_DIR="+repoPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			http.Error(w, string(out), 500)
			return
		}
	}
	// always refresh ssh config so key-only changes stick
	if err := configureRepoSSH(repoPath, sshKeyPath, pub); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	w.Write([]byte("repo settings adjusted"))
}

// GetRepo returns registration state for Terraform Read.
// Body: {"url","ssh-reading-public-key"}. 404 if not a bare repo.
func (s synccache) GetRepo(w http.ResponseWriter, r *http.Request) {
	repo := r.PathValue("repo")
	repoPath := filepath.Join(s.fileroot, repo)
	if !isBareRepository(repoPath) {
		http.Error(w, "repo not registered", http.StatusNotFound)
		return
	}
	url, err := originURL(repoPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"url":                    url,
		"ssh-reading-public-key": gitConfigGet(repoPath, "vaelci.sshKeyPublic"),
	})
}

// DeleteRepo removes the bare repo dir. SSH keys are content-addressed and shared — left alone.
func (s synccache) DeleteRepo(w http.ResponseWriter, r *http.Request) {
	repo := r.PathValue("repo")
	repoPath := filepath.Join(s.fileroot, repo)
	if !isBareRepository(repoPath) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := os.RemoveAll(repoPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s synccache) GetTarball(w http.ResponseWriter, r *http.Request) {
	repo := r.PathValue("repo")
	ref := r.PathValue("ref")

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-%s.tgz"`, repo, ref))

	repoPath := filepath.Join(s.fileroot, repo)

	cmd := exec.Command("git", "archive", "--format=tar.gz", "origin/"+ref)
	cmd.Dir = s.fileroot
	cmd.Env = append(cmd.Env, "GIT_DIR="+repoPath)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, "failed to get output pipe", http.StatusInternalServerError)
		return
	}
	stderr, _ := cmd.StderrPipe()

	// Start the command
	if err := cmd.Start(); err != nil {
		http.Error(w, "archive failed", http.StatusInternalServerError)
		return
	}

	// Stream the output directly to the response
	_, err = io.Copy(w, stdout)
	if err != nil {
		// We can't send an HTTP error at this point as we've already started streaming
		// Log the error instead
		log.Printf("Error streaming tarball: %v", err)
	}

	// Wait for the command to complete
	if err := cmd.Wait(); err != nil {
		// We can't send an HTTP error at this point as we've already started streaming
		// Log the error instead
		log.Printf("Error completing archive: %v", err)
		io.Copy(os.Stdout, stderr)
	}
}
