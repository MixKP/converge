package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/RinZ5/converge/backend/internal/backup"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	backupDir := flag.String("backup-dir", os.Getenv("BACKUP_DIR"), "directory containing verified backups")
	backupName := flag.String("backup", "latest", "backup filename or latest")
	targetHost := flag.String("target-host", "127.0.0.1", "replacement database host")
	targetPort := flag.String("target-port", "5433", "replacement database port")
	targetDB := flag.String("target-db", "converge_restore", "empty replacement database name")
	targetUser := flag.String("target-user", os.Getenv("POSTGRES_USER"), "replacement database user")
	confirm := flag.String("confirm", "", "type RESTORE to host:port/database")
	flag.Parse()
	options := backup.RestoreOptions{
		BackupDir:      *backupDir,
		BackupName:     *backupName,
		SourceHost:     os.Getenv("DB_HOST"),
		SourcePort:     os.Getenv("DB_PORT"),
		TargetHost:     *targetHost,
		TargetPort:     *targetPort,
		TargetDB:       *targetDB,
		TargetUser:     *targetUser,
		TargetPassword: os.Getenv("TARGET_DB_PASSWORD"),
		Confirmation:   *confirm,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
	defer cancel()
	if err := backup.Restore(ctx, options); err != nil {
		fmt.Fprintln(os.Stderr, "Recovery stopped:", err)
		os.Exit(1)
	}
	fmt.Println("Backup restored into the isolated database. Check representative records before switching application traffic.")
}
