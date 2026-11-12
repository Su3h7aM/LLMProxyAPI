package usagedb

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode/utf8"

	coreusage "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/usage"
)

const (
	// maxFailSummaryBytes caps stored failure text. No bodies are stored.
	maxFailSummaryBytes = 4096

	// millisPerHour truncates event timestamps to hour buckets.
	millisPerHour = int64(3600 * 1000)
)

// UsageRow is one redacted usage record ready for storage. Cost fields stay
// NULL until the price book lands (issue 0044). The raw API key, request and
// response bodies, and header values never enter a row.
type UsageRow struct {
	RequestID        string
	TraceID          string
	TsMs             int64
	HourMs           int64
	Provider         string
	Model            string
	Alias            string
	Executor         string
	AuthID           string
	AuthIndex        string
	AuthType         string
	AccountLabel     string
	APIKeyID         string
	AccessProvider   string
	SessionID        string
	Stream           bool
	StatusClass      string
	Failed           bool
	FailCode         string
	FailSummary      string
	LatencyMs        int64
	TTFTMs           int64
	InputTokens      int64
	OutputTokens     int64
	ReasoningTokens  int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	TotalTokens      int64
	ServiceTier      string
	ReasoningEffort  string
	CostMicroUSD     *int64
	CostSource       *string
}

// RecordToRow converts a usage record to a storage row. It is pure: the same
// record always gives the same row.
func RecordToRow(record coreusage.Record) UsageRow {
	key := strings.TrimSpace(record.APIKey)
	row := UsageRow{
		RequestID:        strings.TrimSpace(record.RequestID),
		TraceID:          strings.TrimSpace(record.TraceID),
		TsMs:             record.RequestedAt.UnixMilli(),
		Provider:         orUnknown(record.Provider),
		Model:            orUnknown(record.Model),
		Executor:         orUnknown(record.ExecutorType),
		AuthID:           strings.TrimSpace(record.AuthID),
		AuthIndex:        strings.TrimSpace(record.AuthIndex),
		AuthType:         orUnknown(record.AuthType),
		AccountLabel:     maskAccountLabel(record.Source, key),
		APIKeyID:         apiKeyID(key),
		AccessProvider:   strings.TrimSpace(record.AccessProvider),
		SessionID:        strings.TrimSpace(record.SessionID),
		Stream:           record.Stream,
		StatusClass:      statusClass(record),
		Failed:           record.Failed,
		FailCode:         strings.TrimSpace(record.Fail.Code),
		FailSummary:      failSummary(record.Fail.Body, key),
		LatencyMs:        durationMs(record.Latency.Milliseconds()),
		TTFTMs:           durationMs(record.TTFT.Milliseconds()),
		InputTokens:      record.Detail.InputTokens,
		OutputTokens:     record.Detail.OutputTokens,
		ReasoningTokens:  record.Detail.ReasoningTokens,
		CacheReadTokens:  record.Detail.CacheReadTokens,
		CacheWriteTokens: record.Detail.CacheCreationTokens,
		TotalTokens:      record.Detail.TotalTokens,
		ServiceTier:      serviceTier(record),
		ReasoningEffort:  strings.TrimSpace(record.ReasoningEffort),
	}
	row.Alias = strings.TrimSpace(record.Alias)
	if row.Alias == "" {
		row.Alias = row.Model
	}
	row.HourMs = row.TsMs / millisPerHour * millisPerHour
	return row
}

// apiKeyID returns the first 16 hex of sha256(key), or empty when no key.
func apiKeyID(key string) string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])[:16]
}

// maskAccountLabel masks the source/account label. Email becomes ab***@domain.
// Long token-like text is cut to head...tail. A source that is or contains
// the raw client key becomes empty: no key fragment may enter the row.
func maskAccountLabel(source, apiKey string) string {
	trimmed := strings.TrimSpace(source)
	if trimmed == "" {
		return ""
	}
	if apiKey != "" && strings.Contains(trimmed, apiKey) {
		return ""
	}
	if at := strings.Index(trimmed, "@"); at > 0 && !strings.ContainsAny(trimmed, " \t\r\n/\\") {
		head := trimmed[:at]
		if len(head) > 2 {
			head = head[:2]
		}
		return head + "***@" + trimmed[at+1:]
	}
	if looksTokenLike(trimmed) {
		if len(trimmed) <= 8 {
			return "m:****"
		}
		return "m:" + trimmed[:4] + "..." + trimmed[len(trimmed)-4:]
	}
	return trimmed
}

// looksTokenLike reports text that must not stay in full: long strings
// without separators, or known key prefixes.
func looksTokenLike(value string) bool {
	if strings.HasPrefix(value, "sk-") || strings.HasPrefix(value, "AIza") {
		return true
	}
	if strings.ContainsAny(value, " \t\r\n/\\") {
		return false
	}
	return len(value) >= 32
}

// failSummary cuts failure text to 4096 bytes and removes the raw client key.
func failSummary(body, apiKey string) string {
	trimmed := strings.TrimSpace(body)
	if apiKey != "" && strings.Contains(trimmed, apiKey) {
		trimmed = strings.ReplaceAll(trimmed, apiKey, "[redacted]")
	}
	return truncateUTF8Bytes(strings.TrimSpace(trimmed), maxFailSummaryBytes)
}

func truncateUTF8Bytes(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	limit := maxBytes
	for limit > 0 && !utf8.ValidString(value[:limit]) {
		limit--
	}
	return strings.TrimSpace(value[:limit])
}

// statusClass maps the failure status: 2xx ok, 4xx, 5xx, other.
// Success (Failed == false) is 2xx; the record carries no success status code.
func statusClass(record coreusage.Record) string {
	if !record.Failed {
		return "2xx"
	}
	switch code := record.Fail.StatusCode; {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return "other"
	}
}

func serviceTier(record coreusage.Record) string {
	if tier := strings.TrimSpace(record.EffectiveServiceTier); tier != "" {
		return tier
	}
	if tier := strings.TrimSpace(record.ServiceTier); tier != "" {
		return tier
	}
	return coreusage.DefaultServiceTier
}

func orUnknown(value string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return "unknown"
}

func durationMs(value int64) int64 {
	if value < 0 {
		return 0
	}
	return value
}
