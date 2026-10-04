package chatpolicy

import "testing"

func TestMidTurnInboxEnabled_EnvWins(t *testing.T) {
	if !MidTurnInboxEnabled(true, map[string]interface{}{
		"chat": map[string]interface{}{"inbox_mid_turn_enabled": false},
	}) {
		t.Fatal("env flag should force enabled")
	}
}

func TestMidTurnInboxEnabled_TenantPolicyWhenEnvOff(t *testing.T) {
	if MidTurnInboxEnabled(false, nil) {
		t.Fatal("nil policy with env off => disabled")
	}
	if !MidTurnInboxEnabled(false, map[string]interface{}{
		"chat": map[string]interface{}{"inbox_mid_turn_enabled": true},
	}) {
		t.Fatal("tenant policy should enable")
	}
}
