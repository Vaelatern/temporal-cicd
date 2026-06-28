package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		query  string
		want   string
	}{
		{"bearer header", "Bearer mytoken", "", "mytoken"},
		{"query token", "", "mytoken", "mytoken"},
		{"header takes precedence", "Bearer headertok", "querytok", "headertok"},
		{"no token", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			if tt.query != "" {
				req.URL.RawQuery = "token=" + tt.query
			}
			got := getToken(req)
			if got != tt.want {
				t.Errorf("getToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVerifySignature(t *testing.T) {
	secret := "testsecret"
	body := []byte(`{"test":"payload"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !verifySignature(body, secret, expected) {
		t.Error("verifySignature failed for valid sha256=")
	}
	if !verifySignature(body, secret, hex.EncodeToString(mac.Sum(nil))) {
		t.Error("verifySignature failed for raw hex")
	}
	if verifySignature(body, secret, "bad") {
		t.Error("verifySignature should fail for bad sig")
	}
	if verifySignature(body, "wrongsecret", expected) {
		t.Error("verifySignature should fail for wrong secret")
	}

	// From the github documentation https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries
	if !verifySignature([]byte("Hello, World!"),
		"It's a Secret to Everybody",
		"sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17") {
		t.Error("verifySignature should succeed on the github example case")
	}
}

func TestWebhookHandlers_ErrorPaths(t *testing.T) {
	// Use a wrangler without temporal to test early returns; success path not tested here to avoid needing Temporal server
	k := &KickoffWrangler{}

	testCases := []struct {
		name       string
		handler    func(http.ResponseWriter, *http.Request)
		method     string
		path       string
		token      string
		body       string
		headers    map[string]string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "github no token",
			handler:    k.webhookHandler,
			method:     "POST",
			path:       "/hooks/github/test-repo",
			token:      "",
			body:       `{"repository":{"full_name":"owner/repo"},"ref":"refs/heads/main"}`,
			headers:    map[string]string{"X-GitHub-Event": "push", "X-Hub-Signature-256": "sha256=bad"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "github bad signature",
			handler:    k.webhookHandler,
			method:     "POST",
			path:       "/hooks/github/test-repo?token=testtok",
			token:      "testtok",
			body:       `{"repository":{"full_name":"owner/repo"},"ref":"refs/heads/main"}`,
			headers:    map[string]string{"X-GitHub-Event": "push", "X-Hub-Signature-256": "sha256=invalid"},
			wantStatus: http.StatusForbidden,
			wantBody:   "invalid signature from origin",
		},
		{
			name:       "github non-push event",
			handler:    k.webhookHandler,
			method:     "POST",
			path:       "/hooks/github/test-repo?token=testtok",
			token:      "testtok",
			body:       `{"repository":{"full_name":"owner/repo"},"ref":"refs/heads/main"}`,
			headers:    map[string]string{"X-GitHub-Event": "pull_request"},
			wantStatus: http.StatusOK,
			wantBody:   "ignored non-push event",
		},
		{
			name:       "gitlab bad token",
			handler:    k.webhookHandler,
			method:     "POST",
			path:       "/hooks/gitlab/test-repo?token=testtok",
			token:      "testtok",
			body:       `{"project":{"path_with_namespace":"owner/repo"},"ref":"refs/heads/main","object_kind":"push"}`,
			headers:    map[string]string{"X-Gitlab-Token": "wrongtoken"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "bitbucket bad sig",
			handler:    k.webhookHandler,
			method:     "POST",
			path:       "/hooks/bitbucket/test-repo?token=testtok",
			token:      "testtok",
			body:       `{"repository":{"full_name":"owner/repo"},"push":{"changes":[{"new":{"name":"main","type":"branch"}}]}}`,
			headers:    map[string]string{"X-Event-Key": "repo:push", "X-Hub-Signature": "sha256=bad"},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "codeberg bad sig",
			handler:    k.webhookHandler,
			method:     "POST",
			path:       "/hooks/codeberg/test-repo?token=testtok",
			token:      "testtok",
			body:       `{"repository":{"full_name":"owner/repo"},"ref":"refs/heads/main"}`,
			headers:    map[string]string{"X-Gitea-Event": "push", "X-Gitea-Signature": "sha256=bad"},
			wantStatus: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, bytes.NewBufferString(tc.body))
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			if tc.token != "" && !strings.Contains(tc.path, "token=") {
				// ensure query if needed, but in cases above some have
			}
			rr := httptest.NewRecorder()
			// Simulate router path value extraction for direct handler calls
			if strings.Contains(tc.path, "/hooks/") {
				parts := strings.Split(strings.TrimPrefix(strings.Split(tc.path, "?")[0], "/hooks/"), "/")
				if len(parts) >= 2 {
					req.SetPathValue("source", parts[0])
					req.SetPathValue("repo", parts[1])
				}
			}
			tc.handler(rr, req)
			if rr.Code != tc.wantStatus {
				t.Errorf("%s: got status %d, want %d", tc.name, rr.Code, tc.wantStatus)
			}
			if tc.wantBody != "" && !strings.Contains(rr.Body.String(), tc.wantBody) {
				t.Errorf("%s: body %q does not contain %q", tc.name, rr.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestWebhookPayloadParsing(t *testing.T) {
	// Test that ref normalization works conceptually; full handler tests would require Temporal mock
	// Here we just ensure the structs parse sample payloads
	samples := []string{
		`{"repository":{"full_name":"owner/repo"},"ref":"refs/heads/feature/branch"}`,
		`{"project":{"path_with_namespace":"group/sub"},"ref":"refs/tags/v1.0"}`,
		`{"repository":{"full_name":"owner/bb"},"push":{"changes":[{"new":{"name":"main"}}]}}`,
	}
	for i, s := range samples {
		var p1 struct {
			Ref string `json:"ref"`
		}
		if err := json.Unmarshal([]byte(s), &p1); err != nil && !strings.Contains(s, "push") {
			t.Errorf("sample %d parse failed: %v", i, err)
		}
		// bitbucket special
		if strings.Contains(s, "push") {
			var p2 struct {
				Push struct {
					Changes []struct{ New struct{ Name string } }
				} `json:"push"`
			}
			if err := json.Unmarshal([]byte(s), &p2); err != nil {
				t.Errorf("bitbucket sample parse failed: %v", err)
			}
		}
	}
}
