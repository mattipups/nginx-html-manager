package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SecurityProfile string

const (
	ProfileStatic      SecurityProfile = "static"
	ProfileInteractive SecurityProfile = "interactive-local"
	ProfileAPIEnabled  SecurityProfile = "api-enabled"
)

type FileVersionMeta struct {
	Version   int       `json:"version"`
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	SHA256    string    `json:"sha256"`
	Timestamp time.Time `json:"timestamp"`
}

type ManagedFileMeta struct {
	ID              string            `json:"id"`
	CustomSlug      string            `json:"custom_slug,omitempty"`
	CurrentVersion  int               `json:"current_version"`
	SecurityProfile SecurityProfile   `json:"security_profile"`
	AllowedHosts    []string          `json:"allowed_hosts,omitempty"`
	History         []FileVersionMeta `json:"history"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type VersionManager struct {
	mu       sync.RWMutex
	dataDir  string
	metadata map[string]*ManagedFileMeta
}

func NewVersionManager(dataDir string) *VersionManager {
	return &VersionManager{
		dataDir:  dataDir,
		metadata: make(map[string]*ManagedFileMeta),
	}
}

func (vm *VersionManager) UpdateContent(id string, reader io.Reader, filename string) (*ManagedFileMeta, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	meta, exists := vm.metadata[id]
	if !exists {
		return nil, fmt.Errorf("datei mit ID %s existiert nicht", id)
	}

	nextVersion := meta.CurrentVersion + 1
	histDir := filepath.Join(vm.dataDir, "history", id, fmt.Sprintf("v%d", nextVersion))
	if err := os.MkdirAll(histDir, 0755); err != nil {
		return nil, err
	}

	destPath := filepath.Join(histDir, filename)
	out, err := os.Create(destPath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	size, err := io.Copy(out, reader)
	if err != nil {
		return nil, err
	}

	// Symlink oder aktiven Stand aktualisieren
	activePath := filepath.Join(vm.dataDir, "current", id)
	_ = os.Remove(activePath)
	if err := os.Link(destPath, activePath); err != nil {
		return nil, err
	}

	meta.CurrentVersion = nextVersion
	meta.UpdatedAt = time.Now().UTC()
	meta.History = append(meta.History, FileVersionMeta{
		Version:   nextVersion,
		Filename:  filename,
		Size:      size,
		Timestamp: meta.UpdatedAt,
	})

	return meta, nil
}

func (vm *VersionManager) Rollback(id string, targetVersion int) (*ManagedFileMeta, error) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	meta, exists := vm.metadata[id]
	if !exists {
		return nil, fmt.Errorf("datei nicht gefunden")
	}

	var found *FileVersionMeta
	for _, v := range meta.History {
		if v.Version == targetVersion {
			found = &v
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("version %d existiert nicht", targetVersion)
	}

	histPath := filepath.Join(vm.dataDir, "history", id, fmt.Sprintf("v%d", targetVersion), found.Filename)
	activePath := filepath.Join(vm.dataDir, "current", id)
	_ = os.Remove(activePath)
	if err := os.Link(histPath, activePath); err != nil {
		return nil, err
	}

	meta.CurrentVersion = targetVersion
	meta.UpdatedAt = time.Now().UTC()
	return meta, nil
}
