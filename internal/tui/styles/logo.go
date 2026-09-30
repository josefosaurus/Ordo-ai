package styles

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gentleman-programming/gentle-ai/v4/internal/brand"
)

// gradientColors is the top-to-bottom logo gradient, set by Apply from the
// brand palette.
var gradientColors []lipgloss.Color

// RenderLogo returns the ASCII logo with a top-to-bottom gradient.
func RenderLogo() string {
	logoLines := brand.Current().Logo
	total := len(logoLines)
	if total == 0 {
		return ""
	}

	bands := len(gradientColors)
	if bands == 0 {
		return strings.Join(logoLines, "\n")
	}
	var b strings.Builder

	for i, line := range logoLines {
		bandIdx := (i * bands) / total
		if bandIdx >= bands {
			bandIdx = bands - 1
		}
		style := lipgloss.NewStyle().Foreground(gradientColors[bandIdx])
		b.WriteString(style.Render(line))
		if i < total-1 {
			b.WriteByte('\n')
		}
	}

	return b.String()
}
