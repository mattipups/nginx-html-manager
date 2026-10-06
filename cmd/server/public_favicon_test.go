package main

import (
    "bytes"
    "encoding/base64"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "testing"

    favicons "example.com/nginx-html-manager/logo/favicons"
)

func TestPublicFaviconLinksMatchEmbeddedAssets(t *testing.T) {
    links, err := publicFaviconLinks()
    if err != nil { t.Fatal(err) }
    matches := regexp.MustCompile(`<link rel="icon" href="data:([^;]+);base64,([A-Za-z0-9+/=]+)" type="([^"]+)"(?: sizes="([^"]+)")?>`).FindAllStringSubmatch(links, -1)
    expected := []struct{ name, contentType, sizes string }{
        {"favicon.ico", "image/x-icon", ""},
        {"favicon-16.png", "image/png", "16x16"},
        {"favicon-32.png", "image/png", "32x32"},
        {"favicon-48.png", "image/png", "48x48"},
    }
    if len(matches) != len(expected) { t.Fatalf("got %d icons, want %d", len(matches), len(expected)) }
    for i, icon := range expected {
        match := matches[i]
        if match[1] != icon.contentType || match[3] != icon.contentType || match[4] != icon.sizes { t.Fatalf("incorrect metadata for %s", icon.name) }
        decoded, err := base64.StdEncoding.DecodeString(match[2])
        if err != nil { t.Fatal(err) }
        original, err := favicons.Assets.ReadFile(icon.name)
        if err != nil { t.Fatal(err) }
        if !bytes.Equal(decoded, original) { t.Fatalf("favicon bytes changed: %s", icon.name) }
    }
}

func TestPublicIndexIncludesFaviconsInHead(t *testing.T) {
    a := testApp(t)
    if err := a.renderPublicIndex(); err != nil { t.Fatal(err) }
    data, err := os.ReadFile(filepath.Join(a.dir, "public", "index.html"))
    if err != nil { t.Fatal(err) }
    links, err := publicFaviconLinks()
    if err != nil { t.Fatal(err) }
    document := string(data)
    headEnd := strings.Index(document, "</head>")
    if headEnd < 0 || !strings.Contains(document[:headEnd], links) { t.Fatal("favicon links missing from public document head") }
    if strings.Contains(document, `href="/favicon`) { t.Fatal("public icons must not depend on admin-origin routes") }
}
