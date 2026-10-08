package cli

import (
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/brand"
	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
	"github.com/gentleman-programming/gentle-ai/v4/internal/planner"
)

func TestRenderDryRunIncludesPlatformDecision(t *testing.T) {
	result := InstallResult{
		Selection: model.Selection{Persona: model.PersonaGentleman, Preset: model.PresetFullGentleman},
		Resolved: planner.ResolvedPlan{
			Agents:            []model.AgentID{model.AgentClaudeCode},
			OrderedComponents: []model.ComponentID{model.ComponentEngram},
		},
		Review: planner.ReviewPayload{
			PlatformDecision: planner.PlatformDecision{
				OS:             "linux",
				LinuxDistro:    "ubuntu",
				PackageManager: "apt",
				Supported:      true,
			},
		},
	}

	output := RenderDryRun(result)

	want := "Platform decision: os=linux distro=ubuntu package-manager=apt status=supported"
	if !strings.Contains(output, want) {
		t.Fatalf("RenderDryRun() missing platform decision\noutput=%s", output)
	}
}

func TestRenderDryRunHeaderFollowsBrand(t *testing.T) {
	t.Cleanup(func() { brand.Set(brand.Default()) })

	brand.Set(brand.Default())
	if output := RenderDryRun(InstallResult{}); !strings.HasPrefix(output, "Ordo dry-run\n============\n") {
		t.Fatalf("RenderDryRun() header = %q, want the Ordo dry-run title underlined", strings.SplitN(output, "\n", 3)[:2])
	}

	b := brand.Default()
	b.Name = "Acme"
	brand.Set(b)
	if output := RenderDryRun(InstallResult{}); !strings.HasPrefix(output, "Acme dry-run\n============\n") {
		t.Fatalf("RenderDryRun() header = %q, want the Acme dry-run title underlined", strings.SplitN(output, "\n", 3)[:2])
	}
}
