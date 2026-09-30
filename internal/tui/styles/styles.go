package styles

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/gentleman-programming/gentle-ai/v4/internal/brand"
)

// Palette colors, set from the active brand by Apply. The names are the
// historical Rosé Pine ones; brand.Palette maps them to UI roles.
var (
	ColorOverlay  lipgloss.Color
	ColorText     lipgloss.Color
	ColorSubtext  lipgloss.Color
	ColorLavender lipgloss.Color
	ColorGreen    lipgloss.Color
	ColorPeach    lipgloss.Color
	ColorRed      lipgloss.Color
	ColorMauve    lipgloss.Color
	ColorYellow   lipgloss.Color
)

// Cursor is the prefix used for the currently focused item.
const Cursor = "▸ "

// Tagline returns the welcome screen tagline with the given version.
func Tagline(version string) string {
	return brand.Current().Headline(version)
}

// Pre-built reusable styles, rebuilt by Apply.
var (
	TitleStyle      lipgloss.Style
	HeadingStyle    lipgloss.Style
	HelpStyle       lipgloss.Style
	SubtextStyle    lipgloss.Style
	SelectedStyle   lipgloss.Style
	UnselectedStyle lipgloss.Style
	SuccessStyle    lipgloss.Style
	ErrorStyle      lipgloss.Style
	WarningStyle    lipgloss.Style
	FrameStyle      lipgloss.Style
	PanelStyle      lipgloss.Style
	ProgressFilled  lipgloss.Style
	ProgressEmpty   lipgloss.Style
	PercentStyle    lipgloss.Style
)

func init() { Apply(brand.Default().Palette) }

// Apply sets the palette colors and rebuilds every style from p. Call it
// before rendering so a user's brand override takes effect.
func Apply(p brand.Palette) {
	ColorLavender = lipgloss.Color(p.Primary)
	ColorMauve = lipgloss.Color(p.Accent)
	ColorText = lipgloss.Color(p.Text)
	ColorSubtext = lipgloss.Color(p.Muted)
	ColorOverlay = lipgloss.Color(p.Border)
	ColorGreen = lipgloss.Color(p.Success)
	ColorRed = lipgloss.Color(p.Error)
	ColorYellow = lipgloss.Color(p.Warning)
	ColorPeach = lipgloss.Color(p.Highlight)

	gradientColors = gradientColors[:0]
	for _, c := range p.LogoGradient {
		gradientColors = append(gradientColors, lipgloss.Color(c))
	}

	TitleStyle = lipgloss.NewStyle().Foreground(ColorLavender).Bold(true)
	HeadingStyle = lipgloss.NewStyle().Foreground(ColorMauve).Bold(true)
	HelpStyle = lipgloss.NewStyle().Foreground(ColorSubtext)
	SubtextStyle = lipgloss.NewStyle().Foreground(ColorSubtext)
	SelectedStyle = lipgloss.NewStyle().Foreground(ColorLavender).Bold(true)
	UnselectedStyle = lipgloss.NewStyle().Foreground(ColorText)
	SuccessStyle = lipgloss.NewStyle().Foreground(ColorGreen)
	ErrorStyle = lipgloss.NewStyle().Foreground(ColorRed)
	WarningStyle = lipgloss.NewStyle().Foreground(ColorYellow)
	FrameStyle = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(ColorLavender).
		Padding(1, 2)
	PanelStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorOverlay).
		Padding(0, 1)
	ProgressFilled = lipgloss.NewStyle().Foreground(ColorGreen)
	ProgressEmpty = lipgloss.NewStyle().Foreground(ColorOverlay)
	PercentStyle = lipgloss.NewStyle().Foreground(ColorPeach).Bold(true)
}
