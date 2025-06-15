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
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/Vaelatern/temporal-cicd/internal/aerouter"
	"github.com/Vaelatern/temporal-cicd/internal/basicauth"
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
	cmd.Dir = repoPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		http.Error(w, string(out), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("ok"))
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
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		os.MkdirAll(repoPath, 0755)
		cmd := exec.Command("git", "init", "--bare")
		cmd.Dir = repoPath
		if out, err := cmd.CombinedOutput(); err != nil {
			http.Error(w, string(out), 500)
			return
		}
		cmd = exec.Command("git", "remote", "add", "origin", req.URL)
		cmd.Dir = repoPath
		if out, err := cmd.CombinedOutput(); err != nil {
			http.Error(w, string(out), 500)
			return
		}
		cmd = exec.Command("git", "config", "core.sshCommand", "ssh -F /dev/null -i "+sshKeyPath+" -o StrictHostKeyChecking=accept-new")
		cmd.Dir = repoPath
		if out, err := cmd.CombinedOutput(); err != nil {
			http.Error(w, string(out), 500)
			return
		}
	}

	w.Write([]byte("repo registered"))
}

func (s synccache) GetTarball(w http.ResponseWriter, r *http.Request) {
	repo := r.PathValue("repo")
	ref := r.PathValue("ref")

	// Set response headers
	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s-%s.tar.gz", repo, ref))

	// Create git archive command
	cmd := exec.Command("git", "archive", "--format=tar.gz", "origin/"+ref)
	cmd.Dir = filepath.Join(s.fileroot, repo)

	// Get command output pipe
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

func main() {
	auth := basicauth.AuthCore{KeyDir: "../keys"}
	auth.LoadAuth()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1)
	go func() {
		for range sig {
			auth.ReloadAuth()
		}
	}()

	c := synccache{
		fileroot: "./",
		keypath:  "../ssh-keys",
	}

	r := aerouter.NewRouter()

	r.Use(auth.AuthMiddleware)

	r.HandleFunc("POST /sync/{repo}/{ref}", c.SyncRef)
	r.HandleFunc("PUT /sync/{repo}", c.NewRef)
	r.HandleFunc("GET /download/{repo}/{ref}", c.GetTarball)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	log.Fatal(http.ListenAndServe(":8081", r))
}
