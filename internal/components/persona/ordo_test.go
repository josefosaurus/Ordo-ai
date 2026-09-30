package persona

import (
	"os"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
	"github.com/gentleman-programming/gentle-ai/v4/internal/ordopersona"
)

func readInjectedFiles(t *testing.T, files []string) string {
	t.Helper()
	var all strings.Builder
	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		all.Write(data)
	}
	return all.String()
}

func TestInjectOrdoAddsTeamProfileToNeutral(t *testing.T) {
	home := t.TempDir()
	if err := ordopersona.WriteOverride(home, ordopersona.Persona{Voice: "Terse and kind.", Rules: []string{"Feature flags for risky changes."}}); err != nil {
		t.Fatal(err)
	}

	result, err := Inject(home, codexAdapter(), model.PersonaOrdo)
	if err != nil {
		t.Fatalf("Inject(ordo) error = %v", err)
	}
	got := readInjectedFiles(t, result.Files)

	neutral := strings.TrimSpace(personaContent(model.AgentCodex, model.PersonaNeutral, false))
	firstNeutralLine := strings.SplitN(neutral, "\n", 2)[0]
	for _, want := range []string{firstNeutralLine, "## Ordo Team Profile", "Terse and kind.", "- Feature flags for risky changes."} {
		if !strings.Contains(got, want) {
			t.Fatalf("injected ordo persona missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, ordopersona.Default().Rules[0]) {
		t.Fatal("overridden rules must replace the default rules")
	}
}

func TestInjectOrdoIsIdempotentAndFollowsEdits(t *testing.T) {
	home := t.TempDir()
	first, err := Inject(home, codexAdapter(), model.PersonaOrdo)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(readInjectedFiles(t, first.Files), ordopersona.Default().Voice) {
		t.Fatal("default voice must be installed without an override")
	}
	again, err := Inject(home, codexAdapter(), model.PersonaOrdo)
	if err != nil {
		t.Fatal(err)
	}
	if again.Changed {
		t.Fatal("re-injecting the same ordo persona must be a no-op")
	}

	if err := ordopersona.WriteOverride(home, ordopersona.Persona{Voice: "Now more playful."}); err != nil {
		t.Fatal(err)
	}
	edited, err := Inject(home, codexAdapter(), model.PersonaOrdo)
	if err != nil {
		t.Fatal(err)
	}
	got := readInjectedFiles(t, edited.Files)
	if !edited.Changed || !strings.Contains(got, "Now more playful.") || strings.Contains(got, ordopersona.Default().Voice) {
		t.Fatalf("edited voice must replace the previous one:\n%s", got)
	}
}

func TestInjectPiPersonaMapsOrdoToNeutral(t *testing.T) {
	root := t.TempDir()
	if _, err := InjectPiPersona(root, model.PersonaOrdo); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(PiPersonaConfigPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(content), "{\n  \"mode\": \"neutral\"\n}\n"; got != want {
		t.Fatalf("Pi persona config = %q, want %q", got, want)
	}
}

func TestOrdoUsesNeutralOutputStyle(t *testing.T) {
	style, ok := ResourcePlanFor(model.PersonaOrdo).OutputStyle()
	if !ok || style.Name != "Neutral" {
		t.Fatalf("ResourcePlanFor(ordo) output style = %+v, %v; want Neutral", style, ok)
	}
}
