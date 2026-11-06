package cliproxy

import (
	"testing"

	internalconfig "github.com/router-for-me/LLMProxyAPI/v8/internal/config"
	coreauth "github.com/router-for-me/LLMProxyAPI/v8/sdk/cliproxy/auth"
)

func TestSoonestResetRoutingSelector(t *testing.T) {
	for _, strategy := range []string{"soonest-reset", "soonestreset", " Soonest-Reset "} {
		t.Run(strategy, func(t *testing.T) {
			state := normalizedRoutingRuntimeState(&internalconfig.Config{
				Routing: internalconfig.RoutingConfig{Strategy: strategy},
			})
			if state.strategy != "soonest-reset" {
				t.Fatalf("strategy = %q, want soonest-reset", state.strategy)
			}
			if _, ok := newRoutingSelector(state).(*coreauth.SoonestResetSelector); !ok {
				t.Fatalf("selector type = %T, want *auth.SoonestResetSelector", newRoutingSelector(state))
			}
		})
	}
}

func TestSoonestResetRoutingSelectorKeepsSessionAffinity(t *testing.T) {
	state := normalizedRoutingRuntimeState(&internalconfig.Config{
		Routing: internalconfig.RoutingConfig{Strategy: "soonest-reset", SessionAffinity: true},
	})
	affinity, ok := newRoutingSelector(state).(*coreauth.SessionAffinitySelector)
	if !ok {
		t.Fatalf("selector type = %T, want *auth.SessionAffinitySelector", newRoutingSelector(state))
	}
	t.Cleanup(affinity.Stop)
}
