package main

import (
    "encoding/base64"
    "fmt"
    "net/http"
    "strconv"
    "strings"

    favicons "example.com/nginx-html-manager/logo/favicons"
)

var faviconFiles = map[string]struct{ name, contentType string }{
    "/favicon.ico": {"favicon.ico", "image/x-icon"},
    "/favicon-16.png": {"favicon-16.png", "image/png"},
    "/favicon-32.png": {"favicon-32.png", "image/png"},
    "/favicon-48.png": {"favicon-48.png", "image/png"},
}

// Only these public branding assets bypass auth, never HTML, API or user uploads.
func serveFavicon(w http.ResponseWriter, r *http.Request) bool {
    asset, ok := faviconFiles[r.URL.Path]
    if !ok { return false }
    if r.Method != http.MethodGet && r.Method != http.MethodHead {
        w.Header().Set("Allow", "GET, HEAD")
        w.WriteHeader(http.StatusMethodNotAllowed)
        return true
    }
    data, err := favicons.Assets.ReadFile(asset.name)
    if err != nil {
        fail(w, http.StatusInternalServerError, "Favicon nicht verfügbar")
        return true
    }
    w.Header().Set("Content-Type", asset.contentType)
    w.Header().Set("Content-Length", strconv.Itoa(len(data)))
    w.WriteHeader(http.StatusOK)
    if r.Method != http.MethodHead { _, _ = w.Write(data) }
    return true
}

// Inline the same assets on the public origin without relaxing its image CSP.
func publicFaviconLinks() (string, error) {
    var links strings.Builder
    for _, icon := range []struct{ path, sizes string }{
        {"/favicon.ico", ""},
        {"/favicon-16.png", "16x16"},
        {"/favicon-32.png", "32x32"},
        {"/favicon-48.png", "48x48"},
    } {
        asset := faviconFiles[icon.path]
        data, err := favicons.Assets.ReadFile(asset.name)
        if err != nil {
            return "", fmt.Errorf("read public favicon %s: %w", asset.name, err)
        }
        fmt.Fprintf(&links, `<link rel="icon" href="data:%s;base64,%s" type="%s"`, asset.contentType, base64.StdEncoding.EncodeToString(data), asset.contentType)
        if icon.sizes != "" {
            fmt.Fprintf(&links, ` sizes="%s"`, icon.sizes)
        }
        links.WriteString(">\n")
    }
    return links.String(), nil
}
