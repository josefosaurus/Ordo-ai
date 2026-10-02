package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gentleman-programming/gentle-ai/v4/internal/brand"
	"github.com/gentleman-programming/gentle-ai/v4/internal/ordopersona"
	"github.com/gentleman-programming/gentle-ai/v4/internal/tui/screens"
	"github.com/gentleman-programming/gentle-ai/v4/internal/tui/styles"
)

// customizeHome resolves the home directory whose brand and persona
// overrides the customize screen edits. Package-level for tests.
var customizeHome = homeDir

// CustomizeState is the customize screen's own state. The screen handles
// all of its keys itself (see handleCustomizeKey), so the shared Cursor and
// option-count machinery are not involved.
type CustomizeState struct {
	Brand     brand.Brand
	Persona   ordopersona.Persona
	Cursor    int
	Editing   bool
	Input     string
	InputPos  int
	Status    string
	StatusErr bool
}

func (m Model) startCustomize() (tea.Model, tea.Cmd) {
	m.Customize = CustomizeState{}
	m.Customize.reload()
	m.setScreen(ScreenCustomize)
	return m, nil
}

// reload re-reads both overrides and applies the brand to the running TUI so
// the preview and every other screen reflect the saved state.
func (s *CustomizeState) reload() {
	home := customizeHome()
	brand.Init(home)
	s.Brand = brand.Current()
	styles.Apply(s.Brand.Palette)
	s.Persona, _ = ordopersona.Load(home)
}

func (m Model) renderCustomize() string {
	c := m.Customize
	return screens.RenderCustomize(screens.CustomizeView{
		Brand: c.Brand, Persona: c.Persona, Version: m.Version,
		Cursor: c.Cursor, Editing: c.Editing, Input: c.Input, InputPos: c.InputPos,
		Status: c.Status, StatusErr: c.StatusErr,
	})
}

func (m Model) handleCustomizeKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	c := &m.Customize
	if c.Editing {
		switch msg.Type {
		case tea.KeyEnter:
			c.Editing = false
			field := screens.CustomizeFields()[c.Cursor]
			if err := saveCustomizeField(field, c.Input); err != nil {
				c.Status, c.StatusErr = err.Error(), true
			} else {
				c.Status, c.StatusErr = savedMessage(field), false
			}
			c.reload()
		case tea.KeyEsc:
			c.Editing, c.Status = false, ""
		case tea.KeyBackspace:
			if c.InputPos > 0 {
				r := []rune(c.Input)
				c.Input = string(append(r[:c.InputPos-1], r[c.InputPos:]...))
				c.InputPos--
			}
		case tea.KeyLeft:
			if c.InputPos > 0 {
				c.InputPos--
			}
		case tea.KeyRight:
			if c.InputPos < len([]rune(c.Input)) {
				c.InputPos++
			}
		case tea.KeyRunes, tea.KeySpace:
			typed := msg.Runes
			if msg.Type == tea.KeySpace {
				typed = []rune{' '}
			}
			r := []rune(c.Input)
			next := make([]rune, 0, len(r)+len(typed))
			next = append(append(append(next, r[:c.InputPos]...), typed...), r[c.InputPos:]...)
			c.Input = string(next)
			c.InputPos += len(typed)
		}
		return m, nil
	}

	fields := screens.CustomizeFields()
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "q":
		m.setScreen(ScreenWelcome)
	case "up", "k":
		if c.Cursor > 0 {
			c.Cursor--
		}
	case "down", "j":
		if c.Cursor < len(fields)-1 {
			c.Cursor++
		}
	case "r":
		field := fields[c.Cursor]
		if field == screens.CustomizeBack {
			break
		}
		if err := resetCustomizeField(field); err != nil {
			c.Status, c.StatusErr = err.Error(), true
		} else {
			c.Status, c.StatusErr = field.Label()+" reset to default", false
		}
		c.reload()
	case "enter":
		field := fields[c.Cursor]
		if field == screens.CustomizeBack {
			m.setScreen(ScreenWelcome)
			break
		}
		c.Editing, c.Status = true, ""
		c.Input = field.Value(c.Brand, c.Persona)
		c.InputPos = len([]rune(c.Input))
	}
	return m, nil
}

func savedMessage(field screens.CustomizeField) string {
	if field == screens.CustomizeVoice || field == screens.CustomizeLanguage {
		return field.Label() + " saved; run " + brand.Command + " sync to apply it to your agents"
	}
	return field.Label() + " saved"
}

// saveCustomizeField validates and writes one field through the same
// override files and validation the brand and persona commands use.
func saveCustomizeField(field screens.CustomizeField, value string) error {
	home := customizeHome()
	switch field {
	case screens.CustomizeVoice, screens.CustomizeLanguage:
		o, err := ordopersona.ReadOverride(home)
		if err != nil {
			return err
		}
		if field == screens.CustomizeVoice {
			o.Voice = value
		} else {
			o.ChatLanguage = value
		}
		return ordopersona.WriteOverride(home, o)
	}

	o, err := brand.ReadOverride(home)
	if err != nil {
		return err
	}
	switch field {
	case screens.CustomizeName:
		o.Name = value
	case screens.CustomizeTagline:
		o.Tagline = value
	case screens.CustomizePrimary:
		o.Palette.Primary = strings.TrimSpace(value)
	case screens.CustomizeAccent:
		o.Palette.Accent = strings.TrimSpace(value)
	case screens.CustomizeLogo:
		lines, err := readLogoFile(strings.TrimSpace(value))
		if err != nil {
			return err
		}
		o.Logo = lines
	}
	return brand.WriteOverride(home, o)
}

func resetCustomizeField(field screens.CustomizeField) error {
	home := customizeHome()
	switch field {
	case screens.CustomizeVoice, screens.CustomizeLanguage:
		o, err := ordopersona.ReadOverride(home)
		if err != nil {
			return err
		}
		if field == screens.CustomizeVoice {
			o.Voice = ""
		} else {
			o.ChatLanguage = ""
		}
		return ordopersona.WriteOverride(home, o)
	}
	o, err := brand.ReadOverride(home)
	if err != nil {
		return err
	}
	switch field {
	case screens.CustomizeName:
		o.Name = ""
	case screens.CustomizeTagline:
		o.Tagline = ""
	case screens.CustomizePrimary:
		o.Palette.Primary = ""
	case screens.CustomizeAccent:
		o.Palette.Accent = ""
	case screens.CustomizeLogo:
		o.Logo = nil
	}
	return brand.WriteOverride(home, o)
}

// maxCustomizeLogoBytes bounds logo files read from the customize screen.
const maxCustomizeLogoBytes = 64 * 1024

func readLogoFile(path string) ([]string, error) {
	if path == "" {
		return nil, fmt.Errorf("enter the path to a plain-text logo file")
	}
	if strings.HasPrefix(path, "~/") {
		path = customizeHome() + path[1:]
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > maxCustomizeLogoBytes {
		return nil, fmt.Errorf("logo file is larger than %d bytes", maxCustomizeLogoBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text := strings.TrimRight(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("logo file is empty")
	}
	return strings.Split(text, "\n"), nil
}
