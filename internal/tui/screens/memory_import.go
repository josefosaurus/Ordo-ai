package screens

import (
	"fmt"
	"strings"

	"github.com/gentleman-programming/gentle-ai/v4/internal/memorycmd"
	"github.com/gentleman-programming/gentle-ai/v4/internal/tui/styles"
)

// MemoryImportStep is one step of the import memories flow.
type MemoryImportStep int

const (
	MemoryImportPath MemoryImportStep = iota
	MemoryImportScanning
	MemoryImportProject
	MemoryImportPreview
	MemoryImportRunning
	MemoryImportResult
)

// memoryImportPreviewRows bounds how many entries the preview lists.
const memoryImportPreviewRows = 12

// MemoryImportView is everything RenderMemoryImport needs.
type MemoryImportView struct {
	Step     MemoryImportStep
	Path     string
	Project  string
	Input    string
	InputPos int
	Entries  []memorycmd.PreviewEntry
	Err      string
	Output   string
}

// RenderMemoryImport draws the current step of the import memories flow.
func RenderMemoryImport(v MemoryImportView) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Import memories"))
	b.WriteString("\n")
	b.WriteString(styles.SubtextStyle.Render("Load Markdown, CSV, JSONL or JSON knowledge into your local Engram memory."))
	b.WriteString("\n\n")

	help := "enter: next • esc: back"
	switch v.Step {
	case MemoryImportPath:
		b.WriteString(styles.SelectedStyle.Render(styles.Cursor+"File or directory  ") + inputWithCursor(v.Input, v.InputPos))
		b.WriteString("\n")
		b.WriteString(styles.SubtextStyle.Render("  .md (one memory per \"## \" section), .csv, .jsonl, or a .json file; directories are scanned recursively (without .json)"))
		b.WriteString("\n")
	case MemoryImportScanning:
		b.WriteString(styles.UnselectedStyle.Render("  File or directory  " + v.Input))
		b.WriteString("\n\n")
		b.WriteString("Scanning…\n")
		help = "esc: back"
	case MemoryImportProject:
		b.WriteString(styles.UnselectedStyle.Render("  File or directory  " + v.Path))
		b.WriteString("\n")
		b.WriteString(styles.SelectedStyle.Render(styles.Cursor+"Engram project     ") + inputWithCursor(v.Input, v.InputPos))
		b.WriteString("\n")
	case MemoryImportPreview:
		fmt.Fprintf(&b, "%d entries will be imported into project %s from %s\n\n",
			len(v.Entries), styles.SelectedStyle.Render(v.Project), v.Path)
		for i, e := range v.Entries {
			if i == memoryImportPreviewRows {
				b.WriteString(styles.SubtextStyle.Render(fmt.Sprintf("  … and %d more", len(v.Entries)-i)))
				b.WriteString("\n")
				break
			}
			b.WriteString("  " + styles.SubtextStyle.Render(e.SyncID) + "  " + e.Title + "  " + styles.SubtextStyle.Render("("+e.Source+")"))
			b.WriteString("\n")
		}
		b.WriteString("\n")
		b.WriteString(styles.SubtextStyle.Render("Unchanged files are skipped; edited files update their memories in place."))
		b.WriteString("\n")
		help = "enter: import • esc: back"
	case MemoryImportRunning:
		b.WriteString("Importing… (running engram import)\n")
		help = ""
	case MemoryImportResult:
		if v.Err == "" {
			b.WriteString(styles.SuccessStyle.Render("Import finished"))
			b.WriteString("\n\n")
		}
		if out := strings.TrimSpace(v.Output); out != "" {
			b.WriteString(out)
			b.WriteString("\n")
		}
		help = "press any key to return"
	}

	if v.Err != "" {
		b.WriteString("\n")
		b.WriteString(styles.ErrorStyle.Render(v.Err))
		b.WriteString("\n")
	}
	if help != "" {
		b.WriteString("\n")
		b.WriteString(styles.HelpStyle.Render(help))
	}
	return b.String()
}
