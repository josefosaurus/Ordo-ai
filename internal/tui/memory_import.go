package tui

import (
	"context"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gentleman-programming/gentle-ai/v4/internal/memorycmd"
	"github.com/gentleman-programming/gentle-ai/v4/internal/tui/screens"
)

// memoryImportFunc runs one import of already collected entries;
// memorycmd.ImportEntries unless a test injects a stub through
// Model.memoryImportRun.
type memoryImportFunc func(ctx context.Context, entries []memorycmd.Entry, project string, out io.Writer) error

// MemoryImportState is the state of ScreenMemoryImport. The screen handles
// all of its keys itself (see handleMemoryImportKey).
type MemoryImportState struct {
	Step     screens.MemoryImportStep
	Path     string
	Project  string
	Input    string
	InputPos int
	Entries  []memorycmd.PreviewEntry
	Err      string
	Output   string

	// scanned holds the entries the path scan collected; the preview and the
	// import both use them, so the path is scanned exactly once. scanSeq
	// identifies the current scan so a result that lands after Esc is ignored.
	scanned []memorycmd.Entry
	scanSeq int

	// cancel stops a running import; quitting records a Ctrl+C that waits
	// for the cancelled import to return before the TUI exits.
	cancel   context.CancelFunc
	quitting bool
}

// memoryImportScanDoneMsg reports a finished path scan.
type memoryImportScanDoneMsg struct {
	Seq     int
	Path    string
	Entries []memorycmd.Entry
	Err     error
}

// memoryImportDoneMsg reports a finished import with Engram's output.
type memoryImportDoneMsg struct {
	Output string
	Err    error
}

func (m Model) startMemoryImport() (tea.Model, tea.Cmd) {
	m.MemoryImport = MemoryImportState{}
	m.setScreen(ScreenMemoryImport)
	return m, nil
}

func (m Model) renderMemoryImport() string {
	s := m.MemoryImport
	return screens.RenderMemoryImport(screens.MemoryImportView{
		Step: s.Step, Path: s.Path, Project: s.Project,
		Input: s.Input, InputPos: s.InputPos,
		Entries: s.Entries, Err: s.Err, Output: s.Output,
	})
}

func (m Model) handleMemoryImportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := &m.MemoryImport
	if msg.Type == tea.KeyCtrlC {
		// Quitting mid-import would orphan Engram and skip memorycmd's
		// temp-file cleanup, so cancel and quit once the import returns.
		if s.Step == screens.MemoryImportRunning && s.cancel != nil {
			s.cancel()
			s.quitting = true
			return m, nil
		}
		return m, tea.Quit
	}
	switch s.Step {
	case screens.MemoryImportScanning:
		// The scan is read-only, so Esc just abandons it; its late result
		// no longer matches scanSeq and is ignored.
		if msg.Type == tea.KeyEsc {
			s.scanSeq++
			s.Step, s.Err = screens.MemoryImportPath, ""
		}
		return m, nil
	case screens.MemoryImportRunning:
		return m, nil
	case screens.MemoryImportResult:
		m.setScreen(ScreenWelcome)
		return m, nil
	case screens.MemoryImportPreview:
		switch msg.Type {
		case tea.KeyEsc:
			s.Step, s.Err = screens.MemoryImportProject, ""
			s.setInput(s.Project)
		case tea.KeyEnter:
			s.Step, s.Err = screens.MemoryImportRunning, ""
			ctx, cancel := context.WithCancel(context.Background())
			s.cancel = cancel
			return m, m.runMemoryImport(ctx, s.scanned, s.Project)
		}
		return m, nil
	}

	switch msg.Type {
	case tea.KeyEsc:
		if s.Step == screens.MemoryImportPath {
			m.setScreen(ScreenWelcome)
			return m, nil
		}
		s.Project = s.Input
		s.Step, s.Err = screens.MemoryImportPath, ""
		s.setInput(s.Path)
	case tea.KeyEnter:
		return m, s.submit()
	default:
		if editInput(&s.Input, &s.InputPos, msg) {
			s.Err = ""
		}
	}
	return m, nil
}

