package memorycmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	runEngram = func(ctx context.Context, bin string, args []string, stdout io.Writer) error {
		if ctx.Done() == nil {
			t.Fatal("engram must run with a cancellable context")
		}
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
	runEngram = func(context.Context, string, []string, io.Writer) error {
		t.Fatal("dry run must not run engram")
		return nil
	}

	file := sampleFile(t)
	entries, err := Collect(file)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := Run([]string{"import", file, "--project", "demo", "--dry-run"}, &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := syncID("demo", entries[0].Origin, "Rule") + "  Rule  (golden.md)"
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
	t.Run("engram missing lists searched dirs", func(t *testing.T) {
		stubEngram(t, false, nil)
		oldDirs := homebrewDirs
		t.Cleanup(func() { homebrewDirs = oldDirs })
		homebrewDirs = []string{"/brew/a", "/brew/b"}
		err := Run([]string{"import", sampleFile(t), "--project", "demo"}, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "/brew/a, /brew/b") || !strings.Contains(err.Error(), "ordo install --agent <agent> --component engram") {
			t.Fatalf("err = %v, want searched dirs and install hint", err)
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

	golden := filepath.Join(dir, "golden.md")
	later := fixedTime.Add(time.Hour)
	if err := os.WriteFile(golden, []byte("## One\nfirst, edited\n## Two\nsecond\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(golden, later, later); err != nil {
		t.Fatal(err)
	}
	var edited bytes.Buffer
	if err := Run([]string{"import", dir, "--project", "ordo-test"}, &edited); err != nil {
		t.Fatalf("edited import: %v\n%s", err, edited.String())
	}
	if !strings.Contains(edited.String(), "0 imported, 2 updated, 1 skipped") {
		t.Fatalf("editing a file must update its entries:\n%s", edited.String())
	}

	later = later.Add(time.Hour)
	if err := os.WriteFile(golden, []byte("## One\nfirst, edited\n## Two\nsecond\n## Four\nfourth\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(golden, later, later); err != nil {
		t.Fatal(err)
	}
	var appended bytes.Buffer
	if err := Run([]string{"import", dir, "--project", "ordo-test"}, &appended); err != nil {
		t.Fatalf("appended import: %v\n%s", err, appended.String())
	}
	if !strings.Contains(appended.String(), "1 imported") {
		t.Fatalf("appending a section must import it:\n%s", appended.String())
	}

	// --force pushes content whose mtime did not advance (cp -p, rsync -a).
	if err := os.WriteFile(golden, []byte("## One\nfirst, edited again\n## Two\nsecond\n## Four\nfourth\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(golden, later, later); err != nil {
		t.Fatal(err)
	}
	var stale, forced bytes.Buffer
	if err := Run([]string{"import", dir, "--project", "ordo-test"}, &stale); err != nil {
		t.Fatalf("stale import: %v\n%s", err, stale.String())
	}
	if !strings.Contains(stale.String(), "0 updated") {
		t.Fatalf("an unchanged mtime must not update without --force:\n%s", stale.String())
	}
	if err := Run([]string{"import", dir, "--project", "ordo-test", "--force"}, &forced); err != nil {
		t.Fatalf("forced import: %v\n%s", err, forced.String())
	}
	if !strings.Contains(forced.String(), "0 imported, 4 updated") {
		t.Fatalf("--force must update every entry:\n%s", forced.String())
	}
}

func TestRunForceStampsUpdatedAtNow(t *testing.T) {
	captured := stubEngram(t, true, nil)
	before := time.Now().UTC().Truncate(time.Second)
	if err := Run([]string{"import", sampleFile(t), "--project", "demo", "--force"}, io.Discard); err != nil {
		t.Fatalf("Run: %v", err)
	}
	o := captured.Observations[0]
	if o.CreatedAt != fixedTime.Format(timeLayout) {
		t.Errorf("created_at = %q, want file mtime %q", o.CreatedAt, fixedTime.Format(timeLayout))
	}
	updated, err := time.Parse(timeLayout, o.UpdatedAt)
	if err != nil || updated.Before(before) {
		t.Errorf("updated_at = %q (%v), want >= %v", o.UpdatedAt, err, before)
	}
}

func TestRunDryRunMatchesImportedIDs(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo, "teamA", "faq.md"), "## Q1\na\n## Q2\nb\n")
	writeFile(t, filepath.Join(repo, "teamB", "faq.jsonl"), "{\"title\":\"Q3\",\"content\":\"c\"}\n")

	captured := stubEngram(t, true, nil)
	if err := Run([]string{"import", repo, "--project", "demo"}, io.Discard); err != nil {
		t.Fatalf("Run: %v", err)
	}
	var out bytes.Buffer
	if err := Run([]string{"import", repo, "--project", "demo", "--dry-run"}, &out); err != nil {
		t.Fatalf("dry run: %v", err)
	}
	sources := map[string]string{"Q1": "teamA/faq.md", "Q2": "teamA/faq.md", "Q3": "teamB/faq.jsonl"}
	for _, o := range captured.Observations {
		line := o.SyncID + "  " + o.Title + "  (" + sources[o.Title] + ")"
		if !strings.Contains(out.String(), line) {
			t.Errorf("dry-run output missing %q:\n%s", line, out.String())
		}
	}
}

func TestRunInterruptRemovesTempFile(t *testing.T) {
	stubEngram(t, true, nil)
	var importFile string
	runEngram = func(ctx context.Context, bin string, args []string, stdout io.Writer) error {
		if ctx.Done() == nil {
			t.Fatal("engram must run with a cancellable context")
		}
		importFile = args[1]
		p, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Skipf("cannot find own process: %v", err)
		}
		if err := p.Signal(os.Interrupt); err != nil {
			t.Skipf("cannot signal own process: %v", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(5 * time.Second):
			t.Fatal("interrupt did not cancel the engram context")
			return nil
		}
	}
	err := Run([]string{"import", sampleFile(t), "--project", "demo"}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "interrupted") {
		t.Fatalf("err = %v, want interrupted", err)
	}
	if _, statErr := os.Stat(importFile); !os.IsNotExist(statErr) {
		t.Fatalf("temp import file %s still exists: %v", importFile, statErr)
	}
}
