// Package memorycmd implements `ordo memory`: load curated knowledge from
// Markdown, CSV, or JSONL files into the local Engram memory store. Ordo never
// writes Engram's database itself; it converts the input into Engram's export
// format and hands it to `engram import`.
package memorycmd

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gentleman-programming/gentle-ai/v4/internal/system"
)

const usage = `Load curated knowledge into your local Engram memory.

USAGE
  ordo memory import <path> --project <name> [--type <type>] [--dry-run]

INPUT
  <path> is a file or a directory (scanned recursively, sorted).
  .md      each "## " section is one memory; a file without sections is one
           memory titled by its file name; text before the first section is ignored
  .csv     header with title,content and an optional type column
  .jsonl   one {"title","content","type"} object per line

FLAGS
  --project <name>   Engram project that receives the memories (required)
  --type <type>      type for every memory (default: per-record type, else manual)
  --dry-run          list the entries that would be imported; writes nothing

Re-importing the same input never duplicates memories: unchanged files are
skipped and edited files update their memories in place.
`

const missingEngram = "engram not found on PATH; install it with: ordo install --agent <agent> --component engram"

var (
	// lookPath and runEngram are seams so tests never need a real engram.
	lookPath  = exec.LookPath
	runEngram = func(bin string, args []string, stdout io.Writer) error {
		cmd := exec.Command(bin, args...)
		system.EnsureCommandDir(cmd)
		cmd.Stdout = stdout
		cmd.Stderr = os.Stderr
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
		return errors.New("usage: ordo memory import <path> --project <name> [--type <type>] [--dry-run] (see ordo memory help)")
	}
	name := strings.TrimSpace(*project)
	if name == "" {
		return errors.New("--project is required: usage: ordo memory import <path> --project <name> (see ordo memory help)")
	}

	entries, err := Collect(positional[0])
	if err != nil {
		return err
	}
	export := BuildExport(entries, name, strings.TrimSpace(*typ))

	if *dryRun {
		for i, o := range export.Observations {
			_, _ = fmt.Fprintf(stdout, "%s  %s  (%s)\n", o.SyncID, o.Title, entries[i].Source)
		}
		_, _ = fmt.Fprintf(stdout, "%d entries (dry run, nothing written)\n", len(export.Observations))
		return nil
	}

	bin, err := resolveEngram()
	if err != nil {
		return err
	}
	file, err := writeExport(export)
	if err != nil {
		return err
	}
	defer os.Remove(file)
	if err := runEngram(bin, []string{"import", file}, stdout); err != nil {
		return fmt.Errorf("engram import failed: %w", err)
	}
	return nil
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
	return "", errors.New(missingEngram)
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
