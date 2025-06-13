package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_InvalidAddRepoJSON(t *testing.T) {
	tokenMap = map[string]TokenRule{"test-token": {}} // bypass auth
	req := httptest.NewRequest(http.MethodPut, "/sync/testrepo", strings.NewReader("{"))
	req.Header.Set("Authorization", "Bearer test-token")
	recorder := httptest.NewRecorder()

	handler(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request, got %d", recorder.Code)
	}
}
