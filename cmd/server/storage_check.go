package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type StorageIssueType string

const (
	IssueMissingOriginal StorageIssueType = "MISSING_ORIGINAL"
	IssueMissingMeta     StorageIssueType = "MISSING_META"
	IssueSizeMismatch    StorageIssueType = "SIZE_MISMATCH"
	IssueChecksumMismatch StorageIssueType = "CHECKSUM_MISMATCH"
	IssueMissingLink     StorageIssueType = "MISSING_LINK"
	IssueBrokenLink      StorageIssueType = "BROKEN_LINK"
	IssueOrphanFile      StorageIssueType = "ORPHAN_FILE"
	IssueCorruptMeta     StorageIssueType = "CORRUPT_META"
)

type StorageIssue struct {
	Type        StorageIssueType `json:"type"`
	ID          string           `json:"id,omitempty"`
	Path        string           `json:"path"`
	Description string           `json:"description"`
	CanFix      bool             `json:"canFix"`
	Fixed       bool             `json:"fixed,omitempty"`
	FixAction   string           `json:"fixAction,omitempty"`
}

type StorageReport struct {
	CheckedAt       time.Time      `json:"checkedAt"`
	DataDir         string         `json:"dataDir"`
	MetaCount       int            `json:"metaCount"`
	PublicFileCount int            `json:"publicFileCount"`
	Issues          []StorageIssue `json:"issues"`
	FixedCount      int            `json:"fixedCount"`
}

func (r *StorageReport) HasErrors() bool {
	return len(r.Issues) > 0
}

func (r *StorageReport) AddIssue(issue StorageIssue) {
	r.Issues = append(r.Issues, issue)
}

