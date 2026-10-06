package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVersionManagerLifecycle(t *testing.T) {
	tmpDir := t.TempDir()
	vm := NewVersionManager(tmpDir)

	histDir := filepath.Join(tmpDir, "history", "test-id", "v1")
	if err := os.MkdirAll(histDir, 0755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(histDir, "index.html"), []byte("<h1>V1</h1>"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	vm.metadata["test-id"] = &ManagedFileMeta{
		ID:             "test-id",
		CurrentVersion: 1,
		History: []FileVersionMeta{
			{Version: 1, Filename: "index.html", Size: 11},
		},
	}

	content := strings.NewReader("<h1>V2 Aktualisiert</h1>")
	updated, err := vm.UpdateContent("test-id", content, "index.html")
	if err != nil {
		t.Fatalf("UpdateContent fehlgeschlagen: %v", err)
	}
	if updated.CurrentVersion != 2 {
		t.Errorf("erwartete Version 2, erhalten: %d", updated.CurrentVersion)
	}
	if len(updated.History) != 2 {
		t.Errorf("erwartete 2 Historien-Einträge, erhalten: %d", len(updated.History))
	}

	restored, err := vm.Rollback("test-id", 1)
	if err != nil {
		t.Fatalf("Rollback fehlgeschlagen: %v", err)
	}
	if restored.CurrentVersion != 1 {
		t.Errorf("erwartete Version 1 nach Rollback, erhalten: %d", restored.CurrentVersion)
	}
}
