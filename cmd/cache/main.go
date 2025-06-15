package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Vaelatern/temporal-cicd/internal/aerouter"
	"github.com/Vaelatern/temporal-cicd/internal/basicauth"
)

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
	r.HandleFunc("POST /sync/{repo}", c.AdjustRef)
	r.HandleFunc("GET /download/{repo}/{ref}", c.GetTarball)
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	log.Fatal(http.ListenAndServe(":8081", r))
}
