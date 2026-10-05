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

// memoryImportFunc runs one import; memorycmd.Import unless a test injects
// a stub through Model.memoryImportRun.
type memoryImportFunc func(ctx context.Context, path, project string, out io.Writer) error

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

	// cancel stops a running import; quitting records a Ctrl+C that waits
	// for the cancelled import to return before the TUI exits.
	cancel   context.CancelFunc
	quitting bool
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
			return m, m.runMemoryImport(ctx, s.Path, s.Project)
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
		s.submit()
	default:
		if editInput(&s.Input, &s.InputPos, msg) {
			s.Err = ""
		}
	}
	return m, nil
}

// submit validates the current input step and advances on success; errors
// stay inline on the same step.
func (s *MemoryImportState) submit() {
	value := strings.TrimSpace(s.Input)
	if s.Step == screens.MemoryImportPath {
		if value == "" {
			s.Err = "enter a file or directory to import"
			return
		}
		if strings.HasPrefix(value, "~/") {
			value = homeDir() + value[1:]
		}
		// Validate the path now so a typo is reported where it was typed.
		if _, err := memorycmd.Collect(value); err != nil {
			s.Err = err.Error()
			return
		}
		s.Path, s.Err = value, ""
		s.Step = screens.MemoryImportProject
		if s.Project == "" {
			if cwd, err := os.Getwd(); err == nil {
				s.Project = memorycmd.DefaultProject(cwd)
			}
		}
		s.setInput(s.Project)
		return
	}

	if value == "" {
		s.Err = "enter an Engram project for these memories"
		return
	}
	entries, err := memorycmd.Preview(s.Path, value)
	if err != nil {
		s.Err = err.Error()
		return
	}
	s.Project, s.Entries, s.Err = value, entries, ""
	s.Step = screens.MemoryImportPreview
}

func (s *MemoryImportState) setInput(value string) {
	s.Input, s.InputPos = value, len([]rune(value))
}

// runMemoryImport runs the import off the update loop and reports Engram's
// output, stdout and stderr, as a memoryImportDoneMsg.
func (m Model) runMemoryImport(ctx context.Context, path, project string) tea.Cmd {
	run := m.memoryImportRun
	if run == nil {
		run = memorycmd.Import
	}
	return func() tea.Msg {
		var out strings.Builder
		err := run(ctx, path, project, &out)
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
