package styles

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/gentleman-programming/gentle-ai/v4/internal/brand"
)

func TestApplyRebuildsStylesFromPalette(t *testing.T) {
	t.Cleanup(func() { Apply(brand.Default().Palette) })

	p := brand.Default().Palette
	p.Primary = "#123456"
	p.Error = "#abcdef"
	Apply(p)

	if got := TitleStyle.GetForeground(); got != lipgloss.Color("#123456") {
		t.Fatalf("TitleStyle foreground = %v, want #123456", got)
	}
	if got := FrameStyle.GetBorderTopForeground(); got != lipgloss.Color("#123456") {
		t.Fatalf("FrameStyle border = %v, want #123456", got)
	}
	if got := ErrorStyle.GetForeground(); got != lipgloss.Color("#abcdef") {
		t.Fatalf("ErrorStyle foreground = %v, want #abcdef", got)
	}
}

func TestRenderLogoAndTaglineFollowBrand(t *testing.T) {
	t.Cleanup(func() { brand.Set(brand.Default()) })

	b := brand.Default()
	b.Name = "Acme"
	b.Logo = []string{"ACME-LOGO"}
	brand.Set(b)

	if got := RenderLogo(); !strings.Contains(got, "ACME-LOGO") {
		t.Fatalf("RenderLogo() = %q, want custom logo", got)
	}
	if got := Tagline("v1"); got != "Acme v1" {
		t.Fatalf("Tagline() = %q", got)
	}
}
