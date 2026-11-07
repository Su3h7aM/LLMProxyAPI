package management

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestLatestVersionUsesForkAndReportsMissingRelease(t *testing.T) {
	previous := http.DefaultTransport
	t.Cleanup(func() { http.DefaultTransport = previous })
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if got, want := req.URL.String(), "https://api.github.com/repos/Su3h7aM/LLMProxyAPI/releases/latest"; got != want {
			t.Fatalf("release URL = %q, want %q", got, want)
		}
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("not found")), Header: make(http.Header)}, nil
	})
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/server/latest-version", nil)
	(&Handler{}).GetLatestVersion(ctx)
	if response.Code != http.StatusBadGateway || !strings.Contains(response.Body.String(), `"error":"unexpected_status"`) {
		t.Fatalf("response = %d %s, want 502 unexpected_status", response.Code, response.Body.String())
	}
}

func TestSetLatestReleaseRequestHeaders(t *testing.T) {
	tests := []struct {
		name              string
		githubToken       string
		wantAuthorization string
	}{
		{
			name:              "sets GitHub authorization",
			githubToken:       "release-token",
			wantAuthorization: "Bearer release-token",
		},
		{
			name: "omits authorization without token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_TOKEN", tt.githubToken)
			if tt.githubToken == "" {
				t.Setenv("github_token", "")
			}
			t.Setenv("GITSTORE_GIT_TOKEN", "")
			t.Setenv("GITSTORE_GIT_URL", "")

			req := httptest.NewRequest(http.MethodGet, latestReleaseURL, nil)
			setLatestReleaseRequestHeaders(req)

			if got := req.Header.Get("Authorization"); got != tt.wantAuthorization {
				t.Fatalf("Authorization = %q, want %q", got, tt.wantAuthorization)
			}
			if got := req.Header.Get("Accept"); got != "application/vnd.github+json" {
				t.Fatalf("Accept = %q, want GitHub JSON media type", got)
			}
			if got := req.Header.Get("User-Agent"); got != latestReleaseUserAgent {
				t.Fatalf("User-Agent = %q, want %q", got, latestReleaseUserAgent)
			}
		})
	}
}
