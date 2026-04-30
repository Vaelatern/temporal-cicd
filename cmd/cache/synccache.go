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

	hasher := sha256.New()
	hasher.Write([]byte(req.SSHReadingPrivateKey))
	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)

	sshKeyPath := filepath.Join(s.keypath, hashString)
	if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
		err = ioutil.WriteFile(sshKeyPath, []byte(req.SSHReadingPrivateKey), 0600)
		if err != nil {
			http.Error(w, "failed to write ssh key", 500)
			return
		}
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
		cmd = exec.Command("git", "config", "core.sshCommand", "ssh -F /dev/null -i "+sshKeyPath+" -o StrictHostKeyChecking=accept-new")
		cmd.Dir = s.fileroot
		cmd.Env = append(cmd.Env, "GIT_DIR="+repoPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			http.Error(w, string(out), 500)
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

	hasher := sha256.New()
	hasher.Write([]byte(req.SSHReadingPrivateKey))
	hashBytes := hasher.Sum(nil)
	hashString := hex.EncodeToString(hashBytes)

	absPath, err := filepath.Abs(s.keypath)
	if err != nil {
		http.Error(w, "failed to get ssh key abspath - bad configuration: "+err.Error(), 500)
		return
	}
	sshKeyPath := filepath.Join(absPath, hashString)
	if _, err := os.Stat(sshKeyPath); os.IsNotExist(err) {
		err = ioutil.WriteFile(sshKeyPath, []byte(req.SSHReadingPrivateKey), 0600)
		if err != nil {
			http.Error(w, "failed to write ssh key", 500)
			return
		}
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
		cmd = exec.Command("git", "config", "core.sshCommand", "ssh -F /dev/null -i "+sshKeyPath+" -o StrictHostKeyChecking=accept-new")
		cmd.Dir = s.fileroot
		cmd.Env = append(cmd.Env, "GIT_DIR="+repoPath)
		if out, err := cmd.CombinedOutput(); err != nil {
			http.Error(w, string(out), 500)
			return
		}
		w.Write([]byte("repo settings adjusted"))
	}
	w.Write([]byte("repo settings consistent"))
}

func (s synccache) GetTarball(w http.ResponseWriter, r *http.Request) {
	repo := r.PathValue("repo")
	ref := r.PathValue("ref")

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s.tgz", repo, ref))

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
