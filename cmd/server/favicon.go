package main

import (
    "net/http"
    "strconv"

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
