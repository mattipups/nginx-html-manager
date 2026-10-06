package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func createTestStorage(t *testing.T) (string, page, []byte) {
	t.Helper()
	tmpDir := t.TempDir()
	metaDir := filepath.Join(tmpDir, "meta")
	publicDir := filepath.Join(tmpDir, "public")
	_ = os.MkdirAll(metaDir, 0755)
	_ = os.MkdirAll(publicDir, 0755)

	content := []byte("<!doctype html><html><head><title>Test</title></head><body>OK</body></html>")
	id := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	p := page{
		ID:           id,
		Name:         "test.html",
		Size:         int64(len(content)),
		SHA256:       hashStr,
		Created:      time.Now().UTC(),
		URL:          "http://localhost:8081/pages/" + id + ".html",
		CanonicalURL: "http://localhost:8081/pages/" + id + ".html",
		Profile:      ProfileInteractive,
	}

	origPath := filepath.Join(publicDir, id+".html")
	if err := os.WriteFile(origPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	metaBytes, _ := json.Marshal(p)
	if err := os.WriteFile(filepath.Join(metaDir, id+".json"), metaBytes, 0644); err != nil {
		t.Fatal(err)
	}

	return tmpDir, p, content
}

func TestStorageCheckHealthy(t *testing.T) {
	dir, p, _ := createTestStorage(t)

	slugPath := filepath.Join(dir, "public", "test-slug.html")
	origPath := filepath.Join(dir, "public", p.ID+".html")
	if err := os.Link(origPath, slugPath); err != nil {
		t.Fatal(err)
	}

	p.Slug = "test-slug"
	metaBytes, _ := json.Marshal(p)
	_ = os.WriteFile(filepath.Join(dir, "meta", p.ID+".json"), metaBytes, 0644)

	report, err := checkStorage(dir)
	if err != nil {
		t.Fatalf("checkStorage error: %v", err)
	}
	if report.HasErrors() {
		t.Fatalf("expected healthy storage, got issues: %+v", report.Issues)
	}
}

func TestStorageCheckMissingOriginal(t *testing.T) {
	dir, p, _ := createTestStorage(t)
	origPath := filepath.Join(dir, "public", p.ID+".html")
	if err := os.Remove(origPath); err != nil {
		t.Fatal(err)
	}

	report, err := checkStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Issues) != 1 || report.Issues[0].Type != IssueMissingOriginal {
		t.Fatalf("expected MissingOriginal, got: %+v", report.Issues)
	}
}

func TestStorageCheckMissingMeta(t *testing.T) {
	dir, p, _ := createTestStorage(t)
	metaPath := filepath.Join(dir, "meta", p.ID+".json")
	if err := os.Remove(metaPath); err != nil {
		t.Fatal(err)
	}

	report, err := checkStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, issue := range report.Issues {
		if issue.Type == IssueMissingMeta && issue.ID == p.ID {
			found = true
			if !issue.CanFix {
				t.Errorf("MissingMeta should be repairable")
			}
		}
	}
	if !found {
		t.Fatalf("expected MissingMeta for %s, got: %+v", p.ID, report.Issues)
	}
}

func TestStorageCheckSizeAndChecksumMismatch(t *testing.T) {
	dir, p, _ := createTestStorage(t)
	origPath := filepath.Join(dir, "public", p.ID+".html")
	if err := os.WriteFile(origPath, []byte("<!doctype html><html>corrupted</html>"), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := checkStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	hasChecksum := false
	for _, issue := range report.Issues {
		if issue.Type == IssueChecksumMismatch {
			hasChecksum = true
		}
	}
	if !hasChecksum {
		t.Fatalf("expected ChecksumMismatch, got: %+v", report.Issues)
	}
}

func TestStorageCheckBrokenHardlinkAndRepair(t *testing.T) {
	dir, p, content := createTestStorage(t)
	p.Slug = "demo-link"
	metaBytes, _ := json.Marshal(p)
	_ = os.WriteFile(filepath.Join(dir, "meta", p.ID+".json"), metaBytes, 0644)

	slugPath := filepath.Join(dir, "public", "demo-link.html")
	if err := os.WriteFile(slugPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	report, err := checkStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	foundBroken := false
	for _, issue := range report.Issues {
		if issue.Type == IssueBrokenLink {
			foundBroken = true
		}
	}
	if !foundBroken {
		t.Fatalf("expected BrokenLink, got: %+v", report.Issues)
	}

	var out bytes.Buffer
	in := strings.NewReader("ja\n")
	repairedReport, err := runStorageCheck(dir, true, false, in, &out)
	if err != nil {
		t.Fatalf("runStorageCheck repair failed: %v", err)
	}
	if repairedReport.FixedCount == 0 {
		t.Fatalf("expected repair to fix broken hardlink")
	}

	origStat, _ := os.Stat(filepath.Join(dir, "public", p.ID+".html"))
	slugStat, _ := os.Stat(slugPath)
	if !os.SameFile(origStat, slugStat) {
		t.Fatalf("after repair, slug should be the exact same inode as original")
	}
}

func TestStorageCheckOrphanDetectionAndCleanup(t *testing.T) {
	dir, _, _ := createTestStorage(t)
	orphanPath := filepath.Join(dir, "public", "orphan-manual.html")
	if err := os.WriteFile(orphanPath, []byte("orphan content"), 0644); err != nil {
		t.Fatal(err)
	}

	report, err := checkStorage(dir)
	if err != nil {
		t.Fatal(err)
	}
	foundOrphan := false
	for _, issue := range report.Issues {
		if issue.Type == IssueOrphanFile && strings.Contains(issue.Path, "orphan-manual.html") {
			foundOrphan = true
		}
	}
	if !foundOrphan {
		t.Fatalf("expected orphan file to be reported, got: %+v", report.Issues)
	}

	var out bytes.Buffer
	repairedReport, err := runStorageCheck(dir, true, true, nil, &out)
	if err != nil {
		t.Fatalf("runStorageCheck with confirm failed: %v", err)
	}
	if repairedReport.FixedCount == 0 {
		t.Fatalf("expected orphan file to be cleaned up")
	}
	if _, err := os.Stat(orphanPath); !os.IsNotExist(err) {
		t.Fatalf("orphan file should have been removed")
	}
}
