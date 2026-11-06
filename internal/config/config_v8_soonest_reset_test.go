package config

import "testing"

// The v8 save path checks section names and API-key families only. It does not
// check the value of routing.strategy, so the new value must pass unchanged.
func TestValidateV8ConfigAcceptsSoonestResetStrategy(t *testing.T) {
	for _, strategy := range []string{"soonest-reset", "soonestreset"} {
		data := []byte("config-version: 8\nrouting: {strategy: " + strategy + "}\n")
		if err := ValidateV8Config(data); err != nil {
			t.Fatalf("ValidateV8Config(strategy=%q) error = %v", strategy, err)
		}
	}
}
