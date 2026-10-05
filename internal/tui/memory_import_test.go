package tui

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gentleman-programming/gentle-ai/v4/internal/memorycmd"
	"github.com/gentleman-programming/gentle-ai/v4/internal/system"
	"github.com/gentleman-programming/gentle-ai/v4/internal/tui/screens"
)

// importCall records what the stubbed import received.
type importCall struct {
	path, project string
	calls         int
}

// newMemoryImportModel opens the import screen from the welcome menu with a
// stubbed import, so no test ever runs the real engram.
func newMemoryImportModel(t *testing.T, output string, importErr error) (Model, *importCall) {
	t.Helper()
	call := &importCall{}
	m := NewModel(system.DetectionResult{}, "v-test")
	m.memoryImportRun = func(_ context.Context, path, project string, out io.Writer) error {
		call.path, call.project = path, project
		call.calls++
		_, _ = io.WriteString(out, output)
		return importErr
	}
	m.Cursor = slices.Index(screens.WelcomeOptions(m.UpdateResults, m.UpdateCheckDone, false, 0, true), "Import memories")
	if m.Cursor < 0 {
		t.Fatal("welcome menu has no \"Import memories\" entry")
	}
	m = press(t, m, keyEnter)
	if m.Screen != ScreenMemoryImport || m.MemoryImport.Step != screens.MemoryImportPath {
		t.Fatalf("screen = %v step = %v, want memory import path step", m.Screen, m.MemoryImport.Step)
	}
	return m, call
}

func memoryFile(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "notes.md")
	if err := os.WriteFile(path, []byte("## Rule one\nUse X.\n## Rule two\nUse Y.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func setInput(m Model, s string) Model {
	m.MemoryImport.Input, m.MemoryImport.InputPos = s, len([]rune(s))
	return m
}

// runCmd executes a command and feeds its message back, as Bubbletea would.
func runCmd(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a command")
	}
	next, _ := m.Update(cmd())
	return next.(Model)
}

func TestImportMemoriesSitsDirectlyAboveReceiptDrivenDevelopment(t *testing.T) {
	for _, profiles := range []bool{false, true} {
		opts := screens.WelcomeOptions(nil, true, profiles, 0, true)
		imp := slices.Index(opts, "Import memories")
		rdd := slices.Index(opts, "Receipt-Driven Development")
		if imp < 0 || rdd != imp+1 {
			t.Fatalf("Import memories at %d, RDD at %d; want directly above: %v", imp, rdd, opts)
		}
		// bench/journeys_issue3766.go reaches RDD with 5 up presses from the
		// first entry (wrapping), so RDD must stay 5th from the end.
		if rdd != len(opts)-5 {
			t.Fatalf("RDD at %d, want len-5 = %d: %v", rdd, len(opts)-5, opts)
		}
	}
}

func TestMemoryImportWalksPathProjectPreviewImportResult(t *testing.T) {
	m, call := newMemoryImportModel(t, "Observations: 2 imported, 0 updated, 0 skipped stale\n", nil)
	file := memoryFile(t)

	m = press(t, m, typeText(file), keyEnter)
	if m.MemoryImport.Step != screens.MemoryImportProject || m.MemoryImport.Path != file {
		t.Fatalf("step = %v path = %q, want project step", m.MemoryImport.Step, m.MemoryImport.Path)
	}
	cwd, _ := os.Getwd()
	if want := memorycmd.DefaultProject(cwd); m.MemoryImport.Input != want || want == "" {
		t.Fatalf("project input = %q, want default %q", m.MemoryImport.Input, want)
	}

	m = press(t, setInput(m, "demo"), keyEnter)
	if m.MemoryImport.Step != screens.MemoryImportPreview {
		t.Fatalf("step = %v (err %q), want preview", m.MemoryImport.Step, m.MemoryImport.Err)
	}
	want, err := memorycmd.Preview(file, "demo")
	if err != nil {
		t.Fatal(err)
	}
	view := m.View()
	for _, e := range want {
		if !strings.Contains(view, e.SyncID) || !strings.Contains(view, e.Title) {
			t.Fatalf("preview view missing %s %q:\n%s", e.SyncID, e.Title, view)
		}
	}
	if !strings.Contains(view, "2 entries") {
		t.Fatalf("preview view missing entry count:\n%s", view)
	}

	next, cmd := m.Update(keyEnter)
	m = next.(Model)
	if m.MemoryImport.Step != screens.MemoryImportRunning || !strings.Contains(m.View(), "Importing") {
		t.Fatalf("step = %v, want running with Importing… view", m.MemoryImport.Step)
	}
	if call.calls != 0 {
		t.Fatal("import ran inside Update; it must run as a command")
	}
	m = press(t, m, keyEsc) // ignored while running
	m = runCmd(t, m, cmd)
	if call.calls != 1 || call.path != file || call.project != "demo" {
		t.Fatalf("import call = %+v", call)
	}
	if m.MemoryImport.Step != screens.MemoryImportResult || !strings.Contains(m.View(), "2 imported") {
		t.Fatalf("step = %v, result view:\n%s", m.MemoryImport.Step, m.View())
	}

	m = press(t, m, typeText("x"))
	if m.Screen != ScreenWelcome {
		t.Fatalf("screen = %v, want welcome after any key on result", m.Screen)
	}
}

