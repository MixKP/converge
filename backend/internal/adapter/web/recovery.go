package web

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/RinZ5/converge/backend/internal/backup"
	"github.com/gin-gonic/gin"
)

type RecoveryHandler struct {
	service *backup.Service
	logger  *slog.Logger
}

func NewRecoveryHandler(service *backup.Service, logger *slog.Logger) *RecoveryHandler {
	return &RecoveryHandler{service: service, logger: logger}
}

func (h *RecoveryHandler) Maintenance(c *gin.Context) {
	if c.Request.Method == http.MethodOptions ||
		c.Request.URL.Path == "/api/recovery/backups" && c.Request.Method == http.MethodGet ||
		c.Request.URL.Path == "/api/recovery/restores" && c.Request.Method == http.MethodPost {
		c.Next()
		return
	}
	if !h.service.EnterRequest() {
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "database restore is in progress"})
		return
	}
	defer h.service.LeaveRequest()
	c.Next()
}

func (h *RecoveryHandler) Status(c *gin.Context) {
	status, err := h.service.Status()
	if err != nil {
		h.fail(c, "read backup status", err)
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *RecoveryHandler) RequestBackup(c *gin.Context) {
	err := h.service.Request()
	if errors.Is(err, backup.ErrBusy) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		h.fail(c, "queue backup", err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "backup queued"})
}

func (h *RecoveryHandler) RequestRestore(c *gin.Context) {
	var request struct {
		BackupName   string `json:"backup_name" binding:"required"`
		Confirmation string `json:"confirmation" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "backup name and confirmation are required"})
		return
	}
	err := h.service.RequestLiveRestore(request.BackupName, request.Confirmation)
	if errors.Is(err, backup.ErrBusy) || errors.Is(err, backup.ErrRecoveryPointChanged) {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "restore started"})
}

func (h *RecoveryHandler) fail(c *gin.Context, operation string, err error) {
	h.logger.Error("backup failed", "operation", operation, "error", err)
	c.JSON(http.StatusServiceUnavailable, gin.H{"error": "backup service is unavailable"})
}
