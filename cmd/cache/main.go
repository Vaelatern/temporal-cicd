package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Vaelatern/temporal-cicd/internal/aerouter"
	"github.com/Vaelatern/temporal-cicd/internal/basicauth"
	"github.com/Vaelatern/temporal-cicd/internal/config"
	"github.com/Vaelatern/temporal-cicd/internal/infoall"
)

func main() {
	conf, err := config.LoadConfig()
	if err != nil {
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

	routes := []string{
		"POST /sync/{repo}/{ref}",
		"PUT /sync/{repo}",
		"POST /sync/{repo}",
		"GET /download/{repo}/{ref}",
		"GET /.vaelcicd/info/all",
	}
	r.HandleFunc("POST /sync/{repo}/{ref}", c.SyncRef)
	r.HandleFunc("PUT /sync/{repo}", c.NewRef)
	r.HandleFunc("POST /sync/{repo}", c.AdjustRef)
	r.HandleFunc("GET /download/{repo}/{ref}", c.GetTarball)
	r.Handle("GET /.vaelcicd/info/all", infoall.Handler{
		Service: "cache",
		Routes:  routes,
		Roots: map[string]string{
			"repos":   conf.Dir.Cache,
			"ssh-keys": conf.Dir.SSHKey,
		},
	})
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	log.Printf("[cache] Listening on %s\n", conf.Listen)
	log.Fatal(http.ListenAndServe(conf.Listen, r))
}
