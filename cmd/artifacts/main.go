package main

import (
	"context"
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

	"github.com/sethvargo/go-envconfig"

	"github.com/Vaelatern/temporal-cicd/internal/aerouter"
	"github.com/Vaelatern/temporal-cicd/internal/basicauth"
	"github.com/Vaelatern/temporal-cicd/internal/config"
)

type artifactstore struct {
	fileroot string
}

func (a artifactstore) PutArtifact(w http.ResponseWriter, r *http.Request) {
	upath := r.PathValue("path")
	path := filepath.Join(a.fileroot, upath)
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

}

func (a artifactstore) GetArtifact(w http.ResponseWriter, r *http.Request) {
	upath := r.PathValue("path")
	path := filepath.Join(a.fileroot, upath)
	f, err := os.Open(path)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}
	defer f.Close()
	http.ServeContent(w, r, filepath.Base(path), time.Now(), f)
}

func main() {
	var conf config.Config
	if err := envconfig.Process(context.Background(), &conf); err != nil {
		log.Fatal(err)
	}
	auth := basicauth.AuthCore{KeyDir: conf.Dir.Key}
	auth.LoadAuth()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1)
	go func() {
		for range sig {
			auth.ReloadAuth()
		}
	}()

	a := artifactstore{
		fileroot: conf.Dir.RawArtifact,
	}

	r := aerouter.NewRouter()

	r.Use(auth.AuthMiddleware)

	r.HandleFunc("PUT /{path...}", a.PutArtifact)
	r.HandleFunc("GET /{path...}", a.GetArtifact)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	log.Fatal(http.ListenAndServe(conf.Listen, r))
}
