package memorycmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewOfMatchesImportEntriesSyncIDs(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo, "teamA", "faq.md"), "## Q1\na\n## Q2\nb\n")
	writeFile(t, filepath.Join(repo, "teamB", "faq.jsonl"), "{\"title\":\"Q3\",\"content\":\"c\"}\n")

	entries, err := Collect(repo)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	preview, err := PreviewOf(entries, "demo")
	if err != nil {
		t.Fatalf("PreviewOf: %v", err)
	}
	captured := stubEngram(t, true, nil)
	if err := ImportEntries(context.Background(), entries, "demo", io.Discard); err != nil {
		t.Fatalf("ImportEntries: %v", err)
	}
	if len(preview) != len(captured.Observations) {
		t.Fatalf("preview has %d entries, import sent %d", len(preview), len(captured.Observations))
	}
	sources := map[string]string{"Q1": "teamA/faq.md", "Q2": "teamA/faq.md", "Q3": "teamB/faq.jsonl"}
	for i, o := range captured.Observations {
		p := preview[i]
		if p.SyncID != o.SyncID || p.Title != o.Title || p.Source != sources[o.Title] {
			t.Errorf("preview[%d] = %+v, imported %s %q from %s", i, p, o.SyncID, o.Title, sources[o.Title])
		}
	}
}

// ImportEntries imports exactly the entries it is given; it never rescans
// their source, so a file added after the preview is not imported.
func TestImportEntriesDoesNotRescan(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.md"), "## One\na\n")
	entries, err := Collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "b.md"), "## Two\nb\n")
	captured := stubEngram(t, true, nil)
	if err := ImportEntries(context.Background(), entries, "demo", io.Discard); err != nil {
		t.Fatalf("ImportEntries: %v", err)
	}
	if len(captured.Observations) != 1 || captured.Observations[0].Title != "One" {
		t.Fatalf("imported %+v, want only the collected entry", captured.Observations)
	}
}

func TestImportStreamsEngramOutputAndErrors(t *testing.T) {
	stubEngram(t, true, nil)
	inner := runEngram
	runEngram = func(ctx context.Context, bin string, args []string, stdout, stderr io.Writer) error {
		_, _ = io.WriteString(stderr, "warning: something\n")
		return inner(ctx, bin, args, stdout, stderr)
	}
	entries, err := Collect(sampleFile(t))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := ImportEntries(context.Background(), entries, " demo ", &out); err != nil {
		t.Fatalf("ImportEntries: %v", err)
	}
	if !strings.Contains(out.String(), "1 imported") || !strings.Contains(out.String(), "warning: something") {
		t.Fatalf("Import output = %q, want engram stdout and stderr", out.String())
	}
}

func TestImportAndPreviewReportProblems(t *testing.T) {
	entries, err := Collect(sampleFile(t))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name    string
		run     func() error
		wantErr string
	}{
		{"preview without project", func() error { _, err := PreviewOf(entries, "  "); return err }, "project is required"},
		{"preview without entries", func() error { _, err := PreviewOf(nil, "demo"); return err }, "no entries"},
		{"import without project", func() error { return ImportEntries(context.Background(), entries, "", io.Discard) }, "project is required"},
		{"import without entries", func() error { return ImportEntries(context.Background(), nil, "demo", io.Discard) }, "no entries"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.run(); err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
		})
	}

	t.Run("engram missing", func(t *testing.T) {
		stubEngram(t, false, nil)
		oldDirs := homebrewDirs
		t.Cleanup(func() { homebrewDirs = oldDirs })
		homebrewDirs = nil
		err := ImportEntries(context.Background(), entries, "demo", io.Discard)
		if err == nil || !strings.Contains(err.Error(), "engram not found") {
			t.Fatalf("err = %v, want engram not found", err)
		}
	})
}

func TestDefaultProject(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "my-repo")
	nested := filepath.Join(repo, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	plain := filepath.Join(t.TempDir(), "notes")
	if err := os.Mkdir(plain, 0o755); err != nil {
		t.Fatal(err)
	}

	tests := []struct{ name, dir, want string }{
		{"repo root", repo, "my-repo"},
		{"inside repo", nested, "my-repo"},
		{"outside repo", plain, "notes"},
		{"filesystem root", string(filepath.Separator), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultProject(tt.dir); got != tt.want {
				t.Fatalf("DefaultProject(%q) = %q, want %q", tt.dir, got, tt.want)
			}
		})
	}
}
