package memorycmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// stubEngram replaces the engram seams for one test and records the import
// document engram would have received.
func stubEngram(t *testing.T, found bool, runErr error) *Export {
	t.Helper()
	oldLook, oldRun := lookPath, runEngram
	t.Cleanup(func() { lookPath, runEngram = oldLook, oldRun })
	lookPath = func(string) (string, error) {
		if found {
			return "/fake/engram", nil
		}
		return "", exec.ErrNotFound
	}
	captured := &Export{}
	runEngram = func(bin string, args []string, stdout io.Writer) error {
		if bin != "/fake/engram" || len(args) != 2 || args[0] != "import" {
			t.Fatalf("unexpected engram call: %s %v", bin, args)
		}
		raw, err := os.ReadFile(args[1])
		if err != nil {
			t.Fatalf("read import file: %v", err)
		}
		if err := json.Unmarshal(raw, captured); err != nil {
			t.Fatalf("import file is not json: %v", err)
		}
		_, _ = io.WriteString(stdout, "Observations: 1 imported, 0 updated, 0 skipped stale\n")
		return runErr
	}
	return captured
}

func sampleFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "golden.md")
	writeFile(t, path, "## Rule\nUse X.\n")
	return path
}

func TestRunUsageErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{"unknown subcommand", []string{"nope"}, `unknown memory command "nope"`},
		{"missing path", []string{"import", "--project", "p"}, "usage: ordo memory import"},
		{"missing project", []string{"import", "x.md"}, "--project is required"},
		{"unknown flag", []string{"import", "x.md", "--project", "p", "--bogus"}, "flag provided but not defined"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Run(tt.args, io.Discard)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestRunHelp(t *testing.T) {
	for _, args := range [][]string{nil, {"help"}, {"--help"}} {
		var out bytes.Buffer
		if err := Run(args, &out); err != nil {
			t.Fatalf("Run(%v): %v", args, err)
		}
		if !strings.Contains(out.String(), "ordo memory import <path> --project <name>") {
			t.Errorf("help for %v missing usage line:\n%s", args, out.String())
		}
	}
}

func TestRunDryRunWritesNothingAndNeedsNoEngram(t *testing.T) {
	oldLook, oldRun := lookPath, runEngram
	t.Cleanup(func() { lookPath, runEngram = oldLook, oldRun })
	lookPath = func(string) (string, error) { t.Fatal("dry run must not resolve engram"); return "", nil }
	runEngram = func(string, []string, io.Writer) error { t.Fatal("dry run must not run engram"); return nil }

	var out bytes.Buffer
	if err := Run([]string{"import", sampleFile(t), "--project", "demo", "--dry-run"}, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := syncID("demo", "golden.md", "Rule") + "  Rule  (golden.md)"
	if !strings.Contains(out.String(), want) || !strings.Contains(out.String(), "1 entries (dry run, nothing written)") {
		t.Fatalf("dry-run output:\n%s\nwant line %q", out.String(), want)
	}
}

func TestRunImportsThroughEngram(t *testing.T) {
	captured := stubEngram(t, true, nil)
	var out bytes.Buffer
	if err := Run([]string{"import", sampleFile(t), "--project=demo", "--type", "decision"}, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(captured.Observations) != 1 || captured.Observations[0].Project != "demo" || captured.Observations[0].Type != "decision" {
		t.Fatalf("engram received %+v", captured)
	}
	if !strings.Contains(out.String(), "1 imported") {
		t.Errorf("engram output not streamed: %q", out.String())
	}
}

func TestRunReportsEngramProblems(t *testing.T) {
	t.Run("engram missing", func(t *testing.T) {
		stubEngram(t, false, nil)
		oldDirs := homebrewDirs
		t.Cleanup(func() { homebrewDirs = oldDirs })
		homebrewDirs = nil
		err := Run([]string{"import", sampleFile(t), "--project", "demo"}, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "engram not found on PATH") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("engram fails", func(t *testing.T) {
		stubEngram(t, true, errors.New("exit status 1"))
		err := Run([]string{"import", sampleFile(t), "--project", "demo"}, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "engram import failed: exit status 1") {
			t.Fatalf("err = %v", err)
		}
	})
}

func TestResolveEngramFallsBackToHomebrew(t *testing.T) {
	oldLook, oldDirs := lookPath, homebrewDirs
	t.Cleanup(func() { lookPath, homebrewDirs = oldLook, oldDirs })
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	dir := t.TempDir()
	bin := filepath.Join(dir, "engram")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	homebrewDirs = []string{filepath.Join(t.TempDir(), "missing"), dir}
	got, err := resolveEngram()
	if err != nil || got != bin {
		t.Fatalf("resolveEngram = %q, %v; want %q", got, err, bin)
	}
}

// TestRealEngramReimportIsIdempotent runs the installed engram binary against
// an isolated data directory; it never touches the user's ~/.engram.
func TestRealEngramReimportIsIdempotent(t *testing.T) {
	if _, err := exec.LookPath("engram"); err != nil {
		t.Skip("engram not installed")
	}
	t.Setenv("ENGRAM_DATA_DIR", t.TempDir())
	t.Setenv("HOME", t.TempDir())

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "golden.md"), "## One\nfirst\n## Two\nsecond\n")
	writeFile(t, filepath.Join(dir, "more.jsonl"), "{\"title\":\"Three\",\"content\":\"third\"}\n")

	var first, second bytes.Buffer
	if err := Run([]string{"import", dir, "--project", "ordo-test"}, &first); err != nil {
		t.Fatalf("first import: %v\n%s", err, first.String())
	}
	if !strings.Contains(first.String(), "3 imported") {
		t.Fatalf("first import summary:\n%s", first.String())
	}
	if err := Run([]string{"import", dir, "--project", "ordo-test"}, &second); err != nil {
		t.Fatalf("second import: %v\n%s", err, second.String())
	}
	if !strings.Contains(second.String(), "0 imported, 0 updated, 3 skipped") {
		t.Fatalf("second import must skip every entry:\n%s", second.String())
	}
}
