package main

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

type AuditAction string

const (
	AuditUpload   AuditAction = "UPLOAD"
	AuditUpdate   AuditAction = "UPDATE"
	AuditRollback AuditAction = "ROLLBACK"
	AuditDelete   AuditAction = "DELETE"
	AuditLink     AuditAction = "LINK_CHANGE"
	AuditMetadata AuditAction = "METADATA_CHANGE"
)

type AuditEntry struct {
	Timestamp time.Time   `json:"timestamp"`
	User      string      `json:"user"`
	Role      Role        `json:"role"`
	Action    AuditAction `json:"action"`
	PageID    string      `json:"pageId,omitempty"`
	Details   string      `json:"details"`
	ClientIP  string      `json:"clientIp"`
}

var auditMutex sync.Mutex

func (a *app) logAudit(entry AuditEntry) {
	auditMutex.Lock()
	defer auditMutex.Unlock()

	logPath := filepath.Join(a.dir, "audit.log")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		return
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err == nil {
		_, _ = f.Write(append(data, '\n'))
	}
}

func (a *app) getAuditLogs(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 1000 {
			limit = val
		}
	}

	auditMutex.Lock()
	defer auditMutex.Unlock()

	logPath := filepath.Join(a.dir, "audit.log")
	file, err := os.Open(logPath)
	if err != nil {
		jsonReply(w, 200, []AuditEntry{})
		return
	}
	defer file.Close()

	var entries []AuditEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var e AuditEntry
		if err := json.Unmarshal(scanner.Bytes(), &e); err == nil {
			entries = append(entries, e)
		}
	}

	if len(entries) > limit {
		entries = entries[len(entries)-limit:]
	}

	// Neueste zuerst
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	jsonReply(w, 200, entries)
}
