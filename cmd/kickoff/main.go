package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/Vaelatern/temporal-cicd/internal/aerouter"
	"github.com/Vaelatern/temporal-cicd/internal/basicauth"
)

type KickoffRequest struct {
	Repository   string `json:"repository"`
	Ref          string `json:"ref"`
	BuildPattern string `json:"build-pattern"`
	ApplyPatch   string `json:"compat-patch"`
}

func kickoffHandler(w http.ResponseWriter, r *http.Request) {
	repo := r.PathValue("repo")
	ref := r.PathValue("ref")

	kick := KickoffRequest{
		Repository: repo,
		Ref:        ref,
	}

	payload, err := json.Marshal(kick)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp, err := http.Post("http://temporal-worker:8081/workflow/start", "application/json", bytes.NewBuffer(payload))
	if err != nil || resp.StatusCode >= 300 {
		http.Error(w, "workflow trigger failed", http.StatusBadGateway)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("build triggered"))
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

	r := aerouter.NewRouter()

	r.Use(auth.AuthMiddleware)

	r.HandleFunc("KICKOFF /{repo}/{ref}", kickoffHandler)
	log.Fatal(http.ListenAndServe(":8083", r))
}
