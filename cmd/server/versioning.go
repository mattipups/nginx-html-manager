package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

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

	name, _ := urlPathUnescape(r.Header.Get("X-File-Name"))
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

	// 1. Quota prüfen
	if quotaErr := a.checkQuotaAddition(int64(len(body))); quotaErr != nil {
		fail(w, 413, quotaErr.Error())
		return
	}

	// 2. Bisherigen Stand in Versionshistorie archivieren
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

	// 3. Neuen Stand aktivieren
	nextVersion := p.CurrentVersion + 1
	newHash := sha256.Sum256(body)
	p.CurrentVersion = nextVersion
	p.Name = name
	p.Size = int64(len(body))
	p.SHA256 = hex.EncodeToString(newHash[:])
	p.Created = time.Now().UTC()

	// Atomar in public schreiben
	if err := atomicWrite(filepath.Join(a.dir, "public"), id+".html", body); err != nil {
		fail(w, 500, "Aktualisierung fehlgeschlagen")
		return
	}

	// Hardlink erneuern
	if p.Slug != "" {
		slugPath := filepath.Join(a.dir, "public", p.Slug+".html")
		_ = os.Remove(slugPath)
		_ = os.Link(filepath.Join(a.dir, "public", id+".html"), slugPath)
	}

	// Metadaten speichern
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

	// Aktuellen Stand vor Rollback sichern
	versionsDir := filepath.Join(a.dir, "versions", id)
	_ = os.WriteFile(filepath.Join(versionsDir, fmt.Sprintf("v%d.html", p.CurrentVersion)), archivedData, 0644)

	// Wiederherstellen
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
