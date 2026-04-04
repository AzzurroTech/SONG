package core

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BackupManager handles backup and restore operations
type BackupManager struct {
	store     *Store
	backupDir string
	enabled   bool
	interval  time.Duration
	stopChan  chan struct{}
	doneChan  chan struct{}
}

// BackupManifest describes a backup
type BackupManifest struct {
	ID         string    `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	Type       string    `json:"type"` // scheduled, manual
	Files      []string  `json:"files"`
	Checksum   string    `json:"checksum"`
	SizeBytes  int64     `json:"size_bytes"`
	Compressed bool      `json:"compressed"`
}

// NewBackupManager creates a new backup manager
func NewBackupManager(store *Store, backupDir string, enabled bool, interval time.Duration) *BackupManager {
	return &BackupManager{
		store:     store,
		backupDir: backupDir,
		enabled:   enabled,
		interval:  interval,
		stopChan:  make(chan struct{}),
		doneChan:  make(chan struct{}),
	}
}

// Start begins the scheduled backup routine
func (bm *BackupManager) Start() error {
	if !bm.enabled {
		return nil
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(bm.backupDir, 0700); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	go bm.runScheduler()
	return nil
}

// Stop halts the backup scheduler
func (bm *BackupManager) Stop() {
	close(bm.stopChan)
	<-bm.doneChan
}

// runScheduler executes backups at the configured interval
func (bm *BackupManager) runScheduler() {
	ticker := time.NewTicker(bm.interval)
	defer ticker.Stop()
	defer close(bm.doneChan)

	for {
		select {
		case <-bm.stopChan:
			return
		case <-ticker.C:
			if _, err := bm.CreateBackup("scheduled"); err != nil {
				// Log error (would use LogManager in production)
				fmt.Printf("Scheduled backup failed: %v\n", err)
			}
		}
	}
}

// CreateBackup creates a new backup of all core JSON files
func (bm *BackupManager) CreateBackup(backupType string) (*BackupManifest, error) {
	if !bm.enabled {
		return nil, fmt.Errorf("backups are disabled")
	}

	now := time.Now().UTC()
	backupID := now.Format("20060102-150405")
	backupPath := filepath.Join(bm.backupDir, backupID)

	// Create backup directory
	if err := os.MkdirAll(backupPath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create backup folder: %w", err)
	}

	// Files to backup
	coreFiles := []string{
		"users.json",
		"functions.json",
		"sessions.json",
		"logs.jsonl",
	}

	var backedUpFiles []string
	var totalSize int64

	for _, filename := range coreFiles {
		srcPath := filepath.Join(bm.store.baseDir, filename)

		// Check if file exists
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue // Skip missing files
		}

		// Copy and compress
		dstPath := filepath.Join(backupPath, filename+".gz")
		size, err := bm.compressFile(srcPath, dstPath)
		if err != nil {
			return nil, fmt.Errorf("failed to backup %s: %w", filename, err)
		}

		backedUpFiles = append(backedUpFiles, filename)
		totalSize += size
	}

	// Calculate checksum of the backup directory
	checksum, err := bm.calculateDirectoryChecksum(backupPath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	manifest := BackupManifest{
		ID:         backupID,
		CreatedAt:  now,
		Type:       backupType,
		Files:      backedUpFiles,
		Checksum:   checksum,
		SizeBytes:  totalSize,
		Compressed: true,
	}

	// Write manifest
	manifestPath := filepath.Join(backupPath, "manifest.json")
	manifestData, _ := json.MarshalIndent(manifest, "", "  ")
	if err := ioutil.WriteFile(manifestPath, manifestData, 0600); err != nil {
		return nil, fmt.Errorf("failed to write manifest: %w", err)
	}

	return &manifest, nil
}

// RestoreBackup restores core files from a backup
func (bm *BackupManager) RestoreBackup(backupID string) error {
	backupPath := filepath.Join(bm.backupDir, backupID)

	// Verify backup exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup not found: %s", backupID)
	}

	// Load and verify manifest
	manifestPath := filepath.Join(backupPath, "manifest.json")
	manifestData, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest BackupManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Verify checksum
	currentChecksum, err := bm.calculateDirectoryChecksum(backupPath)
	if err != nil {
		return fmt.Errorf("failed to verify checksum: %w", err)
	}
	if currentChecksum != manifest.Checksum {
		return fmt.Errorf("backup checksum mismatch: possible corruption")
	}

	// Restore each file
	for _, filename := range manifest.Files {
		srcPath := filepath.Join(backupPath, filename+".gz")
		dstPath := filepath.Join(bm.store.baseDir, filename)

		if err := bm.decompressFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to restore %s: %w", filename, err)
		}
	}

	return nil
}

// ListBackups returns all available backups
func (bm *BackupManager) ListBackups() ([]BackupManifest, error) {
	entries, err := ioutil.ReadDir(bm.backupDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []BackupManifest{}, nil
		}
		return nil, err
	}

	var backups []BackupManifest
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		manifestPath := filepath.Join(bm.backupDir, entry.Name(), "manifest.json")
		data, err := ioutil.ReadFile(manifestPath)
		if err != nil {
			continue // Skip backups without manifests
		}

		var manifest BackupManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}

		backups = append(backups, manifest)
	}

	return backups, nil
}

// DeleteBackup removes a backup
func (bm *BackupManager) DeleteBackup(backupID string) error {
	backupPath := filepath.Join(bm.backupDir, backupID)
	return os.RemoveAll(backupPath)
}

// PruneOldBackups removes backups older than the retention period
func (bm *BackupManager) PruneOldBackups(retentionDays int) error {
	backups, err := bm.ListBackups()
	if err != nil {
		return err
	}

	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)
	for _, backup := range backups {
		if backup.CreatedAt.Before(cutoff) {
			if err := bm.DeleteBackup(backup.ID); err != nil {
				fmt.Printf("Failed to delete old backup %s: %v\n", backup.ID, err)
			}
		}
	}

	return nil
}

// compressFile compresses a file using gzip
func (bm *BackupManager) compressFile(src, dst string) (int64, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer dstFile.Close()

	gzWriter := gzip.NewWriter(dstFile)
	defer gzWriter.Close()

	written, err := io.Copy(gzWriter, srcFile)
	if err != nil {
		return 0, err
	}

	return written, nil
}

// decompressFile decompresses a gzip file
func (bm *BackupManager) decompressFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	gzReader, err := gzip.NewReader(srcFile)
	if err != nil {
		return err
	}
	defer gzReader.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, gzReader)
	return err
}

// calculateDirectoryChecksum computes a SHA256 hash of all files in a directory
func (bm *BackupManager) calculateDirectoryChecksum(dir string) (string, error) {
	hasher := sha256.New()

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		// Skip manifest file
		if strings.HasSuffix(path, "manifest.json") {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(hasher, f); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
