package executor

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/MoeForge/moego/ojson"
	"github.com/Su3h7aM/LLMProxyAPI/internal/config"
	"github.com/Su3h7aM/LLMProxyAPI/internal/runtime/executor/helps"
	cliproxyauth "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/auth"
	cliproxyexecutor "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/executor"
	sdktranslator "github.com/Su3h7aM/LLMProxyAPI/sdk/translator"
)

const claudeNativeTTLBetas = "claude-code-20250219,oauth-2025-04-20,interleaved-thinking-2025-05-14"

// nativeTTLCase describes one confirmed native Claude Code request. Each
// cache_control field is a ttl token: "" is a bare marker, "5m" and "1h" are
// explicit. want holds the expected ttl token for the system block and then
// for each user content block, in evaluation order.
type nativeTTLCase struct {
	name         string
	option       bool
	oauth        bool
	subagent     bool
	subagent1h   bool
	probe        bool
	explicitMode bool
	system       string
	content      []string
	want         []string
}

func nativeTTLMarker(ttl string) string {
	if ttl == "" {
		return `{"type":"ephemeral"}`
	}
	return `{"type":"ephemeral","ttl":"` + ttl + `"}`
}

// nativeTTLSystemTTL returns the ttl of the caller system block. Native requests
// gain injected system blocks, so the block is located by its text.
func nativeTTLSystemTTL(body []byte) string {
	for _, block := range ojson.GetBytes(body, "system").Array() {
		if block.Get("text").String() == "You are terse." {
			return block.Get("cache_control.ttl").String()
		}
	}
	return "<missing system block>"
}

func nativeTTLBody(c nativeTTLCase) []byte {
	maxTokens, text := 1024, "hello"
	if c.probe {
		maxTokens, text = 1, "quota"
	}
	blocks := make([]string, 0, len(c.content))
	for _, ttl := range c.content {
		blocks = append(blocks, `{"type":"text","text":"`+text+`","cache_control":`+nativeTTLMarker(ttl)+`}`)
	}
	body := `{"model":"claude-sonnet-5","max_tokens":` + strconv.Itoa(maxTokens) +
		`,"metadata":{"user_id":` + strconv.Quote(claudeNativeHelperUserID) + `}` +
		`,"system":[{"type":"text","text":"You are terse.","cache_control":` + nativeTTLMarker(c.system) + `}]` +
		`,"messages":[{"role":"user","content":[` + strings.Join(blocks, ",") + `]}]`
	if c.explicitMode {
		body += `,"prompt_cache_options":{"mode":"explicit"}`
	}
	return []byte(body + `}`)
}

// runNativeTTLCase sends c through Execute or ExecuteStream and returns the
// upstream body and Anthropic-Beta header.
func runNativeTTLCase(t *testing.T, c nativeTTLCase, stream bool) ([]byte, string) {
	t.Helper()
	var upstreamBody []byte
	var upstreamBeta string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamBody, _ = io.ReadAll(r.Body)
		upstreamBeta = r.Header.Get("Anthropic-Beta")
		if stream {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_1\",\"type\":\"message\",\"role\":\"assistant\",\"model\":\"claude-sonnet-5\"}}\n\nevent: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"msg_1","type":"message","role":"assistant","model":"claude-sonnet-5","content":[{"type":"text","text":"ok"}],"usage":{"input_tokens":1,"output_tokens":1}}`))
	}))
	defer server.Close()

	var auth *cliproxyauth.Auth
	if c.oauth {
		auth = claudeNativeHelperOAuthAuth(server.URL)
	} else {
		auth = &cliproxyauth.Auth{
			ID: "native-ttl-api-key",
			Attributes: map[string]string{
				"api_key":  "sk-ant-api03-native-ttl",
				"base_url": server.URL,
			},
		}
	}

	betas := claudeNativeTTLBetas
	if c.subagent1h {
		betas += ",extended-cache-ttl-2025-04-11"
	}
	headers := claudeNativeHelperHeaders(betas, "gzip, deflate, br, zstd")
	if c.subagent {
		headers.Set("X-Claude-Code-Agent-Id", "agent-native-ttl")
	}
	payload := nativeTTLBody(c)
	cfg := &config.Config{Claude: config.ClaudeConfig{NativeOAuthPromptCache1h: c.option}}
	if det := helps.DetectClaudeCodeRequest(headers, payload, false, cfg); !det.Confirmed {
		t.Fatalf("precondition: request is not a confirmed native Claude Code request")
	}

	executor := NewClaudeExecutor(cfg)
	req := cliproxyexecutor.Request{Model: "claude-sonnet-5", Payload: payload}
	opts := cliproxyexecutor.Options{
		SourceFormat:    sdktranslator.FormatClaude,
		OriginalRequest: payload,
		Headers:         headers,
	}
	if stream {
		result, err := executor.ExecuteStream(context.Background(), auth, req, opts)
		if err != nil {
			t.Fatalf("ExecuteStream() error = %v", err)
		}
		if result != nil && result.Chunks != nil {
			for range result.Chunks {
			}
		}
	} else if _, err := executor.Execute(context.Background(), auth, req, opts); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if upstreamBody == nil {
		t.Fatal("upstream received no request body")
	}
	return upstreamBody, upstreamBeta
}

