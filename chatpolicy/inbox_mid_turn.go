package chatpolicy

import "strings"

// MidTurnInboxEnabled resolves agent inbox (I40) mid-turn gate: deployment env OR tenant policy.
// Env: INBOX_MID_TURN_ENABLED=true forces on for all tenants in this deployment.
// When env is false, tenant policy chat.inbox_mid_turn_enabled applies (default true if unset).
func MidTurnInboxEnabled(envEnabled bool, policy map[string]interface{}) bool {
	if envEnabled {
		return true
	}
	if policy == nil {
		return false
	}
	return readPolicyBoolDefault(policy, "chat", "inbox_mid_turn_enabled", true)
}

func readPolicyBool(policy map[string]interface{}, section, key string) *bool {
	if policy == nil {
		return nil
	}
	rawSection, ok := policy[section]
	if !ok {
		return nil
	}
	m, ok := rawSection.(map[string]interface{})
	if !ok {
		return nil
	}
	raw, ok := m[key]
	if !ok {
		return nil
	}
	switch v := raw.(type) {
	case bool:
		return &v
	case string:
		s := strings.TrimSpace(strings.ToLower(v))
		if s == "true" || s == "1" || s == "yes" {
			return ptrBool(true)
		}
		if s == "false" || s == "0" || s == "no" {
			return ptrBool(false)
		}
	}
	return nil
}

func readPolicyBoolDefault(policy map[string]interface{}, section, key string, fallback bool) bool {
	if v := readPolicyBool(policy, section, key); v != nil {
		return *v
	}
	return fallback
}

func ptrBool(v bool) *bool { return &v }
