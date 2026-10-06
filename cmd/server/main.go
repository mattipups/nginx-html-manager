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
	ID             string          `json:"id"`
	Name           string          `json:"name"`
	Size           int64           `json:"size"`
	SHA256         string          `json:"sha256"`
	Created        time.Time       `json:"created"`
	URL            string          `json:"url"`
	Slug           string          `json:"slug,omitempty"`
	CanonicalURL   string          `json:"canonicalUrl"`
	Profile        SecurityProfile `json:"profile,omitempty"`
	CurrentVersion int             `json:"currentVersion,omitempty"`
	Versions       []PageVersion   `json:"versions,omitempty"`
	Title          string          `json:"title,omitempty"`
	Description    string          `json:"description,omitempty"`
	Tags           []string        `json:"tags,omitempty"`
	Redirects      []string        `json:"redirects,omitempty"`
	Visibility     Visibility      `json:"visibility,omitempty"`
	AllowedGroups  []string        `json:"allowedGroups,omitempty"`
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

func urlPathUnescape(s string) (string, error) {
	return url.PathUnescape(s)
}

func initStorage(dir string) error {
	for _, name := range []string{"public", "meta", "versions"} {
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
`)
	b.WriteString(logoPicture)
	b.WriteString(`<p class="eyebrow">NGINX · HTML-PUBLISHER · 1.5.0</p>
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
<tbody id="page-list">`)
	for _, p := range pages {
		targetFile := p.ID + ".html"
		linkURL := "/pages/" + targetFile
		displayName := p.Name
		if p.Title != "" {
			displayName = p.Title
		} else if p.Slug != "" {
			linkURL = "/pages/" + p.Slug + ".html"
			displayName = p.Slug + " (" + p.Name + ")"
		}
		sizeStr := fmt.Sprintf("%.1f KiB", float64(p.Size)/1024.0)
		badgeClass := "badge-static"
		badgeLabel := "Statisch"
		if p.Profile == ProfileInteractive {
			badgeClass = "badge-interactive"
			badgeLabel = "Interaktiv"
		} else if p.Profile == ProfileAPIEnabled {
			badgeClass = "badge-api"
			badgeLabel = "Interaktiv (API)"
		}
		b.WriteString(fmt.Sprintf(`<tr><td><strong><a href="%s">%s</a></strong></td><td><span class="badge %s">%s</span></td><td>%s</td><td>%s</td><td><a class="action" href="%s">Öffnen</a></td></tr>`,
			linkURL, html.EscapeString(displayName), badgeClass, badgeLabel, sizeStr, p.Created.Format("2006-01-02 15:04"), linkURL))
	}
	b.WriteString(`</tbody>
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
</html>`)

	return atomicWrite(filepath.Join(a.dir, "public"), "index.html", []byte(b.String()))
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "init-storage" {
		if err := initStorage(env("DATA_DIR", "/data")); err != nil {
			log.Fatal(err)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		c := http.Client{Timeout: 2 * time.Second}
		r, err := c.Get("http://127.0.0.1:8080/healthz")
		if err != nil {
			os.Exit(1)
		}
		defer r.Body.Close()
		if r.StatusCode != 200 {
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && (os.Args[1] == "verify-storage" || os.Args[1] == "check-storage") {
		dir := env("DATA_DIR", "/data")
		fix := false
		confirm := false
		for _, arg := range os.Args[2:] {
			switch arg {
			case "--fix", "-f":
				fix = true
			case "--yes", "-y", "--confirm":
				confirm = true
			}
		}
		report, err := runStorageCheck(dir, fix, confirm, os.Stdin, os.Stdout)
		if err != nil {
			log.Fatalf("Speicherprüfung fehlgeschlagen: %v", err)
		}
		if report.HasErrors() && !fix {
			os.Exit(1)
		}
		return
	}
	password, err := os.ReadFile(env("ADMIN_PASSWORD_FILE", "/run/secrets/admin_password"))
	if err != nil {
		log.Fatal("cannot read administrator password file")
	}
	secret := strings.TrimRight(string(password), "\r\n")
	if len(secret) < 16 {
		log.Fatal("administrator password must contain at least 16 bytes")
	}
	a := &app{
		dir:         env("DATA_DIR", "/data"),
		user:        env("ADMIN_USER", "admin"),
		password:    secret,
		origin:      env("ADMIN_ORIGIN", "http://localhost:8080"),
		publicURL:   env("PUBLIC_URL", "http://localhost:8081"),
		maxBytes:    positive("MAX_UPLOAD_BYTES", 5*1024*1024),
		maxPages:    int(positive("MAX_PAGES", 500)),
		uploadSlots: make(chan struct{}, 1),
	}
	if !validOrigin(a.origin) || !validOrigin(a.publicURL) || a.origin == a.publicURL {
		log.Fatal("ADMIN_ORIGIN and PUBLIC_URL must be distinct origins without trailing slash")
	}
	if err := initStorage(a.dir); err != nil {
		log.Fatal(err)
	}
	_ = a.renderPublicIndex()

	server := &http.Server{
		Addr:              ":8080",
		Handler:           a.handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    16 * 1024,
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			log.Printf("shutdown: %v", err)
		}
	}()
	log.Print("HTML manager listening on :8080")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func jsonReply(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, message string) {
	jsonReply(w, code, map[string]string{"error": message})
}

func equal(a, b string) bool {
	x, y := sha256.Sum256([]byte(a)), sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(x[:], y[:]) == 1
}

func (a *app) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; connect-src 'self'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
		if serveFavicon(w, r) {
			return
		}
		if r.URL.Path == "/healthz" && r.Method == http.MethodGet {
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok\n"))
			return
		}

		if strings.HasPrefix(r.URL.Path, "/verify-access") && r.Method == http.MethodGet {
			a.checkPublicAccess(w, r)
			return
		}

		user, ok := a.authenticate(r)
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="HTML-Verwaltung", charset="UTF-8"`)
			fail(w, 401, "Anmeldung erforderlich")
			return
		}
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Header.Get("Origin") != a.origin {
			fail(w, 403, "Origin nicht erlaubt")
			return
		}

		switch {
		case r.URL.Path == "/api/pages" && r.Method == http.MethodGet:
			a.mu.Lock()
			defer a.mu.Unlock()
			pages, err := a.list()
			if err != nil {
				fail(w, 500, "Dateiliste konnte nicht geladen werden")
				return
			}
			jsonReply(w, 200, pages)

		case r.URL.Path == "/api/upload" && r.Method == http.MethodPost:
			if !user.CanUpload() {
				fail(w, 403, "Berechtigung verweigert: Rolle 'viewer' darf keine Dateien hochladen")
				return
			}
			a.upload(w, r, user)

		case strings.HasPrefix(r.URL.Path, "/api/pages/") && strings.HasSuffix(r.URL.Path, "/update") && r.Method == http.MethodPut:
			if !user.CanUpdate() {
				fail(w, 403, "Berechtigung verweigert: Keine Editierrechte")
				return
			}
			a.updateExistingPage(w, r, user)

		case strings.HasPrefix(r.URL.Path, "/api/pages/") && strings.HasSuffix(r.URL.Path, "/versions") && r.Method == http.MethodGet:
			a.listVersions(w, r)

		case strings.HasPrefix(r.URL.Path, "/api/pages/") && strings.Contains(r.URL.Path, "/rollback/") && r.Method == http.MethodPost:
			if !user.CanUpdate() {
				fail(w, 403, "Berechtigung verweigert: Keine Rollback-Rechte")
				return
			}
			a.rollbackVersion(w, r, user)

		case strings.HasPrefix(r.URL.Path, "/api/pages/") && strings.HasSuffix(r.URL.Path, "/metadata") && r.Method == http.MethodPut:
			if !user.CanUpdate() {
				fail(w, 403, "Berechtigung verweigert: Keine Editierrechte")
				return
			}
			a.updateMetadata(w, r, user)

		case strings.HasPrefix(r.URL.Path, "/api/pages/") && strings.HasSuffix(r.URL.Path, "/access") && r.Method == http.MethodPut:
			if user.Role != RoleAdmin {
				fail(w, 403, "Berechtigung verweigert: Nur Administratoren können Zugriffsregeln ändern")
				return
			}
			a.updateAccessPolicy(w, r, user)

		case r.URL.Path == "/api/storage/quota" && r.Method == http.MethodGet:
			a.getQuotaHandler(w, r)

		case r.URL.Path == "/api/audit" && r.Method == http.MethodGet:
			if !user.CanViewAudit() {
				fail(w, 403, "Berechtigung verweigert: Audit-Protokoll nur für Administratoren")
				return
			}
			a.getAuditLogs(w, r)

		case strings.HasPrefix(r.URL.Path, "/api/pages/") && strings.HasSuffix(r.URL.Path, "/link") && r.Method == http.MethodPut:
			if !user.CanUpdate() {
				fail(w, 403, "Berechtigung verweigert")
				return
			}
			a.changeLink(w, r)

		case strings.HasPrefix(r.URL.Path, "/api/pages/") && strings.HasSuffix(r.URL.Path, "/download") && (r.Method == http.MethodGet || r.Method == http.MethodHead):
			a.download(w, r)

		case strings.HasPrefix(r.URL.Path, "/api/pages/") && r.Method == http.MethodDelete:
			if !user.CanDelete() {
				fail(w, 403, "Berechtigung verweigert: Nur Administratoren dürfen Seiten löschen")
				return
			}
			a.remove(w, r, user)

		case (r.URL.Path == "/logo-light.png" || r.URL.Path == "/logo-dark.png") && (r.Method == http.MethodGet || r.Method == http.MethodHead):
			name := map[string]string{"/logo-light.png": "builder-light-1400x700.png", "/logo-dark.png": "builder-dark-1400x700.png"}[r.URL.Path]
			data, err := branding.Assets.ReadFile(name)
			if err != nil {
				fail(w, 500, "Logo nicht verfügbar")
				return
			}
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(200)
			if r.Method != http.MethodHead {
				_, _ = w.Write(data)
			}

		case r.Method == http.MethodGet || r.Method == http.MethodHead:
			name := map[string]string{"/": "index.html", "/app.js": "app.js", "/style.css": "style.css"}[r.URL.Path]
			if name == "" {
				fail(w, 404, "Nicht gefunden")
				return
			}
			data, err := web.ReadFile("web/" + name)
			if err != nil {
				fail(w, 500, "Oberfläche nicht verfügbar")
				return
			}
			types := map[string]string{"index.html": "text/html; charset=utf-8", "app.js": "text/javascript; charset=utf-8", "style.css": "text/css; charset=utf-8"}
			w.Header().Set("Content-Type", types[name])
			w.WriteHeader(200)
			if r.Method != http.MethodHead {
				_, _ = w.Write(data)
			}

		default:
			fail(w, 405, "Methode nicht erlaubt")
		}
	})
}

