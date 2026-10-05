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

func TestPreviewMatchesImportedSyncIDs(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(repo, "teamA", "faq.md"), "## Q1\na\n## Q2\nb\n")
	writeFile(t, filepath.Join(repo, "teamB", "faq.jsonl"), "{\"title\":\"Q3\",\"content\":\"c\"}\n")

	preview, err := Preview(repo, "demo")
	if err != nil {
		t.Fatalf("Preview: %v", err)
	}
	captured := stubEngram(t, true, nil)
	if err := Import(context.Background(), repo, "demo", io.Discard); err != nil {
		t.Fatalf("Import: %v", err)
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

func TestImportStreamsEngramOutputAndErrors(t *testing.T) {
	stubEngram(t, true, nil)
	inner := runEngram
	runEngram = func(ctx context.Context, bin string, args []string, stdout, stderr io.Writer) error {
		_, _ = io.WriteString(stderr, "warning: something\n")
		return inner(ctx, bin, args, stdout, stderr)
	}
	var out bytes.Buffer
	if err := Import(context.Background(), sampleFile(t), " demo ", &out); err != nil {
		t.Fatalf("Import: %v", err)
	}
	if !strings.Contains(out.String(), "1 imported") || !strings.Contains(out.String(), "warning: something") {
		t.Fatalf("Import output = %q, want engram stdout and stderr", out.String())
	}
}

func TestImportAndPreviewReportProblems(t *testing.T) {
	empty := t.TempDir()
	tests := []struct {
		name    string
		run     func() error
		wantErr string
	}{
		{"preview without project", func() error { _, err := Preview(sampleFile(t), "  "); return err }, "project is required"},
		{"preview without entries", func() error { _, err := Preview(empty, "demo"); return err }, "no entries found"},
		{"import without project", func() error { return Import(context.Background(), sampleFile(t), "", io.Discard) }, "project is required"},
		{"import missing path", func() error {
			return Import(context.Background(), filepath.Join(empty, "nope.md"), "demo", io.Discard)
		}, "no such file"},
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
		err := Import(context.Background(), sampleFile(t), "demo", io.Discard)
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
