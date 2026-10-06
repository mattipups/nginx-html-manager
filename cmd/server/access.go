package main

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type Visibility string

const (
	VisibilityPublic        Visibility = "public"
	VisibilityAuthenticated Visibility = "authenticated"
	VisibilityRestricted    Visibility = "restricted"
)

type AccessPolicy struct {
	Visibility    Visibility `json:"visibility"`
	AllowedGroups []string   `json:"allowedGroups,omitempty"`
}

func (a *app) updateAccessPolicy(w http.ResponseWriter, r *http.Request, user *AuthUser) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/pages/"), "/access")
	if !idRE.MatchString(id) {
		fail(w, 400, "Ungültige Datei-ID")
		return
	}

	var policy AccessPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		fail(w, 400, "Ungültiges JSON")
		return
	}

	if policy.Visibility != VisibilityPublic && policy.Visibility != VisibilityAuthenticated && policy.Visibility != VisibilityRestricted {
		policy.Visibility = VisibilityPublic
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	p, err := a.storedPage(id)
	if err != nil {
		pageError(w, err)
		return
	}

	p.Visibility = policy.Visibility
	p.AllowedGroups = policy.AllowedGroups

	metaBytes, _ := json.Marshal(p)
	_ = atomicWrite(filepath.Join(a.dir, "meta"), id+".json", metaBytes)
	_ = a.renderPublicIndex()

	a.logAudit(AuditEntry{
		Timestamp: time.Now().UTC(),
		User:      user.Username,
		Role:      user.Role,
		Action:    AuditMetadata,
		PageID:    id,
		Details:   "Zugriffsrichtlinie geändert: " + string(policy.Visibility),
		ClientIP:  r.RemoteAddr,
	})

	jsonReply(w, 200, p)
}

func (a *app) checkPublicAccess(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("file")
	if target == "" {
		target = strings.TrimPrefix(r.URL.Path, "/verify-access/")
	}
	target = strings.TrimSuffix(target, ".html")

	a.mu.Lock()
	pages, _ := a.list()
	a.mu.Unlock()

	var matched *page
	for _, p := range pages {
		if p.ID == target || (p.Slug != "" && p.Slug == target) {
			matched = &p
			break
		}
	}

	// Falls unbekannt oder öffentlich -> Freigabe
	if matched == nil || matched.Visibility == "" || matched.Visibility == VisibilityPublic {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("public: access granted\n"))
		return
	}

	user, authenticated := a.authenticate(r)
	if !authenticated {
		w.Header().Set("WWW-Authenticate", `Basic realm="HTML-Access", charset="UTF-8"`)
		fail(w, 401, "Geschützte Seite: Anmeldung erforderlich")
		return
	}

	if matched.Visibility == VisibilityAuthenticated {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("authenticated: access granted\n"))
		return
	}

	if matched.Visibility == VisibilityRestricted {
		if user.Role == RoleAdmin {
			w.WriteHeader(200)
			return
		}
		for _, required := range matched.AllowedGroups {
			for _, userGroup := range user.Groups {
				if userGroup == required {
					w.WriteHeader(200)
					return
				}
			}
		}
		fail(w, 403, "Zugriff verweigert: Sie verfügen nicht über die erforderliche Gruppenberechtigung")
		return
	}

	w.WriteHeader(200)
}
