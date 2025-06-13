package main

import (
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Vaelatern/temporal-cicd/internal/basicauth"
)

func handler(w http.ResponseWriter, r *http.Request) {
	if !basicauth.Authorize(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	reposRoot := "/repos"
	switch {
	case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/sync/"):
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/sync/"), "/")
		if len(parts) != 2 {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		repo, ref := parts[0], parts[1]
		namespacePath := filepath.Join(reposRoot, repo)
		cmd := exec.Command("git", "fetch", "origin", ref+":"+ref)
		cmd.Dir = namespacePath
		out, err := cmd.CombinedOutput()
		if err != nil {
			http.Error(w, string(out), 500)
			return
		}
		w.Write([]byte("ok"))

	case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/sync/"):
		// Add repo with SSH key: skipped for brevity
		http.Error(w, "not implemented", 501)

	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/download/"):
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/download/"), "/")
		if len(parts) != 2 {
			http.Error(w, "invalid path", http.StatusBadRequest)
			return
		}
		repo, ref := parts[0], parts[1]
		tarPath := filepath.Join("/tmp", repo+"-"+ref+".tar.gz")
		cmd := exec.Command("git", "archive", "--format=tar.gz", "-o", tarPath, ref)
		cmd.Dir = filepath.Join(reposRoot, repo)
		err := cmd.Run()
		if err != nil {
			http.Error(w, "archive failed", 500)
			return
		}
		f, err := os.Open(tarPath)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}
		defer f.Close()
		http.ServeContent(w, r, filepath.Base(tarPath), time.Now(), f)

	default:
		http.NotFound(w, r)
	}
}

func main() {
	basicauth.LoadAuth()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1)
	go func() {
		for range sig {
			basicauth.ReloadAuth()
		}
	}()

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
