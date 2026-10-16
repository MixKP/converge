package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type RestoreOptions struct {
	BackupDir      string
	BackupName     string
	SourceHost     string
	SourcePort     string
	TargetHost     string
	TargetPort     string
	TargetDB       string
	TargetUser     string
	TargetPassword string
	Confirmation   string
}

type commandRunner func(context.Context, string, []string, []string) ([]byte, error)

func Restore(ctx context.Context, options RestoreOptions) error {
	return restore(ctx, options, false, command)
}

func RestoreLive(ctx context.Context, options RestoreOptions) error {
	return restore(ctx, options, true, command)
}

func command(ctx context.Context, name string, args, env []string) ([]byte, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Env = append(os.Environ(), env...)
	return command.CombinedOutput()
}

func restore(ctx context.Context, options RestoreOptions, live bool, run commandRunner) error {
	if options.SourceHost == "" || options.SourcePort == "" || options.TargetHost == "" || options.TargetPort == "" || options.TargetDB == "" || options.TargetUser == "" || options.TargetPassword == "" {
		return errors.New("source and target database connection settings are required")
	}
	if !live && sameEndpoint(options.SourceHost, options.SourcePort, options.TargetHost, options.TargetPort) {
		return errors.New("refusing to restore to the production database server")
	}
	expected := "RESTORE to " + options.TargetHost + ":" + options.TargetPort + "/" + options.TargetDB
	if live {
		if !sameEndpoint(options.SourceHost, options.SourcePort, options.TargetHost, options.TargetPort) {
			return errors.New("live restore target must be the configured database server")
		}
		expected = "RESTORE LIVE DATA from " + options.BackupName
	}
	if options.Confirmation != expected {
		return fmt.Errorf("confirmation must be %q", expected)
	}
	archive, err := verifiedArchive(options.BackupDir, options.BackupName)
	if err != nil {
		return err
	}
	env := []string{"PGHOST=" + options.TargetHost, "PGPORT=" + options.TargetPort, "PGDATABASE=" + options.TargetDB, "PGUSER=" + options.TargetUser, "PGPASSWORD=" + options.TargetPassword}
	serverOutput, err := run(ctx, "psql", []string{"-AtX", "-v", "ON_ERROR_STOP=1", "-c", "SHOW server_version_num"}, env)
	if err != nil {
		return fmt.Errorf("read replacement PostgreSQL version: %w: %s", err, strings.TrimSpace(string(serverOutput)))
	}
	serverNumber, err := strconv.Atoi(strings.TrimSpace(string(serverOutput)))
	if err != nil {
		return fmt.Errorf("invalid replacement PostgreSQL version: %w", err)
	}
	clientOutput, err := run(ctx, "pg_restore", []string{"--version"}, env)
	if err != nil {
		return fmt.Errorf("read pg_restore version: %w: %s", err, strings.TrimSpace(string(clientOutput)))
	}
	clientVersion, err := clientMajor(clientOutput)
	if err != nil {
		return fmt.Errorf("invalid pg_restore version: %w", err)
	}
	if clientVersion != serverNumber/10000 {
		return fmt.Errorf("pg_restore major version %d must match replacement PostgreSQL major version %d", clientVersion, serverNumber/10000)
	}
	if err := runChecked(ctx, run, "pg_restore", []string{"--list", archive}, env); err != nil {
		return fmt.Errorf("check archive: %w", err)
	}
	if !live {
		output, err := run(ctx, "psql", []string{"-AtX", "-v", "ON_ERROR_STOP=1", "-c", "SELECT count(*) FROM pg_tables WHERE schemaname NOT IN ('pg_catalog', 'information_schema')"}, env)
		if err != nil {
			return fmt.Errorf("check replacement database: %w: %s", err, strings.TrimSpace(string(output)))
		}
		if strings.TrimSpace(string(output)) != "0" {
			return errors.New("replacement database is not empty")
		}
	}
	args := []string{"--exit-on-error", "--single-transaction", "--no-owner", "--no-acl"}
	if live {
		args = append(args, "--clean", "--if-exists")
	}
	args = append(args, "--dbname="+options.TargetDB, archive)
	if err := runChecked(ctx, run, "pg_restore", args, env); err != nil {
		return fmt.Errorf("restore archive: %w", err)
	}
	for _, table := range []string{"users", "teachers", "branches", "bookings"} {
		if err := runChecked(ctx, run, "psql", []string{"-AtX", "-v", "ON_ERROR_STOP=1", "-c", "SELECT count(*) FROM " + table}, env); err != nil {
			if live {
				return fmt.Errorf("live data was replaced but validation of %s failed: %w", table, err)
			}
			return fmt.Errorf("validate %s: %w", table, err)
		}
	}
	return nil
}

func runChecked(ctx context.Context, run commandRunner, name string, args, env []string) error {
	output, err := run(ctx, name, args, env)
	if err != nil {
		return fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(output)))
	}
	return nil
}

func sameEndpoint(sourceHost, sourcePort, targetHost, targetPort string) bool {
	if sourcePort != targetPort {
		return false
	}
	sourceHost = strings.ToLower(strings.TrimSpace(sourceHost))
	targetHost = strings.ToLower(strings.TrimSpace(targetHost))
	if sourceHost == targetHost {
		return true
	}
	loopback := func(host string) bool {
		if host == "localhost" {
			return true
		}
		ip := net.ParseIP(host)
		return ip != nil && ip.IsLoopback()
	}
	return loopback(sourceHost) && loopback(targetHost)
}

func verifiedArchive(dir, requested string) (string, error) {
	if dir == "" {
		return "", errors.New("backup directory is required")
	}
	name := requested
	if name == "" || name == "latest" {
		data, err := os.ReadFile(filepath.Join(dir, "latest"))
		if err != nil {
			return "", fmt.Errorf("read latest backup: %w", err)
		}
		name = strings.TrimSpace(string(data))
	}
	if name == "" || !filepath.IsLocal(name) || filepath.Base(name) != name || !strings.HasSuffix(name, ".dump") {
		return "", errors.New("backup name must be a .dump filename in the backup directory")
	}
	archive := filepath.Join(dir, name)
	checksum, err := os.ReadFile(archive + ".sha256")
	if err != nil {
		return "", fmt.Errorf("read backup checksum: %w", err)
	}
	fields := strings.Fields(string(checksum))
	if len(fields) != 2 || fields[1] != name {
		return "", errors.New("backup checksum entry does not match archive")
	}
	expected, err := hex.DecodeString(fields[0])
	if err != nil || len(expected) != sha256.Size {
		return "", errors.New("backup checksum is invalid")
	}
	file, err := os.Open(archive)
	if err != nil {
		return "", fmt.Errorf("open backup: %w", err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash backup: %w", err)
	}
	if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), fields[0]) {
		return "", errors.New("backup checksum does not match archive")
	}
	return archive, nil
}
