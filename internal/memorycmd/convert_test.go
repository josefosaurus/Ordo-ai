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
	exp := BuildExport(entries, "demo", BuildOptions{})
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
	if overridden := BuildExport(entries, "demo", BuildOptions{TypeOverride: "pattern"}); overridden.Observations[1].Type != "pattern" {
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
	a, _ := json.Marshal(BuildExport(first, "demo", BuildOptions{}))
	b, _ := json.Marshal(BuildExport(second, "demo", BuildOptions{}))
	if string(a) != string(b) {
		t.Fatalf("export not deterministic:\n%s\n%s", a, b)
	}
}

func TestCollectSkipsHiddenAndVendoredDirs(t *testing.T) {
	root := filepath.Join(t.TempDir(), ".golden") // a dotted root is still scanned
	writeFile(t, filepath.Join(root, "a.md"), "## A\na\n")
	writeFile(t, filepath.Join(root, "sub", "b.md"), "## B\nb\n")
	for _, skipped := range []string{".git", ".hidden", "node_modules", "vendor", filepath.Join("sub", "node_modules")} {
		writeFile(t, filepath.Join(root, skipped, "x.md"), "## Skipped "+skipped+"\nx\n")
	}
	entries, err := Collect(root)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if got := strings.Join(titles(entries), "|"); got != "A|B" {
		t.Fatalf("titles = %q, want A|B", got)
	}

	t.Chdir(root)
	entries, err = Collect(".")
	if err != nil {
		t.Fatalf("Collect(.): %v", err)
	}
	if got := strings.Join(titles(entries), "|"); got != "A|B" {
		t.Fatalf("Collect(.) titles = %q, want A|B", got)
	}
}

func TestCollectWalkErrorNamesFileAndRecord(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "good.md"), "## A\na\n")
	writeFile(t, filepath.Join(dir, "sub", "bad.jsonl"), "{\"title\":\"ok\",\"content\":\"c\"}\n{nope}\n")
	_, err := Collect(dir)
	if err == nil || !strings.Contains(err.Error(), "sub/bad.jsonl: line 2:") {
		t.Fatalf("err = %v, want naming sub/bad.jsonl line 2", err)
	}
}

func idsOf(t *testing.T, path string) []string {
	t.Helper()
	entries, err := Collect(path)
	if err != nil {
		t.Fatalf("Collect(%s): %v", path, err)
	}
	var ids []string
	for _, o := range BuildExport(entries, "demo", BuildOptions{}).Observations {
		ids = append(ids, o.SyncID)
	}
	return ids
}

func TestSyncIDIsIndependentOfImportRoot(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(repo, "docs", "faq.md")
	writeFile(t, file, "## Q\nanswer\n")

	direct := idsOf(t, file)
	viaParent := idsOf(t, filepath.Join(repo, "docs"))
	viaRepo := idsOf(t, repo)
	if direct[0] != viaParent[0] || direct[0] != viaRepo[0] {
		t.Fatalf("ids differ by import root: direct=%s parent=%s repo=%s", direct[0], viaParent[0], viaRepo[0])
	}
	if want := syncID("demo", "docs/faq.md", "Q"); direct[0] != want {
		t.Fatalf("id = %s, want repo-relative identity %s", direct[0], want)
	}

	entries, err := Collect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if entries[0].Source != "docs/faq.md" {
		t.Errorf("display source = %q, want docs/faq.md", entries[0].Source)
	}
}

func TestSyncIDDistinguishesSameBasenameInDifferentDirs(t *testing.T) {
	repo := t.TempDir()
	if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(repo, "teamA", "faq.md")
	b := filepath.Join(repo, "teamB", "faq.md")
	writeFile(t, a, "## Q\nanswer\n")
	writeFile(t, b, "## Q\nanswer\n")
	if idsOf(t, a)[0] == idsOf(t, b)[0] {
		t.Fatal("teamA/faq.md and teamB/faq.md share a sync_id")
	}
}

func TestSyncIDOutsideGitUsesAbsolutePath(t *testing.T) {
	dir := t.TempDir()
	for d := dir; ; d = filepath.Dir(d) {
		if _, err := os.Lstat(filepath.Join(d, ".git")); err == nil {
			t.Skipf("temp dir %s is inside a git repository", dir)
		}
		if filepath.Dir(d) == d {
			break
		}
	}
	file := filepath.Join(dir, "faq.md")
	writeFile(t, file, "## Q\nanswer\n")
	abs, err := filepath.Abs(file)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := idsOf(t, file)[0], syncID("demo", filepath.ToSlash(abs), "Q"); got != want {
		t.Fatalf("id = %s, want absolute-path identity %s", got, want)
	}
}

func TestCollectFollowsSymlinkForModTime(t *testing.T) {
	target := filepath.Join(t.TempDir(), "real.md")
	writeFile(t, target, "## Linked\nbody\n")
	dir := t.TempDir()
	if err := os.Symlink(target, filepath.Join(dir, "linked.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	entries, err := Collect(dir)
	if err != nil {
		t.Fatalf("Collect: %v", err)
	}
	if len(entries) != 1 || !entries[0].ModTime.Equal(fixedTime) {
		t.Fatalf("entries = %+v, want ModTime of the symlink target %v", entries, fixedTime)
	}
}

func TestCollectStripsUTF8BOM(t *testing.T) {
	tests := []struct {
		file, content, want string
	}{
		{"x.md", "\uFEFF## First\na\n## Second\nb\n", "First|Second"},
		{"x.jsonl", "\uFEFF{\"title\":\"J\",\"content\":\"c\"}\n", "J"},
		{"x.csv", "\uFEFFtitle,content\nC,c\n", "C"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), tt.file)
			writeFile(t, path, tt.content)
			entries, err := Collect(path)
			if err != nil {
				t.Fatalf("Collect: %v", err)
			}
			if got := strings.Join(titles(entries), "|"); got != tt.want {
				t.Fatalf("titles = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseMarkdownFences(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantTitles string
	}{
		{
			name:       "longer outer fence keeps a shorter inner fence as content",
			content:    "## A\n````md\n```\n## inside\n```\n````\n## B\nb\n",
			wantTitles: "A|B",
		},
		{
			name:       "tilde fence is not closed by backticks",
			content:    "## A\n~~~\n```\n## inside\n~~~\n## B\nb\n",
			wantTitles: "A|B",
		},
		{
			name:       "backtick fence is not closed by tildes",
			content:    "## A\n```\n~~~\n## inside\n```\n## B\nb\n",
			wantTitles: "A|B",
		},
		{
			name:       "closing fence with trailing text does not close",
			content:    "## A\n```\n``` not a close\n## inside\n```\n## B\nb\n",
			wantTitles: "A|B",
		},
		{
			// CommonMark: an unclosed fence runs to the end of the document,
			// so later headings stay content of the open section.
			name:       "unclosed fence keeps the rest of the file as content",
			content:    "## A\n```\n## not a heading\n",
			wantTitles: "A",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries, err := parseMarkdown([]byte(tt.content), "f")
			if err != nil {
				t.Fatalf("parseMarkdown: %v", err)
			}
			if got := strings.Join(titles(entries), "|"); got != tt.wantTitles {
				t.Fatalf("titles = %q, want %q", got, tt.wantTitles)
			}
		})
	}
}
