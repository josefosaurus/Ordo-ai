package memorycmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// PreviewEntry is one memory as `ordo memory import --dry-run` lists it.
type PreviewEntry struct {
	SyncID string
	Title  string
	Source string
}

// PreviewOf returns entries, as Collect returned them, with the sync IDs they
// would import under for project. It writes nothing and does not need engram.
func PreviewOf(entries []Entry, project string) ([]PreviewEntry, error) {
	name, err := requireImport(entries, project)
	if err != nil {
		return nil, err
	}
	return previewEntries(entries, name), nil
}

// ImportEntries loads entries, as Collect returned them, into the Engram
// project through `engram import`, with each memory's own type and no forced
// update. It never rescans their source, so it imports exactly what PreviewOf
// listed. Engram's stdout and stderr both go to out.
func ImportEntries(ctx context.Context, entries []Entry, project string, out io.Writer) error {
	name, err := requireImport(entries, project)
	if err != nil {
		return err
	}
	return importEntries(ctx, entries, name, BuildOptions{}, out, out)
}

// DefaultProject suggests an Engram project for dir: the name of the git
// repository containing it, else the directory's own name. It returns "" for
// a filesystem root.
func DefaultProject(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = filepath.Clean(dir)
	}
	root := originResolver{roots: map[string]string{}}.gitRoot(abs)
	if root == "" {
		root = abs
	}
	if filepath.Dir(root) == root {
		return ""
	}
	return filepath.Base(root)
}

// requireImport validates an import request and returns the trimmed project.
func requireImport(entries []Entry, project string) (string, error) {
	name := strings.TrimSpace(project)
	if name == "" {
		return "", errors.New("project is required")
	}
	if len(entries) == 0 {
		return "", errors.New("no entries to import")
	}
	return name, nil
}

func previewEntries(entries []Entry, project string) []PreviewEntry {
	preview := make([]PreviewEntry, 0, len(entries))
	for _, e := range entries {
		preview = append(preview, PreviewEntry{SyncID: entrySyncID(project, e), Title: e.Title, Source: e.Source})
	}
	return preview
}

// importEntries converts entries to Engram's export format and runs
// `engram import` on a temporary copy, which is always removed.
func importEntries(ctx context.Context, entries []Entry, project string, opts BuildOptions, stdout, stderr io.Writer) error {
	export := BuildExport(entries, project, opts)

	bin, err := resolveEngram()
	if err != nil {
		return err
	}
	file, err := writeExport(export)
	if err != nil {
		return err
	}
	defer os.Remove(file)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	if err := runEngram(ctx, bin, []string{"import", file}, stdout, stderr); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("engram import interrupted: %w", err)
		}
		return fmt.Errorf("engram import failed: %w", err)
	}
	return nil
}
