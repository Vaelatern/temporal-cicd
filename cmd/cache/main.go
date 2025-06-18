package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sethvargo/go-envconfig"

	"github.com/Vaelatern/temporal-cicd/internal/aerouter"
	"github.com/Vaelatern/temporal-cicd/internal/basicauth"
	"github.com/Vaelatern/temporal-cicd/internal/config"
)

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

	c := synccache{
		fileroot: conf.Dir.Cache,
		keypath:  conf.Dir.SSHKey,
	}

	r := aerouter.NewRouter()

	r.Use(auth.AuthMiddleware)

	r.HandleFunc("POST /sync/{repo}/{ref}", c.SyncRef)
	r.HandleFunc("PUT /sync/{repo}", c.NewRef)
	r.HandleFunc("POST /sync/{repo}", c.AdjustRef)
	r.HandleFunc("GET /download/{repo}/{ref}", c.GetTarball)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	log.Printf("[cache] Listening on %s\n", conf.Listen)
	log.Fatal(http.ListenAndServe(conf.Listen, r))
}
