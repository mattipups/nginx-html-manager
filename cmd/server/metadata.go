package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type MetadataUpdateRequest struct {
	Title       *string  `json:"title"`
	Description *string  `json:"description"`
	Tags        []string `json:"tags"`
	NewSlug     *string  `json:"newSlug"`
	RedirectOld bool     `json:"redirectOld"`
}

func (a *app) updateMetadata(w http.ResponseWriter, r *http.Request, user *AuthUser) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/pages/"), "/metadata")
	if !idRE.MatchString(id) {
		fail(w, 400, "Ungültige Datei-ID")
		return
	}

	var req MetadataUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		fail(w, 400, "Ungültiger JSON-Body")
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	p, err := a.storedPage(id)
	if err != nil {
		pageError(w, err)
		return
	}

	if req.Title != nil {
		p.Title = strings.TrimSpace(*req.Title)
	}
	if req.Description != nil {
		p.Description = strings.TrimSpace(*req.Description)
	}
	if req.Tags != nil {
		cleanTags := make([]string, 0, len(req.Tags))
		for _, t := range req.Tags {
			t = strings.TrimSpace(strings.ToLower(t))
			if t != "" && len(t) <= 30 {
				cleanTags = append(cleanTags, t)
			}
		}
		p.Tags = cleanTags
	}

	// Slug umbenennen mit optionaler Weiterleitung
	if req.NewSlug != nil && *req.NewSlug != p.Slug {
		newSlug := *req.NewSlug
		if newSlug != "" && !validSlug(newSlug) {
			fail(w, 400, "Ungültiger URL-Slug")
			return
		}

		oldSlug := p.Slug
		publicDir := filepath.Join(a.dir, "public")

		// Alten Hardlink entfernen oder zu HTML-Weiterleitung umwandeln
		if oldSlug != "" {
			oldPath := filepath.Join(publicDir, oldSlug+".html")
			_ = os.Remove(oldPath)

			if req.RedirectOld && newSlug != "" {
				targetURL := "/pages/" + newSlug + ".html"
				redirectHTML := fmt.Sprintf(`<!doctype html>
<html lang="de"><head><meta charset="utf-8">
<meta http-equiv="refresh" content="0; url=%s">
<title>Weiterleitung nach %s</title></head>
<body><p>Weiterleitung nach <a href="%s">%s</a> …</p></body></html>`, targetURL, targetURL, targetURL, targetURL)
				_ = os.WriteFile(oldPath, []byte(redirectHTML), 0644)
				p.Redirects = append(p.Redirects, oldSlug)
			}
		}

		// Neuen Hardlink anlegen
		if newSlug != "" {
			newPath := filepath.Join(publicDir, newSlug+".html")
			origPath := filepath.Join(publicDir, id+".html")
			if err := os.Link(origPath, newPath); err != nil {
				fail(w, 409, "URL-Slug bereits belegt")
				return
			}
		}

		p.Slug = newSlug
		a.pageLinks(&p)

		a.logAudit(AuditEntry{
			Timestamp: time.Now().UTC(),
			User:      user.Username,
			Role:      user.Role,
			Action:    AuditLink,
			PageID:    id,
			Details:   fmt.Sprintf("Slug geändert: '%s' -> '%s' (Weiterleitung: %v)", oldSlug, newSlug, req.RedirectOld),
			ClientIP:  r.RemoteAddr,
		})
	}

	metaBytes, _ := json.Marshal(p)
	_ = atomicWrite(filepath.Join(a.dir, "meta"), id+".json", metaBytes)
	_ = a.renderPublicIndex()

	a.logAudit(AuditEntry{
		Timestamp: time.Now().UTC(),
		User:      user.Username,
		Role:      user.Role,
		Action:    AuditMetadata,
		PageID:    id,
		Details:   fmt.Sprintf("Metadaten gepflegt: Titel='%s', Tags=%v", p.Title, p.Tags),
		ClientIP:  r.RemoteAddr,
	})

	jsonReply(w, 200, p)
}
