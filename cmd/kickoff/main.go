package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go.temporal.io/sdk/client"
	temporal_envconfig "go.temporal.io/sdk/contrib/envconfig"
	temporal_log "go.temporal.io/sdk/log"

	"github.com/Vaelatern/temporal-cicd/internal/aerouter"
	"github.com/Vaelatern/temporal-cicd/internal/basicauth"
	"github.com/Vaelatern/temporal-cicd/internal/config"
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
	sharedSecrets  *config.SharedSecrets
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

func getToken(r *http.Request) string {
	token := r.Header.Get("Authorization")
	if strings.HasPrefix(token, "Bearer ") {
		return strings.TrimPrefix(token, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

// verifySignature computes HMAC-SHA256 of the body using secret and compares
// it (constant-time) to the provided signature string.
// It is only called when a signature header was present from the provider.
// Callers guard the call with `if signature != "" && !verifySignature(...)`.
func verifySignature(body []byte, secret, signature string) bool {
	if signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	sig := strings.TrimPrefix(signature, "sha256=")
	sig = strings.TrimPrefix(sig, "sha1=")
	sig = strings.TrimPrefix(sig, "sha256 ") // some variants
	return hmac.Equal([]byte(expected), []byte(sig))
}

func (k KickoffWrangler) kickoffHandler(w http.ResponseWriter, r *http.Request) {
	repo := r.PathValue("repo")
	ref := r.PathValue("ref")

	var kick KickoffRequest
	if r.Body != nil && r.ContentLength > 0 {
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

	k.triggerWorkflow(kick, w)
}

func (k *KickoffWrangler) triggerWorkflow(kick KickoffRequest, w http.ResponseWriter) {
	// Allow overrides (including changing the whole repository we actually mean if we want to play like that)
	if k.overrideFS != nil {
		for _, file := range []string{kick.Repository + ".json", fmt.Sprintf("%s.%s.json", kick.Repository, kick.Ref)} {
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
	if kick.BuildPattern == "" {
		kick.BuildPattern = "GenericVaelCiCdStart"
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

// webhookHandler is the single entry point for all git provider webhooks.
// Path: POST /hooks/{source}/{repo}?token=...
// The {repo} from the path is the canonical name used in our system (overrides anything in the payload).
// {source} selects the verification + parsing logic (github, gitlab, bitbucket, codeberg).
func (k *KickoffWrangler) webhookHandler(w http.ResponseWriter, r *http.Request) {
	source := r.PathValue("source")
	repo := r.PathValue("repo")

	token := getToken(r)
	if token == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var kick KickoffRequest
	kick.Repository = repo

	secret := ""
	if k.sharedSecrets != nil {
		secret = k.sharedSecrets.Get(source, repo)
	}

	switch source {
	case "github":
		k.handleGitHub(w, r, body, secret, &kick)
	case "gitlab":
		k.handleGitLab(w, r, body, secret, &kick)
	case "bitbucket":
		k.handleBitbucket(w, r, body, secret, &kick)
	case "codeberg":
		k.handleCodeberg(w, r, body, secret, &kick)
	default:
		http.Error(w, "unknown source: "+source, http.StatusBadRequest)
		return
	}

	if kick.Ref != "" {
		k.triggerWorkflow(kick, w)
	}
}

// The following handle* methods contain the provider-specific signature verification
// and ref extraction logic. They are called from webhookHandler.

func (k *KickoffWrangler) handleGitHub(w http.ResponseWriter, r *http.Request, body []byte, secret string, kick *KickoffRequest) {
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		signature = r.Header.Get("X-Hub-Signature")
	}
	if signature != "" && !verifySignature(body, secret, signature) {
		http.Error(w, "invalid signature from origin", http.StatusForbidden)
		return
	}

	event := r.Header.Get("X-GitHub-Event")
	if event != "push" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ignored non-push event"))
		return
	}

	var payload struct {
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
		Ref     string `json:"ref"`
		Deleted bool   `json:"deleted"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if payload.Deleted || payload.Ref == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ref := strings.TrimPrefix(payload.Ref, "refs/heads/")
	ref = strings.TrimPrefix(ref, "refs/tags/")
	kick.Ref = ref
}

func (k *KickoffWrangler) handleGitLab(w http.ResponseWriter, r *http.Request, body []byte, secret string, kick *KickoffRequest) {
	tokenHeader := r.Header.Get("X-Gitlab-Token")
	signature := r.Header.Get("X-Gitlab-Signature")
	if signature == "" {
		signature = r.Header.Get("webhook-signature")
	}
	verified := false
	if tokenHeader != "" && tokenHeader == secret {
		verified = true
	} else if signature != "" && verifySignature(body, secret, signature) {
		verified = true
	}
	if !verified {
		http.Error(w, "invalid token/signature from origin", http.StatusForbidden)
		return
	}

	objectKind := r.Header.Get("X-Gitlab-Event")
	if objectKind != "Push Hook" && objectKind != "push" {
		var check struct{ ObjectKind string `json:"object_kind"` }
		if json.Unmarshal(body, &check) == nil && check.ObjectKind != "push" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ignored non-push event"))
			return
		}
	}

	var payload struct {
		Project struct {
			PathWithNamespace string `json:"path_with_namespace"`
		} `json:"project"`
		Ref string `json:"ref"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if payload.Ref == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ref := strings.TrimPrefix(payload.Ref, "refs/heads/")
	ref = strings.TrimPrefix(ref, "refs/tags/")
	kick.Ref = ref
}

func (k *KickoffWrangler) handleBitbucket(w http.ResponseWriter, r *http.Request, body []byte, secret string, kick *KickoffRequest) {
	signature := r.Header.Get("X-Hub-Signature")
	if signature == "" {
		signature = r.Header.Get("X-Hub-Signature-256")
	}
	if signature != "" && !verifySignature(body, secret, signature) {
		http.Error(w, "invalid signature from origin", http.StatusForbidden)
		return
	}

	eventKey := r.Header.Get("X-Event-Key")
	if eventKey != "repo:push" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ignored non-push event"))
		return
	}

	var payload struct {
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
		Push struct {
			Changes []struct {
				New struct {
					Name string `json:"name"`
					Type string `json:"type"`
				} `json:"new"`
			} `json:"changes"`
		} `json:"push"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if len(payload.Push.Changes) == 0 || payload.Push.Changes[0].New.Name == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ref := payload.Push.Changes[0].New.Name
	ref = strings.TrimPrefix(ref, "refs/heads/")
	kick.Ref = ref
}

func (k *KickoffWrangler) handleCodeberg(w http.ResponseWriter, r *http.Request, body []byte, secret string, kick *KickoffRequest) {
	signature := r.Header.Get("X-Gitea-Signature")
	if signature == "" {
		signature = r.Header.Get("X-Hub-Signature")
		if signature == "" {
			signature = r.Header.Get("X-Hub-Signature-256")
		}
	}
	if signature != "" && !verifySignature(body, secret, signature) {
		http.Error(w, "invalid signature from origin", http.StatusForbidden)
		return
	}

	event := r.Header.Get("X-Gitea-Event")
	if event != "push" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ignored non-push event"))
		return
	}

	var payload struct {
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
		Ref     string `json:"ref"`
		Deleted bool   `json:"deleted"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if payload.Deleted || payload.Ref == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	ref := strings.TrimPrefix(payload.Ref, "refs/heads/")
	ref = strings.TrimPrefix(ref, "refs/tags/")
	kick.Ref = ref
}

func logger() temporal_log.Logger {
	return temporal_log.NewStructuredLogger(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})))
}

func main() {
	conf, err := config.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	auth := basicauth.AuthCore{KeyDir: conf.Dir.Key}
	auth.LoadAuth()

	sharedSecrets := &config.SharedSecrets{Dir: conf.Dir.SharedSecrets}
	sharedSecrets.Load()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGUSR1)
	go func() {
		for range sig {
			auth.ReloadAuth()
			sharedSecrets.Reload()
		}
	}()

	opts := temporal_envconfig.MustLoadDefaultClientOptions()
	opts.Logger = logger()
	c, err := client.Dial(opts)
	if err != nil {
		fmt.Printf("Failed to create Temporal client: %v\n", err)
		os.Exit(1)
	}
	defer c.Close()

	k := KickoffWrangler{
		overrideFS:     os.DirFS(conf.Dir.CustomKickoff),
		temporalClient: c,
		sharedSecrets:  sharedSecrets,
	}

	r := aerouter.NewRouter()
	r.Use(auth.AuthMiddleware)

	r.HandleFunc("KICKOFF /{repo}/{ref}", k.kickoffHandler)
	r.HandleFunc("KICKOFF /", k.kickoffHandler)

	r.HandleFunc("POST /hooks/{source}/{repo}", k.webhookHandler)

	log.Printf("[kickoff] Listening on %s\n", conf.Listen)
	log.Fatal(http.ListenAndServe(conf.Listen, r))
}
