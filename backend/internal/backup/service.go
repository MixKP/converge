package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Status struct {
	LastSuccess   *time.Time `json:"last_success"`
	LastFailure   *time.Time `json:"last_failure"`
	Latest        string     `json:"latest"`
	Queued        bool       `json:"queued"`
	Running       bool       `json:"running"`
	Restoring     bool       `json:"restoring"`
	RestoreBackup string     `json:"restore_backup"`
	RestoreError  string     `json:"restore_error"`
	RestoredAt    *time.Time `json:"restored_at"`
}

type Service struct {
	dir       string
	interval  time.Duration
	retention time.Duration
	logger    *slog.Logger
	mu        sync.Mutex
	// tradeoff: maintenance gates one API process -> use shared coordination before running multiple API instances.
	gate          sync.RWMutex
	restoring     atomic.Bool
	restoreError  string
	restoreBackup string
	restoredAt    *time.Time
	afterRestore  func() error
	queued        bool
	running       bool
	wake          chan struct{}
	run           func(context.Context, string, ...string) ([]byte, error)
}

func New(dir string, interval, retention time.Duration, logger *slog.Logger) *Service {
	s := &Service{dir: dir, interval: interval, retention: retention, logger: logger, wake: make(chan struct{}, 1)}
	s.run = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Env = append(os.Environ(), "PGHOST="+env("DB_HOST", "localhost"), "PGPORT="+env("DB_PORT", "5432"), "PGUSER="+env("POSTGRES_USER", "postgres"), "PGPASSWORD="+os.Getenv("POSTGRES_PASSWORD"), "PGDATABASE="+env("POSTGRES_DB", "postgres"))
		output, err := cmd.CombinedOutput()
		if err != nil {
			return output, fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(output)))
		}
		return output, nil
	}
	return s
}

