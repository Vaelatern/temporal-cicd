package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Vaelatern/temporal-cicd/internal/basicauth"
)

func handler(w http.ResponseWriter, r *http.Request) {
	if !basicauth.Authorize(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	root := "/artifacts"
	switch r.Method {
	case http.MethodPut:
		path := filepath.Join(root, r.URL.Path)
		os.MkdirAll(filepath.Dir(path), 0755)
		f, err := os.Create(path)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer f.Close()
		h := sha256.New()
		mw := io.MultiWriter(f, h)
		io.Copy(mw, r.Body)
		hash := hex.EncodeToString(h.Sum(nil))
		log.Printf("[upload] path=%s hash=%s", path, hash)
		w.Write([]byte(hash))

	case http.MethodGet:
		path := filepath.Join(root, r.URL.Path)
		f, err := os.Open(path)
		if err != nil {
			http.Error(w, "not found", 404)
			return
		}
		defer f.Close()
		http.ServeContent(w, r, filepath.Base(path), time.Now(), f)

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
	log.Fatal(http.ListenAndServe(":8081", nil))
}
