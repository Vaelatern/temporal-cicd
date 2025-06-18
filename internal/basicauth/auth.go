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
	KeyDir   string
	authMu   sync.RWMutex
	tokenMap map[string]TokenRule
}

type yamlConfig map[string][]string

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
	dir, _ := os.ReadDir(a.KeyDir)
	for _, f := range dir {
		if f.IsDir() || (!strings.HasSuffix(f.Name(), ".yaml") && !strings.HasSuffix(f.Name(), ".yml")) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(a.KeyDir, f.Name()))
		if err != nil {
			log.Println("[auth] failed reading " + f.Name())
			continue
		}
		err = a.loadTokens(b)
		if err != nil {
			log.Println("[auth] failed yaml parse on " + f.Name())
			continue
		}
		log.Println("[auth] loaded a file")
	}
	log.Println("[auth] keys loaded")
}

func (a *AuthCore) ReloadAuth() {
	a.LoadAuth()
}

// NOT REMOTELY CONSTANT TIME
// Please forgive me. This is just better than fully open.
func (a *AuthCore) Authorize(r *http.Request) bool {
	a.authMu.RLock()
	defer a.authMu.RUnlock()
	token := r.Header.Get("Authorization")
	if !strings.HasPrefix(token, "Bearer ") {
		return false
	}
	token = strings.TrimPrefix(token, "Bearer ")
	rule, ok := a.tokenMap[token]
	if !ok {
		return false
	}
	path := r.Method + " " + r.URL.Path
	for _, re := range rule.Regexps {
		if re.MatchString(path) {
			return true
		}
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
