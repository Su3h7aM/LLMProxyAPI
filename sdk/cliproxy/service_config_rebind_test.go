package cliproxy

import (
	"testing"

	internalconfig "github.com/Su3h7aM/LLMProxyAPI/internal/config"
	coreauth "github.com/Su3h7aM/LLMProxyAPI/sdk/cliproxy/auth"
)

func TestNormalizedRoutingRuntimeStateSessionAffinityRebindOn(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		rebind string
		want   string
	}{
		{name: "unset defaults to quota", rebind: "", want: coreauth.SessionAffinityRebindOnQuota},
		{name: "unknown defaults to quota", rebind: "sometimes", want: coreauth.SessionAffinityRebindOnQuota},
		{name: "explicit quota", rebind: "quota", want: coreauth.SessionAffinityRebindOnQuota},
		{name: "explicit any", rebind: "any", want: coreauth.SessionAffinityRebindOnAny},
		{name: "explicit any with case and spaces", rebind: " ANY ", want: coreauth.SessionAffinityRebindOnAny},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			state := normalizedRoutingRuntimeState(&internalconfig.Config{Routing: internalconfig.RoutingConfig{
				SessionAffinity:         true,
				SessionAffinityRebindOn: testCase.rebind,
			}})
			if state.sessionAffinityRebindOn != testCase.want {
				t.Fatalf("sessionAffinityRebindOn = %q, want %q", state.sessionAffinityRebindOn, testCase.want)
			}
		})
	}
}