func (a *app) list() ([]page, error) {
	entries, err := os.ReadDir(filepath.Join(a.dir, "meta"))
	if err != nil {
		return nil, err
	}
	pages := make([]page, 0, len(entries))
	for _, e := range entries {
		id := strings.TrimSuffix(e.Name(), ".json")
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || !idRE.MatchString(id) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(a.dir, "meta", e.Name()))
		if err != nil {
			return nil, err
		}
		var p page
		if err := json.Unmarshal(data, &p); err != nil || p.ID != id {
			return nil, fmt.Errorf("invalid metadata: %s", id)
		}
		if _, err := os.Stat(filepath.Join(a.dir, "public", id+".html")); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return nil, err
		}
		if p.Slug != "" && !validSlug(p.Slug) {
			return nil, fmt.Errorf("invalid slug: %s", id)
		}
		a.pageLinks(&p)
		pages = append(pages, p)
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Created.After(pages[j].Created) })
	return pages, nil
}

func validName(name string) bool {
	return utf8.ValidString(name) && len(name) > 0 && len(name) <= 180 &&
		!strings.ContainsAny(name, "/\\\x00\r\n") &&
		(strings.HasSuffix(strings.ToLower(name), ".html") || strings.HasSuffix(strings.ToLower(name), ".htm"))
}

