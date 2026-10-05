package tui

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
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
	entries []memorycmd.Entry
	project string
	calls   int
}

// newMemoryImportModel opens the import screen from the welcome menu with a
// stubbed import, so no test ever runs the real engram.
func newMemoryImportModel(t *testing.T, output string, importErr error) (Model, *importCall) {
	t.Helper()
	call := &importCall{}
	m := NewModel(system.DetectionResult{}, "v-test")
	m.memoryImportRun = func(_ context.Context, entries []memorycmd.Entry, project string, out io.Writer) error {
		call.entries, call.project = entries, project
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

// enterPath types path, presses Enter and feeds the scan result back.
func enterPath(t *testing.T, m Model, path string) Model {
	t.Helper()
	next, cmd := m.Update(typeText(path))
	next, cmd = next.(Model).Update(keyEnter)
	return runCmd(t, next.(Model), cmd)
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

	m = press(t, m, typeText(file))
	next, scan := m.Update(keyEnter)
	m = next.(Model)
	if m.MemoryImport.Step != screens.MemoryImportScanning || !strings.Contains(m.View(), "Scanning") {
		t.Fatalf("step = %v, want scanning with Scanning… view:\n%s", m.MemoryImport.Step, m.View())
	}
	if len(m.MemoryImport.scanned) != 0 {
		t.Fatal("path was scanned inside Update; it must scan as a command")
	}
	m = press(t, m, typeText("zzz")) // ignored while scanning
	if m.MemoryImport.Input != file {
		t.Fatalf("input changed while scanning: %q", m.MemoryImport.Input)
	}
	m = runCmd(t, m, scan)
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
	entries, err := memorycmd.Collect(file)
	if err != nil {
		t.Fatal(err)
	}
	want, err := memorycmd.PreviewOf(entries, "demo")
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
	if call.calls != 1 || len(call.entries) != 2 || call.project != "demo" {
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
	m = enterPath(t, m, file)
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
		m = enterPath(t, m, t.TempDir())
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
		m = enterPath(t, m, memoryFile(t))
		m = press(t, setInput(m, "  "), keyEnter)
		if m.MemoryImport.Step != screens.MemoryImportProject || !strings.Contains(m.View(), "enter an Engram project") {
			t.Fatalf("step = %v view:\n%s", m.MemoryImport.Step, m.View())
		}
	})
	t.Run("import fails", func(t *testing.T) {
		m, _ := newMemoryImportModel(t, "", errors.New("engram not found on PATH"))
		m = enterPath(t, m, memoryFile(t))
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
	m.memoryImportRun = func(ctx context.Context, _ []memorycmd.Entry, _ string, _ io.Writer) error {
		<-ctx.Done()
		cancelled = true
		return ctx.Err()
	}
	m = enterPath(t, m, memoryFile(t))
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

// A second Ctrl+C while a cancelled import is still returning quits at once,
// so a hung Engram can never trap the user.
func TestMemoryImportSecondCtrlCQuitsImmediately(t *testing.T) {
	m, _ := newMemoryImportModel(t, "", nil)
	m.memoryImportRun = func(ctx context.Context, _ []memorycmd.Entry, _ string, _ io.Writer) error {
		<-ctx.Done()
		return ctx.Err()
	}
	m = enterPath(t, m, memoryFile(t))
	m = press(t, setInput(m, "demo"), keyEnter)
	next, _ := m.Update(keyEnter)
	m = next.(Model)

	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = next.(Model)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("second Ctrl+C returned no command, want tea.Quit")
	}
	if _, quit := cmd().(tea.QuitMsg); !quit {
		t.Fatalf("second Ctrl+C cmd = %T, want tea.Quit", cmd())
	}
}

// The preview and the import come from one scan: a file added to the
// directory after the preview must not reach the import.
func TestMemoryImportScansOnceForPreviewAndImport(t *testing.T) {
	m, call := newMemoryImportModel(t, "", nil)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("## One\na\n## Two\nb\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m = enterPath(t, m, dir)
	m = press(t, setInput(m, "demo"), keyEnter)
	if m.MemoryImport.Step != screens.MemoryImportPreview {
		t.Fatalf("step = %v (err %q), want preview", m.MemoryImport.Step, m.MemoryImport.Err)
	}
	preview := m.MemoryImport.Entries
	if err := os.WriteFile(filepath.Join(dir, "b.md"), []byte("## Three\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	next, cmd := m.Update(keyEnter)
	m = runCmd(t, next.(Model), cmd)
	if call.calls != 1 {
		t.Fatalf("import calls = %d, want 1", call.calls)
	}
	imported, err := memorycmd.PreviewOf(call.entries, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(imported) != len(preview) {
		t.Fatalf("imported %d entries, previewed %d", len(imported), len(preview))
	}
	for i := range preview {
		if imported[i].SyncID != preview[i].SyncID {
			t.Fatalf("imported[%d] = %s, previewed %s", i, imported[i].SyncID, preview[i].SyncID)
		}
	}
}

func TestMemoryImportEscWhileScanningIgnoresStaleResult(t *testing.T) {
	m, _ := newMemoryImportModel(t, "", nil)
	file := memoryFile(t)
	m = press(t, m, typeText(file))
	next, scan := m.Update(keyEnter)
	m = next.(Model)
	if m.MemoryImport.Step != screens.MemoryImportScanning {
		t.Fatalf("step = %v, want scanning", m.MemoryImport.Step)
	}

	m = press(t, m, keyEsc)
	if m.Screen != ScreenMemoryImport || m.MemoryImport.Step != screens.MemoryImportPath || m.MemoryImport.Input != file {
		t.Fatalf("esc while scanning: screen = %v step = %v input = %q", m.Screen, m.MemoryImport.Step, m.MemoryImport.Input)
	}
	stale := scan()

	// A new scan of a different path is under way when the stale result lands.
	other := t.TempDir()
	m = setInput(m, other)
	next, _ = m.Update(keyEnter)
	m = next.(Model)
	next, _ = m.Update(stale)
	m = next.(Model)
	if m.MemoryImport.Step != screens.MemoryImportScanning || len(m.MemoryImport.scanned) != 0 {
		t.Fatalf("stale scan result applied: step = %v scanned = %d", m.MemoryImport.Step, len(m.MemoryImport.scanned))
	}

	// Also ignored on the path step.
	m = press(t, m, keyEsc)
	next, _ = m.Update(stale)
	m = next.(Model)
	if m.MemoryImport.Step != screens.MemoryImportPath || m.MemoryImport.Path != "" {
		t.Fatalf("stale scan result applied on path step: step = %v path = %q", m.MemoryImport.Step, m.MemoryImport.Path)
	}
}

func TestMemoryImportCtrlCWhileScanningQuits(t *testing.T) {
	m, _ := newMemoryImportModel(t, "", nil)
	m = press(t, m, typeText(memoryFile(t)))
	next, _ := m.Update(keyEnter)
	_, cmd := next.(Model).Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("Ctrl+C while scanning returned no command")
	}
	if _, quit := cmd().(tea.QuitMsg); !quit {
		t.Fatalf("Ctrl+C while scanning: cmd = %T, want tea.Quit", cmd())
	}
}

func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	tests := []struct{ name, in, home, want string }{
		{"tilde slash", "~/x", home, home + "/x"},
		{"bare tilde", "~", home, home},
		{"absolute", "/abs/x", home, "/abs/x"},
		{"tilde user is not expanded", "~bob/x", home, "~bob/x"},
		{"empty home keeps tilde slash", "~/x", "", "~/x"},
		{"empty home keeps bare tilde", "~", "", "~"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := expandHome(tt.in, tt.home); got != tt.want {
				t.Fatalf("expandHome(%q, %q) = %q, want %q", tt.in, tt.home, got, tt.want)
			}
		})
	}
}

// The path step expands ~ against the user's home directory.
func TestMemoryImportExpandsTildeInPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("home directory comes from USERPROFILE on Windows")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, "notes.md"), []byte("## One\na\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, _ := newMemoryImportModel(t, "", nil)
	m = enterPath(t, m, "~/notes.md")
	if want := home + "/notes.md"; m.MemoryImport.Step != screens.MemoryImportProject || m.MemoryImport.Path != want {
		t.Fatalf("step = %v path = %q (err %q), want project step with %q", m.MemoryImport.Step, m.MemoryImport.Path, m.MemoryImport.Err, want)
	}
}
