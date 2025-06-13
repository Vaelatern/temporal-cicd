package basicauth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthorize_InvalidToken(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/download/some/repo", nil)
	req.Header.Set("Authorization", "Bearer invalid")

	if Authorize(req) {
		t.Errorf("Expected unauthorized for invalid token")
	}
}

func TestHandler_Unauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/download/repo/ref", nil)
	recorder := httptest.NewRecorder()

	handler(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %d", recorder.Code)
	}
}

func TestLoadAuth_BadFile(t *testing.T) {
	LoadAuth() // Expect no panic even if directory or files are malformed
}
