package opencodeplugin

import (
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/brand"
)

func TestLogoPluginLabelFollowsBrand(t *testing.T) {
	t.Cleanup(func() { brand.Set(brand.Default()) })

	brand.Set(brand.Default())
	if got, want := LogoPluginLabel(), "Ordo logo TUI plugin"; got != want {
		t.Fatalf("LogoPluginLabel() = %q, want %q", got, want)
	}

	b := brand.Default()
	b.Name = "Acme"
	brand.Set(b)
	if got, want := LogoPluginLabel(), "Acme logo TUI plugin"; got != want {
		t.Fatalf("LogoPluginLabel() = %q, want %q", got, want)
	}
}
