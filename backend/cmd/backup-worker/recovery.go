package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	previewPath        = "/var/lib/converge-preview/database"
	previewArchivePath = "/var/lib/converge-preview/preview.dump"
)

type physicalBackup struct {
	Label       string    `json:"label"`
	CompletedAt time.Time `json:"completed_at"`
}

type physicalStatus struct {
	Backups            []physicalBackup `json:"backups"`
	ArchiveSegment     string           `json:"archive_segment"`
	LastArchivedAt     *time.Time       `json:"last_archived_at"`
	LastFailedAt       *time.Time       `json:"last_failed_at"`
	ArchiveStatusError string           `json:"archive_status_error,omitempty"`
	RepositoryError    string           `json:"repository_error,omitempty"`
}

type previewStatus struct {
	ID               string         `json:"id,omitempty"`
	State            string         `json:"state"`
	Target           time.Time      `json:"target,omitempty"`
	EventID          int64          `json:"event_id,omitempty"`
	Position         string         `json:"position,omitempty"`
	SelectionMatched bool           `json:"selection_matched,omitempty"`
	StartedAt        *time.Time     `json:"started_at,omitempty"`
	FinishedAt       *time.Time     `json:"finished_at,omitempty"`
	Counts           map[string]int `json:"counts,omitempty"`
	AbsentEvents     []auditEvent   `json:"absent_events,omitempty"`
	ComparisonError  string         `json:"comparison_error,omitempty"`
	Error            string         `json:"error,omitempty"`
}

type auditEvent struct {
	ID            int64     `json:"id"`
	OccurredAt    time.Time `json:"occurred_at"`
	Resource      string    `json:"resource"`
	Operation     string    `json:"operation"`
	RowKey        string    `json:"row_key"`
	ChangedFields []string  `json:"changed_fields"`
}

type recoveryServer struct {
	token     string
	mu        sync.Mutex
	archiveMu sync.RWMutex
	preview   previewStatus
}

func serveRecovery(ctx context.Context, token string) {
	s := &recoveryServer{token: token, preview: previewStatus{State: "idle"}}
	mux := http.NewServeMux()
	mux.HandleFunc("/status", s.authorize(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.status(w, r)
	}))
	mux.HandleFunc("/previews", s.authorize(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			s.previewStatus(w, r)
		case http.MethodPost:
			s.startPreview(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/previews/archive", s.authorize(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		s.previewArchive(w, r)
	}))
	server := &http.Server{Addr: ":8099", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("recovery service stopped: %v", err)
	}
}

