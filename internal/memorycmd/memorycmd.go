// Package memorycmd implements `ordo memory`: load curated knowledge from
// Markdown, CSV, JSONL, or JSON files into the local Engram memory store. Ordo never
// writes Engram's database itself; it converts the input into Engram's export
// format and hands it to `engram import`.
package memorycmd

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gentleman-programming/gentle-ai/v4/internal/system"
)

const usage = `Load curated knowledge into your local Engram memory.

USAGE
  ordo memory import <path> --project <name> [--type <type>] [--title-field <name>]
                     [--include-json] [--force] [--dry-run]

INPUT
  <path> is a file or a directory (scanned recursively, sorted; hidden
  directories, node_modules and vendor are skipped; .json files are skipped
  unless --include-json is given, since they are often configuration).
  .md      each "## " section is one memory; a file without sections is one
           memory titled by its file name; text before the first section is ignored
  .csv     header with title,content and an optional type column
  .jsonl   one {"title","content","type"} object per line
  .json    an array of objects (or an object holding one array); each object
           is one memory titled by title, titulo or name (prefixed by id), and
           its content field or, without one, its other fields as labeled text

FLAGS
  --project <name>   Engram project that receives the memories (required)
  --type <type>      type for every memory (default: per-record type, else manual)
  --title-field <name>
                     .json field that titles each record (default: title,
                     titulo or name, combined with id)
  --include-json     also read .json files when <path> is a directory
  --force            update every memory even if its file's mtime did not
                     advance (after cp -p, rsync -a or archive extraction)
  --dry-run          list the entries that would be imported; writes nothing

Re-importing the same input never duplicates memories: unchanged files are
skipped and edited files update their memories in place.
`

var (
	// lookPath and runEngram are seams so tests never need a real engram.
	lookPath  = exec.LookPath
	runEngram = func(ctx context.Context, bin string, args []string, stdout, stderr io.Writer) error {
		cmd := exec.CommandContext(ctx, bin, args...)
		system.EnsureCommandDir(cmd)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		return cmd.Run()
	}
	// homebrewDirs are checked when engram is not on PATH, matching where
	// `ordo install` places it through Homebrew.
	homebrewDirs = []string{"/opt/homebrew/bin", "/usr/local/bin", "/home/linuxbrew/.linuxbrew/bin"}
)

// Run dispatches a memory subcommand.
func Run(args []string, stdout io.Writer) error {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stdout, usage)
		return nil
	}
	switch args[0] {
	case "help", "--help", "-h":
		_, _ = fmt.Fprint(stdout, usage)
		return nil
	case "import":
		return runImport(args[1:], stdout)
	default:
		return fmt.Errorf("unknown memory command %q (see ordo memory help)", args[0])
	}
}

func runImport(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("memory import", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	project := fs.String("project", "", "")
	typ := fs.String("type", "", "")
	dryRun := fs.Bool("dry-run", false, "")
	force := fs.Bool("force", false, "")
	titleField := fs.String("title-field", "", "")
	includeJSON := fs.Bool("include-json", false, "")

	// Accept flags before or after the path.
	var positional []string
	for {
		if err := fs.Parse(args); err != nil {
			return fmt.Errorf("%w (see ordo memory help)", err)
		}
		args = fs.Args()
		if len(args) == 0 {
			break
		}
		positional = append(positional, args[0])
		args = args[1:]
	}
	if len(positional) != 1 {
		return errors.New("usage: ordo memory import <path> --project <name> [--type <type>] [--title-field <name>] [--include-json] [--force] [--dry-run] (see ordo memory help)")
	}
	name := strings.TrimSpace(*project)
	if name == "" {
		return errors.New("--project is required: usage: ordo memory import <path> --project <name> (see ordo memory help)")
	}

	entries, err := collect(positional[0], collectOptions{
		titleField:  strings.TrimSpace(*titleField),
		includeJSON: *includeJSON,
	})
	if err != nil {
		return err
	}
	if *dryRun {
		for _, e := range previewEntries(entries, name) {
			_, _ = fmt.Fprintf(stdout, "%s  %s  (%s)\n", e.SyncID, e.Title, e.Source)
		}
		_, _ = fmt.Fprintf(stdout, "%d entries (dry run, nothing written)\n", len(entries))
		return nil
	}
	opts := BuildOptions{TypeOverride: strings.TrimSpace(*typ)}
	if *force {
		opts.ForceAt = time.Now()
	}

	// Cancel engram on Ctrl-C or SIGTERM so the temp-file removal still runs.
	// There is no timeout: large imports may legitimately be slow.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return importEntries(ctx, entries, name, opts, stdout, os.Stderr)
}

func resolveEngram() (string, error) {
	if path, err := lookPath("engram"); err == nil {
		return path, nil
	}
	for _, dir := range homebrewDirs {
		candidate := filepath.Join(dir, "engram")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return candidate, nil
		}
	}
	searched := "PATH"
	if len(homebrewDirs) > 0 {
		searched += " or in " + strings.Join(homebrewDirs, ", ")
	}
	return "", fmt.Errorf("engram not found on %s; install it with: ordo install --agent <agent> --component engram", searched)
}

func writeExport(export Export) (string, error) {
	f, err := os.CreateTemp("", "ordo-memory-import-*.json")
	if err != nil {
		return "", err
	}
	enc := json.NewEncoder(f)
	if err := enc.Encode(export); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}
