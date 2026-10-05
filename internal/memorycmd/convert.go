package memorycmd

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"
)

// maxContentChars bounds one memory's content so a malformed source cannot
// flood the memory store with a single oversized observation.
const maxContentChars = 8000

const (
	exportVersion = "0.2.0"
	defaultType   = "manual"
	defaultScope  = "project"
	timeLayout    = "2006-01-02 15:04:05"
)

// Entry is one memory read from a source file.
type Entry struct {
	Title   string
	Content string
	Type    string    // optional per-record type; empty means default
	Source  string    // display path: slash-separated, relative to the import root
	Origin  string    // identity path for sync_id; see originPath
	ModTime time.Time // source file modification time
}

// Export mirrors the JSON document accepted by `engram import`.
type Export struct {
	Version      string        `json:"version"`
	ExportedAt   string        `json:"exported_at"`
	Sessions     []Session     `json:"sessions"`
	Observations []Observation `json:"observations"`
	Prompts      []any         `json:"prompts"`
}

// Session is the single import session every observation belongs to.
type Session struct {
	ID        string `json:"id"`
	Project   string `json:"project"`
	Directory string `json:"directory"`
	StartedAt string `json:"started_at"`
}

// Observation is one memory in Engram's export format.
type Observation struct {
	SyncID    string  `json:"sync_id"`
	SessionID string  `json:"session_id"`
	Type      string  `json:"type"`
	Title     string  `json:"title"`
	Content   string  `json:"content"`
	Project   string  `json:"project"`
	Scope     string  `json:"scope"`
	TopicKey  *string `json:"topic_key"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// Collect reads every supported file under path (a file or a directory,
// walked in sorted order) and returns its validated entries. Directory walks
// skip hidden directories (".git", ".cache", ...) and vendored trees
// ("node_modules", "vendor") below the root. Any invalid in-scope file fails
// the whole collection, naming the file and the record.
func Collect(path string) ([]Entry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	origins := originResolver{roots: map[string]string{}}
	var entries []Entry
	if !info.IsDir() {
		if !supported(path) {
			return nil, fmt.Errorf("%s: unsupported file type (use .md, .csv or .jsonl)", path)
		}
		entries, err = readFile(path, filepath.Base(path), origins.origin(path), info.ModTime())
		if err != nil {
			return nil, err
		}
	} else {
		err = filepath.WalkDir(path, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				if p != path && skipDir(d.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if !supported(p) {
				return nil
			}
			rel, relErr := filepath.Rel(path, p)
			if relErr != nil {
				return relErr
			}
			// os.Stat follows symlinks, so a linked file carries its target's mtime.
			fi, statErr := os.Stat(p)
			if statErr != nil {
				return statErr
			}
			found, readErr := readFile(p, filepath.ToSlash(rel), origins.origin(p), fi.ModTime())
			if readErr != nil {
				return readErr
			}
			entries = append(entries, found...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("%s: no entries found (expected .md, .csv or .jsonl content)", path)
	}
	return entries, nil
}

// skipDir reports whether a directory below the import root is out of scope.
func skipDir(name string) bool {
	return strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor"
}

// originResolver computes the identity path behind each sync_id: the file path
// relative to its enclosing git repository root, or the cleaned absolute path
// outside a repository. It never depends on the import root, so importing a
// file directly or through any parent directory yields the same sync_id, and
// same-named files in different directories stay distinct. Repository roots
// are cached per directory.
type originResolver struct {
	roots map[string]string // directory -> git root ("" when none)
}

func (r originResolver) origin(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	if root := r.gitRoot(filepath.Dir(abs)); root != "" {
		if rel, err := filepath.Rel(root, abs); err == nil {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(abs)
}

func (r originResolver) gitRoot(dir string) string {
	if root, ok := r.roots[dir]; ok {
		return root
	}
	var root string
	if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
		root = dir
	} else if parent := filepath.Dir(dir); parent != dir {
		root = r.gitRoot(parent)
	}
	r.roots[dir] = root
	return root
}

func supported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".csv", ".jsonl":
		return true
	}
	return false
}

// readFile parses one source file and validates its entries. Errors name the
// source path and, where known, the row or line and the entry title.
func readFile(path, source, origin string, modTime time.Time) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// Editors on Windows often prepend a UTF-8 BOM; it would hide the first
	// Markdown heading and break JSON decoding.
	data = bytes.TrimPrefix(data, []byte("\uFEFF"))
	var entries []Entry
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md":
		entries, err = parseMarkdown(data, strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
	case ".csv":
		entries, err = parseCSV(data)
	case ".jsonl":
		entries, err = parseJSONL(data)
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}
	seen := map[string]bool{}
	for i := range entries {
		if seen[entries[i].Title] {
			return nil, fmt.Errorf("%s: duplicate entry %q (titles must be unique per file)", source, entries[i].Title)
		}
		seen[entries[i].Title] = true
		entries[i].Source = source
		entries[i].Origin = origin
		entries[i].ModTime = modTime
	}
	return entries, nil
}

func validate(e Entry) error {
	if e.Title == "" {
		return errors.New("title is empty")
	}
	if e.Content == "" {
		return fmt.Errorf("entry %q: content is empty", e.Title)
	}
	if utf8.RuneCountInString(e.Content) > maxContentChars {
		return fmt.Errorf("entry %q: content exceeds %d characters", e.Title, maxContentChars)
	}
	return nil
}

// parseMarkdown turns each `## ` section into one entry. Text before the
// first section is ignored; a file without sections becomes one entry titled
// by the file name. Headings inside fenced code blocks are content. Fences
// follow CommonMark: a fence of N backticks or tildes closes only on a line of
// at least N of the same character with nothing but whitespace after it, and
// an unclosed fence runs to the end of the file.
func parseMarkdown(data []byte, fileTitle string) ([]Entry, error) {
	var (
		entries []Entry
		current *Entry
		body    []string
		fence   fenceMarker
		hasHead bool
	)
	flush := func() {
		if current != nil {
			current.Content = strings.TrimSpace(strings.Join(body, "\n"))
			entries = append(entries, *current)
		}
		body = nil
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if fence.open() {
			if fence.closedBy(line) {
				fence = fenceMarker{}
			}
		} else if opened := openFence(line); opened.open() {
			fence = opened
		} else if strings.HasPrefix(line, "## ") {
			flush()
			hasHead = true
			current = &Entry{Title: strings.TrimSpace(strings.TrimPrefix(line, "## "))}
			continue
		}
		body = append(body, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !hasHead {
		current = &Entry{Title: fileTitle}
	}
	flush()
	for _, e := range entries {
		if err := validate(e); err != nil {
			return nil, err
		}
	}
	return entries, nil
}

// fenceMarker is an open code fence: its character and run length.
type fenceMarker struct {
	char byte
	n    int
}

func (f fenceMarker) open() bool { return f.n > 0 }

// openFence returns the fence a line opens, or a zero marker.
func openFence(line string) fenceMarker {
	trimmed := strings.TrimLeft(line, " \t")
	if trimmed == "" || (trimmed[0] != '`' && trimmed[0] != '~') {
		return fenceMarker{}
	}
	n := len(trimmed) - len(strings.TrimLeft(trimmed, trimmed[:1]))
	if n < 3 {
		return fenceMarker{}
	}
	return fenceMarker{char: trimmed[0], n: n}
}

// closedBy reports whether line closes the fence.
func (f fenceMarker) closedBy(line string) bool {
	trimmed := strings.TrimLeft(line, " \t")
	rest := strings.TrimLeft(trimmed, string(f.char))
	return len(trimmed)-len(rest) >= f.n && strings.TrimSpace(rest) == ""
}

// parseCSV reads rows under a header that names title and content columns
// (case-insensitive) and an optional type column.
func parseCSV(data []byte) ([]Entry, error) {
	r := csv.NewReader(bytes.NewReader(data))
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err == io.EOF {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	col := map[string]int{}
	for i, name := range header {
		col[strings.ToLower(strings.TrimSpace(strings.TrimPrefix(name, "\uFEFF")))] = i
	}
	titleCol, okTitle := col["title"]
	contentCol, okContent := col["content"]
	if !okTitle || !okContent {
		return nil, errors.New("header must include title and content columns")
	}
	typeCol, hasType := col["type"]
	field := func(rec []string, i int) string {
		if i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
		return ""
	}
	var entries []Entry
	for row := 2; ; row++ {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		e := Entry{Title: field(rec, titleCol), Content: field(rec, contentCol)}
		if hasType {
			e.Type = field(rec, typeCol)
		}
		if err := validate(e); err != nil {
			return nil, fmt.Errorf("row %d: %w", row, err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// parseJSONL reads one {"title","content","type"} object per non-blank line.
func parseJSONL(data []byte) ([]Entry, error) {
	var entries []Entry
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for line := 1; scanner.Scan(); line++ {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		var rec struct {
			Title   string `json:"title"`
			Content string `json:"content"`
			Type    string `json:"type"`
		}
		if err := json.Unmarshal([]byte(text), &rec); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		e := Entry{Title: strings.TrimSpace(rec.Title), Content: strings.TrimSpace(rec.Content), Type: strings.TrimSpace(rec.Type)}
		if err := validate(e); err != nil {
			return nil, fmt.Errorf("line %d: %w", line, err)
		}
		entries = append(entries, e)
	}
	return entries, scanner.Err()
}

// BuildExport converts entries into one Engram import document bound to a
// single session for project. typeOverride, when set, replaces every record's
// type; otherwise the record type or "manual" is used. Observations keep the
// order of entries, which Collect returns in sorted path order.
//
// Timestamps come from each source file's modification time, so re-importing
// an unchanged file is skipped by Engram and editing a file updates its
// entries in place (Engram updates only when updated_at is newer). ForceAt
// overrides updated_at for edits whose mtime did not advance.
// BuildOptions tunes BuildExport.
type BuildOptions struct {
	// TypeOverride, when set, replaces every record's type.
	TypeOverride string
	// ForceAt, when non-zero, becomes every observation's updated_at so Engram
	// applies edits whose file modification time did not advance.
	ForceAt time.Time
}

func BuildExport(entries []Entry, project string, opts BuildOptions) Export {
	sessionID := "ordo-import-" + project
	var earliest, latest time.Time
	observations := make([]Observation, 0, len(entries))
	for _, e := range entries {
		ts := e.ModTime.UTC().Truncate(time.Second)
		if earliest.IsZero() || ts.Before(earliest) {
			earliest = ts
		}
		if ts.After(latest) {
			latest = ts
		}
		typ := opts.TypeOverride
		if typ == "" {
			typ = e.Type
		}
		if typ == "" {
			typ = defaultType
		}
		stamp, updated := ts.Format(timeLayout), ts
		if !opts.ForceAt.IsZero() {
			updated = opts.ForceAt.UTC().Truncate(time.Second)
		}
		if updated.After(latest) {
			latest = updated
		}
		observations = append(observations, Observation{
			SyncID:    entrySyncID(project, e),
			SessionID: sessionID,
			Type:      typ,
			Title:     e.Title,
			Content:   e.Content,
			Project:   project,
			Scope:     defaultScope,
			CreatedAt: stamp,
			UpdatedAt: updated.Format(timeLayout),
		})
	}
	return Export{
		Version:      exportVersion,
		ExportedAt:   latest.Format(timeLayout),
		Sessions:     []Session{{ID: sessionID, Project: project, StartedAt: earliest.Format(timeLayout)}},
		Observations: observations,
	}
}

// syncIDHexLen is how many hex characters of the hash form a sync_id. It
// matches the look of Engram's own obs-<16hex> ids, and a 64-bit space is
// ample for per-project curated sets. Changing it re-keys every imported
// memory, so later imports would duplicate them.
const syncIDHexLen = 16

// entrySyncID is the sync_id an entry imports under for project.
func entrySyncID(project string, e Entry) string {
	return syncID(project, e.Origin, e.Title)
}

// syncID derives Engram's dedupe key from where an entry came from, so the
// same entry always maps to the same observation.
func syncID(project, origin, key string) string {
	sum := sha256.Sum256([]byte(project + "\x00" + origin + "\x00" + key))
	return "obs-" + hex.EncodeToString(sum[:])[:syncIDHexLen]
}
