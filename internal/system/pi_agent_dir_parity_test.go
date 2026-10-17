package system_test

import (
	"path/filepath"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/agents/pi"
	"github.com/gentleman-programming/gentle-ai/v4/internal/system"
)

// TestScanConfigsPiPathMatchesAgentConfigPath guards the replicated
// PI_CODING_AGENT_DIR resolution in ScanConfigs against drifting from
// pi.AgentConfigPath, which system cannot import (pi imports system).
func TestScanConfigsPiPathMatchesAgentConfigPath(t *testing.T) {
	for _, override := range []string{
		"~",
		"~/.gentle-shell/agent",
		filepath.Join(t.TempDir(), "isolated", "agent"),
		"relative/agent",
	} {
		t.Run(override, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("PI_CODING_AGENT_DIR", override)

			var got string
			for _, state := range system.ScanConfigs(home) {
				if state.Agent == "pi" {
					got = state.Path
				}
			}
			if want := pi.AgentConfigPath(home); got != want {
				t.Fatalf("ScanConfigs pi path = %q, want pi.AgentConfigPath %q", got, want)
			}
		})
	}
}
