package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Vaelatern/temporal-cicd/internal/basicauth"
)

type KickoffRequest struct {
	Repository string `json:"repository"`
	Ref        string `json:"ref"`
}

func kickoffHandler(w http.ResponseWriter, r *http.Request) {
	if !basicauth.Authorize(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/kickoff/"), "/")
	if len(parts) != 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	repo := parts[0]
	ref := parts[1]

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
	basicauth.LoadAuth()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1)
	go func() {
		for range sig {
			basicauth.ReloadAuth()
		}
	}()

	http.HandleFunc("/kickoff/", kickoffHandler)
	log.Fatal(http.ListenAndServe(":8082", nil))
}
