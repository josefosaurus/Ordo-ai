package personacmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/ordopersona"
)

func run(t *testing.T, home string, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	err := Run(args, home, &out)
	return out.String(), err
}

func TestSetAddRemoveShowReset(t *testing.T) {
	home := t.TempDir()
	if _, err := run(t, home, "set", "voice", "Friendly mentor."); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, home, "set", "language", "Spanish"); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, home, "add-rule", "Ship behind feature flags.")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ordo sync") {
		t.Fatalf("edits must say how to apply them: %q", out)
	}

	p, warnings := ordopersona.Load(home)
	defaults := ordopersona.Default().Rules
	if len(warnings) != 0 || p.Voice != "Friendly mentor." || p.ChatLanguage != "Spanish" {
		t.Fatalf("persona after set = %+v, %v", p, warnings)
	}
	if len(p.Rules) != len(defaults)+1 || p.Rules[len(p.Rules)-1] != "Ship behind feature flags." {
		t.Fatalf("add-rule must keep defaults and append: %q", p.Rules)
	}

	if _, err := run(t, home, "remove-rule", "1"); err != nil {
		t.Fatal(err)
	}
	if p, _ := ordopersona.Load(home); p.Rules[0] == defaults[0] {
		t.Fatal("remove-rule 1 must drop the first rule")
	}

	out, err = run(t, home, "show")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"voice     Friendly mentor.", "language  Spanish", "Ship behind feature flags.", "--- what agents receive ---", "Team Profile", ordopersona.OverridePath(home)} {
		if !strings.Contains(out, want) {
			t.Fatalf("show missing %q:\n%s", want, out)
		}
	}

	if _, err := run(t, home, "reset", "voice"); err != nil {
		t.Fatal(err)
	}
	if p, _ := ordopersona.Load(home); p.Voice != ordopersona.Default().Voice || p.ChatLanguage != "Spanish" {
		t.Fatalf("reset voice must keep other fields: %+v", p)
	}
	if _, err := run(t, home, "reset"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ordopersona.OverridePath(home)); !os.IsNotExist(err) {
		t.Fatal("full reset must remove the override")
	}
}

func TestRejectsInvalidInputWithoutWriting(t *testing.T) {
	home := t.TempDir()
	cases := [][]string{
		{"set", "voice", "two\nlines"},
		{"set", "voice", "sneaky <!-- marker"},
		{"set", "mood", "x"},
		{"add-rule", strings.Repeat("x", ordopersona.MaxRuleRunes+1)},
		{"remove-rule", "0"},
		{"remove-rule", "99"},
		{"remove-rule", "abc"},
		{"set", "voice"},
		{"frobnicate"},
	}
	for _, args := range cases {
		if _, err := run(t, home, args...); err == nil {
			t.Fatalf("Run(%q) succeeded, want error", args)
		}
	}
	if _, err := os.Stat(ordopersona.OverridePath(home)); !os.IsNotExist(err) {
		t.Fatal("invalid input must not create an override")
	}
}

func TestBrokenOverrideRefusesEditsAndResetRecovers(t *testing.T) {
	home := t.TempDir()
	path := ordopersona.OverridePath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("voice: [broken\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, home, "show")
	if err != nil || !strings.Contains(out, "warning:") {
		t.Fatalf("show must warn and fall back: %q, %v", out, err)
	}
	if _, err := run(t, home, "set", "voice", "x"); err == nil {
		t.Fatal("set on a broken override must refuse")
	}
	if _, err := run(t, home, "reset"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("reset must remove the broken override")
	}
}
