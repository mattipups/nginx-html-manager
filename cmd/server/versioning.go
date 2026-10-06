package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
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

	activeDir := filepath.Join(vm.dataDir, "current")
	if err := os.MkdirAll(activeDir, 0755); err != nil {
		return nil, err
	}
	activePath := filepath.Join(activeDir, id)
	_ = os.Remove(activePath)
	if err := os.Link(destPath, activePath); err != nil {
		content, readErr := os.ReadFile(destPath)
		if readErr != nil {
			return nil, err
		}
		if writeErr := os.WriteFile(activePath, content, 0644); writeErr != nil {
			return nil, writeErr
		}
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
	activeDir := filepath.Join(vm.dataDir, "current")
	if err := os.MkdirAll(activeDir, 0755); err != nil {
		return nil, err
	}
	activePath := filepath.Join(activeDir, id)
	_ = os.Remove(activePath)
	if err := os.Link(histPath, activePath); err != nil {
		content, readErr := os.ReadFile(histPath)
		if readErr != nil {
			return nil, err
		}
		if writeErr := os.WriteFile(activePath, content, 0644); writeErr != nil {
			return nil, writeErr
		}
	}

	meta.CurrentVersion = targetVersion
	meta.UpdatedAt = time.Now().UTC()
	return meta, nil
}

type PageVersion struct {
	Version   int       `json:"version"`
	Filename  string    `json:"filename"`
	Size      int64     `json:"size"`
	SHA256    string    `json:"sha256"`
	Created   time.Time `json:"created"`
	Author    string    `json:"author,omitempty"`
}

func (a *app) updateExistingPage(w http.ResponseWriter, r *http.Request, user *AuthUser) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/pages/"), "/update")
	if !idRE.MatchString(id) {
		fail(w, 400, "Ungültige Datei-ID")
		return
	}

	name, _ := url.PathUnescape(r.Header.Get("X-File-Name"))
	if name == "" {
		name = id + ".html"
	}
	if !validName(name) {
		fail(w, 400, "Ungültiger Dateiname")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, a.maxBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fail(w, 400, "Inhalt konnte nicht gelesen werden")
		return
	}
	if len(body) == 0 || !htmlRE.Match(body) {
		fail(w, 400, "Gültiges HTML erforderlich")
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	p, err := a.storedPage(id)
	if err != nil {
		pageError(w, err)
		return
	}

	if quotaErr := a.checkQuotaAddition(int64(len(body))); quotaErr != nil {
		fail(w, 413, quotaErr.Error())
		return
	}

	if p.CurrentVersion < 1 {
		p.CurrentVersion = 1
	}
	versionsDir := filepath.Join(a.dir, "versions", id)
	_ = os.MkdirAll(versionsDir, 0755)

	archiveName := fmt.Sprintf("v%d.html", p.CurrentVersion)
	currentOriginal := filepath.Join(a.dir, "public", id+".html")
	currentData, _ := os.ReadFile(currentOriginal)
	if len(currentData) > 0 {
		_ = os.WriteFile(filepath.Join(versionsDir, archiveName), currentData, 0644)
	}

	p.Versions = append(p.Versions, PageVersion{
		Version:  p.CurrentVersion,
		Filename: p.Name,
		Size:     p.Size,
		SHA256:   p.SHA256,
		Created:  p.Created,
		Author:   user.Username,
	})

	nextVersion := p.CurrentVersion + 1
	newHash := sha256.Sum256(body)
	p.CurrentVersion = nextVersion
	p.Name = name
	p.Size = int64(len(body))
	p.SHA256 = hex.EncodeToString(newHash[:])
	p.Created = time.Now().UTC()

	if err := atomicWrite(filepath.Join(a.dir, "public"), id+".html", body); err != nil {
		fail(w, 500, "Aktualisierung fehlgeschlagen")
		return
	}

	if p.Slug != "" {
		slugPath := filepath.Join(a.dir, "public", p.Slug+".html")
		_ = os.Remove(slugPath)
		_ = os.Link(filepath.Join(a.dir, "public", id+".html"), slugPath)
	}

	a.pageLinks(&p)
	metaBytes, _ := json.Marshal(p)
	_ = atomicWrite(filepath.Join(a.dir, "meta"), id+".json", metaBytes)

	_ = a.renderPublicIndex()

	a.logAudit(AuditEntry{
		Timestamp: time.Now().UTC(),
		User:      user.Username,
		Role:      user.Role,
		Action:    AuditUpdate,
		PageID:    id,
		Details:   fmt.Sprintf("Aktualisiert auf Version %d (%s, %d Bytes)", nextVersion, name, p.Size),
		ClientIP:  r.RemoteAddr,
	})

	jsonReply(w, 200, p)
}

func (a *app) listVersions(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/pages/"), "/versions")
	if !idRE.MatchString(id) {
		fail(w, 400, "Ungültige Datei-ID")
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	p, err := a.storedPage(id)
	if err != nil {
		pageError(w, err)
		return
	}

	list := make([]PageVersion, len(p.Versions))
	copy(list, p.Versions)
	sort.Slice(list, func(i, j int) bool {
		return list[i].Version > list[j].Version
	})

	jsonReply(w, 200, map[string]any{
		"id":             id,
		"currentVersion": p.CurrentVersion,
		"versions":       list,
	})
}

func (a *app) rollbackVersion(w http.ResponseWriter, r *http.Request, user *AuthUser) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/pages/"), "/")
	if len(parts) < 3 || parts[1] != "rollback" {
		fail(w, 400, "Ungültiger Pfad")
		return
	}
	id := parts[0]
	targetVersion, err := strconv.Atoi(parts[2])
	if err != nil || targetVersion < 1 {
		fail(w, 400, "Ungültige Zielversion")
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	p, err := a.storedPage(id)
	if err != nil {
		pageError(w, err)
		return
	}

	archivePath := filepath.Join(a.dir, "versions", id, fmt.Sprintf("v%d.html", targetVersion))
	archivedData, err := os.ReadFile(archivePath)
	if err != nil {
		fail(w, 404, fmt.Sprintf("Archivversion v%d nicht vorhanden", targetVersion))
		return
	}

	versionsDir := filepath.Join(a.dir, "versions", id)
	_ = os.WriteFile(filepath.Join(versionsDir, fmt.Sprintf("v%d.html", p.CurrentVersion)), archivedData, 0644)

	if err := atomicWrite(filepath.Join(a.dir, "public"), id+".html", archivedData); err != nil {
		fail(w, 500, "Rollback fehlgeschlagen")
		return
	}

	if p.Slug != "" {
		slugPath := filepath.Join(a.dir, "public", p.Slug+".html")
		_ = os.Remove(slugPath)
		_ = os.Link(filepath.Join(a.dir, "public", id+".html"), slugPath)
	}

	hash := sha256.Sum256(archivedData)
	p.CurrentVersion = targetVersion
	p.Size = int64(len(archivedData))
	p.SHA256 = hex.EncodeToString(hash[:])
	p.Created = time.Now().UTC()

	a.pageLinks(&p)
	metaBytes, _ := json.Marshal(p)
	_ = atomicWrite(filepath.Join(a.dir, "meta"), id+".json", metaBytes)

	_ = a.renderPublicIndex()

	a.logAudit(AuditEntry{
		Timestamp: time.Now().UTC(),
		User:      user.Username,
		Role:      user.Role,
		Action:    AuditRollback,
		PageID:    id,
		Details:   fmt.Sprintf("Rollback auf Version %d durchgeführt", targetVersion),
		ClientIP:  r.RemoteAddr,
	})

	jsonReply(w, 200, p)
}
