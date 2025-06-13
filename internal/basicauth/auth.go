package basicauth

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type TokenRule struct {
	Regexps []*regexp.Regexp
}

var (
	authMu   sync.RWMutex
	tokenMap map[string]TokenRule
)

func LoadAuth() {
	authMu.Lock()
	defer authMu.Unlock()
	tokenMap = make(map[string]TokenRule)
	dir, _ := os.ReadDir("/keys.d")
	for _, f := range dir {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".yaml") {
			continue
		}
		b, err := os.ReadFile(filepath.Join("/keys.d", f.Name()))
		if err != nil {
			continue
		}
		lines := strings.Split(string(b), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			token := strings.TrimSpace(parts[0])
			regexps := strings.Split(parts[1], ",")
			var compiled []*regexp.Regexp
			for _, r := range regexps {
				re, err := regexp.Compile(strings.TrimSpace(r))
				if err == nil {
					compiled = append(compiled, re)
				}
			}
			tokenMap[token] = TokenRule{Regexps: compiled}
		}
	}
	log.Println("[auth] keys loaded")
}

func ReloadAuth() {
	LoadAuth()
}

func Authorize(r *http.Request) bool {
	authMu.RLock()
	defer authMu.RUnlock()
	token := r.Header.Get("Authorization")
	if !strings.HasPrefix(token, "Bearer ") {
		return false
	}
	token = strings.TrimPrefix(token, "Bearer ")
	rule, ok := tokenMap[token]
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
