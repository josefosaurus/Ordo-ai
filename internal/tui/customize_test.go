package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gentleman-programming/gentle-ai/v4/internal/brand"
	"github.com/gentleman-programming/gentle-ai/v4/internal/ordopersona"
	"github.com/gentleman-programming/gentle-ai/v4/internal/system"
	"github.com/gentleman-programming/gentle-ai/v4/internal/tui/screens"
	"github.com/gentleman-programming/gentle-ai/v4/internal/tui/styles"
)

func newCustomizeModel(t *testing.T) (Model, string) {
	t.Helper()
	home := t.TempDir()
	orig := customizeHome
	customizeHome = func() string { return home }
	t.Cleanup(func() {
		customizeHome = orig
		brand.Set(brand.Default())
		styles.Apply(brand.Default().Palette)
	})
	m := NewModel(system.DetectionResult{}, "v-test")
	next, _ := m.startCustomize()
	return next.(Model), home
}

func press(t *testing.T, m Model, keys ...tea.KeyMsg) Model {
	t.Helper()
	for _, k := range keys {
		next, _ := m.Update(k)
		m = next.(Model)
	}
	return m
}

func typeText(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

var (
	keyEnter = tea.KeyMsg{Type: tea.KeyEnter}
	keyEsc   = tea.KeyMsg{Type: tea.KeyEsc}
	keyDown  = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}
	keyReset = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")}
)

func clearInput(m Model) Model {
	m.Customize.Input, m.Customize.InputPos = "", 0
	return m
}

func TestCustomizeIsOfferedOnWelcomeBeforeQuit(t *testing.T) {
	opts := screens.WelcomeOptions(nil, false, false, 0, true)
	if len(opts) < 2 || opts[len(opts)-2] != "Customize brand & persona" || opts[len(opts)-1] != "Quit" {
		t.Fatalf("welcome options end = %q", opts[len(opts)-2:])
	}
}

func TestCustomizeEditsNameWithLivePreview(t *testing.T) {
	m, home := newCustomizeModel(t)
	if m.Screen != ScreenCustomize || !strings.Contains(m.View(), "Customize brand & persona") {
		t.Fatalf("not on customize screen: %v", m.Screen)
	}

	m = press(t, m, keyEnter) // edit Name
	if !m.Customize.Editing || m.Customize.Input != "Ordo" {
		t.Fatalf("editing=%v input=%q, want prefilled current name", m.Customize.Editing, m.Customize.Input)
	}
	m = clearInput(m)
	m = press(t, m, typeText("Acme"), tea.KeyMsg{Type: tea.KeySpace}, typeText("Corp"), keyEnter)

	if m.Customize.StatusErr || m.Customize.Brand.Name != "Acme Corp" {
		t.Fatalf("status=%q err=%v name=%q", m.Customize.Status, m.Customize.StatusErr, m.Customize.Brand.Name)
	}
	if b, _ := brand.Load(home); b.Name != "Acme Corp" {
		t.Fatalf("override not written: %q", b.Name)
	}
	if !strings.Contains(m.View(), "Acme Corp v-test") {
		t.Fatal("preview headline must show the new name")
	}
}

func TestCustomizeRejectsInvalidColorWithoutWriting(t *testing.T) {
	m, home := newCustomizeModel(t)
	m = press(t, m, keyDown, keyDown, keyDown, keyEnter) // Primary color
	m = clearInput(m)
	m = press(t, m, typeText("red"), keyEnter)

	if !m.Customize.StatusErr || !strings.Contains(m.Customize.Status, "#RRGGBB") {
		t.Fatalf("status=%q err=%v, want a color validation error", m.Customize.Status, m.Customize.StatusErr)
	}
	if _, err := os.Stat(brand.OverridePath(home)); !os.IsNotExist(err) {
		t.Fatal("invalid color must not create an override")
	}
}

func TestCustomizeSavesPersonaAndResets(t *testing.T) {
	m, home := newCustomizeModel(t)
	for i := 0; i < int(screens.CustomizeVoice); i++ {
		m = press(t, m, keyDown)
	}
	m = press(t, m, keyEnter)
	m = clearInput(m)
	m = press(t, m, typeText("Friendly"), keyEnter)

	if p, _ := ordopersona.Load(home); p.Voice != "Friendly" {
		t.Fatalf("voice = %q", p.Voice)
	}
	if !strings.Contains(m.Customize.Status, "sync") {
		t.Fatalf("persona save must say how to apply it: %q", m.Customize.Status)
	}

	m = press(t, m, keyReset)
	if p, _ := ordopersona.Load(home); p.Voice != ordopersona.Default().Voice {
		t.Fatalf("reset voice = %q", p.Voice)
	}
}

func TestCustomizeLogoFromFileAndEscape(t *testing.T) {
	m, home := newCustomizeModel(t)
	logo := filepath.Join(home, "logo.txt")
	if err := os.WriteFile(logo, []byte("ACME\n----\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m = press(t, m, keyDown, keyDown, keyEnter) // Logo file
	m = press(t, m, typeText(logo), keyEnter)
	if m.Customize.StatusErr || !strings.Contains(m.View(), "ACME") {
		t.Fatalf("logo not applied: %q", m.Customize.Status)
	}

	m = press(t, m, keyEnter, typeText("ignored"), keyEsc) // cancel an edit
	if m.Customize.Editing || m.Screen != ScreenCustomize {
		t.Fatal("esc while editing must cancel and stay on the screen")
	}
	m = press(t, m, keyEsc)
	if m.Screen != ScreenWelcome {
		t.Fatalf("esc must return to welcome, got %v", m.Screen)
	}
}
