package basicauth

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestLoadTokens(t *testing.T) {
	// Create a temporary YAML file content
	yamlContent := []byte(`
token1:
  - ^GET /api/.*
  - ^POST /api/users$
token2:
  - ^GET /public/.*
`)

	auth := &AuthCore{
		tokenMap: make(map[string]TokenRule),
	}

	// Test successful YAML parsing and regex compilation
	err := auth.loadTokens(yamlContent)
	if err != nil {
		t.Fatalf("loadTokens failed: %v", err)
	}

	// Verify tokenMap contents
	if len(auth.tokenMap) != 2 {
		t.Errorf("Expected 2 tokens, got %d", len(auth.tokenMap))
	}

	// Check token1
	if rule, ok := auth.tokenMap["token1"]; !ok {
		t.Error("Expected token1 in tokenMap")
	} else if len(rule.Regexps) != 2 {
		t.Errorf("Expected 2 regexps for token1, got %d", len(rule.Regexps))
	}

	// Check token2
	if rule, ok := auth.tokenMap["token2"]; !ok {
		t.Error("Expected token2 in tokenMap")
	} else if len(rule.Regexps) != 1 {
		t.Errorf("Expected 1 regexp for token2, got %d", len(rule.Regexps))
	}

	// Test invalid YAML
	invalidYAML := []byte(`invalid: yaml: content`)
	err = auth.loadTokens(invalidYAML)
	if err == nil {
		t.Error("Expected error for invalid YAML, got none")
	}
}

func TestLoadAuth(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := ioutil.TempDir("", "auth_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a sample YAML file
	yamlContent := `
token1:
  - ^GET /api/.*
  - ^POST /api/users$
`
	yamlFile := filepath.Join(tmpDir, "test.yaml")
	err = ioutil.WriteFile(yamlFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}

	// Create AuthCore instance
	auth := &AuthCore{
		KeyDir:   tmpDir,
		tokenMap: make(map[string]TokenRule),
	}

	// Test LoadAuth
	auth.LoadAuth()

	// Verify tokenMap
	if len(auth.tokenMap) != 1 {
		t.Errorf("Expected 1 token, got %d", len(auth.tokenMap))
	}

	if rule, ok := auth.tokenMap["token1"]; !ok {
		t.Error("Expected token1 in tokenMap")
	} else if len(rule.Regexps) != 2 {
		t.Errorf("Expected 2 regexps for token1, got %d", len(rule.Regexps))
	}

	// Test with invalid YAML file
	invalidYAMLFile := filepath.Join(tmpDir, "invalid.yaml")
	err = ioutil.WriteFile(invalidYAMLFile, []byte(`invalid: yaml: content`), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid YAML file: %v", err)
	}

	auth.LoadAuth() // Should log error but continue
}

func TestAuthorize(t *testing.T) {
	auth := &AuthCore{
		tokenMap: make(map[string]TokenRule),
	}

	// Setup test token with regexps
	re1, _ := regexp.Compile(`^GET /api/.*`)
	re2, _ := regexp.Compile(`^POST /api/users$`)
	auth.tokenMap["valid_token"] = TokenRule{
		Regexps: []*regexp.Regexp{re1, re2},
	}

	tests := []struct {
		name           string
		header         string
		method         string
		path           string
		expectedResult bool
	}{
		{
			name:           "Valid token and matching path",
			header:         "Bearer valid_token",
			method:         "GET",
			path:           "/api/users",
			expectedResult: true,
		},
		{
			name:           "Valid token and matching POST path",
			header:         "Bearer valid_token",
			method:         "POST",
			path:           "/api/users",
			expectedResult: true,
		},
		{
			name:           "Invalid token",
			header:         "Bearer invalid_token",
			method:         "GET",
			path:           "/api/users",
			expectedResult: false,
		},
		{
			name:           "Missing Bearer prefix",
			header:         "valid_token",
			method:         "GET",
			path:           "/api/users",
			expectedResult: false,
		},
		{
			name:           "Non-matching path",
			header:         "Bearer valid_token",
			method:         "GET",
			path:           "/other/path",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("Authorization", tt.header)
			result := auth.Authorize(req)
			if result != tt.expectedResult {
				t.Errorf("Authorize() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

func TestAuthMiddleware(t *testing.T) {
	auth := &AuthCore{
		tokenMap: make(map[string]TokenRule),
	}

	// Setup test token
	re, _ := regexp.Compile(`^GET /api/.*`)
	auth.tokenMap["valid_token"] = TokenRule{
		Regexps: []*regexp.Regexp{re},
	}

	// Create a test handler
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Success")
	})

	middleware := auth.AuthMiddleware(nextHandler)

	// Test successful authorization
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer valid_token")
	rr := httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Middleware returned wrong status code: got %v, want %v", status, http.StatusOK)
	}
	if !strings.Contains(rr.Body.String(), "Success") {
		t.Errorf("Middleware did not call next handler")
	}

	// Test unauthorized request
	req = httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer invalid_token")
	rr = httptest.NewRecorder()
	middleware.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("Middleware returned wrong status code: got %v, want %v", status, http.StatusUnauthorized)
	}
	if !strings.Contains(rr.Body.String(), "unauthorized") {
		t.Errorf("Middleware did not return unauthorized message")
	}
}

func TestReloadAuth(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := ioutil.TempDir("", "auth_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a sample YAML file
	yamlContent := `
token1:
  - ^GET /api/.*
`
	yamlFile := filepath.Join(tmpDir, "test.yaml")
	err = ioutil.WriteFile(yamlFile, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write YAML file: %v", err)
	}

	auth := &AuthCore{
		KeyDir:   tmpDir,
		tokenMap: make(map[string]TokenRule),
	}

	// Test ReloadAuth
	auth.ReloadAuth()

	if len(auth.tokenMap) != 1 {
		t.Errorf("Expected 1 token after ReloadAuth, got %d", len(auth.tokenMap))
	}
}