func (s *Service) SetAfterRestore(afterRestore func() error) {
	s.afterRestore = afterRestore
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func (s *Service) Ready() error {
	if s.dir == "" {
		return errors.New("backup directory is not configured")
	}
	info, err := os.Stat(s.dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errors.New("backup path is not a directory")
	}
	return nil
}

func (s *Service) Status() (Status, error) {
	if err := s.Ready(); err != nil {
		return Status{}, err
	}
	success, err := readTime(filepath.Join(s.dir, "last_success"))
	if err != nil {
		return Status{}, err
	}
	failure, err := readTime(filepath.Join(s.dir, "last_failure"))
	if err != nil {
		return Status{}, err
	}
	latest, err := os.ReadFile(filepath.Join(s.dir, "latest"))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Status{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return Status{LastSuccess: success, LastFailure: failure, Latest: strings.TrimSpace(string(latest)), Queued: s.queued, Running: s.running, Restoring: s.restoring.Load(), RestoreBackup: s.restoreBackup, RestoreError: s.restoreError, RestoredAt: s.restoredAt}, nil
}

func readTime(path string) (*time.Time, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data)))
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (s *Service) Request() error {
	if err := s.Ready(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.queued || s.running || s.restoring.Load() {
		return ErrBusy
	}
	s.queued = true
	s.wake <- struct{}{}
	return nil
}

var ErrBusy = errors.New("a backup or restore is already queued or running")
var ErrRecoveryPointChanged = errors.New("the latest backup changed; reload and confirm the new recovery point")

type PreviewArchiveFetcher func(context.Context, io.Writer) (string, error)

func (s *Service) EnterRequest() bool {
	if s.restoring.Load() || !s.gate.TryRLock() {
		return false
	}
	if s.restoring.Load() {
		s.gate.RUnlock()
		return false
	}
	return true
}

func (s *Service) LeaveRequest() {
	s.gate.RUnlock()
}

func (s *Service) RequestLiveRestore(name, confirmation string) error {
	if err := s.Ready(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.queued || s.running || s.restoring.Load() {
		return ErrBusy
	}
	latest, err := os.ReadFile(filepath.Join(s.dir, "latest"))
	if err != nil {
		return fmt.Errorf("read latest backup: %w", err)
	}
	if name == "" || name != strings.TrimSpace(string(latest)) {
		return ErrRecoveryPointChanged
	}
	if confirmation != "RESTORE LIVE DATA from "+name {
		return errors.New("confirmation does not match the selected backup")
	}
	s.restoreError = ""
	s.restoreBackup = name
	s.restoredAt = nil
	s.restoring.Store(true)
	go s.runLiveRestore(name)
	return nil
}

func (s *Service) RequestPreviewRestore(id, confirmation string, fetch PreviewArchiveFetcher) error {
	if err := s.Ready(); err != nil {
		return err
	}
	previewID, err := strconv.ParseInt(id, 10, 64)
	if err != nil || previewID <= 0 || fetch == nil {
		return errors.New("a valid recovery preview is required")
	}
	if confirmation != "APPLY WAL PREVIEW "+id {
		return errors.New("confirmation does not match the selected recovery preview")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.queued || s.running || s.restoring.Load() {
		return ErrBusy
	}
	s.restoreError = ""
	s.restoreBackup = "WAL preview " + id
	s.restoredAt = nil
	s.restoring.Store(true)
	go s.runPreviewRestore(id, fetch)
	return nil
}

func (s *Service) runPreviewRestore(id string, fetch PreviewArchiveFetcher) {
	s.gate.Lock()
	defer s.gate.Unlock()
	defer s.restoring.Store(false)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()
	name := "wal-preview-" + id + ".dump"
	archive := filepath.Join(s.dir, name)
	defer os.Remove(archive)
	defer os.Remove(archive + ".sha256")
	err := s.savePreviewArchive(ctx, name, fetch)
	if err == nil {
		err = s.backupPreserving(ctx, name)
	}
	if err == nil {
		err = writeTime(filepath.Join(s.dir, "last_success"))
	}
	if err == nil {
		err = s.restoreLiveArchive(ctx, name)
	}
	s.mu.Lock()
	if err != nil {
		s.restoreError = err.Error()
		s.logger.Error("WAL preview restore failed", "error", err)
	} else {
		now := time.Now().UTC()
		s.restoredAt = &now
		s.logger.Info("WAL preview restored", "preview_id", id)
	}
	s.mu.Unlock()
}

func (s *Service) savePreviewArchive(ctx context.Context, name string, fetch PreviewArchiveFetcher) error {
	tmp, err := os.CreateTemp(s.dir, ".wal-preview-*.partial")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	hash := sha256.New()
	expected, err := fetch(ctx, io.MultiWriter(tmp, hash))
	closeErr := tmp.Close()
	if err != nil {
		return fmt.Errorf("download recovery preview: %w", err)
	}
	if closeErr != nil {
		return closeErr
	}
	if expected == "" || !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), expected) {
		return errors.New("recovery preview archive checksum does not match")
	}
	if _, err := s.run(ctx, "pg_restore", "--list", tmp.Name()); err != nil {
		return fmt.Errorf("verify recovery preview archive: %w", err)
	}
	final := filepath.Join(s.dir, name)
	if err := os.Rename(tmp.Name(), final); err != nil {
		return err
	}
	return writeAtomic(final+".sha256", []byte(expected+"  "+name+"\n"))
}

func (s *Service) runLiveRestore(name string) {
	s.gate.Lock()
	defer s.gate.Unlock()
	defer s.restoring.Store(false)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()
	err := s.backupPreserving(ctx, name)
	if err == nil {
		err = writeTime(filepath.Join(s.dir, "last_success"))
	}
	if err == nil {
		err = s.restoreLiveArchive(ctx, name)
	}
	s.mu.Lock()
	if err != nil {
		s.restoreError = err.Error()
		s.logger.Error("live restore failed", "error", err)
	} else {
		now := time.Now().UTC()
		s.restoredAt = &now
		s.logger.Info("live restore completed", "backup", name)
	}
	s.mu.Unlock()
}

func (s *Service) restoreLiveArchive(ctx context.Context, name string) error {
	err := RestoreLive(ctx, RestoreOptions{
		BackupDir: s.dir, BackupName: name,
		SourceHost: env("DB_HOST", "localhost"), SourcePort: env("DB_PORT", "5432"),
		TargetHost: env("DB_HOST", "localhost"), TargetPort: env("DB_PORT", "5432"),
		TargetDB: env("POSTGRES_DB", "postgres"), TargetUser: env("POSTGRES_USER", "postgres"),
		TargetPassword: os.Getenv("POSTGRES_PASSWORD"), Confirmation: "RESTORE LIVE DATA from " + name,
	})
	if err == nil && s.afterRestore != nil {
		if setupErr := s.afterRestore(); setupErr != nil {
			return fmt.Errorf("live data was replaced but post-restore setup failed: %w", setupErr)
		}
	}
	return err
}

func (s *Service) Start(ctx context.Context) {
	go func() {
		timer := time.NewTimer(0)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
			case <-s.wake:
			}
			s.mu.Lock()
			if s.restoring.Load() {
				s.mu.Unlock()
				timer.Reset(s.interval)
				continue
			}
			select {
			case <-s.wake:
			default:
			}
			s.queued = false
			s.running = true
			s.mu.Unlock()
			jobCtx, cancel := context.WithTimeout(ctx, 2*time.Hour)
			err := s.backup(jobCtx)
			cancel()
			if err != nil {
				s.logger.Error("backup failed", "error", err)
				if writeErr := writeTime(filepath.Join(s.dir, "last_failure")); writeErr != nil {
					s.logger.Error("record backup failure", "error", writeErr)
				}
			} else {
				if writeErr := writeTime(filepath.Join(s.dir, "last_success")); writeErr != nil {
					s.logger.Error("record backup success", "error", writeErr)
				} else {
					if removeErr := os.Remove(filepath.Join(s.dir, "last_failure")); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
						s.logger.Error("clear backup failure", "error", removeErr)
					}
					s.logger.Info("backup completed")
				}
			}
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
			timer.Reset(s.interval)
		}
	}()
}

