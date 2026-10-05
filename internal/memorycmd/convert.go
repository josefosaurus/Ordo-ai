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
	"sort"
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
	Source  string    // slash-separated path relative to the import root
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
// walked in sorted order) and returns its validated entries.
func Collect(path string) ([]Entry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	if !info.IsDir() {
		if !supported(path) {
			return nil, fmt.Errorf("%s: unsupported file type (use .md, .csv or .jsonl)", path)
		}
		entries, err = readFile(path, filepath.Base(path), info.ModTime())
		if err != nil {
			return nil, err
		}
	} else {
		err = filepath.WalkDir(path, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !supported(p) {
				return nil
			}
			rel, relErr := filepath.Rel(path, p)
			if relErr != nil {
				return relErr
			}
			fi, statErr := d.Info()
			if statErr != nil {
				return statErr
			}
			found, readErr := readFile(p, filepath.ToSlash(rel), fi.ModTime())
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

func supported(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".csv", ".jsonl":
		return true
	}
	return false
}

// readFile parses one source file and validates its entries. Errors name the
// source path and, where known, the row or line and the entry title.
func readFile(path, source string, modTime time.Time) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
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
// by the file name. Headings inside fenced code blocks are content.
func parseMarkdown(data []byte, fileTitle string) ([]Entry, error) {
	var (
		entries []Entry
		current *Entry
		body    []string
		inFence bool
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
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
		}
		if !inFence && strings.HasPrefix(line, "## ") {
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
// type; otherwise the record type or "manual" is used.
//
// Timestamps come from each source file's modification time, so re-importing
// an unchanged file is skipped by Engram and editing a file updates its
// entries in place (Engram updates only when updated_at is newer).
func BuildExport(entries []Entry, project, typeOverride string) Export {
	sessionID := "ordo-import-" + project
	sorted := append([]Entry(nil), entries...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Source < sorted[j].Source })

	var earliest, latest time.Time
	observations := make([]Observation, 0, len(sorted))
	for _, e := range sorted {
		ts := e.ModTime.UTC().Truncate(time.Second)
		if earliest.IsZero() || ts.Before(earliest) {
			earliest = ts
		}
		if ts.After(latest) {
			latest = ts
		}
		typ := typeOverride
		if typ == "" {
			typ = e.Type
		}
		if typ == "" {
			typ = defaultType
		}
		stamp := ts.Format(timeLayout)
		observations = append(observations, Observation{
			SyncID:    syncID(project, e.Source, e.Title),
			SessionID: sessionID,
			Type:      typ,
			Title:     e.Title,
			Content:   e.Content,
			Project:   project,
			Scope:     defaultScope,
			CreatedAt: stamp,
			UpdatedAt: stamp,
		})
	}
	return Export{
		Version:      exportVersion,
		ExportedAt:   latest.Format(timeLayout),
		Sessions:     []Session{{ID: sessionID, Project: project, StartedAt: earliest.Format(timeLayout)}},
		Observations: observations,
	}
}

// syncID derives Engram's dedupe key from where an entry came from, so the
// same entry always maps to the same observation.
func syncID(project, source, key string) string {
	sum := sha256.Sum256([]byte(project + "\x00" + source + "\x00" + key))
	return "obs-" + hex.EncodeToString(sum[:])[:16]
}
