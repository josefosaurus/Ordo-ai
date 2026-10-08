package persona

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/agents"
	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
)

func readOutputStyleSetting(t *testing.T, path string) any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", path, err)
	}
	var settings map[string]any
	if err := json.Unmarshal(raw, &settings); err != nil {
		t.Fatalf("Unmarshal(%q) error = %v", path, err)
	}
	return settings["outputStyle"]
}

func TestInjectClaudeGentlemanSelectsMentorOutputStyle(t *testing.T) {
	home := t.TempDir()

	if _, err := Inject(home, claudeAdapter(), model.PersonaGentleman); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	style, err := os.ReadFile(filepath.Join(home, ".claude", "output-styles", "gentleman.md"))
	if err != nil {
		t.Fatalf("ReadFile(output style) error = %v", err)
	}
	if !strings.Contains(string(style), "name: Mentor\n") {
		t.Fatalf("output style frontmatter does not name Mentor:\n%s", style)
	}
	if got := readOutputStyleSetting(t, filepath.Join(home, ".claude", "settings.json")); got != "Mentor" {
		t.Fatalf("settings.json outputStyle = %v, want Mentor", got)
	}
}

// A machine installed by an earlier version carries outputStyle "Gentleman".
// Both install and sync must switch it to "Mentor" instead of leaving a value
// that no longer names any managed output style.
func TestInjectMigratesLegacyGentlemanOutputStyleToMentor(t *testing.T) {
	for _, tc := range []struct {
		name   string
		inject func(string, agents.Adapter, model.PersonaID) (InjectionResult, error)
	}{
		{name: "install", inject: Inject},
		{name: "sync", inject: InjectForSync},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			settingsDir := filepath.Join(home, ".claude")
			if err := os.MkdirAll(settingsDir, 0o755); err != nil {
				t.Fatalf("MkdirAll() error = %v", err)
			}
			settingsPath := filepath.Join(settingsDir, "settings.json")
			if err := os.WriteFile(settingsPath, []byte(`{"outputStyle":"Gentleman","theme":"dark"}`), 0o644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			result, err := tc.inject(home, claudeAdapter(), model.PersonaGentleman)
			if err != nil {
				t.Fatalf("inject error = %v", err)
			}
			if !result.Changed {
				t.Fatal("inject changed = false, want the legacy outputStyle migrated")
			}
			if got := readOutputStyleSetting(t, settingsPath); got != "Mentor" {
				t.Fatalf("outputStyle = %v, want Mentor", got)
			}
		})
	}
}

func TestRemoveManagedOutputStyleSetting(t *testing.T) {
	for _, tc := range []struct {
		name        string
		value       string
		wantRemoved bool
	}{
		{name: "legacy Gentleman", value: "Gentleman", wantRemoved: true},
		{name: "current Mentor", value: "Mentor", wantRemoved: true},
		{name: "user style", value: "MyCustom", wantRemoved: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "settings.json")
			if err := os.WriteFile(path, []byte(`{"outputStyle":"`+tc.value+`","theme":"dark"}`), 0o644); err != nil {
				t.Fatalf("WriteFile() error = %v", err)
			}

			removed, err := removeManagedOutputStyleSetting(path)
			if err != nil {
				t.Fatalf("removeManagedOutputStyleSetting() error = %v", err)
			}
			if removed != tc.wantRemoved {
				t.Fatalf("removed = %v, want %v", removed, tc.wantRemoved)
			}
			got := readOutputStyleSetting(t, path)
			if tc.wantRemoved && got != nil {
				t.Fatalf("outputStyle = %v, want removed", got)
			}
			if !tc.wantRemoved && got != tc.value {
				t.Fatalf("outputStyle = %v, want %q kept", got, tc.value)
			}
		})
	}
}

func TestInjectVSCodeWritesOrdoPersonaMarker(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	vscodeAdapter, err := agents.NewAdapter("vscode-copilot")
	if err != nil {
		t.Fatalf("NewAdapter(vscode-copilot) error = %v", err)
	}
	if _, err := Inject(home, vscodeAdapter, model.PersonaGentleman); err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	content, err := os.ReadFile(vscodeAdapter.SystemPromptFile(home))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.HasPrefix(string(content), "---\nname: Ordo Persona\n") {
		t.Fatalf("instructions file does not start with the Ordo persona marker:\n%s", content)
	}
}

// An instructions file written by an earlier version starts with the legacy
// "Gentle AI Persona" marker. Re-running injection must replace that preamble
// and keep the managed sections that follow it.
func TestInjectVSCodeReplacesLegacyPersonaMarker(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))

	vscodeAdapter, err := agents.NewAdapter("vscode-copilot")
	if err != nil {
		t.Fatalf("NewAdapter(vscode-copilot) error = %v", err)
	}
	path := vscodeAdapter.SystemPromptFile(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	legacy := strings.Join([]string{
		"---",
		"name: Gentle AI Persona",
		"description: Teaching-oriented persona with SDD orchestration and Engram protocol",
		"applyTo: \"**\"",
		"---",
		"",
		"## Personality",
		"Senior Architect mentor persona.",
		"",
		"<!-- gentle-ai:sdd-orchestrator -->",
		"SDD stays.",
		"<!-- /gentle-ai:sdd-orchestrator -->",
	}, "\n") + "\n"
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := InjectForSync(home, vscodeAdapter, model.PersonaNeutral); err != nil {
		t.Fatalf("InjectForSync() error = %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	text := string(raw)
	if strings.Contains(text, "Gentle AI Persona") {
		t.Fatalf("legacy persona marker survived re-injection:\n%s", text)
	}
	if !strings.Contains(text, "name: Ordo Persona") {
		t.Fatalf("instructions file missing the Ordo persona marker:\n%s", text)
	}
	if !strings.Contains(text, "<!-- gentle-ai:sdd-orchestrator -->\nSDD stays.") {
		t.Fatalf("managed section lost during re-injection:\n%s", text)
	}
}