func writeTime(path string) error {
	return writeAtomic(path, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"))
}

func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".backup-status-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err = tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func (s *Service) checkClientVersion(ctx context.Context) error {
	serverOutput, err := s.run(ctx, "psql", "-AtX", "-v", "ON_ERROR_STOP=1", "-c", "SHOW server_version_num")
	if err != nil {
		return fmt.Errorf("read PostgreSQL server version: %w", err)
	}
	serverNumber, err := strconv.Atoi(strings.TrimSpace(string(serverOutput)))
	if err != nil {
		return fmt.Errorf("invalid PostgreSQL server version: %w", err)
	}
	serverMajor := serverNumber / 10000
	for _, tool := range []string{"pg_dump", "pg_restore"} {
		output, err := s.run(ctx, tool, "--version")
		if err != nil {
			return fmt.Errorf("read %s version: %w", tool, err)
		}
		major, err := clientMajor(output)
		if err != nil {
			return fmt.Errorf("invalid %s version: %w", tool, err)
		}
		if major != serverMajor {
			return fmt.Errorf("%s major version %d must match PostgreSQL server major version %d", tool, major, serverMajor)
		}
	}
	return nil
}

func clientMajor(output []byte) (int, error) {
	for _, field := range strings.Fields(string(output)) {
		if major, err := strconv.Atoi(strings.Split(field, ".")[0]); err == nil {
			return major, nil
		}
	}
	return 0, errors.New("no numeric version found")
}

func (s *Service) backup(ctx context.Context) error {
	return s.backupPreserving(ctx, "")
}

func (s *Service) backupPreserving(ctx context.Context, preserve string) error {
	if err := s.Ready(); err != nil {
		return err
	}
	if err := s.checkClientVersion(ctx); err != nil {
		return err
	}
	lock := filepath.Join(s.dir, ".backup.lock")
	if err := os.Mkdir(lock, 0700); err != nil {
		return fmt.Errorf("backup lock: %w", err)
	}
	defer os.Remove(lock)
	name := env("POSTGRES_DB", "postgres") + "_" + time.Now().UTC().Format("20060102T150405.000000000Z") + "_" + strconv.Itoa(os.Getpid()) + ".dump"
	partial := filepath.Join(s.dir, "."+name+".partial")
	final := filepath.Join(s.dir, name)
	defer os.Remove(partial)
	if _, err := s.run(ctx, "pg_dump", "--format=custom", "--file="+partial); err != nil {
		return err
	}
	if _, err := s.run(ctx, "pg_restore", "--list", partial); err != nil {
		return err
	}
	file, err := os.Open(partial)
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, err = io.Copy(hash, file)
	file.Close()
	if err != nil {
		return err
	}
	if err := os.Rename(partial, final); err != nil {
		return err
	}
	checksum := hex.EncodeToString(hash.Sum(nil)) + "  " + name + "\n"
	if err := writeAtomic(final+".sha256", []byte(checksum)); err != nil {
		return err
	}
	if err := writeAtomic(filepath.Join(s.dir, "latest"), []byte(name+"\n")); err != nil {
		return err
	}
	s.cleanup(name, preserve)
	return nil
}

func (s *Service) cleanup(latest, preserve string) {
	files, err := filepath.Glob(filepath.Join(s.dir, env("POSTGRES_DB", "postgres")+"_*.dump"))
	if err != nil {
		s.logger.Error("list old backups", "error", err)
		return
	}
	for _, path := range files {
		if filepath.Base(path) == latest || filepath.Base(path) == preserve {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) > s.retention {
			if err := os.Remove(path); err != nil {
				s.logger.Error("remove old backup", "error", err)
				continue
			}
			if err := os.Remove(path + ".sha256"); err != nil && !errors.Is(err, os.ErrNotExist) {
				s.logger.Error("remove old checksum", "error", err)
			}
		}
	}
}
