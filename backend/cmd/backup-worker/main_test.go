package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestBackupCycleAndHealth(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	clock := func() time.Time { return now }
	var calls []string
	run := func(_ context.Context, args ...string) error {
		calls = append(calls, args[len(args)-1])
		return nil
	}
	initialized, backedUp, err := backupCycle(context.Background(), false, run, dir, clock)
	if err != nil || !initialized || !backedUp {
		t.Fatalf("first backup: initialized=%v backedUp=%v err=%v", initialized, backedUp, err)
	}
	if !slices.Equal(calls, []string{"stanza-create", "check", "backup"}) {
		t.Fatalf("unexpected command sequence: %v", calls)
	}
	if err := healthy(dir, now); err != nil {
		t.Fatalf("fresh backup is unhealthy: %v", err)
	}
	calls = nil
	clock = func() time.Time { return now.Add(checkEvery) }
	_, backedUp, err = backupCycle(context.Background(), initialized, run, dir, clock)
	if err != nil || backedUp || !slices.Equal(calls, []string{"check"}) {
		t.Fatalf("fresh backup was repeated: backedUp=%v calls=%v err=%v", backedUp, calls, err)
	}

	calls = nil
	failedRun := func(_ context.Context, args ...string) error {
		calls = append(calls, args[len(args)-1])
		if args[len(args)-1] == "check" {
			return errors.New("bucket unavailable")
		}
		return nil
	}
	clock = func() time.Time { return now }
	initialized, backedUp, err = backupCycle(context.Background(), initialized, failedRun, dir, clock)
	if err == nil || !initialized || !slices.Equal(calls, []string{"check"}) {
		t.Fatalf("failed check: initialized=%v calls=%v err=%v", initialized, calls, err)
	}
	if err := healthy(dir, now); err == nil {
		t.Fatal("failed backup still reports healthy")
	}
	if _, err := os.Stat(filepath.Join(dir, "last-success")); err != nil {
		t.Fatalf("previous successful backup was lost: %v", err)
	}
	_, backedUp, err = backupCycle(context.Background(), initialized, run, dir, clock)
	if err != nil || !backedUp {
		t.Fatalf("backup did not recover after failure: %v", err)
	}
	if err := healthy(dir, now); err != nil {
		t.Fatalf("recovered backup is unhealthy: %v", err)
	}
	if err := healthy(dir, now.Add(backupEvery+3*time.Hour)); err == nil {
		t.Fatal("stale backup still reports healthy")
	}
}

func TestBackupDueRetriesFailureAndInvalidClock(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	check := func(want bool) {
		t.Helper()
		got, err := backupDue(dir, now)
		if err != nil || got != want {
			t.Fatalf("backupDue() = %t, %v; want %t", got, err, want)
		}
	}
	check(true)
	if err := os.WriteFile(filepath.Join(dir, "last-success"), []byte(now.Format(time.RFC3339Nano)), 0600); err != nil {
		t.Fatal(err)
	}
	check(false)
	if err := os.WriteFile(filepath.Join(dir, "last-failure"), []byte(now.Format(time.RFC3339Nano)), 0600); err != nil {
		t.Fatal(err)
	}
	check(true)
	if err := os.Remove(filepath.Join(dir, "last-failure")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "last-success"), []byte("invalid timestamp"), 0600); err != nil {
		t.Fatal(err)
	}
	check(true)
	if err := os.WriteFile(filepath.Join(dir, "last-success"), []byte(now.Add(time.Hour).Format(time.RFC3339Nano)), 0600); err != nil {
		t.Fatal(err)
	}
	check(true)
}
