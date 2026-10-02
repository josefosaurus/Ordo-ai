package screens

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gentleman-programming/gentle-ai/v4/internal/brand"
	"github.com/gentleman-programming/gentle-ai/v4/internal/ordopersona"
	"github.com/gentleman-programming/gentle-ai/v4/internal/tui/styles"
)

// CustomizeField identifies one editable row on the customize screen.
type CustomizeField int

const (
	CustomizeName CustomizeField = iota
	CustomizeTagline
	CustomizeLogo
	CustomizePrimary
	CustomizeAccent
	CustomizeVoice
	CustomizeLanguage
	CustomizeBack
)

// CustomizeFields lists the rows in display order; the last one is Back.
func CustomizeFields() []CustomizeField {
	return []CustomizeField{CustomizeName, CustomizeTagline, CustomizeLogo, CustomizePrimary, CustomizeAccent, CustomizeVoice, CustomizeLanguage, CustomizeBack}
}

// Label is the row label shown on the customize screen.
func (f CustomizeField) Label() string {
	switch f {
	case CustomizeName:
		return "Name"
	case CustomizeTagline:
		return "Tagline"
	case CustomizeLogo:
		return "Logo file"
	case CustomizePrimary:
		return "Primary color"
	case CustomizeAccent:
		return "Accent color"
	case CustomizeVoice:
		return "Persona voice"
	case CustomizeLanguage:
		return "Chat language"
	default:
		return "Back"
	}
}

// Value returns the field's current value for display and as the starting
// text when editing. The logo is edited by file path, so it starts empty.
func (f CustomizeField) Value(b brand.Brand, p ordopersona.Persona) string {
	switch f {
	case CustomizeName:
		return b.Name
	case CustomizeTagline:
		return b.Tagline
	case CustomizePrimary:
		return b.Palette.Primary
	case CustomizeAccent:
		return b.Palette.Accent
	case CustomizeVoice:
		return p.Voice
	case CustomizeLanguage:
		return p.ChatLanguage
	default:
		return ""
	}
}

// customizeValueWidth truncates long values (such as the persona voice) on
// display; editing always shows the full text.
const customizeValueWidth = 60

// CustomizeView is everything RenderCustomize needs.
type CustomizeView struct {
	Brand     brand.Brand
	Persona   ordopersona.Persona
	Version   string
	Cursor    int
	Editing   bool
	Input     string
	InputPos  int
	Status    string
	StatusErr bool
}

// RenderCustomize draws the live preview and the editable rows.
func RenderCustomize(v CustomizeView) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Customize brand & persona"))
	b.WriteString("\n")
	b.WriteString(styles.SubtextStyle.Render("Saved per user in ~/.gentle-ai. Code, docs, and commits stay in English."))
	b.WriteString("\n\n")

	// Live preview: logo, headline, and the two edited color roles.
	b.WriteString(styles.RenderLogo())
	b.WriteString("\n\n")
	b.WriteString(styles.SubtextStyle.Render(v.Brand.Headline(v.Version)))
	b.WriteString("\n")
	b.WriteString(swatch("primary", v.Brand.Palette.Primary))
	b.WriteString("   ")
	b.WriteString(swatch("accent", v.Brand.Palette.Accent))
	b.WriteString("\n\n")

	for idx, field := range CustomizeFields() {
		focused := idx == v.Cursor
		if field == CustomizeBack {
			b.WriteString("\n")
			cursor := -1
			if focused {
				cursor = 0
			}
			b.WriteString(renderOptions([]string{field.Label()}, cursor))
			continue
		}
		label := field.Label() + strings.Repeat(" ", max(0, 15-len(field.Label())))
		value := field.Value(v.Brand, v.Persona)
		switch {
		case focused && v.Editing:
			value = inputWithCursor(v.Input, v.InputPos)
		case field == CustomizeLogo:
			value = styles.SubtextStyle.Render("enter a path to a plain-text logo")
		case field == CustomizeLanguage && value == "":
			value = styles.SubtextStyle.Render("match the user's language")
		case value == "":
			value = styles.SubtextStyle.Render("(empty)")
		case len([]rune(value)) > customizeValueWidth:
			value = string([]rune(value)[:customizeValueWidth-1]) + "…"
		}
		line := label + value
		if focused {
			b.WriteString(styles.SelectedStyle.Render(styles.Cursor + label))
			b.WriteString(value)
		} else {
			b.WriteString(styles.UnselectedStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}

	if v.Status != "" {
		b.WriteString("\n")
		if v.StatusErr {
			b.WriteString(styles.ErrorStyle.Render(v.Status))
		} else {
			b.WriteString(styles.SuccessStyle.Render(v.Status))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	if v.Editing {
		b.WriteString(styles.HelpStyle.Render("enter: save • esc: cancel"))
	} else {
		b.WriteString(styles.HelpStyle.Render("j/k: navigate • enter: edit • r: reset field • esc: back"))
	}
	return b.String()
}

func swatch(name, color string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render("■■") + " " + styles.SubtextStyle.Render(name+" "+color)
}

func inputWithCursor(text string, pos int) string {
	runes := []rune(text)
	if pos > len(runes) {
		pos = len(runes)
	}
	return string(runes[:pos]) + styles.SelectedStyle.Render("|") + string(runes[pos:])
}
