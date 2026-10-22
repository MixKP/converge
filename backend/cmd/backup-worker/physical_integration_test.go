//go:build integration

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/RinZ5/converge/backend/internal/adapter/db"
	"github.com/RinZ5/converge/backend/internal/adapter/token"
	"github.com/RinZ5/converge/backend/internal/adapter/web"
	"github.com/RinZ5/converge/backend/internal/backup"
	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

func TestPhysicalBackupWALPreview(t *testing.T) {
	if os.Getenv("CONVERGE_PHYSICAL_DRILL") != "1" {
		t.Skip("set CONVERGE_PHYSICAL_DRILL=1 to run the isolated Silo/WAL drill")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	nw, err := network.New(ctx)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, nw.Remove(context.Background())) })
	certificate, key := siloTestCertificate(t)
	credentials := map[string]string{"MINIO_ROOT_USER": "recovery_test", "MINIO_ROOT_PASSWORD": "recovery-test-password"}
	silo, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "pgsty/minio:RELEASE.2026-08-04T00-00-00Z",
			Cmd:   []string{"server", "/data", "--certs-dir", "/certs", "--console-address", ":9001"},
			Env:   credentials,
			Files: []testcontainers.ContainerFile{
				{Reader: bytes.NewReader(certificate), ContainerFilePath: "/certs/public.crt", FileMode: 0644},
				{Reader: bytes.NewReader(key), ContainerFilePath: "/certs/private.key", FileMode: 0600},
			},
			ExposedPorts:   []string{"9000/tcp"},
			WaitingFor:     wait.ForListeningPort("9000/tcp").WithStartupTimeout(2 * time.Minute),
			Networks:       []string{nw.Name},
			NetworkAliases: map[string][]string{nw.Name: {"silo"}},
		}, Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, silo.Terminate(context.Background())) })
	containerExec(t, ctx, silo, "mc", "--insecure", "alias", "set", "local", "https://127.0.0.1:9000", credentials["MINIO_ROOT_USER"], credentials["MINIO_ROOT_PASSWORD"])
	containerExec(t, ctx, silo, "mc", "--insecure", "mb", "local/converge-integration")

	config := fmt.Sprintf(`[converge]
pg1-path=/var/lib/postgresql/data
pg1-socket-path=/var/run/postgresql

[global]
repo1-type=s3
repo1-path=/converge
repo1-s3-bucket=converge-integration
repo1-s3-endpoint=silo
repo1-storage-port=9000
repo1-storage-verify-tls=n
repo1-s3-region=us-east-1
repo1-s3-uri-style=path
repo1-s3-key=%s
repo1-s3-key-secret=%s
repo1-cipher-type=aes-256-cbc
repo1-cipher-pass=recovery-integration-only-passphrase
repo1-retention-full=2
log-level-file=off
`, credentials["MINIO_ROOT_USER"], credentials["MINIO_ROOT_PASSWORD"])
	unique := fmt.Sprintf("converge-dr-integration-%d", time.Now().UnixNano())
	dataVolume, socketVolume := unique+"-data", unique+"-socket"
	for _, volume := range []string{dataVolume, socketVolume} {
		output, err := exec.Command("docker", "volume", "create", volume).CombinedOutput()
		require.NoError(t, err, string(output))
		t.Cleanup(func() {
			output, err := exec.Command("docker", "volume", "rm", volume).CombinedOutput()
			require.NoError(t, err, string(output))
		})
	}
	pgMounts := testcontainers.Mounts(
		testcontainers.VolumeMount(dataVolume, "/var/lib/postgresql/data"),
		testcontainers.VolumeMount(socketVolume, "/var/run/postgresql"),
	)
	configuration := testcontainers.ContainerFile{Reader: strings.NewReader(config), ContainerFilePath: "/etc/pgbackrest/pgbackrest.conf", FileMode: 0644}
	postgres, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "converge-postgres-backup:integration",
			Env:          map[string]string{"POSTGRES_USER": "test", "POSTGRES_PASSWORD": "test", "POSTGRES_DB": "converge_test", "PGUSER": "test"},
			Cmd:          []string{"postgres", "-c", "wal_level=replica", "-c", "archive_mode=on", "-c", "archive_command=pgbackrest --stanza=converge archive-push %p", "-c", "archive_timeout=15min"},
			Files:        []testcontainers.ContainerFile{configuration},
			Mounts:       pgMounts,
			ExposedPorts: []string{"5432/tcp"},
			WaitingFor:   wait.ForListeningPort("5432/tcp").WithStartupTimeout(2 * time.Minute),
			Networks:     []string{nw.Name},
		}, Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, postgres.Terminate(context.Background())) })
	host, err := postgres.Host(ctx)
	require.NoError(t, err)
	port, err := postgres.MappedPort(ctx, "5432/tcp")
	require.NoError(t, err)
	connection := fmt.Sprintf("postgres://test:test@%s:%s/converge_test?sslmode=disable", host, port.Port())
	database, err := db.InitDBWithConnString(connection)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, database.Close()) })
	require.NoError(t, db.AutoMigrate(database))
	_, err = database.Exec(`INSERT INTO branches (name, capacity) VALUES ('base', 10)`)
	require.NoError(t, err)

	worker, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "converge-postgres-backup:integration",
			User:         "postgres",
			Entrypoint:   []string{"/usr/local/bin/converge-backup-worker"},
			Env:          map[string]string{"PGUSER": "test", "PGDATABASE": "converge_test", "PITR_WORKER_TOKEN": "recovery-test-token"},
			Files:        []testcontainers.ContainerFile{{Reader: strings.NewReader(config), ContainerFilePath: "/etc/pgbackrest/pgbackrest.conf", FileMode: 0644}},
			Mounts:       pgMounts,
			ExposedPorts: []string{"8099/tcp"},
			WaitingFor:   wait.ForListeningPort("8099/tcp").WithStartupTimeout(2 * time.Minute),
			Networks:     []string{nw.Name},
		}, Started: true,
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, worker.Terminate(context.Background())) })
	workerHost, err := worker.Host(ctx)
	require.NoError(t, err)
	workerPort, err := worker.MappedPort(ctx, "8099/tcp")
	require.NoError(t, err)
	workerURL := fmt.Sprintf("http://%s:%s", workerHost, workerPort.Port())
	client := &http.Client{Timeout: 20 * time.Second}
	call := func(method, path string, body any, output any) int {
		t.Helper()
		var reader io.Reader
		if body != nil {
			data, err := json.Marshal(body)
			require.NoError(t, err)
			reader = bytes.NewReader(data)
		}
		request, err := http.NewRequestWithContext(ctx, method, workerURL+path, reader)
		require.NoError(t, err)
		request.Header.Set("Authorization", "Bearer recovery-test-token")
		request.Header.Set("Content-Type", "application/json")
		response, err := client.Do(request)
		require.NoError(t, err)
		defer response.Body.Close()
		data, err := io.ReadAll(response.Body)
		require.NoError(t, err)
		if output != nil && response.StatusCode < 300 {
			require.NoError(t, json.Unmarshal(data, output), string(data))
		}
		return response.StatusCode
	}
	var physical physicalStatus
	require.Eventually(t, func() bool {
		physical = physicalStatus{}
		return call(http.MethodGet, "/status", nil, &physical) == http.StatusOK && len(physical.Backups) > 0
	}, 3*time.Minute, 2*time.Second)

	_, err = database.Exec(`INSERT INTO branches (name, capacity) VALUES ('after backup', 20)`)
	require.NoError(t, err)
	var eventID int64
	var eventAt time.Time
	require.NoError(t, database.QueryRow(`SELECT id, occurred_at FROM admin_audit_event ORDER BY id DESC LIMIT 1`).Scan(&eventID, &eventAt))
	require.Eventually(t, func() bool { return time.Now().After(eventAt.Add(time.Second)) }, 3*time.Second, 10*time.Millisecond)
	_, err = database.Exec(`UPDATE branches SET capacity = capacity WHERE name = 'base'`)
	require.NoError(t, err)
	previousSegment := physical.ArchiveSegment
	_, err = database.Exec(`SELECT pg_switch_wal()`)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		physical = physicalStatus{}
		return call(http.MethodGet, "/status", nil, &physical) == http.StatusOK && physical.ArchiveSegment != "" && physical.ArchiveSegment != previousSegment
	}, 2*time.Minute, time.Second)
	for _, check := range []struct {
		position string
		count    int
	}{{"before", 1}, {"after", 2}} {
		target := eventAt.Add(time.Second)
		if check.position == "before" {
			target = eventAt.Add(-time.Microsecond)
		}
		var started previewStatus
		require.Equal(t, http.StatusAccepted, call(http.MethodPost, "/previews", map[string]any{
			"target": target, "event_id": eventID, "event_at": eventAt, "position": check.position,
		}, &started))
		var preview previewStatus
		require.Eventually(t, func() bool {
			preview = previewStatus{}
			return call(http.MethodGet, "/previews", nil, &preview) == http.StatusOK && preview.ID == started.ID && preview.State != "running"
		}, 3*time.Minute, time.Second)
		if preview.State != "ready" {
			logs, logErr := worker.Logs(ctx)
			if logErr == nil {
				content, _ := io.ReadAll(logs)
				t.Logf("backup worker logs: %s", content)
				logs.Close()
			}
		}
		require.Equal(t, "ready", preview.State, preview.Error)
		require.True(t, preview.SelectionMatched)
		require.Equal(t, check.count, preview.Counts["branches"])
	}
	var readyPreview previewStatus
	require.Equal(t, http.StatusOK, call(http.MethodGet, "/previews", nil, &readyPreview))
	var liveCount int
	require.NoError(t, database.QueryRow(`SELECT count(*) FROM branches`).Scan(&liveCount))
	require.Equal(t, 2, liveCount)
	require.Equal(t, http.StatusBadRequest, call(http.MethodPost, "/previews", map[string]any{
		"target": physical.Backups[0].CompletedAt.Add(-time.Second),
	}, nil))
	_, err = database.Exec(`INSERT INTO branches (name, capacity) VALUES ('to be removed', 30)`)
	require.NoError(t, err)
	t.Setenv("DB_HOST", host)
	t.Setenv("DB_PORT", port.Port())
	t.Setenv("POSTGRES_USER", "test")
	t.Setenv("POSTGRES_PASSWORD", "test")
	t.Setenv("POSTGRES_DB", "converge_test")
	t.Setenv("PGHOST", host)
	t.Setenv("PGPORT", port.Port())
	t.Setenv("PGUSER", "test")
	t.Setenv("PGPASSWORD", "test")
	t.Setenv("PGDATABASE", "converge_test")
	t.Setenv("PITR_WORKER_URL", workerURL)
	t.Setenv("PITR_WORKER_TOKEN", "recovery-test-token")
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := backup.New(t.TempDir(), time.Hour, 24*time.Hour, logger)
	service.SetAfterRestore(func() error { return db.RecoverAfterLiveRestore(database) })
	applyHandler := web.NewRecoveryTimelineHandler(database, service, logger)
	jwt := token.NewJWT([]byte("physical-recovery-integration-secret"), time.Hour)
	adminToken, err := jwt.Issue(shared.Principal{UserID: 1, Role: shared.RoleAdmin})
	require.NoError(t, err)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	admin := router.Group("/api", web.AuthMiddleware(jwt), web.RequireRole(shared.RoleAdmin))
	admin.POST("/recovery/previews/apply", applyHandler.ApplyPreview)
	applyBody, err := json.Marshal(map[string]string{"id": readyPreview.ID, "confirmation": "APPLY WAL PREVIEW " + readyPreview.ID})
	require.NoError(t, err)
	applyRequest := httptest.NewRequest(http.MethodPost, "/api/recovery/previews/apply", bytes.NewReader(applyBody))
	applyRequest.Header.Set("Authorization", "Bearer "+adminToken)
	applyRequest.Header.Set("Content-Type", "application/json")
	applyResponse := httptest.NewRecorder()
	router.ServeHTTP(applyResponse, applyRequest)
	require.Equal(t, http.StatusAccepted, applyResponse.Code, applyResponse.Body.String())
	require.Eventually(t, func() bool {
		status, err := service.Status()
		return err == nil && !status.Restoring
	}, 2*time.Minute, 100*time.Millisecond)
	applyStatus, err := service.Status()
	require.NoError(t, err)
	require.Empty(t, applyStatus.RestoreError)
	require.NoError(t, database.QueryRow(`SELECT count(*) FROM branches`).Scan(&liveCount))
	require.Equal(t, 2, liveCount)

	var failuresBefore int64
	var archivedBefore string
	require.NoError(t, database.QueryRow(`SELECT failed_count, COALESCE(last_archived_wal, '') FROM pg_stat_archiver`).Scan(&failuresBefore, &archivedBefore))
	require.NoError(t, silo.Stop(ctx, nil))
	_, err = database.Exec(`INSERT INTO branches (name, capacity) VALUES ('during outage', 30)`)
	require.NoError(t, err)
	_, err = database.Exec(`SELECT pg_switch_wal()`)
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		var failures int64
		return database.QueryRow(`SELECT failed_count FROM pg_stat_archiver`).Scan(&failures) == nil && failures > failuresBefore
	}, time.Minute, time.Second)
	require.NoError(t, silo.Start(ctx))
	require.Eventually(t, func() bool {
		var archived string
		return database.QueryRow(`SELECT COALESCE(last_archived_wal, '') FROM pg_stat_archiver`).Scan(&archived) == nil && archived != "" && archived != archivedBefore
	}, 2*time.Minute, time.Second)
}

func containerExec(t *testing.T, ctx context.Context, container testcontainers.Container, command ...string) {
	t.Helper()
	code, output, err := container.Exec(ctx, command)
	require.NoError(t, err)
	data, err := io.ReadAll(output)
	require.NoError(t, err)
	require.Zero(t, code, string(data))
}

func siloTestCertificate(t *testing.T) ([]byte, []byte) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	require.NoError(t, err)
	template := x509.Certificate{
		SerialNumber: serial, Subject: pkix.Name{CommonName: "silo"}, DNSNames: []string{"silo"},
		NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)
	certificate := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	key := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	return certificate, key
}