func TestMemoryImportEscGoesBackOneStep(t *testing.T) {
	m, _ := newMemoryImportModel(t, "", nil)
	file := memoryFile(t)
	m = press(t, m, typeText(file), keyEnter)
	m = press(t, setInput(m, "demo"), keyEnter)
	if m.MemoryImport.Step != screens.MemoryImportPreview {
		t.Fatalf("step = %v, want preview", m.MemoryImport.Step)
	}

	m = press(t, m, keyEsc)
	if m.MemoryImport.Step != screens.MemoryImportProject || m.MemoryImport.Input != "demo" {
		t.Fatalf("esc from preview: step = %v input = %q", m.MemoryImport.Step, m.MemoryImport.Input)
	}
	m = press(t, m, keyEsc)
	if m.MemoryImport.Step != screens.MemoryImportPath || m.MemoryImport.Input != file {
		t.Fatalf("esc from project: step = %v input = %q", m.MemoryImport.Step, m.MemoryImport.Input)
	}
	m = press(t, m, keyEsc)
	if m.Screen != ScreenWelcome {
		t.Fatalf("esc from path: screen = %v, want welcome", m.Screen)
	}
}

func TestMemoryImportShowsInlineErrors(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		m, _ := newMemoryImportModel(t, "", nil)
		m = press(t, m, keyEnter)
		if m.MemoryImport.Step != screens.MemoryImportPath || !strings.Contains(m.View(), "enter a file or directory") {
			t.Fatalf("step = %v view:\n%s", m.MemoryImport.Step, m.View())
		}
	})
	t.Run("no entries", func(t *testing.T) {
		m, _ := newMemoryImportModel(t, "", nil)
		m = press(t, m, typeText(t.TempDir()), keyEnter)
		if m.MemoryImport.Step != screens.MemoryImportPath || !strings.Contains(m.View(), "no entries found") {
			t.Fatalf("step = %v view:\n%s", m.MemoryImport.Step, m.View())
		}
		// Typing again clears the stale error.
		m = press(t, m, typeText("x"))
		if m.MemoryImport.Err != "" {
			t.Fatalf("error not cleared on edit: %q", m.MemoryImport.Err)
		}
	})
	t.Run("empty project", func(t *testing.T) {
		m, _ := newMemoryImportModel(t, "", nil)
		m = press(t, m, typeText(memoryFile(t)), keyEnter)
		m = press(t, setInput(m, "  "), keyEnter)
		if m.MemoryImport.Step != screens.MemoryImportProject || !strings.Contains(m.View(), "enter an Engram project") {
			t.Fatalf("step = %v view:\n%s", m.MemoryImport.Step, m.View())
		}
	})
	t.Run("import fails", func(t *testing.T) {
		m, _ := newMemoryImportModel(t, "", errors.New("engram not found on PATH"))
		m = press(t, m, typeText(memoryFile(t)), keyEnter)
		m = press(t, setInput(m, "demo"), keyEnter)
		next, cmd := m.Update(keyEnter)
		m = runCmd(t, next.(Model), cmd)
		if m.MemoryImport.Step != screens.MemoryImportResult || !strings.Contains(m.View(), "engram not found on PATH") {
			t.Fatalf("step = %v view:\n%s", m.MemoryImport.Step, m.View())
		}
	})
}

// Ctrl+C while Engram runs must cancel the import and wait for it to return,
// so Engram stops and memorycmd removes its temporary export file, before the
// TUI quits.
func TestMemoryImportCtrlCCancelsRunningImportBeforeQuitting(t *testing.T) {
	m, _ := newMemoryImportModel(t, "", nil)
	cancelled := false
	m.memoryImportRun = func(ctx context.Context, _, _ string, _ io.Writer) error {
		<-ctx.Done()
		cancelled = true
		return ctx.Err()
	}
	m = press(t, m, typeText(memoryFile(t)), keyEnter)
	m = press(t, setInput(m, "demo"), keyEnter)
	next, importCmd := m.Update(keyEnter)
	m = next.(Model)
	if m.MemoryImport.Step != screens.MemoryImportRunning {
		t.Fatalf("step = %v, want running", m.MemoryImport.Step)
	}

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = next.(Model)
	if cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Fatal("Ctrl+C quit while the import was still running")
		}
	}

	next, cmd = m.Update(importCmd())
	if !cancelled {
		t.Fatal("Ctrl+C did not cancel the running import")
	}
	if cmd == nil {
		t.Fatal("no quit after the cancelled import returned")
	}
	if _, quit := cmd().(tea.QuitMsg); !quit {
		t.Fatalf("after the cancelled import returned, cmd = %T, want tea.Quit", cmd())
	}
	_ = next
}
