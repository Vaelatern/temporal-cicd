package basicauth

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type TokenRule struct {
	Regexps []*regexp.Regexp
}

type AuthCore struct {
	KeyDir        string
	Debug         bool
	ShowSensitive bool
	authMu        sync.RWMutex
	tokenMap      map[string]TokenRule
}

type yamlConfig map[string][]string

// GetToken extracts the auth token the same way Authorize does:
// Authorization: Bearer <token>, else ?token=.
func GetToken(r *http.Request) string {
	token := r.Header.Get("Authorization")
	if strings.HasPrefix(token, "Bearer ") {
		return strings.TrimPrefix(token, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

func (a *AuthCore) loadTokens(data []byte) error {
	var config yamlConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	for token, regexps := range config {
		var compiled []*regexp.Regexp
		for _, r := range regexps {
			re, err := regexp.Compile(r)
			if err == nil {
				compiled = append(compiled, re)
			} else {
				log.Println("[auth] [regex] failed compiling " + r)
			}
		}
		a.tokenMap[token] = TokenRule{Regexps: compiled}
	}

	return nil
}

func (a *AuthCore) LoadAuth() {
	a.authMu.Lock()
	defer a.authMu.Unlock()
	a.tokenMap = make(map[string]TokenRule)
	dir, err := os.ReadDir(a.KeyDir)
	if err != nil {
		log.Printf("[auth] failed reading key dir %s: %v", a.KeyDir, err)
		return
	}
	for _, f := range dir {
		if f.IsDir() || (!strings.HasSuffix(f.Name(), ".yaml") && !strings.HasSuffix(f.Name(), ".yml")) {
			continue
		}
		path := filepath.Join(a.KeyDir, f.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			log.Printf("[auth] failed reading %s: %v", path, err)
			continue
		}
		err = a.loadTokens(b)
		if err != nil {
			log.Printf("[auth] failed yaml parse on %s: %v", path, err)
			continue
		}
		log.Printf("[auth] loaded %s", path)
	}
	log.Printf("[auth] keys loaded from %s (%d tokens)", a.KeyDir, len(a.tokenMap))
}

func (a *AuthCore) ReloadAuth() {
	a.LoadAuth()
}

// NOT REMOTELY CONSTANT TIME
// Please forgive me. This is just better than fully open.
func (a *AuthCore) Authorize(r *http.Request) bool {
	a.authMu.RLock()
	defer a.authMu.RUnlock()
	token := GetToken(r)
	if token == "" {
		if a.Debug {
			log.Printf("[auth] no token for %s %s", r.Method, r.URL.Path)
		}
		return false
	}
	rule, ok := a.tokenMap[token]
	path := r.Method + " " + r.URL.Path
	if !ok {
		if a.Debug {
			if a.ShowSensitive {
				log.Printf("[auth] unknown token=%q for path=%q", token, path)
			} else {
				log.Printf("[auth] unknown token for path=%q", path)
			}
		}
		return false
	}
	patterns := make([]string, len(rule.Regexps))
	for i, re := range rule.Regexps {
		patterns[i] = re.String()
	}
	if a.Debug {
		if a.ShowSensitive {
			log.Printf("[auth] token=%q path=%q considering patterns=%q", token, path, patterns)
		} else {
			log.Printf("[auth] path=%q considering patterns=%q", path, patterns)
		}
	}
	for _, re := range rule.Regexps {
		if re.MatchString(path) {
			if a.Debug {
				log.Printf("[auth] matched pattern=%q path=%q", re.String(), path)
			}
			return true
		}
	}
	if a.Debug {
		log.Printf("[auth] no pattern matched path=%q", path)
	}
	return false
}

func (a *AuthCore) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.Authorize(r) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