// submit validates the current input step and advances on success; errors
// stay inline on the same step. On the path step it starts the scan and
// returns the command that runs it.
func (s *MemoryImportState) submit() tea.Cmd {
	value := strings.TrimSpace(s.Input)
	if s.Step == screens.MemoryImportPath {
		if value == "" {
			s.Err = "enter a file or directory to import"
			return nil
		}
		s.scanSeq++
		s.scanned = nil
		s.Step, s.Err = screens.MemoryImportScanning, ""
		return scanMemoryImport(s.scanSeq, expandHome(value, homeDir()))
	}

	if value == "" {
		s.Err = "enter an Engram project for these memories"
		return nil
	}
	entries, err := memorycmd.PreviewOf(s.scanned, value)
	if err != nil {
		s.Err = err.Error()
		return nil
	}
	s.Project, s.Entries, s.Err = value, entries, ""
	s.Step = screens.MemoryImportPreview
	return nil
}

// expandHome expands a leading "~" or "~/" against home. An empty home
// leaves path unchanged rather than turning "~/x" into "/x".
func expandHome(path, home string) string {
	if home == "" {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return home + path[1:]
	}
	return path
}

// scanMemoryImport collects the entries under path off the update loop, so a
// large directory never freezes the TUI.
func scanMemoryImport(seq int, path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := memorycmd.Collect(path)
		return memoryImportScanDoneMsg{Seq: seq, Path: path, Entries: entries, Err: err}
	}
}

func (m Model) handleMemoryImportScanDone(msg memoryImportScanDoneMsg) (tea.Model, tea.Cmd) {
	s := &m.MemoryImport
	if m.Screen != ScreenMemoryImport || s.Step != screens.MemoryImportScanning || msg.Seq != s.scanSeq {
		return m, nil
	}
	if msg.Err != nil {
		s.Step, s.Err = screens.MemoryImportPath, msg.Err.Error()
		return m, nil
	}
	s.Path, s.scanned, s.Err = msg.Path, msg.Entries, ""
	s.Step = screens.MemoryImportProject
	if s.Project == "" {
		if cwd, err := os.Getwd(); err == nil {
			s.Project = memorycmd.DefaultProject(cwd)
		}
	}
	s.setInput(s.Project)
	return m, nil
}

func (s *MemoryImportState) setInput(value string) {
	s.Input, s.InputPos = value, len([]rune(value))
}

// runMemoryImport runs the import off the update loop and reports Engram's
// output, stdout and stderr, as a memoryImportDoneMsg.
func (m Model) runMemoryImport(ctx context.Context, entries []memorycmd.Entry, project string) tea.Cmd {
	run := m.memoryImportRun
	if run == nil {
		run = memorycmd.ImportEntries
	}
	return func() tea.Msg {
		var out strings.Builder
		err := run(ctx, entries, project, &out)
		return memoryImportDoneMsg{Output: out.String(), Err: err}
	}
}

func (m Model) handleMemoryImportDone(msg memoryImportDoneMsg) (tea.Model, tea.Cmd) {
	if m.Screen != ScreenMemoryImport || m.MemoryImport.Step != screens.MemoryImportRunning {
		return m, nil
	}
	s := &m.MemoryImport
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	if s.quitting {
		return m, tea.Quit
	}
	s.Step, s.Output, s.Err = screens.MemoryImportResult, msg.Output, ""
	if msg.Err != nil {
		s.Err = msg.Err.Error()
	}
	return m, nil
}

// editInput applies one editing key to a single-line input and reports
// whether the key was an edit.
func editInput(input *string, pos *int, msg tea.KeyMsg) bool {
	r := []rune(*input)
	switch msg.Type {
	case tea.KeyBackspace:
		if *pos > 0 {
			*input = string(append(r[:*pos-1], r[*pos:]...))
			*pos--
		}
	case tea.KeyLeft:
		if *pos > 0 {
			*pos--
		}
		return false
	case tea.KeyRight:
		if *pos < len(r) {
			*pos++
		}
		return false
	case tea.KeyRunes, tea.KeySpace:
		typed := msg.Runes
		if msg.Type == tea.KeySpace {
			typed = []rune{' '}
		}
		next := make([]rune, 0, len(r)+len(typed))
		next = append(append(append(next, r[:*pos]...), typed...), r[*pos:]...)
		*input = string(next)
		*pos += len(typed)
	default:
		return false
	}
	return true
}
