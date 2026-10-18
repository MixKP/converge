package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecoveryPreviewRequiresTokenAndPastTarget(t *testing.T) {
	s := &recoveryServer{token: "test-token", preview: previewStatus{State: "idle"}}
	handler := s.authorize(s.startPreview)
	request := func(token string) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/previews", strings.NewReader(`{"target":"2099-01-01T00:00:00Z"}`))
		r.Header.Set("Authorization", token)
		return r
	}
	unauthorized := httptest.NewRecorder()
	handler(unauthorized, request("Bearer wrong"))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("missing authorization: got %d", unauthorized.Code)
	}
	invalid := httptest.NewRecorder()
	handler(invalid, request("Bearer test-token"))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("future target: got %d", invalid.Code)
	}
}

func TestPreviewArchiveRejectsStaleID(t *testing.T) {
	s := &recoveryServer{token: "test-token", preview: previewStatus{ID: "123", State: "ready"}}
	request := httptest.NewRequest(http.MethodGet, "/previews/archive?id=122", nil)
	request.Header.Set("Authorization", "Bearer test-token")
	response := httptest.NewRecorder()
	s.authorize(s.previewArchive)(response, request)
	if response.Code != http.StatusConflict {
		t.Fatalf("stale preview: got %d", response.Code)
	}
}
