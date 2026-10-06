package main

import (
    "bytes"
    "net/http"
    "strconv"
    "testing"
)

func TestFaviconRoutes(t *testing.T) {
    a := testApp(t)
    for _, tc := range []struct{ path, contentType string; size int }{
        {"/favicon.ico", "image/x-icon", 5267},
        {"/favicon-16.png", "image/png", 645},
        {"/favicon-32.png", "image/png", 1605},
        {"/favicon-48.png", "image/png", 2905},
    } {
        t.Run(tc.path, func(t *testing.T) {
            w := request(a, "GET", tc.path, "", "", "", false)
            if w.Code != http.StatusOK || w.Header().Get("Content-Type") != tc.contentType || w.Body.Len() != tc.size {
                t.Fatalf("GET: code=%d type=%s size=%d", w.Code, w.Header().Get("Content-Type"), w.Body.Len())
            }
            signature := []byte{137, 80, 78, 71, 13, 10, 26, 10}
            if tc.contentType == "image/x-icon" { signature = []byte{0, 0, 1, 0} }
            if !bytes.HasPrefix(w.Body.Bytes(), signature) { t.Fatal("incorrect image signature") }
            h := request(a, "HEAD", tc.path, "", "", "", false)
            if h.Code != http.StatusOK || h.Body.Len() != 0 || h.Header().Get("Content-Length") != strconv.Itoa(tc.size) {
                t.Fatalf("HEAD: code=%d length=%s", h.Code, h.Header().Get("Content-Length"))
            }
            for _, method := range []string{"POST", "PUT", "DELETE"} {
                w := request(a, method, tc.path, "", "", "", false)
                if w.Code != http.StatusMethodNotAllowed || w.Header().Get("Allow") != "GET, HEAD" { t.Fatal(method, w.Code) }
            }
        })
    }
}
func TestFaviconDoesNotBypassAdminAuth(t *testing.T) {
    a := testApp(t)
    for _, path := range []string{"/", "/api/pages", "/logo-light.png", "/favicon-unknown.png"} {
        if w := request(a, "GET", path, "", "", "", false); w.Code != http.StatusUnauthorized { t.Fatal(path, w.Code) }
    }
}
