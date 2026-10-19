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

func TestRecoveryPreviewRejectsInvalidEventSelection(t *testing.T) {
	s := &recoveryServer{preview: previewStatus{State: "idle"}}
	for _, body := range []string{
		`{"target":"2026-10-18T12:00:00Z","event_id":3,"position":"after"}`,
		`{"target":"2026-10-18T12:00:00Z","event_id":3,"event_at":"2026-10-18T11:59:59Z","position":"around"}`,
		`{"target":"2026-10-18T12:00:00Z","position":"before"}`,
		`{"target":"2026-10-18T12:00:00Z","event_id":-1}`,
	} {
		response := httptest.NewRecorder()
		s.startPreview(response, httptest.NewRequest(http.MethodPost, "/previews", strings.NewReader(body)))
		if response.Code != http.StatusBadRequest {
			t.Errorf("request %s: got %d, want 400", body, response.Code)
		}
		if s.preview.State != "idle" {
			t.Fatal("invalid selection started a preview")
		}
	}
}
