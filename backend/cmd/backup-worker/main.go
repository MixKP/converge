package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const (
	statusDir   = "/var/lib/converge-backup"
	backupEvery = 24 * time.Hour
	checkEvery  = 15 * time.Minute
)

type commandRunner func(context.Context, ...string) error

func main() {
	if len(os.Args) > 1 {
		if len(os.Args) != 2 || os.Args[1] != "health" {
			log.Fatal("usage: converge-backup-worker [health]")
		}
		if err := healthy(statusDir, time.Now()); err != nil {
			log.Fatal(err)
		}
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	initialized := false
	for ctx.Err() == nil {
		var err error
		var backedUp bool
		initialized, backedUp, err = backupCycle(ctx, initialized, runPgBackRest, statusDir, time.Now)
		if err != nil {
			log.Printf("physical backup or archive check failed: %v", err)
		} else if backedUp {
			log.Print("physical backup completed")
		}
		timer := time.NewTimer(checkEvery)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}

func runPgBackRest(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "pgbackrest", append([]string{"--stanza=converge"}, args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pgbackrest %v: %w", args, err)
	}
	return nil
}

func backupCycle(ctx context.Context, initialized bool, run commandRunner, dir string, now func() time.Time) (bool, bool, error) {
	if !initialized {
		if err := run(ctx, "stanza-create"); err != nil {
			return false, false, recordFailure(dir, fmt.Errorf("initialize stanza: %w", err))
		}
		initialized = true
	}
	if err := run(ctx, "check"); err != nil {
		return initialized, false, recordFailure(dir, fmt.Errorf("check archiving: %w", err))
	}
	due, err := backupDue(dir, now())
	if err != nil {
		return initialized, false, recordFailure(dir, err)
	}
	if !due {
		return initialized, false, nil
	}
	if err := run(ctx, "--type=full", "backup"); err != nil {
		return initialized, false, recordFailure(dir, fmt.Errorf("create full backup: %w", err))
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return initialized, false, fmt.Errorf("create backup status directory: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "last-success"), []byte(now().UTC().Format(time.RFC3339Nano)), 0600); err != nil {
		return initialized, false, recordFailure(dir, fmt.Errorf("record successful backup: %w", err))
	}
	if err := os.Remove(filepath.Join(dir, "last-failure")); err != nil && !errors.Is(err, os.ErrNotExist) {
		return initialized, false, fmt.Errorf("clear failed backup status: %w", err)
	}
	return initialized, true, nil
}

func backupDue(dir string, now time.Time) (bool, error) {
	if _, err := os.Stat(filepath.Join(dir, "last-failure")); err == nil {
		return true, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("read last backup failure: %w", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "last-success"))
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("read last successful backup: %w", err)
	}
	last, err := time.Parse(time.RFC3339Nano, string(data))
	return err != nil || last.After(now) || now.Sub(last) >= backupEvery, nil
}

func recordFailure(dir string, cause error) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("%w; record backup failure: %v", cause, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "last-failure"), []byte(time.Now().UTC().Format(time.RFC3339Nano)), 0600); err != nil {
		return fmt.Errorf("%w; record backup failure: %v", cause, err)
	}
	return cause
}

func healthy(dir string, now time.Time) error {
	if _, err := os.Stat(filepath.Join(dir, "last-failure")); err == nil {
		return errors.New("the last physical backup attempt failed")
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read backup failure status: %w", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "last-success"))
	if err != nil {
		return fmt.Errorf("read last successful backup: %w", err)
	}
	last, err := time.Parse(time.RFC3339Nano, string(data))
	if err != nil {
		return fmt.Errorf("parse last successful backup: %w", err)
	}
	if last.After(now) || now.Sub(last) > backupEvery+2*time.Hour {
		return fmt.Errorf("last successful physical backup is outside the expected interval: %s", last.Format(time.RFC3339))
	}
	return nil
}
