package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

type StorageQuotaInfo struct {
	TotalBytes    int64   `json:"totalBytes"`
	PublicBytes   int64   `json:"publicBytes"`
	VersionsBytes int64   `json:"versionsBytes"`
	MetaBytes     int64   `json:"metaBytes"`
	MaxBytes      int64   `json:"maxBytes"`
	UsedPercent   float64 `json:"usedPercent"`
	WarnPercent   float64 `json:"warnPercent"`
	IsWarning     bool    `json:"isWarning"`
	IsExceeded    bool    `json:"isExceeded"`
}

func dirSize(path string) int64 {
	var total int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}

func (a *app) calculateStorageQuota() StorageQuotaInfo {
	publicBytes := dirSize(filepath.Join(a.dir, "public"))
	versionsBytes := dirSize(filepath.Join(a.dir, "versions"))
	metaBytes := dirSize(filepath.Join(a.dir, "meta"))
	total := publicBytes + versionsBytes + metaBytes

	maxBytes := positive("MAX_STORAGE_BYTES", 10*1024*1024*1024) // 10 GB Default
	warnPercent := float64(positive("STORAGE_WARN_PERCENT", 80))

	usedPct := 0.0
	if maxBytes > 0 {
		usedPct = (float64(total) / float64(maxBytes)) * 100.0
	}

	return StorageQuotaInfo{
		TotalBytes:    total,
		PublicBytes:   publicBytes,
		VersionsBytes: versionsBytes,
		MetaBytes:     metaBytes,
		MaxBytes:      maxBytes,
		UsedPercent:   usedPct,
		WarnPercent:   warnPercent,
		IsWarning:     usedPct >= warnPercent,
		IsExceeded:    total >= maxBytes,
	}
}

func (a *app) checkQuotaAddition(addition int64) error {
	quota := a.calculateStorageQuota()
	if quota.TotalBytes+addition > quota.MaxBytes {
		return fmt.Errorf("Speicherkontingent erschöpft: %d Bytes überschreiten das Limit von %d Bytes", quota.TotalBytes+addition, quota.MaxBytes)
	}
	return nil
}

func (a *app) getQuotaHandler(w http.ResponseWriter, _ *http.Request) {
	a.mu.Lock()
	defer a.mu.Unlock()
	quota := a.calculateStorageQuota()
	jsonReply(w, 200, quota)
}
