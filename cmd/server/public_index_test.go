package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderPublicIndex(t *testing.T) {
	a := testApp(t)
	publish(t, a)

	if err := a.renderPublicIndex(); err != nil {
		t.Fatalf("renderPublicIndex failed: %v", err)
	}

	indexPath := filepath.Join(a.dir, "public", "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("failed to read generated public/index.html: %v", err)
	}

	htmlStr := string(content)
	if !strings.Contains(htmlStr, "Veröffentlichte Seiten") {
		t.Errorf("missing title in public index")
	}
	if !strings.Contains(htmlStr, "demo.html") {
		t.Errorf("missing demo.html in public index table")
	}
	if !strings.Contains(htmlStr, "badge") {
		t.Errorf("missing badge styling in public index")
	}
}