func (s *recoveryServer) authorize(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if len(provided) != len(s.token) || subtle.ConstantTimeCompare([]byte(provided), []byte(s.token)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func runOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func readPhysicalStatus(ctx context.Context) (physicalStatus, error) {
	output, err := runOutput(ctx, "pgbackrest", "--stanza=converge", "--output=json", "info")
	if err != nil {
		return physicalStatus{}, err
	}
	var info []struct {
		Archive []struct {
			Max string `json:"max"`
		} `json:"archive"`
		Backup []struct {
			Label     string `json:"label"`
			Error     bool   `json:"error"`
			Timestamp struct {
				Stop int64 `json:"stop"`
			} `json:"timestamp"`
		} `json:"backup"`
		Status struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"status"`
	}
	if err := json.Unmarshal(output, &info); err != nil {
		return physicalStatus{}, fmt.Errorf("parse pgBackRest information: %w", err)
	}
	if len(info) != 1 {
		return physicalStatus{}, errors.New("pgBackRest stanza was not found")
	}
	status := physicalStatus{Backups: []physicalBackup{}}
	if info[0].Status.Code != 0 {
		status.RepositoryError = info[0].Status.Message
	}
	for _, item := range info[0].Backup {
		if !item.Error && item.Timestamp.Stop > 0 {
			status.Backups = append(status.Backups, physicalBackup{Label: item.Label, CompletedAt: time.Unix(item.Timestamp.Stop, 0).UTC()})
		}
	}
	sort.Slice(status.Backups, func(i, j int) bool { return status.Backups[i].CompletedAt.Before(status.Backups[j].CompletedAt) })
	if len(info[0].Archive) > 0 {
		status.ArchiveSegment = info[0].Archive[len(info[0].Archive)-1].Max
	}
	output, err = runOutput(ctx, "psql", "-AtX", "-F", "|", "-c", "SELECT COALESCE(extract(epoch from last_archived_time)::bigint, 0), COALESCE(extract(epoch from last_failed_time)::bigint, 0) FROM pg_stat_archiver")
	if err != nil {
		status.ArchiveStatusError = "PostgreSQL archiver status is unavailable"
	} else {
		parts := strings.Split(strings.TrimSpace(string(output)), "|")
		if len(parts) == 2 {
			if seconds, parseErr := strconv.ParseInt(parts[0], 10, 64); parseErr == nil && seconds > 0 {
				t := time.Unix(seconds, 0).UTC()
				status.LastArchivedAt = &t
			}
			if seconds, parseErr := strconv.ParseInt(parts[1], 10, 64); parseErr == nil && seconds > 0 {
				t := time.Unix(seconds, 0).UTC()
				status.LastFailedAt = &t
			}
		} else {
			status.ArchiveStatusError = "PostgreSQL archiver status is unavailable"
		}
	}
	return status, nil
}

func (s *recoveryServer) status(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	status, err := readPhysicalStatus(ctx)
	if err != nil {
		log.Printf("read recovery status: %v", err)
		http.Error(w, "physical backup status is unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *recoveryServer) previewStatus(w http.ResponseWriter, _ *http.Request) {
	s.mu.Lock()
	status := s.preview
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, status)
}

func (s *recoveryServer) previewArchive(w http.ResponseWriter, r *http.Request) {
	s.archiveMu.RLock()
	defer s.archiveMu.RUnlock()
	s.mu.Lock()
	ready := s.preview.State == "ready" && s.preview.ID != "" && s.preview.ID == r.URL.Query().Get("id")
	s.mu.Unlock()
	if !ready {
		http.Error(w, "preview is no longer available", http.StatusConflict)
		return
	}
	file, err := os.Open(previewArchivePath)
	if err != nil {
		http.Error(w, "preview archive is unavailable", http.StatusServiceUnavailable)
		return
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		http.Error(w, "preview archive is unreadable", http.StatusServiceUnavailable)
		return
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "preview archive is unreadable", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("X-Archive-Sha256", hex.EncodeToString(hash.Sum(nil)))
	if _, err := io.Copy(w, file); err != nil {
		log.Printf("stream preview archive: %v", err)
	}
}

func (s *recoveryServer) startPreview(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Target   time.Time `json:"target"`
		EventID  int64     `json:"event_id"`
		EventAt  time.Time `json:"event_at"`
		Position string    `json:"position"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&request); err != nil || request.Target.IsZero() || request.Target.After(time.Now()) || request.EventID < 0 || (request.EventID > 0 && (request.EventAt.IsZero() || request.Position != "before" && request.Position != "after")) || (request.EventID == 0 && request.Position != "") {
		http.Error(w, "a past target time is required", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	status, err := readPhysicalStatus(ctx)
	cancel()
	if err != nil || status.RepositoryError != "" || len(status.Backups) == 0 {
		http.Error(w, "no usable physical backup is available", http.StatusServiceUnavailable)
		return
	}
	if request.Target.Before(status.Backups[0].CompletedAt) {
		message := "Selected time is before the first physical backup. Choose a later time."
		if request.EventID > 0 {
			message = "This change is before the first physical backup. Choose a later change or create a new backup."
		}
		http.Error(w, message, http.StatusBadRequest)
		return
	}
	s.mu.Lock()
	if s.preview.State == "running" {
		s.mu.Unlock()
		http.Error(w, "a preview is already running", http.StatusConflict)
		return
	}
	now := time.Now().UTC()
	s.preview = previewStatus{ID: strconv.FormatInt(now.UnixNano(), 10), State: "running", Target: request.Target.UTC(), EventID: request.EventID, Position: request.Position, StartedAt: &now}
	started := s.preview
	s.mu.Unlock()
	go s.runPreview(request.Target.UTC(), request.EventID, request.EventAt.UTC(), request.Position)
	writeJSON(w, http.StatusAccepted, started)
}

func (s *recoveryServer) runPreview(target time.Time, eventID int64, eventAt time.Time, position string) {
	s.archiveMu.Lock()
	defer s.archiveMu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()
	counts, absentEvents, comparisonError, err := restorePreview(ctx, target, eventID, eventAt, position)
	finished := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.preview.FinishedAt = &finished
	if err != nil {
		s.preview.State = "failed"
		if errors.Is(err, errPreviewArchiveUnavailable) {
			s.preview.Error = "Could not archive the latest WAL to backup storage. Check storage availability and retry the preview."
		} else if errors.Is(err, errSelectionMismatch) {
			s.preview.Error = "Archived WAL did not reach the selected change at this time. Choose another recovery point or wait for archiving."
		} else {
			s.preview.Error = "Recovery preview failed. Check the backup worker logs for the cause."
		}
		log.Printf("recovery preview failed: %v", err)
	} else {
		s.preview.State = "ready"
		s.preview.SelectionMatched = eventID > 0
		s.preview.Counts = counts
		s.preview.AbsentEvents = absentEvents
		s.preview.ComparisonError = comparisonError
	}
}

var errSelectionMismatch = errors.New("selected audit event did not match recovered preview")

var errPreviewArchiveUnavailable = errors.New("preview WAL archiving failed")

func preparePreviewWAL(ctx context.Context, run func(context.Context, string, ...string) ([]byte, error)) error {
	probeCtx, cancelProbe := context.WithTimeout(ctx, 5*time.Second)
	output, err := run(probeCtx, "psql", "-AtX", "-v", "ON_ERROR_STOP=1", "-c", "SELECT pg_is_in_recovery()")
	cancelProbe()
	if err != nil {
		log.Printf("primary unavailable; preview will use existing archived WAL: %v", err)
		return nil
	}
	switch strings.TrimSpace(string(output)) {
	case "t":
		return nil
	case "f":
	default:
		return fmt.Errorf("%w: unexpected primary recovery state", errPreviewArchiveUnavailable)
	}
	archiveCtx, cancelArchive := context.WithTimeout(ctx, 90*time.Second)
	defer cancelArchive()
	if _, err := run(archiveCtx, "psql", "-AtX", "-v", "ON_ERROR_STOP=1", "-c", "SELECT pg_logical_emit_message(true, 'converge-preview', 'recovery boundary')"); err != nil {
		return fmt.Errorf("%w: commit recovery boundary: %v", errPreviewArchiveUnavailable, err)
	}
	if _, err := run(archiveCtx, "pgbackrest", "--stanza=converge", "--archive-timeout=60", "check"); err != nil {
		return fmt.Errorf("%w: %v", errPreviewArchiveUnavailable, err)
	}
	return nil
}

func restorePreview(ctx context.Context, target time.Time, eventID int64, eventAt time.Time, position string) (map[string]int, []auditEvent, string, error) {
	if err := preparePreviewWAL(ctx, runOutput); err != nil {
		return nil, nil, "", err
	}
	if err := os.Remove(previewArchivePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, nil, "", fmt.Errorf("clear previous preview archive: %w", err)
	}
	if err := os.RemoveAll(previewPath); err != nil {
		return nil, nil, "", fmt.Errorf("clear previous preview: %w", err)
	}
	if err := os.MkdirAll(previewPath, 0700); err != nil {
		return nil, nil, "", fmt.Errorf("create preview directory: %w", err)
	}
	_, err := runOutput(ctx, "pgbackrest", "--stanza=converge", "--pg1-path="+previewPath, "--type=time", "--target="+target.Format("2006-01-02 15:04:05.000000+00"), "--target-action=promote", "restore")
	if err != nil {
		return nil, nil, "", fmt.Errorf("restore preview: %w", err)
	}
	start := exec.CommandContext(ctx, "pg_ctl", "-D", previewPath, "-o", "-c listen_addresses='' -c unix_socket_directories=/tmp -c default_transaction_read_only=on -p 5435", "-w", "start")
	start.Stdout, start.Stderr = os.Stdout, os.Stderr
	if err := start.Run(); err != nil {
		return nil, nil, "", fmt.Errorf("start preview database: %w", err)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if _, stopErr := runOutput(stopCtx, "pg_ctl", "-D", previewPath, "-m", "immediate", "-w", "stop"); stopErr != nil {
			log.Printf("stop preview database: %v", stopErr)
		}
	}()
	if err := waitForPreviewRecovery(ctx); err != nil {
		return nil, nil, "", err
	}
	if eventID > 0 {
		query := "SELECT EXISTS (SELECT 1 FROM admin_audit_event WHERE id = " + strconv.FormatInt(eventID, 10) + " AND occurred_at = '" + eventAt.Format(time.RFC3339Nano) + "'::timestamptz)"
		output, queryErr := runOutput(ctx, "psql", "-h", "/tmp", "-p", "5435", "-d", os.Getenv("PGDATABASE"), "-AtX", "-v", "ON_ERROR_STOP=1", "-c", query)
		if queryErr != nil {
			return nil, nil, "", fmt.Errorf("check selected event in preview: %w", queryErr)
		}
		present := strings.TrimSpace(string(output)) == "t"
		if present != (position == "after") {
			return nil, nil, "", errSelectionMismatch
		}
	}
	counts := make(map[string]int)
	for _, table := range []string{"users", "teachers", "branches", "bookings"} {
		cmd := exec.CommandContext(ctx, "psql", "-h", "/tmp", "-p", "5435", "-d", os.Getenv("PGDATABASE"), "-AtX", "-v", "ON_ERROR_STOP=1", "-c", "SELECT count(*) FROM "+table)
		output, queryErr := cmd.CombinedOutput()
		if queryErr != nil {
			return nil, nil, "", fmt.Errorf("check preview %s: %w: %s", table, queryErr, strings.TrimSpace(string(output)))
		}
		count, parseErr := strconv.Atoi(strings.TrimSpace(string(output)))
		if parseErr != nil {
			return nil, nil, "", fmt.Errorf("read preview %s count: %w", table, parseErr)
		}
		counts[table] = count
	}
	const previewIDsQuery = "SELECT COALESCE(json_agg(id), '[]'::json) FROM (SELECT id FROM admin_audit_event ORDER BY id DESC LIMIT 100) recent"
	output, err := runOutput(ctx, "psql", "-h", "/tmp", "-p", "5435", "-d", os.Getenv("PGDATABASE"), "-AtX", "-v", "ON_ERROR_STOP=1", "-c", previewIDsQuery)
	if err != nil {
		return nil, nil, "", fmt.Errorf("check preview audit history: %w", err)
	}
	var previewIDs []int64
	if err := json.Unmarshal(bytes.TrimSpace(output), &previewIDs); err != nil {
		return nil, nil, "", fmt.Errorf("decode preview audit history: %w", err)
	}
	if _, err := runOutput(ctx, "pg_dump", "-h", "/tmp", "-p", "5435", "-d", os.Getenv("PGDATABASE"), "--format=custom", "--file="+previewArchivePath); err != nil {
		return nil, nil, "", fmt.Errorf("export recovered preview: %w", err)
	}
	if _, err := runOutput(ctx, "pg_restore", "--list", previewArchivePath); err != nil {
		return nil, nil, "", fmt.Errorf("verify recovered preview archive: %w", err)
	}
	const liveEventsQuery = "SELECT COALESCE(json_agg(row_to_json(recent)), '[]'::json) FROM (SELECT id, occurred_at, resource, operation, row_key, changed_fields FROM admin_audit_event ORDER BY id DESC LIMIT 100) recent"
	output, err = runOutput(ctx, "psql", "-AtX", "-v", "ON_ERROR_STOP=1", "-c", liveEventsQuery)
	if err != nil {
		log.Printf("compare recovery preview with live audit: %v", err)
		return counts, nil, "Current database is unavailable; changes could not be compared.", nil
	}
	var liveEvents []auditEvent
	if err := json.Unmarshal(bytes.TrimSpace(output), &liveEvents); err != nil {
		return nil, nil, "", fmt.Errorf("decode live audit history: %w", err)
	}
	present := make(map[int64]bool, len(previewIDs))
	for _, id := range previewIDs {
		present[id] = true
	}
	absent := make([]auditEvent, 0)
	for _, event := range liveEvents {
		if !present[event.ID] {
			absent = append(absent, event)
		}
	}
	return counts, absent, "", nil
}

func waitForPreviewRecovery(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	for {
		output, err := runOutput(ctx, "psql", "-h", "/tmp", "-p", "5435", "-d", os.Getenv("PGDATABASE"), "-AtX", "-v", "ON_ERROR_STOP=1", "-c", "SELECT pg_is_in_recovery()")
		if err == nil && strings.TrimSpace(string(output)) == "f" {
			return nil
		}
		if err != nil {
			if _, statusErr := runOutput(ctx, "pg_ctl", "-D", previewPath, "status"); statusErr != nil {
				return fmt.Errorf("preview database stopped before recovery completed: %w", err)
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for preview WAL recovery: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
