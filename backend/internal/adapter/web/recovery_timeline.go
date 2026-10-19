package web

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/RinZ5/converge/backend/internal/backup"
	"github.com/gin-gonic/gin"
)

type RecoveryTimelineHandler struct {
	db     *sql.DB
	backup *backup.Service
	client *http.Client
	url    string
	token  string
	logger *slog.Logger
}

type workerStatusError struct {
	code    int
	message string
}

func (e workerStatusError) Error() string {
	return fmt.Sprintf("physical recovery worker returned %d", e.code)
}

func NewRecoveryTimelineHandler(db *sql.DB, backupService *backup.Service, logger *slog.Logger) *RecoveryTimelineHandler {
	return &RecoveryTimelineHandler{
		db: db, backup: backupService, logger: logger, client: &http.Client{Timeout: 20 * time.Second},
		url: strings.TrimRight(os.Getenv("PITR_WORKER_URL"), "/"), token: os.Getenv("PITR_WORKER_TOKEN"),
	}
}

func (h *RecoveryTimelineHandler) ApplyPreview(c *gin.Context) {
	var request struct {
		ID           string `json:"id" binding:"required"`
		Confirmation string `json:"confirmation" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "preview and confirmation are required"})
		return
	}
	var preview struct {
		ID               string `json:"id"`
		State            string `json:"state"`
		EventID          int64  `json:"event_id"`
		SelectionMatched bool   `json:"selection_matched"`
	}
	var previewJSON json.RawMessage
	if err := h.worker(c.Request.Context(), http.MethodGet, "/previews", nil, &previewJSON); err != nil || json.Unmarshal(previewJSON, &preview) != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "recovery preview is unavailable"})
		return
	}
	if preview.State != "ready" || preview.ID != request.ID || (preview.EventID > 0 && !preview.SelectionMatched) {
		c.JSON(http.StatusConflict, gin.H{"error": "recovery preview changed; preview again before applying"})
		return
	}
	err := h.backup.RequestPreviewRestore(request.ID, request.Confirmation, func(ctx context.Context, dst io.Writer) (string, error) {
		url := h.url + "/previews/archive?id=" + request.ID
		fetchRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return "", err
		}
		fetchRequest.Header.Set("Authorization", "Bearer "+h.token)
		response, err := (&http.Client{Timeout: 2 * time.Hour}).Do(fetchRequest)
		if err != nil {
			return "", err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return "", workerStatusError{code: response.StatusCode}
		}
		if _, err := io.Copy(dst, response.Body); err != nil {
			return "", err
		}
		return response.Header.Get("X-Archive-Sha256"), nil
	})
	if errors.Is(err, backup.ErrBusy) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "WAL preview restore started"})
}

type auditEvent struct {
	ID            int64     `json:"id"`
	OccurredAt    time.Time `json:"occurred_at"`
	Resource      string    `json:"resource"`
	Operation     string    `json:"operation"`
	RowKey        string    `json:"row_key"`
	ChangedFields []string  `json:"changed_fields"`
}

func (h *RecoveryTimelineHandler) Timeline(c *gin.Context) {
	var after *time.Time
	if raw := c.Query("after"); raw != "" {
		parsed, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "after must be an ISO timestamp"})
			return
		}
		after = &parsed
	}
	beforeID := int64(0)
	var beforeAt *time.Time
	if raw := c.Query("before_id"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "before_id must be positive"})
			return
		}
		beforeID = parsed
		parsedAt, err := time.Parse(time.RFC3339Nano, c.Query("before_at"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "before_at must be an ISO timestamp"})
			return
		}
		beforeAt = &parsedAt
	}
	rows, err := h.db.QueryContext(c.Request.Context(), `
		SELECT id, occurred_at, resource, operation, row_key, array_to_json(changed_fields)::text
		FROM admin_audit_event
		WHERE ($1::timestamptz IS NULL OR occurred_at > $1)
		  AND ($2::timestamptz IS NULL OR (occurred_at, id) < ($2, $3))
		ORDER BY occurred_at DESC, id DESC LIMIT 100`, after, beforeAt, beforeID)
	if err != nil {
		h.logger.Error("read recovery audit events", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "recovery timeline is unavailable"})
		return
	}
	defer rows.Close()
	events := make([]auditEvent, 0)
	for rows.Next() {
		var event auditEvent
		var fields string
		if err := rows.Scan(&event.ID, &event.OccurredAt, &event.Resource, &event.Operation, &event.RowKey, &fields); err != nil {
			h.logger.Error("scan recovery audit event", "error", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "recovery timeline is unavailable"})
			return
		}
		if err := json.Unmarshal([]byte(fields), &event.ChangedFields); err != nil {
			h.logger.Error("decode recovery audit event", "error", err)
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "recovery timeline is unavailable"})
			return
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		h.logger.Error("read recovery audit events", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "recovery timeline is unavailable"})
		return
	}
	rows.Close()
	var physical json.RawMessage
	physicalError := ""
	if err := h.worker(c.Request.Context(), http.MethodGet, "/status", nil, &physical); err != nil {
		physicalError = "Physical backup status is unavailable"
		h.logger.Warn("read physical backup status", "error", err)
	}
	c.JSON(http.StatusOK, gin.H{"events": events, "physical": physical, "physical_error": physicalError})
}

func (h *RecoveryTimelineHandler) Preview(c *gin.Context) {
	var request struct {
		Target   time.Time `json:"target"`
		EventID  int64     `json:"event_id"`
		EventAt  time.Time `json:"event_at"`
		Position string    `json:"position"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid recovery preview request"})
		return
	}
	if request.EventID != 0 {
		if request.EventID < 0 || (request.Position != "before" && request.Position != "after") || !request.Target.IsZero() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "select a recorded change and Before or After"})
			return
		}
		if err := h.db.QueryRowContext(c.Request.Context(), "SELECT occurred_at FROM admin_audit_event WHERE id = $1", request.EventID).Scan(&request.EventAt); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				c.JSON(http.StatusNotFound, gin.H{"error": "this change is no longer in the current history; refresh the list"})
			} else {
				h.logger.Error("read recovery audit event", "error", err)
				c.JSON(http.StatusServiceUnavailable, gin.H{"error": "recovery timeline is unavailable"})
			}
			return
		}
		request.Target = request.EventAt
		if request.Position == "before" {
			request.Target = request.Target.Add(-time.Microsecond)
		} else {
			request.Target = request.Target.Add(time.Second)
		}
	} else if request.Position != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a recorded change is required"})
		return
	}
	if request.Target.IsZero() || request.Target.After(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a past target time is required"})
		return
	}
	var result json.RawMessage
	if err := h.worker(c.Request.Context(), http.MethodPost, "/previews", request, &result); err != nil {
		var workerErr workerStatusError
		if errors.As(err, &workerErr) {
			switch workerErr.code {
			case http.StatusBadRequest:
				message := workerErr.message
				if message == "" {
					message = "invalid recovery target"
				}
				c.JSON(http.StatusBadRequest, gin.H{"error": message})
				return
			case http.StatusConflict:
				c.JSON(http.StatusConflict, gin.H{"error": "a recovery preview is already running"})
				return
			}
		}
		h.logger.Error("start recovery preview", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "could not start isolated recovery preview"})
		return
	}
	c.Data(http.StatusAccepted, "application/json", result)
}

func (h *RecoveryTimelineHandler) PreviewStatus(c *gin.Context) {
	var result json.RawMessage
	if err := h.worker(c.Request.Context(), http.MethodGet, "/previews", nil, &result); err != nil {
		h.logger.Warn("read recovery preview status", "error", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "recovery preview status is unavailable"})
		return
	}
	c.Data(http.StatusOK, "application/json", result)
}

func (h *RecoveryTimelineHandler) worker(ctx context.Context, method, path string, body any, result *json.RawMessage) error {
	if h.url == "" || h.token == "" {
		return fmt.Errorf("physical recovery worker is not configured")
	}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	request, err := http.NewRequestWithContext(ctx, method, h.url+path, reader)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+h.token)
	request.Header.Set("Content-Type", "application/json")
	response, err := h.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return workerStatusError{code: response.StatusCode, message: strings.TrimSpace(string(message))}
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if !json.Valid(data) {
		return fmt.Errorf("physical recovery worker returned invalid JSON")
	}
	*result = data
	return nil
}