func atomicWrite(dir, name string, data []byte) error {
	f, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if err = f.Chmod(0644); err != nil {
		f.Close()
		return err
	}
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(dir, name))
}

func (a *app) upload(w http.ResponseWriter, r *http.Request, user *AuthUser) {
	select {
	case a.uploadSlots <- struct{}{}:
		defer func() { <-a.uploadSlots }()
	default:
		fail(w, 429, "Ein anderer Import läuft bereits")
		return
	}
	name, err := url.PathUnescape(r.Header.Get("X-File-Name"))
	if err != nil || !validName(name) {
		fail(w, 400, "Nur .html/.htm-Dateien mit einfachem Dateinamen sind erlaubt")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, a.maxBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			fail(w, 413, "Datei überschreitet das Upload-Limit")
		} else {
			fail(w, 400, "Datei konnte nicht gelesen werden")
		}
		return
	}
	if len(body) == 0 || !utf8.Valid(body) || strings.ContainsRune(string(body), 0) || !htmlRE.Match(body) {
		fail(w, 400, "UTF-8 HTML mit <!doctype html> oder <html> erforderlich")
		return
	}
	profile := SecurityProfile(r.Header.Get("X-Security-Profile"))
	if profile == "" {
		profile = ProfileInteractive
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	// Quota-Prüfung
	if quotaErr := a.checkQuotaAddition(int64(len(body))); quotaErr != nil {
		fail(w, 413, quotaErr.Error())
		return
	}

	pages, err := a.list()
	if err != nil {
		fail(w, 500, "Dateiliste konnte nicht geladen werden")
		return
	}
	if len(pages) >= a.maxPages {
		fail(w, 409, "Maximale Anzahl veröffentlichter Seiten erreicht")
		return
	}
	random := make([]byte, 16)
	if _, err := rand.Read(random); err != nil {
		fail(w, 500, "ID konnte nicht erzeugt werden")
		return
	}
	id := hex.EncodeToString(random)
	hash := sha256.Sum256(body)
	p := page{
		ID:             id,
		Name:           name,
		Size:           int64(len(body)),
		SHA256:         hex.EncodeToString(hash[:]),
		Created:        time.Now().UTC(),
		URL:            a.publicURL + "/pages/" + id + ".html",
		Profile:        profile,
		CanonicalURL:   a.publicURL + "/pages/" + id + ".html",
		CurrentVersion: 1,
		Visibility:     VisibilityPublic,
	}
	a.pageLinks(&p)
	metadata, _ := json.Marshal(p)
	if err := atomicWrite(filepath.Join(a.dir, "meta"), id+".json", metadata); err != nil {
		fail(w, 500, "Metadaten konnten nicht gespeichert werden")
		return
	}
	if err := atomicWrite(filepath.Join(a.dir, "public"), id+".html", body); err != nil {
		_ = os.Remove(filepath.Join(a.dir, "meta", id+".json"))
		fail(w, 500, "Datei konnte nicht gespeichert werden")
		return
	}
	_ = a.renderPublicIndex()

	a.logAudit(AuditEntry{
		Timestamp: time.Now().UTC(),
		User:      user.Username,
		Role:      user.Role,
		Action:    AuditUpload,
		PageID:    id,
		Details:   fmt.Sprintf("Neu hochgeladen: %s (%d Bytes)", name, p.Size),
		ClientIP:  r.RemoteAddr,
	})

	jsonReply(w, 201, p)
}

func (a *app) remove(w http.ResponseWriter, r *http.Request, user *AuthUser) {
	id := strings.TrimPrefix(r.URL.Path, "/api/pages/")
	if !idRE.MatchString(id) {
		fail(w, 400, "Ungültige Datei-ID")
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.removePublishedFiles(id); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fail(w, 404, "Datei nicht gefunden")
		} else {
			fail(w, 500, "Löschen fehlgeschlagen")
		}
		return
	}
	if err := os.Remove(filepath.Join(a.dir, "meta", id+".json")); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("metadata cleanup: %v", err)
	}
	_ = os.RemoveAll(filepath.Join(a.dir, "versions", id))
	_ = a.renderPublicIndex()

	a.logAudit(AuditEntry{
		Timestamp: time.Now().UTC(),
		User:      user.Username,
		Role:      user.Role,
		Action:    AuditDelete,
		PageID:    id,
		Details:   "Seite und zugehörige Versionen gelöscht",
		ClientIP:  r.RemoteAddr,
	})

	w.WriteHeader(http.StatusNoContent)
}
