package main

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	branding "example.com/nginx-html-manager/logo/logos"
)

//go:embed web/*
var web embed.FS

var idRE = regexp.MustCompile(`^[a-f0-9]{32}$`)
var htmlRE = regexp.MustCompile(`(?i)<(?:!doctype\s+html\b|html(?:\s|>))`)

type page struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Size         int64           `json:"size"`
	SHA256       string          `json:"sha256"`
	Created      time.Time       `json:"created"`
	URL          string          `json:"url"`
	Slug         string          `json:"slug,omitempty"`
	CanonicalURL string          `json:"canonicalUrl"`
	Profile      SecurityProfile `json:"profile,omitempty"`
}

type app struct {
	dir, user, password, origin, publicURL string
	maxBytes                               int64
	maxPages                               int
	mu                                     sync.Mutex
	uploadSlots                            chan struct{}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func positive(key string, fallback int64) int64 {
	v, err := strconv.ParseInt(env(key, strconv.FormatInt(fallback, 10)), 10, 64)
	if err != nil || v < 1 {
		log.Fatalf("invalid %s", key)
	}
	return v
}

func validOrigin(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && u.User == nil && u.Path == "" && u.RawQuery == "" && u.Fragment == ""
}

func initStorage(dir string) error {
	for _, name := range []string{"public", "meta"} {
		if err := os.MkdirAll(filepath.Join(dir, name), 0755); err != nil {
			return err
		}
	}
	return nil
}

func (a *app) renderPublicIndex() error {
	pages, err := a.list()
	if err != nil {
		return err
	}
	faviconLinks, err := publicFaviconLinks()
	if err != nil {
		return err
	}
	logoPicture, err := publicLogoPicture()
	if err != nil {
		return err
	}
	var b strings.Builder
	b.WriteString(`<!doctype html>
<html lang="de"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Veröffentlichte Seiten · NGINX HTML Manager</title>
`)
	b.WriteString(faviconLinks)
	b.WriteString(`<style>
*{box-sizing:border-box}body{margin:0;font:16px/1.5 system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#f1f5f9;color:#172033}
main{max-width:1100px;margin:36px auto;padding:0 20px}
header{margin-bottom:24px}
.brand{display:block;max-width:360px}
.brand img{display:block;width:100%;height:auto}
h1{font-size:clamp(24px,4vw,36px);margin:8px 0;font-weight:700}
.eyebrow{font-size:12px;letter-spacing:.15em;color:#2563eb;font-weight:700;text-transform:uppercase;margin:0}
section{background:#fff;border:1px solid #dce3ec;border-radius:14px;padding:24px;margin:24px 0;box-shadow:0 1px 3px rgba(0,0,0,0.05)}
.heading{display:flex;align-items:center;justify-content:space-between;gap:20px;margin-bottom:16px}
.heading h2{margin:0;font-size:20px}
input[type=search]{display:block;width:100%;padding:10px 14px;margin:12px 0 20px;border:1px solid #bcc8d6;border-radius:8px;font:inherit;background:#fff}
.table-wrap{overflow:auto}
table{width:100%;border-collapse:collapse}
th,td{padding:12px 14px;text-align:left;border-bottom:1px solid #e2e8f0}
th{font-size:13px;color:#56657c;text-transform:uppercase;letter-spacing:0.05em;font-weight:600}
td:first-child{max-width:280px;overflow-wrap:anywhere;font-weight:500}
a.action{padding:6px 14px;border:1px solid #bcc8d6;border-radius:6px;background:#eff6ff;color:#1e40af;cursor:pointer;font:inherit;text-decoration:none;display:inline-block;font-size:14px;font-weight:500}
a.action:hover{background:#dbeafe;border-color:#93c5fd}
.badge{display:inline-block;padding:2px 8px;border-radius:4px;font-size:12px;font-weight:600}
.badge-static{background:#e2e8f0;color:#334155}
.badge-interactive{background:#dbeafe;color:#1e40af}
.badge-api{background:#fef3c7;color:#92400e}
footer{margin:32px 0;color:#56657c;font-size:14px;text-align:center}
@media(prefers-color-scheme:dark){
body{background:#111827;color:#f1f5f9}
section{background:#1f2937;border-color:#374151}
.eyebrow{color:#F69370}
input[type=search]{background:#111827;color:#f1f5f9;border-color:#4b5563}
th{color:#9ca3af}
th,td{border-color:#374151}
a.action{background:#1e293b;color:#93c5fd;border-color:#475569}
a.action:hover{background:#334155}
footer{color:#9ca3af}
.badge-static{background:#374151;color:#e5e7eb}
.badge-interactive{background:#1e3a5f;color:#bfdbfe}
.badge-api{background:#451a03;color:#fde68a}
}
</style>
</head>
<body>
<main>
<header>
`)\n\tb.WriteString(logoPicture)\n\tb.WriteString(`<p class="eyebrow">NGINX · HTML-PUBLISHER · 1.5.0</p>
<h1>Veröffentlichte Seiten</h1>
</header>
<section>
<div class="heading">
<h2>Verfügbare Web-Inhalte</h2>
</div>
<input id="search" type="search" placeholder="Dateiname oder URL filtern …" oninput="filterTable()">
<div class="table-wrap">
<table>
<thead><tr><th>Datei / Link</th><th>Profil</th><th>Größe</th><th>Datum (UTC)</th><th>Aktion</th></tr></thead>
<tbody id="page-list">`)\n\tfor _, p := range pages {\n\t\ttargetFile := p.ID + ".html"\n\t\tlinkURL := "/pages/" + targetFile\n\t\tdisplayName := p.Name\n\t\tif p.Slug != "" {\n\t\t\tlinkURL = "/pages/" + p.Slug + ".html"\n\t\t\tdisplayName = p.Slug + " (" + p.Name + ")"\n\t\t}\n\t\tsizeStr := fmt.Sprintf("%.1f KiB", float64(p.Size)/1024.0)\n\t\tbadgeClass := "badge-static"\n\t\tbadgeLabel := "Statisch"\n\t\tif p.Profile == ProfileInteractive {\n\t\t\tbadgeClass = "badge-interactive"\n\t\t\tbadgeLabel = "Interaktiv"\n\t\t} else if p.Profile == ProfileAPIEnabled {\n\t\t\tbadgeClass = "badge-api"\n\t\t\tbadgeLabel = "Interaktiv (API)"\n\t\t}\n\t\tb.WriteString(fmt.Sprintf(`<tr><td><strong><a href="%s">%s</a></strong></td><td><span class="badge %s">%s</span></td><td>%s</td><td>%s</td><td><a class="action" href="%s">Öffnen</a></td></tr>`,\n\t\t\tlinkURL, html.EscapeString(displayName), badgeClass, badgeLabel, sizeStr, p.Created.Format("2006-01-02 15:04"), linkURL))\n\t}\n\tb.WriteString(`</tbody>
</table>
</div>
</section>
<footer>Öffentlicher Bereich · Bereitgestellt über NGINX Port 8081</footer>
<script>
function filterTable(){
    var q = document.getElementById('search').value.toLowerCase();
    var rows = document.querySelectorAll('#page-list tr');
    rows.forEach(function(r){
        r.style.display = r.textContent.toLowerCase().includes(q) ? '' : 'none';
    });
}
</script>
</main>
</body>
</html>`)\n\n\treturn atomicWrite(filepath.Join(a.dir, "public"), "index.html", []byte(b.String()))\n}\n\nfunc main() {\n\tif len(os.Args) > 1 && os.Args[1] == "init-storage" {\n\t\tif err := initStorage(env("DATA_DIR", "/data")); err != nil {\n\t\t\tlog.Fatal(err)\n\t\t}\n\t\treturn\n\t}\n\tif len(os.Args) > 1 && os.Args[1] == "healthcheck" {\n\t\tc := http.Client{Timeout: 2 * time.Second}\n\t\tr, err := c.Get("http://127.0.0.1:8080/healthz")\n\t\tif err != nil {\n\t\t\tos.Exit(1)\n\t\t}\n\t\tdefer r.Body.Close()\n\t\tif r.StatusCode != 200 {\n\t\t\tos.Exit(1)\n\t\t}\n\t\treturn\n\t}\n\tif len(os.Args) > 1 && (os.Args[1] == "verify-storage" || os.Args[1] == "check-storage") {\n\t\tdir := env("DATA_DIR", "/data")\n\t\tfix := false\n\t\tconfirm := false\n\t\tfor _, arg := range os.Args[2:] {\n\t\t\tswitch arg {\n\t\t\tcase "--fix", "-f":\n\t\t\t\tfix = true\n\t\t\tcase "--yes", "-y", "--confirm":\n\t\t\t\tconfirm = true\n\t\t\t}\n\t\t}\n\t\treport, err := runStorageCheck(dir, fix, confirm, os.Stdin, os.Stdout)\n\t\tif err != nil {\n\t\t\tlog.Fatalf("Speicherprüfung fehlgeschlagen: %v", err)\n\t\t}\n\t\tif report.HasErrors() && !fix {\n\t\t\tos.Exit(1)\n\t\t}\n\t\treturn\n\t}\n\tpassword, err := os.ReadFile(env("ADMIN_PASSWORD_FILE", "/run/secrets/admin_password"))\n\tif err != nil {\n\t\tlog.Fatal("cannot read administrator password file")\n\t}\n\tsecret := strings.TrimRight(string(password), "\r\n")\n\tif len(secret) < 16 {\n\t\tlog.Fatal("administrator password must contain at least 16 bytes")\n\t}\n\ta := &app{\n\t\tdir:         env("DATA_DIR", "/data"),\n\t\tuser:        env("ADMIN_USER", "admin"),\n\t\tpassword:    secret,\n\t\torigin:      env("ADMIN_ORIGIN", "http://localhost:8080"),\n\t\tpublicURL:   env("PUBLIC_URL", "http://localhost:8081"),\n\t\tmaxBytes:    positive("MAX_UPLOAD_BYTES", 5*1024*1024),\n\t\tmaxPages:    int(positive("MAX_PAGES", 500)),\n\t\tuploadSlots: make(chan struct{}, 1),\n\t}\n\tif !validOrigin(a.origin) || !validOrigin(a.publicURL) || a.origin == a.publicURL {\n\t\tlog.Fatal("ADMIN_ORIGIN and PUBLIC_URL must be distinct origins without trailing slash")\n\t}\n\tif err := initStorage(a.dir); err != nil {\n\t\tlog.Fatal(err)\n\t}\n\t_ = a.renderPublicIndex()\n\n\tserver := &http.Server{\n\t\tAddr:              ":8080",\n\t\tHandler:           a.handler(),\n\t\tReadHeaderTimeout: 5 * time.Second,\n\t\tReadTimeout:       30 * time.Second,\n\t\tWriteTimeout:      30 * time.Second,\n\t\tIdleTimeout:       60 * time.Second,\n\t\tMaxHeaderBytes:    16 * 1024,\n\t}\n\tctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)\n\tdefer stop()\n\tgo func() {\n\t\t<-ctx.Done()\n\t\tshutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)\n\t\tdefer cancel()\n\t\tif err := server.Shutdown(shutdown); err != nil {\n\t\t\tlog.Printf("shutdown: %v", err)\n\t\t}\n\t}()\n\tlog.Print("HTML manager listening on :8080")\n\tif err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {\n\t\tlog.Fatal(err)\n\t}\n}\n\nfunc jsonReply(w http.ResponseWriter, code int, v any) {\n\tw.Header().Set("Content-Type", "application/json; charset=utf-8")\n\tw.WriteHeader(code)\n\t_ = json.NewEncoder(w).Encode(v)\n}\n\nfunc fail(w http.ResponseWriter, code int, message string) {\n\tjsonReply(w, code, map[string]string{"error": message})\n}\n\nfunc equal(a, b string) bool {\n\tx, y := sha256.Sum256([]byte(a)), sha256.Sum256([]byte(b))\n\treturn subtle.ConstantTimeCompare(x[:], y[:]) == 1\n}\n\nfunc (a *app) handler() http.Handler {\n\treturn http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {\n\t\tw.Header().Set("X-Content-Type-Options", "nosniff")\n\t\tw.Header().Set("Referrer-Policy", "no-referrer")\n\t\tw.Header().Set("Cache-Control", "no-store")\n\t\tw.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")\n\t\tif serveFavicon(w, r) {\n\t\t\treturn\n\t\t}\n\t\tif r.URL.Path == "/healthz" && r.Method == http.MethodGet {\n\t\t\tw.WriteHeader(200)\n\t\t\t_, _ = w.Write([]byte("ok\n"))\n\t\t\treturn\n\t\t}\n\t\tuser, pass, ok := r.BasicAuth()\n\t\tvalidUser, validPass := equal(user, a.user), equal(pass, a.password)\n\t\tif !ok || !validUser || !validPass {\n\t\t\tw.Header().Set("WWW-Authenticate", `Basic realm="HTML-Verwaltung", charset="UTF-8"`)\n\t\t\tfail(w, 401, "Anmeldung erforderlich")\n\t\t\treturn\n\t\t}\n\t\tif r.Method != http.MethodGet && r.Method != http.MethodHead && r.Header.Get("Origin") != a.origin {\n\t\t\tfail(w, 403, "Origin nicht erlaubt")\n\t\t\treturn\n\t\t}\n\t\tswitch {\n\t\tcase r.URL.Path == "/api/pages" && r.Method == http.MethodGet:\n\t\t\ta.mu.Lock()\n\t\t\tdefer a.mu.Unlock()\n\t\t\tpages, err := a.list()\n\t\t\tif err != nil {\n\t\t\t\tfail(w, 500, "Dateiliste konnte nicht geladen werden")\n\t\t\t\treturn\n\t\t\t}\n\t\t\tjsonReply(w, 200, pages)\n\t\tcase r.URL.Path == "/api/upload" && r.Method == http.MethodPost:\n\t\t\ta.upload(w, r)\n\t\tcase strings.HasPrefix(r.URL.Path, "/api/pages/") && strings.HasSuffix(r.URL.Path, "/link") && r.Method == http.MethodPut:\n\t\t\ta.changeLink(w, r)\n\t\tcase strings.HasPrefix(r.URL.Path, "/api/pages/") && strings.HasSuffix(r.URL.Path, "/download") && (r.Method == http.MethodGet || r.Method == http.MethodHead):\n\t\t\ta.download(w, r)\n\t\tcase strings.HasPrefix(r.URL.Path, "/api/pages/") && r.Method == http.MethodDelete:\n\t\t\ta.remove(w, r)\n\t\tcase (r.URL.Path == "/logo-light.png" || r.URL.Path == "/logo-dark.png") && (r.Method == http.MethodGet || r.Method == http.MethodHead):\n\t\t\tname := map[string]string{"/logo-light.png": "builder-light-1400x700.png", "/logo-dark.png": "builder-dark-1400x700.png"}[r.URL.Path]\n\t\t\tdata, err := branding.Assets.ReadFile(name)\n\t\t\tif err != nil {\n\t\t\t\tfail(w, 500, "Logo nicht verfügbar")\n\t\t\t\treturn\n\t\t\t}\n\t\t\tw.Header().Set("Content-Type", "image/png")\n\t\t\tw.WriteHeader(200)\n\t\t\tif r.Method != http.MethodHead {\n\t\t\t\t_, _ = w.Write(data)\n\t\t\t}\n\t\tcase r.Method == http.MethodGet || r.Method == http.MethodHead:\n\t\t\tname := map[string]string{"/": "index.html", "/app.js": "app.js", "/style.css": "style.css"}[r.URL.Path]\n\t\t\tif name == "" {\n\t\t\t\tfail(w, 404, "Nicht gefunden")\n\t\t\t\treturn\n\t\t\t}\n\t\t\tdata, err := web.ReadFile("web/" + name)\n\t\t\tif err != nil {\n\t\t\t\tfail(w, 500, "Oberfläche nicht verfügbar")\n\t\t\t\treturn\n\t\t\t}\n\t\t\ttypes := map[string]string{"index.html": "text/html; charset=utf-8", "app.js": "text/javascript; charset=utf-8", "style.css": "text/css; charset=utf-8"}\n\t\t\tw.Header().Set("Content-Type", types[name])\n\t\t\tw.WriteHeader(200)\n\t\t\tif r.Method != http.MethodHead {\n\t\t\t\t_, _ = w.Write(data)\n\t\t\t}\n\t\tdefault:\n\t\t\tfail(w, 405, "Methode nicht erlaubt")\n\t\t}\n\t})\n}\n\nfunc (a *app) list() ([]page, error) {\n\tentries, err := os.ReadDir(filepath.Join(a.dir, "meta"))\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\tpages := make([]page, 0, len(entries))\n\tfor _, e := range entries {\n\t\tid := strings.TrimSuffix(e.Name(), ".json")\n\t\tif e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || !idRE.MatchString(id) {\n\t\t\tcontinue\n\t\t}\n\t\tdata, err := os.ReadFile(filepath.Join(a.dir, "meta", e.Name()))\n\t\tif err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t\tvar p page\n\t\tif err := json.Unmarshal(data, &p); err != nil || p.ID != id {\n\t\t\treturn nil, fmt.Errorf("invalid metadata: %s", id)\n\t\t}\n\t\tif _, err := os.Stat(filepath.Join(a.dir, "public", id+".html")); errors.Is(err, os.ErrNotExist) {\n\t\t\tcontinue\n\t\t} else if err != nil {\n\t\t\treturn nil, err\n\t\t}\n\t\tif p.Slug != "" && !validSlug(p.Slug) {\n\t\t\treturn nil, fmt.Errorf("invalid slug: %s", id)\n\t\t}\n\t\ta.pageLinks(&p)\n\t\tpages = append(pages, p)\n\t}\n\tsort.Slice(pages, func(i, j int) bool { return pages[i].Created.After(pages[j].Created) })\n\treturn pages, nil\n}\n\nfunc validName(name string) bool {\n\treturn utf8.ValidString(name) && len(name) > 0 && len(name) <= 180 &&\n\t\t!strings.ContainsAny(name, "/\\\x00\r\n") &&\n\t\t(strings.HasSuffix(strings.ToLower(name), ".html") || strings.HasSuffix(strings.ToLower(name), ".htm"))\n}\n\nfunc atomicWrite(dir, name string, data []byte) error {\n\tf, err := os.CreateTemp(dir, ".upload-*")\n\tif err != nil {\n\t\treturn err\n\t}\n\tdefer os.Remove(f.Name())\n\tif err = f.Chmod(0644); err != nil {\n\t\tf.Close()\n\t\treturn err\n\t}\n\tif _, err = f.Write(data); err != nil {\n\t\tf.Close()\n\t\treturn err\n\t}\n\tif err = f.Sync(); err != nil {\n\t\tf.Close()\n\t\treturn err\n\t}\n\tif err = f.Close(); err != nil {\n\t\treturn err\n\t}\n\treturn os.Rename(f.Name(), filepath.Join(dir, name))\n}\n\nfunc (a *app) upload(w http.ResponseWriter, r *http.Request) {\n\tselect {\n\tcase a.uploadSlots <- struct{}{}:\n\t\tdefer func() { <-a.uploadSlots }()\n\tdefault:\n\t\tfail(w, 429, "Ein anderer Import läuft bereits")\n\t\treturn\n\t}\n\tname, err := url.PathUnescape(r.Header.Get("X-File-Name"))\n\tif err != nil || !validName(name) {\n\t\tfail(w, 400, "Nur .html/.htm-Dateien mit einfachem Dateinamen sind erlaubt")\n\t\treturn\n\t}\n\tr.Body = http.MaxBytesReader(w, r.Body, a.maxBytes)\n\tbody, err := io.ReadAll(r.Body)\n\tif err != nil {\n\t\tvar limit *http.MaxBytesError\n\t\tif errors.As(err, &limit) {\n\t\t\tfail(w, 413, "Datei überschreitet das Upload-Limit")\n\t\t} else {\n\t\t\tfail(w, 400, "Datei konnte nicht gelesen werden")\n\t\t}\n\t\treturn\n\t}\n\tif len(body) == 0 || !utf8.Valid(body) || strings.ContainsRune(string(body), 0) || !htmlRE.Match(body) {\n\t\tfail(w, 400, "UTF-8 HTML mit <!doctype html> oder <html> erforderlich")\n\t\treturn\n\t}\n\tprofile := SecurityProfile(r.Header.Get("X-Security-Profile"))\n\tif profile == "" {\n\t\tprofile = ProfileInteractive\n\t}\n\n\ta.mu.Lock()\n\tdefer a.mu.Unlock()\n\tpages, err := a.list()\n\tif err != nil {\n\t\tfail(w, 500, "Dateiliste konnte nicht geladen werden")\n\t\treturn\n\t}\n\tif len(pages) >= a.maxPages {\n\t\tfail(w, 409, "Maximale Anzahl veröffentlichter Seiten erreicht")\n\t\treturn\n\t}\n\trandom := make([]byte, 16)\n\tif _, err := rand.Read(random); err != nil {\n\t\tfail(w, 500, "ID konnte nicht erzeugt werden")\n\t\treturn\n\t}\n\tid := hex.EncodeToString(random)\n\thash := sha256.Sum256(body)\n\tp := page{\n\t\tID:           id,\n\t\tName:         name,\n\t\tSize:         int64(len(body)),\n\t\tSHA256:       hex.EncodeToString(hash[:]),\n\t\tCreated:      time.Now().UTC(),\n\t\tURL:          a.publicURL + "/pages/" + id + ".html",\n\t\tProfile:      profile,\n\t\tCanonicalURL: a.publicURL + "/pages/" + id + ".html",\n\t}\n\ta.pageLinks(&p)\n\tmetadata, _ := json.Marshal(p)\n\tif err := atomicWrite(filepath.Join(a.dir, "meta"), id+".json", metadata); err != nil {\n\t\tfail(w, 500, "Metadaten konnten nicht gespeichert werden")\n\t\treturn\n\t}\n\tif err := atomicWrite(filepath.Join(a.dir, "public"), id+".html", body); err != nil {\n\t\t_ = os.Remove(filepath.Join(a.dir, "meta", id+".json"))\n\t\tfail(w, 500, "Datei konnte nicht gespeichert werden")\n\t\treturn\n\t}\n\t_ = a.renderPublicIndex()\n\tjsonReply(w, 201, p)\n}\n\nfunc (a *app) remove(w http.ResponseWriter, r *http.Request) {\n\tid := strings.TrimPrefix(r.URL.Path, "/api/pages/")\n\tif !idRE.MatchString(id) {\n\t\tfail(w, 400, "Ungültige Datei-ID")\n\t\treturn\n\t}\n\ta.mu.Lock()\n\tdefer a.mu.Unlock()\n\tif err := a.removePublishedFiles(id); err != nil {\n\t\tif errors.Is(err, os.ErrNotExist) {\n\t\t\tfail(w, 404, "Datei nicht gefunden")\n\t\t} else {\n\t\t\tfail(w, 500, "Löschen fehlgeschlagen")\n\t\t}\n\t\treturn\n\t}\n\tif err := os.Remove(filepath.Join(a.dir, "meta", id+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {\n\t\tlog.Printf("metadata cleanup: %v", err)\n\t}\n\t_ = a.renderPublicIndex()\n\tw.WriteHeader(http.StatusNoContent)\n}\n