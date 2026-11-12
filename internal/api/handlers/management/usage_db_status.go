package management

import (
	"net/http"

	"github.com/Su3h7aM/LLMProxyAPI/internal/usagedb"
	"github.com/gin-gonic/gin"
)

// usageDBStatus is the JSON shape of the usage database status route.
type usageDBStatus struct {
	Enabled       bool   `json:"enabled"`
	Path          string `json:"path"`
	SizeBytes     int64  `json:"size_bytes"`
	SchemaVersion int    `json:"schema_version"`
	Events        int64  `json:"events"`
	OldestTsMs    int64  `json:"oldest_ts_ms"`
	NewestTsMs    int64  `json:"newest_ts_ms"`
	BufferDepth   int    `json:"buffer_depth"`
	BufferCap     int    `json:"buffer_capacity"`
	Dropped       int64  `json:"dropped"`
	WriteErrors   int64  `json:"write_errors"`
	LastError     string `json:"last_error"`
	RetentionDays int64  `json:"retention_days"`
	LastPruneMs   int64  `json:"last_prune_ts_ms"`
}

// GetUsageDBStatus reports the embedded usage database state.
func (h *Handler) GetUsageDBStatus(c *gin.Context) {
	if h == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "handler unavailable"})
		return
	}
	snapshot := usagedb.Status()
	c.JSON(http.StatusOK, usageDBStatus{
		Enabled:       snapshot.Enabled,
		Path:          snapshot.Path,
		SizeBytes:     snapshot.SizeBytes,
		SchemaVersion: snapshot.SchemaVersion,
		Events:        snapshot.Events,
		OldestTsMs:    snapshot.OldestTsMs,
		NewestTsMs:    snapshot.NewestTsMs,
		BufferDepth:   snapshot.BufferDepth,
		BufferCap:     snapshot.BufferCap,
		Dropped:       snapshot.Dropped,
		WriteErrors:   snapshot.WriteErrors,
		LastError:     snapshot.LastError,
		RetentionDays: snapshot.RetentionDays,
		LastPruneMs:   snapshot.LastPruneMs,
	})
}
