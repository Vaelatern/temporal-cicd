package config

import (
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// SharedSecrets holds per-source webhook shared secrets (used for signature
// verification). Secrets are loaded from a directory of YAML files or a single
// yaml file. Reload is supported via SIGUSR1.
//
// Config is a YAML list of entries (per-provider + per-repo via regex):
//
//   - provider: github     # optional; omit/empty to match any provider
//     key: AAAAAAAA        # the shared secret value
//     valid-repo: org/.*   # optional regexp; omit to match any repo
//
//   - key: BBBBBBBB        # minimal: matches any provider, any repo
type SharedSecrets struct {
	Dir string

	mu      sync.RWMutex
	secrets []sharedSecretEntry
}

type sharedSecretEntry struct {
	Provider  string `yaml:"provider"`
	Key       string `yaml:"key"`
	ValidRepo string `yaml:"valid-repo"`
}

func (s *SharedSecrets) Load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.secrets = []sharedSecretEntry{}

	if s.Dir == "" {
		return
	}

	info, err := os.Stat(s.Dir)
	if err != nil {
		log.Printf("[shared-secrets] directory not found: %s", s.Dir)
		return
	}

	if !info.IsDir() {
		// treat as single file
		s.loadFile(s.Dir)
		return
	}

	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		log.Printf("[shared-secrets] failed to read dir %s: %v", s.Dir, err)
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}
		path := filepath.Join(s.Dir, name)
		s.loadFile(path)
	}

	log.Printf("[shared-secrets] loaded %d secrets", len(s.secrets))
}

func (s *SharedSecrets) loadFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("[shared-secrets] failed to read %s: %v", path, err)
		return
	}

	var listCfg []sharedSecretEntry
	if err := yaml.Unmarshal(data, &listCfg); err != nil {
		log.Printf("[shared-secrets] failed to parse %s: %v", path, err)
		return
	}
	for _, e := range listCfg {
		if e.Key == "" {
			continue
		}
		prov := strings.ToLower(strings.TrimSpace(e.Provider))
		valid := strings.TrimSpace(e.ValidRepo)
		s.secrets = append(s.secrets, sharedSecretEntry{
			Provider:  prov,
			Key:       e.Key,
			ValidRepo: valid,
		})
	}
}

func (s *SharedSecrets) Reload() {
	s.Load()
}

// Get returns the shared secret for the given provider (e.g. "github", "gitlab")
// and repo name (the canonical repo identifier from the webhook path).
// Matches the first entry where the provider matches (or entry omits provider)
// and the repo matches the valid-repo regexp (defaults to *).
// Returns empty string if no matching secret is configured.
func (s *SharedSecrets) Get(provider, repo string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	provider = strings.ToLower(provider)
	for _, e := range s.secrets {
		if e.Provider != "" && e.Provider != provider {
			continue
		}
		if matchesRepo(e.ValidRepo, repo) {
			return e.Key
		}
	}
	return ""
}

func matchesRepo(pattern, repo string) bool {
	if pattern == "" || pattern == ".*" {
		return true
	}
	if repo == "" {
		return false
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	return re.MatchString(repo)
}
