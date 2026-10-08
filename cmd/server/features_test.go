package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUpdateAndRollbackLifecycle(t *testing.T) {
	dir, p, _ := createTestStorage(t)
	a := &app{
		dir:         dir,
		user:        "admin",
		password:    "secret-admin-pass123",
		origin:      "http://localhost:8080",
		publicURL:   "http://localhost:8081",
		maxBytes:    5 * 1024 * 1024,
		maxPages:    500,
		uploadSlots: make(chan struct{}, 1),
	}

	user := &AuthUser{Username: "admin", Role: RoleAdmin}

	updateContent := "<!doctype html><html><body>Version 2 Inhalt</body></html>"
	req := httptest.NewRequest("PUT", "/api/pages/"+p.ID+"/update", strings.NewReader(updateContent))
	req.Header.Set("X-File-Name", "app-v2.html")
	w := httptest.NewRecorder()

	a.updateExistingPage(w, req, user)
	if w.Code != 200 {
		t.Fatalf("update failed: %d, body: %s", w.Code, w.Body.String())
	}

	archiveV1 := filepath.Join(dir, "versions", p.ID, "v1.html")
	if _, err := os.Stat(archiveV1); err != nil {
		t.Fatalf("expected v1 archive to exist: %v", err)
	}

	reqVer := httptest.NewRequest("GET", "/api/pages/"+p.ID+"/versions", nil)
	wVer := httptest.NewRecorder()
	a.listVersions(wVer, reqVer)
	if wVer.Code != 200 {
		t.Fatalf("versions listing failed: %d", wVer.Code)
	}

	reqRoll := httptest.NewRequest("POST", "/api/pages/"+p.ID+"/rollback/1", nil)
	wRoll := httptest.NewRecorder()
	a.rollbackVersion(wRoll, reqRoll, user)
	if wRoll.Code != 200 {
		t.Fatalf("rollback failed: %d, %s", wRoll.Code, wRoll.Body.String())
	}

	currentData, _ := os.ReadFile(filepath.Join(dir, "public", p.ID+".html"))
	if !strings.Contains(string(currentData), "OK") {
		t.Fatalf("rollback should restore initial content with 'OK', got: %s", string(currentData))
	}
}

func TestMetadataAndRedirect(t *testing.T) {
	dir, p, _ := createTestStorage(t)
	a := &app{
		dir:         dir,
		user:        "admin",
		password:    "secret-admin-pass123",
		origin:      "http://localhost:8080",
		publicURL:   "http://localhost:8081",
		maxBytes:    5 * 1024 * 1024,
		maxPages:    500,
		uploadSlots: make(chan struct{}, 1),
	}

	user := &AuthUser{Username: "admin", Role: RoleAdmin}

	p.Slug = "old-name"
	_ = os.Link(filepath.Join(dir, "public", p.ID+".html"), filepath.Join(dir, "public", "old-name.html"))
	metaBytes, _ := json.Marshal(p)
	_ = os.WriteFile(filepath.Join(dir, "meta", p.ID+".json"), metaBytes, 0644)

	title := "Mein Tool"
	desc := "Eine tolle Anwendung"
	tags := []string{"k8s", "argocd"}
	newSlug := "new-tool"
	bodyData, _ := json.Marshal(MetadataUpdateRequest{
		Title:       &title,
		Description: &desc,
		Tags:        tags,
		NewSlug:     &newSlug,
		RedirectOld: true,
	})

	req := httptest.NewRequest("PUT", "/api/pages/"+p.ID+"/metadata", bytes.NewReader(bodyData))
	w := httptest.NewRecorder()
	a.updateMetadata(w, req, user)
	if w.Code != 200 {
		t.Fatalf("metadata update failed: %d, %s", w.Code, w.Body.String())
	}

	oldFile, err := os.ReadFile(filepath.Join(dir, "public", "old-name.html"))
	if err != nil || !strings.Contains(string(oldFile), "http-equiv=\"refresh\"") {
		t.Fatalf("expected redirect file for old-name.html")
	}

	newStat, err := os.Stat(filepath.Join(dir, "public", "new-tool.html"))
	origStat, _ := os.Stat(filepath.Join(dir, "public", p.ID+".html"))
	if err != nil || !os.SameFile(newStat, origStat) {
		t.Fatalf("expected new-tool.html to be a hardlink to original")
	}
}

func TestAuditLogRecording(t *testing.T) {
	dir, p, _ := createTestStorage(t)
	a := &app{dir: dir}
	user := &AuthUser{Username: "test-editor", Role: RoleEditor}

	a.logAudit(AuditEntry{
		Timestamp: time.Now().UTC(),
		User:      user.Username,
		Role:      user.Role,
		Action:    AuditUpload,
		PageID:    p.ID,
		Details:   "Neuer Upload",
		ClientIP:  "127.0.0.1",
	})

	req := httptest.NewRequest("GET", "/api/audit", nil)
	w := httptest.NewRecorder()
	a.getAuditLogs(w, req)
	if w.Code != 200 {
		t.Fatalf("audit logs get failed: %d", w.Code)
	}

	var entries []AuditEntry
	_ = json.Unmarshal(w.Body.Bytes(), &entries)
	if len(entries) != 1 || entries[0].User != "test-editor" || entries[0].Action != AuditUpload {
		t.Fatalf("unexpected audit entry: %+v", entries)
	}
}

func TestStorageQuotaCalculation(t *testing.T) {
	dir, _, _ := createTestStorage(t)
	a := &app{dir: dir}

	quota := a.calculateStorageQuota()
	if quota.TotalBytes == 0 || quota.MaxBytes == 0 {
		t.Fatalf("expected calculated quota: %+v", quota)
	}
}
