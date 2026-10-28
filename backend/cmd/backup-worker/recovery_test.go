package main

import (
	"context"
	"errors"
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

func TestPreparePreviewWAL(t *testing.T) {
	for _, tc := range []struct {
		name      string
		role      string
		failAt    int
		calls     int
		wantError bool
	}{
		{"writable primary archives boundary", "f", 0, 3, false},
		{"offline primary uses existing archive", "", 1, 1, false},
		{"standby uses existing archive", "t", 0, 1, false},
		{"boundary failure stops preparation", "f", 2, 2, true},
		{"archive failure stops preparation", "f", 3, 3, true},
		{"invalid role fails safely", "unknown", 0, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			run := func(_ context.Context, name string, args ...string) ([]byte, error) {
				calls++
				if calls == tc.failAt {
					return nil, errors.New("unavailable")
				}
				command := name + " " + strings.Join(args, " ")
				switch calls {
				case 1:
					if !strings.Contains(command, "SELECT pg_is_in_recovery()") {
						t.Fatalf("unexpected probe: %s", command)
					}
					return []byte(tc.role + "\n"), nil
				case 2:
					if !strings.Contains(command, "pg_logical_emit_message(true,") {
						t.Fatalf("missing transactional boundary: %s", command)
					}
				case 3:
					if command != "pgbackrest --stanza=converge --archive-timeout=60 check" {
						t.Fatalf("unexpected archive check: %s", command)
					}
				default:
					t.Fatalf("unexpected command: %s", command)
				}
				return nil, nil
			}
			err := preparePreviewWAL(context.Background(), run)
			if errors.Is(err, errPreviewArchiveUnavailable) != tc.wantError {
				t.Fatalf("unexpected archive result: %v", err)
			}
			if calls != tc.calls {
				t.Fatalf("got %d calls, expected %d", calls, tc.calls)
			}
		})
	}
}
