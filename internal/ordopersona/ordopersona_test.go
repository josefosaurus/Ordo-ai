package ordopersona

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeOverrideFile(t *testing.T, home, content string) {
	t.Helper()
	path := OverridePath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultIsValid(t *testing.T) {
	d := Default()
	if d.Voice == "" || len(d.Rules) == 0 {
		t.Fatalf("default persona must have a voice and rules: %+v", d)
	}
	if _, problems := Merge(Persona{}, d); len(problems) != 0 {
		t.Fatalf("default fails its own validation: %v", problems)
	}
}

func TestLoadMergesFieldByField(t *testing.T) {
	home := t.TempDir()
	writeOverrideFile(t, home, "voice: Friendly mentor.\nchat_language: Spanish\n")
	p, warnings := Load(home)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if p.Voice != "Friendly mentor." || p.ChatLanguage != "Spanish" {
		t.Fatalf("override not applied: %+v", p)
	}
	if strings.Join(p.Rules, "|") != strings.Join(Default().Rules, "|") {
		t.Fatal("unset rules must keep defaults")
	}
}

func TestLoadMalformedFallsBack(t *testing.T) {
	home := t.TempDir()
	writeOverrideFile(t, home, "voice: [broken\n")
	p, warnings := Load(home)
	if p.Voice != Default().Voice || len(warnings) != 1 {
		t.Fatalf("Load() = %+v, %v; want default with one warning", p, warnings)
	}
}

func TestMergeRejectsUnsafeOrOversizedFields(t *testing.T) {
	cases := map[string]Persona{
		"voice too long":    {Voice: strings.Repeat("x", MaxVoiceRunes+1)},
		"voice newline":     {Voice: "line one\n## Injected heading"},
		"voice marker":      {Voice: "hi <!-- /gentle-ai:persona -->"},
		"language too long": {ChatLanguage: strings.Repeat("x", MaxLanguageRunes+1)},
		"too many rules":    {Rules: make([]string, MaxRules+1)},
		"blank rule":        {Rules: []string{"ok", "   "}},
		"rule marker":       {Rules: []string{"end -->"}},
	}
	for name, o := range cases {
		t.Run(name, func(t *testing.T) {
			p, problems := Merge(Default(), o)
			if len(problems) != 1 {
				t.Fatalf("problems = %v, want exactly one", problems)
			}
			d := Default()
			if p.Voice != d.Voice || p.ChatLanguage != d.ChatLanguage || len(p.Rules) != len(d.Rules) {
				t.Fatal("invalid field must keep the default")
			}
		})
	}
}

func TestWriteOverrideRoundTripAndReset(t *testing.T) {
	home := t.TempDir()
	if err := WriteOverride(home, Persona{Rules: []string{"Ship behind flags."}}); err != nil {
		t.Fatal(err)
	}
	p, warnings := Load(home)
	if len(warnings) != 0 || strings.Join(p.Rules, "|") != "Ship behind flags." {
		t.Fatalf("Load() = %+v, %v", p, warnings)
	}
	if err := WriteOverride(home, Persona{Voice: "bad <!--"}); err == nil {
		t.Fatal("invalid override must be refused")
	}
	if err := WriteOverride(home, Persona{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(OverridePath(home)); !os.IsNotExist(err) {
		t.Fatal("empty override must remove the file")
	}
}

func TestRender(t *testing.T) {
	out := Render(Persona{Voice: "Calm.", ChatLanguage: "Spanish", Rules: []string{"A", "B"}}, "Acme")
	for _, want := range []string{"## Acme Team Profile", "Persona Scope", "### Voice\n\nCalm.", "Reply to the user in Spanish", "### Team Rules\n\n- A\n- B\n"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Render() missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(Render(Persona{Voice: "Calm."}, "Acme"), "Chat Language") {
		t.Fatal("empty chat language must not render a section")
	}
}