func checkStorage(dataDir string) (*StorageReport, error) {
	report := &StorageReport{
		CheckedAt: time.Now().UTC(),
		DataDir:   dataDir,
		Issues:    make([]StorageIssue, 0),
	}

	metaDir := filepath.Join(dataDir, "meta")
	publicDir := filepath.Join(dataDir, "public")

	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return nil, fmt.Errorf("meta-Verzeichnis nicht verfügbar: %w", err)
	}
	if err := os.MkdirAll(publicDir, 0755); err != nil {
		return nil, fmt.Errorf("public-Verzeichnis nicht verfügbar: %w", err)
	}

	metaEntries, err := os.ReadDir(metaDir)
	if err != nil {
		return nil, fmt.Errorf("meta-Verzeichnis konnte nicht gelesen werden: %w", err)
	}

	knownPublicFiles := make(map[string]bool)
	knownPublicFiles["index.html"] = true

	pagesByID := make(map[string]page)

	for _, me := range metaEntries {
		if me.IsDir() || !strings.HasSuffix(me.Name(), ".json") {
			continue
		}
		id := strings.TrimSuffix(me.Name(), ".json")
		if !idRE.MatchString(id) {
			report.AddIssue(StorageIssue{
				Type:        IssueCorruptMeta,
				ID:          id,
				Path:        filepath.Join("meta", me.Name()),
				Description: fmt.Sprintf("Metadaten-Dateiname '%s' entspricht nicht dem ID-Schema (32 Hex-Zeichen)", me.Name()),
				CanFix:      false,
			})
			continue
		}

		report.MetaCount++
		metaPath := filepath.Join(metaDir, me.Name())
		metaBytes, err := os.ReadFile(metaPath)
		if err != nil {
			report.AddIssue(StorageIssue{
				Type:        IssueCorruptMeta,
				ID:          id,
				Path:        filepath.Join("meta", me.Name()),
				Description: fmt.Sprintf("Metadaten konnten nicht gelesen werden: %v", err),
				CanFix:      false,
			})
			continue
		}

		var p page
		if err := json.Unmarshal(metaBytes, &p); err != nil {
			report.AddIssue(StorageIssue{
				Type:        IssueCorruptMeta,
				ID:          id,
				Path:        filepath.Join("meta", me.Name()),
				Description: fmt.Sprintf("Metadaten-JSON ungültig: %v", err),
				CanFix:      false,
			})
			continue
		}
		if p.ID != id {
			report.AddIssue(StorageIssue{
				Type:        IssueCorruptMeta,
				ID:          id,
				Path:        filepath.Join("meta", me.Name()),
				Description: fmt.Sprintf("Metadaten-ID (%s) stimmt nicht mit Dateiname (%s) überein", p.ID, id),
				CanFix:      false,
			})
			continue
		}

		pagesByID[id] = p
		originalFileName := id + ".html"
		knownPublicFiles[originalFileName] = true
		origPath := filepath.Join(publicDir, originalFileName)

		origStat, err := os.Stat(origPath)
		if errors.Is(err, os.ErrNotExist) {
			report.AddIssue(StorageIssue{
				Type:        IssueMissingOriginal,
				ID:          id,
				Path:        filepath.Join("public", originalFileName),
				Description: fmt.Sprintf("Zugehörige Originaldatei fehlt für Seite '%s' (ID %s)", p.Name, id),
				CanFix:      false,
			})
			continue
		} else if err != nil {
			report.AddIssue(StorageIssue{
				Type:        IssueMissingOriginal,
				ID:          id,
				Path:        filepath.Join("public", originalFileName),
				Description: fmt.Sprintf("Originaldatei konnte nicht gelesen werden: %v", err),
				CanFix:      false,
			})
			continue
		}

		if origStat.Size() != p.Size {
			report.AddIssue(StorageIssue{
				Type:        IssueSizeMismatch,
				ID:          id,
				Path:        filepath.Join("public", originalFileName),
				Description: fmt.Sprintf("Dateigröße weicht ab: Datei hat %d Bytes, Metadaten verzeichnen %d Bytes", origStat.Size(), p.Size),
				CanFix:      false,
			})
		}

		actualHash, err := hashFile(origPath)
		if err != nil {
			report.AddIssue(StorageIssue{
				Type:        IssueChecksumMismatch,
				ID:          id,
				Path:        filepath.Join("public", originalFileName),
				Description: fmt.Sprintf("SHA-256 Prüfsumme konnte nicht berechnet werden: %v", err),
				CanFix:      false,
			})
		} else if actualHash != p.SHA256 {
			report.AddIssue(StorageIssue{
				Type:        IssueChecksumMismatch,
				ID:          id,
				Path:        filepath.Join("public", originalFileName),
				Description: fmt.Sprintf("SHA-256 Prüfsumme ungültig: Datei=%s, Metadaten=%s", actualHash, p.SHA256),
				CanFix:      false,
			})
		}

		if p.Slug != "" {
			slugFileName := p.Slug + ".html"
			knownPublicFiles[slugFileName] = true
			slugPath := filepath.Join(publicDir, slugFileName)

			slugStat, err := os.Stat(slugPath)
			if errors.Is(err, os.ErrNotExist) {
				report.AddIssue(StorageIssue{
					Type:        IssueMissingLink,
					ID:          id,
					Path:        filepath.Join("public", slugFileName),
					Description: fmt.Sprintf("Eigener Link (Hardlink) '%s' fehlt für Seite '%s'", slugFileName, p.Name),
					CanFix:      true,
					FixAction:   fmt.Sprintf("Hardlink von %s nach %s wiederherstellen", originalFileName, slugFileName),
				})
			} else if err == nil {
				if !os.SameFile(origStat, slugStat) {
					report.AddIssue(StorageIssue{
						Type:        IssueBrokenLink,
						ID:          id,
						Path:        filepath.Join("public", slugFileName),
						Description: fmt.Sprintf("Eigener Link '%s' verweist nicht auf dieselbe Inode wie die Originaldatei (kein echter Hardlink)", slugFileName),
						CanFix:      true,
						FixAction:   fmt.Sprintf("Datei %s durch echten Hardlink auf %s ersetzen", slugFileName, originalFileName),
					})
				}
			}
		}
	}

	publicEntries, err := os.ReadDir(publicDir)
	if err != nil {
		return nil, fmt.Errorf("public-Verzeichnis konnte nicht gelesen werden: %w", err)
	}

	for _, pe := range publicEntries {
		if pe.IsDir() {
			continue
		}
		name := pe.Name()
		report.PublicFileCount++

		if strings.HasSuffix(name, ".html") {
			base := strings.TrimSuffix(name, ".html")
			if idRE.MatchString(base) {
				if _, ok := pagesByID[base]; !ok {
					report.AddIssue(StorageIssue{
						Type:        IssueMissingMeta,
						ID:          base,
						Path:        filepath.Join("public", name),
						Description: fmt.Sprintf("HTML-Originaldatei '%s' existiert, aber es fehlen zugehörige Metadaten in meta/", name),
						CanFix:      true,
						FixAction:   fmt.Sprintf("Metadaten in meta/%s.json aus Dateidaten rekonstruieren", base),
					})
					knownPublicFiles[name] = true
				}
			}
		}

		if !knownPublicFiles[name] {
			report.AddIssue(StorageIssue{
				Type:        IssueOrphanFile,
				Path:        filepath.Join("public", name),
				Description: fmt.Sprintf("Verwaiste Datei '%s' in public/ ohne Bezug zu Metadaten oder Systemfunktion", name),
				CanFix:      true,
				FixAction:   fmt.Sprintf("Verwaiste Datei %s entfernen", filepath.Join("public", name)),
			})
		}
	}

	sort.Slice(report.Issues, func(i, j int) bool {
		return report.Issues[i].Path < report.Issues[j].Path
	})

	return report, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func repairStorage(dataDir string, report *StorageReport) error {
	publicDir := filepath.Join(dataDir, "public")
	metaDir := filepath.Join(dataDir, "meta")

	for i := range report.Issues {
		issue := &report.Issues[i]
		if !issue.CanFix {
			continue
		}

		switch issue.Type {
		case IssueMissingLink:
			if issue.ID != "" {
				origPath := filepath.Join(publicDir, issue.ID+".html")
				targetPath := filepath.Join(dataDir, issue.Path)
				if err := os.Link(origPath, targetPath); err == nil {
					issue.Fixed = true
					report.FixedCount++
				}
			}

		case IssueBrokenLink:
			if issue.ID != "" {
				origPath := filepath.Join(publicDir, issue.ID+".html")
				targetPath := filepath.Join(dataDir, issue.Path)
				_ = os.Remove(targetPath)
				if err := os.Link(origPath, targetPath); err == nil {
					issue.Fixed = true
					report.FixedCount++
				}
			}

		case IssueMissingMeta:
			if issue.ID != "" {
				htmlPath := filepath.Join(publicDir, issue.ID+".html")
				stat, err := os.Stat(htmlPath)
				if err == nil {
					hash, hashErr := hashFile(htmlPath)
					if hashErr == nil {
						p := page{
							ID:           issue.ID,
							Name:         issue.ID + ".html",
							Size:         stat.Size(),
							SHA256:       hash,
							Created:      stat.ModTime().UTC(),
							Profile:      ProfileInteractive,
							CanonicalURL: "/pages/" + issue.ID + ".html",
							URL:          "/pages/" + issue.ID + ".html",
						}
						data, mErr := json.MarshalIndent(p, "", "  ")
						if mErr == nil {
							if wErr := atomicWrite(metaDir, issue.ID+".json", data); wErr == nil {
								issue.Fixed = true
								report.FixedCount++
							}
						}
					}
				}
			}

		case IssueOrphanFile:
			filePath := filepath.Join(dataDir, issue.Path)
			if err := os.Remove(filePath); err == nil {
				issue.Fixed = true
				report.FixedCount++
			}
		}
	}

	return nil
}

