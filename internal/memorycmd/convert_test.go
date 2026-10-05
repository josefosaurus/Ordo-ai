package memorycmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

var fixedTime = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, fixedTime, fixedTime); err != nil {
		t.Fatal(err)
	}
}

func titles(entries []Entry) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.Title)
	}
	return out
}

func TestCollectMarkdown(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantTitles  []string
		wantContent []string
	}{
		{
			name:        "sections split by level-two headings, preamble ignored",
			content:     "# Golden\n\nIntro text.\n\n## First rule\nUse X.\n\n## Second rule\nAvoid Y.\n",
			wantTitles:  []string{"First rule", "Second rule"},
			wantContent: []string{"Use X.", "Avoid Y."},
		},
		{
			name:        "no level-two heading imports the whole file titled by file name",
			content:     "Just one note.\n",
			wantTitles:  []string{"notes"},
			wantContent: []string{"Just one note."},
		},
		{
			name:        "headings inside code fences are content",
			content:     "## Shell\n```sh\n## not a heading\n```\n",
			wantTitles:  []string{"Shell"},
			wantContent: []string{"```sh\n## not a heading\n```"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "notes.md")
			writeFile(t, path, tt.content)
			entries, err := Collect(path)
			if err != nil {
				t.Fatalf("Collect: %v", err)
			}
			if got := titles(entries); strings.Join(got, "|") != strings.Join(tt.wantTitles, "|") {
				t.Fatalf("titles = %q, want %q", got, tt.wantTitles)
			}
			for i, e := range entries {
				if e.Content != tt.wantContent[i] {
					t.Errorf("entry %d content = %q, want %q", i, e.Content, tt.wantContent[i])
				}
				if e.Source != "notes.md" {
					t.Errorf("entry %d source = %q, want notes.md", i, e.Source)
				}
			}
		})
	}
}

func TestCollectCSVAndJSONL(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.csv"), "Title,Content,Type\nCSV one,\"multi, comma\",decision\nCSV two,plain,\n")
	writeFile(t, filepath.Join(dir, "b.jsonl"), "{\"title\":\"JSON one\",\"content\":\"c1\",\"type\":\"bugfix\"}\n\n{\"title\":\"JSON two\",\"content\":\"c2\"}\n")
	writeFile(t, filepath.Join(dir, "sub", "c.md"), "## Nested\nbody\n")
	writeFile(t, filepath.Join(dir, "ignored.txt"), "## Not imported\nx\n")

	entries, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	want := []string{"CSV one", "CSV two", "JSON one", "JSON two", "Nested"}
	if got := titles(entries); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("titles = %q, want %q", got, want)
	}
	if entries[0].Content != "multi, comma" || entries[0].Type != "decision" {
		t.Errorf("csv entry = %+v", entries[0])
	}
	if entries[1].Type != "" {
		t.Errorf("empty csv type should stay empty, got %q", entries[1].Type)
	}
	if entries[2].Type != "bugfix" {
		t.Errorf("jsonl type = %q, want bugfix", entries[2].Type)
	}
	if entries[4].Source != "sub/c.md" {
		t.Errorf("nested source = %q, want sub/c.md", entries[4].Source)
	}
}

func TestCollectValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		content string
		wantErr string
	}{
		{"empty markdown section", "x.md", "## Empty\n\n## Full\nok\n", `x.md: entry "Empty": content is empty`},
		{"csv missing column", "x.csv", "title,body\na,b\n", "x.csv: header must include title and content"},
		{"csv empty title", "x.csv", "title,content\n,b\n", "x.csv: row 2: title is empty"},
		{"jsonl invalid json", "x.jsonl", "{nope}\n", "x.jsonl: line 1:"},
		{"jsonl empty content", "x.jsonl", "{\"title\":\"t\",\"content\":\" \"}\n", `x.jsonl: line 1: entry "t": content is empty`},
		{"content too large", "x.jsonl", "{\"title\":\"big\",\"content\":\"" + strings.Repeat("a", maxContentChars+1) + "\"}\n", `x.jsonl: line 1: entry "big": content exceeds 8000 characters`},
		{"duplicate title in one file", "x.md", "## Same\na\n## Same\nb\n", `x.md: duplicate entry "Same"`},
		{"no entries", "x.jsonl", "\n", "no entries found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tt.file)
			writeFile(t, path, tt.content)
			_, err := Collect(path)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestCollectRejectsUnsupportedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.txt")
	writeFile(t, path, "hi")
	if _, err := Collect(path); err == nil || !strings.Contains(err.Error(), "unsupported file type") {
		t.Fatalf("err = %v", err)
	}
}

func TestBuildExportShapeAndDefaults(t *testing.T) {
	entries := []Entry{
		{Title: "A", Content: "a", Source: "n.md", ModTime: fixedTime},
		{Title: "B", Content: "b", Type: "decision", Source: "n.md", ModTime: fixedTime},
	}
	exp := BuildExport(entries, "demo", "")
	if exp.Version != "0.2.0" || len(exp.Sessions) != 1 || exp.Sessions[0].ID != "ordo-import-demo" || exp.Sessions[0].Project != "demo" {
		t.Fatalf("unexpected session/version: %+v", exp)
	}
	if exp.Observations[0].Type != "manual" || exp.Observations[1].Type != "decision" {
		t.Errorf("types = %q,%q", exp.Observations[0].Type, exp.Observations[1].Type)
	}
	for _, o := range exp.Observations {
		if o.SessionID != "ordo-import-demo" || o.Project != "demo" || o.Scope != "project" {
			t.Errorf("observation binding wrong: %+v", o)
		}
		if o.CreatedAt != "2026-01-02 03:04:05" || o.UpdatedAt != o.CreatedAt {
			t.Errorf("timestamps = %q/%q", o.CreatedAt, o.UpdatedAt)
		}
		if !regexp.MustCompile(`^obs-[0-9a-f]{16}$`).MatchString(o.SyncID) {
			t.Errorf("sync_id %q does not match obs-<16hex>", o.SyncID)
		}
	}
	if overridden := BuildExport(entries, "demo", "pattern"); overridden.Observations[1].Type != "pattern" {
		t.Errorf("--type should override record type, got %q", overridden.Observations[1].Type)
	}

	raw, err := json.Marshal(exp)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"prompts":null`, `"topic_key":null`, `"sync_id":`, `"exported_at":`} {
		if !strings.Contains(string(raw), key) {
			t.Errorf("json missing %s: %s", key, raw)
		}
	}
}

func TestSyncIDIsDeterministicAndScoped(t *testing.T) {
	base := syncID("demo", "n.md", "A")
	if base != syncID("demo", "n.md", "A") {
		t.Fatal("sync id not deterministic")
	}
	for name, other := range map[string]string{
		"project": syncID("other", "n.md", "A"),
		"source":  syncID("demo", "m.md", "A"),
		"key":     syncID("demo", "n.md", "B"),
	} {
		if other == base {
			t.Errorf("changing %s did not change the sync id", name)
		}
	}
}

func TestCollectIsDeterministicAcrossRuns(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "b.md"), "## B\nb\n")
	writeFile(t, filepath.Join(dir, "a.md"), "## A\na\n")
	first, err := Collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Collect(dir)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(BuildExport(first, "demo", ""))
	b, _ := json.Marshal(BuildExport(second, "demo", ""))
	if string(a) != string(b) {
		t.Fatalf("export not deterministic:\n%s\n%s", a, b)
	}
}