func TestClaudeNativeOAuthPromptCache1h(t *testing.T) {
	cases := []nativeTTLCase{
		{
			name:    "option on upgrades bare markers on OAuth",
			option:  true,
			oauth:   true,
			system:  "",
			content: []string{"", ""},
			want:    []string{"1h", "1h", "1h"},
		},
		{
			name:    "option off keeps bare markers on OAuth",
			option:  false,
			oauth:   true,
			system:  "",
			content: []string{"", ""},
			want:    []string{"", "", ""},
		},
		{
			name:    "option on keeps API key credentials unchanged",
			option:  true,
			oauth:   false,
			system:  "",
			content: []string{"", ""},
			want:    []string{"", "", ""},
		},
		{
			name:    "explicit 5m system marker keeps later bare markers bare",
			option:  true,
			oauth:   true,
			system:  "5m",
			content: []string{"", ""},
			want:    []string{"5m", "", ""},
		},
		{
			name:    "explicit 5m content marker keeps later bare marker bare",
			option:  true,
			oauth:   true,
			system:  "",
			content: []string{"5m", ""},
			want:    []string{"1h", "5m", ""},
		},
		{
			name:    "explicit 1h marker stays 1h",
			option:  true,
			oauth:   true,
			system:  "",
			content: []string{"1h", ""},
			want:    []string{"1h", "1h", "1h"},
		},
		{
			name:       "subagent with 1h beta is upgraded",
			option:     true,
			oauth:      true,
			subagent:   true,
			subagent1h: true,
			system:     "",
			content:    []string{"", ""},
			want:       []string{"1h", "1h", "1h"},
		},
		{
			name:     "subagent without 1h keeps 5m",
			option:   true,
			oauth:    true,
			subagent: true,
			system:   "",
			content:  []string{"", ""},
			want:     []string{"", "", ""},
		},
		{
			name:    "probe keeps 5m",
			option:  true,
			oauth:   true,
			probe:   true,
			system:  "",
			content: []string{""},
			want:    []string{"", ""},
		},
		{
			name:         "explicit prompt cache mode keeps caller markers",
			option:       true,
			oauth:        true,
			explicitMode: true,
			system:       "",
			content:      []string{"", ""},
			want:         []string{"", "", ""},
		},
	}

	for _, c := range cases {
		for _, stream := range []bool{false, true} {
			mode := "execute"
			if stream {
				mode = "stream"
			}
			t.Run(c.name+"/"+mode, func(t *testing.T) {
				body, beta := runNativeTTLCase(t, c, stream)

				got := []string{nativeTTLSystemTTL(body)}
				for i := range c.content {
					got = append(got, ojson.GetBytes(body, "messages.0.content."+strconv.Itoa(i)+".cache_control.ttl").String())
				}
				has1h := false
				for i := range got {
					if got[i] != c.want[i] {
						t.Fatalf("ttl token %d = %q, want %q; body: %s", i, got[i], c.want[i], body)
					}
					has1h = has1h || c.want[i] == "1h"
				}
				if has1h && !strings.Contains(beta, claudeExtendedCacheTTLBeta) {
					t.Fatalf("1h ttl sent without %s beta; Anthropic-Beta = %q", claudeExtendedCacheTTLBeta, beta)
				}
			})
		}
	}
}
