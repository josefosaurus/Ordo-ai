package cli

import (
	"errors"
	"strings"
	"testing"
)

func TestTrustGentlemanTapFormulaIsScopedAndBestEffort(t *testing.T) {
	restore := runCommand
	t.Cleanup(func() { runCommand = restore })

	var got []string
	runCommand = func(name string, args ...string) error {
		got = append([]string{name}, args...)
		return errors.New("Error: Unknown command: trust") // older Homebrew
	}
	trustGentlemanTapFormula("engram") // must not panic or surface the error

	if len(got) != 4 || !strings.HasSuffix(got[0], "brew") || got[1] != "trust" || got[2] != "--formula" || got[3] != "gentleman-programming/tap/engram" {
		t.Fatalf("trust command = %q, want brew trust --formula gentleman-programming/tap/engram (formula-scoped, never the whole tap)", got)
	}
}

func TestWithTapTrustHint(t *testing.T) {
	if withTapTrustHint(nil, "gga") != nil {
		t.Fatal("nil error must stay nil")
	}
	other := errors.New("run command \"brew install engram\": exit status 1\noutput: Error: No available formula")
	if got := withTapTrustHint(other, "engram"); got != other {
		t.Fatalf("unrelated failures must pass through unchanged: %v", got)
	}
	untrusted := errors.New("run command \"brew reinstall gga\": exit status 1\noutput: Error: Refusing to load formula gentleman-programming/tap/gga from untrusted tap gentleman-programming/tap.")
	got := withTapTrustHint(untrusted, "gga")
	if !errors.Is(got, untrusted) {
		t.Fatal("hint must wrap the original error")
	}
	for _, want := range []string{"brew trust --formula gentleman-programming/tap/gga", "ordo sync"} {
		if !strings.Contains(got.Error(), want) {
			t.Fatalf("hint missing %q: %v", want, got)
		}
	}
}
