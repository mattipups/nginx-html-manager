package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var slugRE = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$`)

func validSlug(slug string) bool { return slugRE.MatchString(slug) && !idRE.MatchString(slug) }

func (a *app) pageLinks(p *page) {
	p.CanonicalURL = a.publicURL + "/pages/" + p.ID + ".html"
	p.URL = p.CanonicalURL
	if p.Slug != "" {
		p.URL = a.publicURL + "/pages/" + p.Slug + ".html"
	}
}

func (a *app) storedPage(id string) (page, error) {
	var p page
	data, err := os.ReadFile(filepath.Join(a.dir, "meta", id+".json"))
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal(data, &p); err != nil {
		return p, err
	}
	if p.ID != id || !validName(p.Name) || (p.Slug != "" && !validSlug(p.Slug)) {
		return p, fmt.Errorf("invalid metadata")
	}
	if _, err := os.Stat(filepath.Join(a.dir, "public", id+".html")); err != nil {
		return p, err
	}
	a.pageLinks(&p)
	return p, nil
}

func pageError(w http.ResponseWriter, err error) {
	if errors.Is(err, os.ErrNotExist) {
		fail(w, 404, "Datei nicht gefunden")
	} else {
		fail(w, 500, "Dateidaten nicht verfügbar")
	}
}

func (a *app) changeLink(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/pages/"), "/link")
	if !idRE.MatchString(id) {
		fail(w, 400, "Ungültige Datei-ID")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1024))
	if err != nil {
		var limit *http.MaxBytesError
		if errors.As(err, &limit) {
			fail(w, 413, "Anfrage zu groß")
		} else {
			fail(w, 400, "Anfrage nicht lesbar")
		}
		return
	}
	var input struct {
		Slug *string `json:"slug"`
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || input.Slug == nil {
		fail(w, 400, "JSON mit slug erforderlich")
		return
	}
	if decoder.Decode(&struct{}{}) != io.EOF || (*input.Slug != "" && !validSlug(*input.Slug)) {
		fail(w, 400, "URL-Name: 1–64 Kleinbuchstaben/Ziffern/Bindestriche; keine ID, kein Bindestrich am Rand")
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	p, err := a.storedPage(id)
	if err != nil {
		pageError(w, err)
		return
	}
	if p.Slug == *input.Slug {
		jsonReply(w, 200, p)
		return
	}
	dir := filepath.Join(a.dir, "public")
	original := filepath.Join(dir, id+".html")
	oldSlug, newSlug := p.Slug, *input.Slug
	newPath := filepath.Join(dir, newSlug+".html")
	if newSlug != "" {
		if err := os.Link(original, newPath); err != nil {
			if errors.Is(err, os.ErrExist) {
				fail(w, 409, "URL-Name bereits belegt")
			} else {
				fail(w, 500, "Link konnte nicht erstellt werden")
			}
			return
		}
	}
	if oldSlug != "" {
		if err := os.Remove(filepath.Join(dir, oldSlug+".html")); err != nil && !errors.Is(err, os.ErrNotExist) {
			if newSlug != "" {
				_ = os.Remove(newPath)
			}
			fail(w, 500, "Bisheriger Link konnte nicht entfernt werden")
			return
		}
	}
	p.Slug = newSlug
	a.pageLinks(&p)
	metadata, err := json.Marshal(p)
	if err == nil {
		err = atomicWrite(filepath.Join(a.dir, "meta"), id+".json", metadata)
	}
	if err != nil {
		if newSlug != "" {
			_ = os.Remove(newPath)
		}
		if oldSlug != "" {
			if restore := os.Link(original, filepath.Join(dir, oldSlug+".html")); restore != nil {
				log.Printf("alias rollback: %v", restore)
			}
		}
		fail(w, 500, "Linkänderung konnte nicht gespeichert werden")
		return
	}
	_ = a.renderPublicIndex()
	jsonReply(w, 200, p)
}

func (a *app) download(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/pages/"), "/download")
	if !idRE.MatchString(id) {
		fail(w, 400, "Ungültige Datei-ID")
		return
	}
	a.mu.Lock()
	p, err := a.storedPage(id)
	if err != nil {
		a.mu.Unlock()
		pageError(w, err)
		return
	}
	file, err := os.Open(filepath.Join(a.dir, "public", id+".html"))
	a.mu.Unlock()
	if err != nil {
		pageError(w, err)
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		pageError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": p.Name}))
	http.ServeContent(w, r, p.Name, info.ModTime(), file)
}

func (a *app) removePublishedFiles(id string) error {
	p, err := a.storedPage(id)
	if err != nil {
		return err
	}
	dir := filepath.Join(a.dir, "public")
	original := filepath.Join(dir, id+".html")
	if p.Slug != "" {
		if err := os.Remove(filepath.Join(dir, p.Slug+".html")); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := os.Remove(original); err != nil {
		if p.Slug != "" {
			_ = os.Link(original, filepath.Join(dir, p.Slug+".html"))
		}
		return err
	}
	return nil
}
