package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/sethvargo/go-envconfig"
	"go.temporal.io/sdk/client"

	"github.com/Vaelatern/temporal-cicd/internal/aerouter"
	"github.com/Vaelatern/temporal-cicd/internal/basicauth"
	"github.com/Vaelatern/temporal-cicd/internal/config"
	"github.com/Vaelatern/temporal-cicd/internal/temporal"
)

type KickoffRequest struct {
	Repository   string `json:"repository"`
	Ref          string `json:"ref"`
	BuildPattern string `json:"build-pattern"`
	ApplyPatch   string `json:"compat-patch"`
}

type KickoffWrangler struct {
	temporalClient client.Client
	overrideFS     fs.FS
}

func (k *KickoffRequest) merge(k2 KickoffRequest) {
	if k2.Repository != "" {
		k.Repository = k2.Repository
	}
	if k2.Ref != "" {
		k.Ref = k2.Ref
	}
	if k2.BuildPattern != "" {
		k.BuildPattern = k2.BuildPattern
	}
	if k2.ApplyPatch != "" {
		k.ApplyPatch = k2.ApplyPatch
	}
}

func (k KickoffWrangler) kickoffHandler(w http.ResponseWriter, r *http.Request) {
	repo := r.PathValue("repo")
	ref := r.PathValue("ref")

	var kick KickoffRequest
	if r.Body != nil {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&kick); err != nil {
			log.Println("[json] failed to decode request body: ", err)
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
	}

	urlParams := KickoffRequest{
		Repository: repo,
		Ref:        ref,
	}
	kick.merge(urlParams)

	// Allow overrides (including changing the whole repository we actually mean if we want to play like that)
	if k.overrideFS != nil {
		for _, file := range []string{repo + ".json", fmt.Sprintf("%s.%s.json", repo, ref)} {
			if data, err := fs.ReadFile(k.overrideFS, file); err == nil {
				var repoOverride KickoffRequest
				if err := json.Unmarshal(data, &repoOverride); err == nil {
					kick.merge(repoOverride)
				} else {
					log.Println("[kickoff] failed to parse repo override file: ", file, err)
				}
			}
		}
	}

	if kick.Repository == "" || kick.Ref == "" {
		http.Error(w, "invalid request body, no repository or git ref provided", http.StatusBadRequest)
		return
	}

	opts := client.StartWorkflowOptions{
		TaskQueue: "basic-builder",
		ID:        fmt.Sprintf("!build! %s %s", kick.Repository, kick.Ref),
	}
	_, err := k.temporalClient.ExecuteWorkflow(context.Background(), opts, kick.BuildPattern, kick)
	if err != nil {
		log.Println("[temporal] workflow failed: ", err)
		http.Error(w, "workflow trigger failed", http.StatusBadGateway)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("build triggered"))
	log.Printf("[kickoff] Kicked off %s, %s\n", kick.Repository, kick.Ref)
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

	c, err := temporal.EasyClient(temporal.Logger())
	if err != nil {
		fmt.Printf("Failed to create Temporal client: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	k := KickoffWrangler{
		overrideFS:     os.DirFS(conf.Dir.CustomKickoff),
		temporalClient: c,
	}

	r := aerouter.NewRouter()
	r.Use(auth.AuthMiddleware)

	r.HandleFunc("KICKOFF /{repo}/{ref}", k.kickoffHandler)
	r.HandleFunc("KICKOFF /", k.kickoffHandler)

	log.Printf("[kickoff] Listening on %s\n", conf.Listen)
	log.Fatal(http.ListenAndServe(conf.Listen, r))
}