func runStorageCheck(dataDir string, fix, confirm bool, in io.Reader, out io.Writer) (*StorageReport, error) {
	fmt.Fprintf(out, "=== Speicherprüfung: %s ===\n\n", dataDir)

	report, err := checkStorage(dataDir)
	if err != nil {
		return nil, err
	}

	fmt.Fprintf(out, "Geprüfte Metadaten: %d\n", report.MetaCount)
	fmt.Fprintf(out, "Geprüfte Dateien in public/: %d\n", report.PublicFileCount)
	fmt.Fprintf(out, "Gefundene Abweichungen: %d\n\n", len(report.Issues))

	if len(report.Issues) == 0 {
		fmt.Fprintf(out, "[OK] Datenbestand ist vollständig konsistent. Keine Fehler erkannt.\n")
		return report, nil
	}

	fmt.Fprintf(out, "%-18s | %-18s | %-30s | %s\n", "STATUS", "TYP", "PFAD", "DETAILS / REPARATUR-VORSCHAU")
	fmt.Fprintf(out, "%s\n", strings.Repeat("-", 90))

	repairableCount := 0
	for _, issue := range report.Issues {
		flag := "[FEHLER]"
		actionNote := ""
		if issue.CanFix {
			flag = "[REPARIERBAR]"
			actionNote = " -> Vorschau: " + issue.FixAction
			repairableCount++
		}
		fmt.Fprintf(out, "%-18s | %-18s | %-30s | %s%s\n", flag, issue.Type, issue.Path, issue.Description, actionNote)
	}
	fmt.Fprintln(out)

	if !fix {
		fmt.Fprintf(out, "Hinweis: Dies war ein reiner Prüflauf (Vorschau/Dry-Run).\n")
		if repairableCount > 0 {
			fmt.Fprintf(out, "%d Problem(e) können mit 'verify-storage --fix' automatisch behoben werden.\n", repairableCount)
		}
		return report, nil
	}

	if repairableCount == 0 {
		fmt.Fprintf(out, "Keine der gefundenen Abweichungen kann automatisch behoben werden.\n")
		return report, nil
	}

	if !confirm {
		fmt.Fprintf(out, "Möchten Sie die Reparatur für %d Problem(e) jetzt anwenden? [j/N]: ", repairableCount)
		var answer string
		if _, scanErr := fmt.Fscanln(in, &answer); scanErr != nil || (strings.ToLower(answer) != "j" && strings.ToLower(answer) != "ja" && strings.ToLower(answer) != "y" && strings.ToLower(answer) != "yes") {
			fmt.Fprintf(out, "Reparatur abgebrochen. Keine Änderungen durchgeführt.\n")
			return report, nil
		}
	}

	if err := repairStorage(dataDir, report); err != nil {
		return report, fmt.Errorf("Fehler bei der Reparatur: %w", err)
	}

	fmt.Fprintf(out, "\n[ERFOLG] Reparatur abgeschlossen: %d von %d reparierbaren Problemen behoben.\n", report.FixedCount, repairableCount)
	return report, nil
}
