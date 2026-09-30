package brand

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeOverrideFile(t *testing.T, home, content string) {
	t.Helper()
	path := OverridePath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultIsOrdo(t *testing.T) {
	d := Default()
	if d.Name != "Ordo" || d.Attribution != "based on Gentle AI" {
		t.Fatalf("Default() = %q / %q", d.Name, d.Attribution)
	}
	if len(d.Logo) == 0 || len(d.Palette.LogoGradient) == 0 {
		t.Fatal("default logo and gradient must be set")
	}
	if _, warnings := Merge(Brand{}, Override{Logo: d.Logo, Palette: d.Palette}); len(warnings) != 0 {
		t.Fatalf("default brand fails its own validation: %v", warnings)
	}
}

func TestCommandIsOrdo(t *testing.T) {
	if Command != "ordo" {
		t.Fatalf("Command = %q, want %q", Command, "ordo")
	}
}

func TestLoadWithoutOverrideReturnsDefault(t *testing.T) {
	b, warnings := Load(t.TempDir())
	if len(warnings) != 0 || b.Name != Default().Name {
		t.Fatalf("Load() = %q, %v", b.Name, warnings)
	}
}

func TestLoadMergesFieldByField(t *testing.T) {
	home := t.TempDir()
	writeOverrideFile(t, home, "name: Acme\npalette:\n  primary: \"#ff0000\"\n")
	b, warnings := Load(home)
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v", warnings)
	}
	if b.Name != "Acme" || b.Palette.Primary != "#ff0000" {
		t.Fatalf("override not applied: %q %q", b.Name, b.Palette.Primary)
	}
	d := Default()
	if b.Palette.Accent != d.Palette.Accent || strings.Join(b.Logo, "\n") != strings.Join(d.Logo, "\n") {
		t.Fatal("unset fields must keep defaults")
	}
}

func TestLoadMalformedFileFallsBackToDefault(t *testing.T) {
	home := t.TempDir()
	writeOverrideFile(t, home, "name: [unclosed\n")
	b, warnings := Load(home)
	if b.Name != Default().Name || len(warnings) != 1 {
		t.Fatalf("Load() = %q, %v; want default with one warning", b.Name, warnings)
	}
}

func TestMergeRejectsInvalidFieldsIndividually(t *testing.T) {
	cases := map[string]Override{
		"name too long":       {Name: strings.Repeat("x", MaxNameRunes+1)},
		"name escape":         {Name: "Acme\x1b[31m"},
		"tagline too long":    {Tagline: strings.Repeat("x", MaxTaglineRunes+1)},
		"logo too many lines": {Logo: make([]string, MaxLogoLines+1)},
		"logo too wide":       {Logo: []string{strings.Repeat("█", MaxLogoWidth+1)}},
		"logo escape":         {Logo: []string{"ok", "\x1b]0;pwned\a"}},
		"bad color":           {Palette: Palette{Primary: "red"}},
		"bad gradient":        {Palette: Palette{LogoGradient: []string{"#fff"}}},
	}
	for name, o := range cases {
		t.Run(name, func(t *testing.T) {
			b, warnings := Merge(Default(), o)
			if len(warnings) != 1 {
				t.Fatalf("warnings = %v, want exactly one", warnings)
			}
			d := Default()
			if b.Name != d.Name || b.Palette.Primary != d.Palette.Primary || len(b.Logo) != len(d.Logo) {
				t.Fatal("invalid field must keep the default")
			}
		})
	}
}

func TestOverrideCannotChangeAttribution(t *testing.T) {
	home := t.TempDir()
	writeOverrideFile(t, home, "name: Acme\nattribution: nobody\n")
	b, _ := Load(home)
	if b.Attribution != Default().Attribution {
		t.Fatalf("attribution = %q, want %q", b.Attribution, Default().Attribution)
	}
}

func TestWriteOverrideRoundTripAndReset(t *testing.T) {
	home := t.TempDir()
	if err := WriteOverride(home, Override{Name: "Acme", Tagline: "Build calmly"}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(OverridePath(home))
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 && os.PathSeparator == '/' {
		t.Fatalf("override mode = %v, want owner-only", perm)
	}
	b, warnings := Load(home)
	if len(warnings) != 0 || b.Name != "Acme" || b.Tagline != "Build calmly" {
		t.Fatalf("Load() = %+v, %v", b, warnings)
	}
	if err := WriteOverride(home, Override{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(OverridePath(home)); !os.IsNotExist(err) {
		t.Fatalf("empty override must remove the file, stat err = %v", err)
	}
}

func TestWriteOverrideRejectsInvalid(t *testing.T) {
	home := t.TempDir()
	if err := WriteOverride(home, Override{Palette: Palette{Error: "nope"}}); err == nil {
		t.Fatal("expected validation error")
	}
	if _, err := os.Stat(OverridePath(home)); !os.IsNotExist(err) {
		t.Fatal("invalid override must not be written")
	}
}

func TestHeadline(t *testing.T) {
	b := Default()
	if got := b.Headline("v1.2.3"); got != "Ordo v1.2.3 · based on Gentle AI" {
		t.Fatalf("Headline() = %q", got)
	}
	b.Tagline = "Build calmly"
	if got := b.Headline(""); got != "Ordo — Build calmly · based on Gentle AI" {
		t.Fatalf("Headline() = %q", got)
	}
}

func TestInitAndCurrent(t *testing.T) {
	t.Cleanup(func() { Set(Default()) })
	home := t.TempDir()
	writeOverrideFile(t, home, "name: Acme\npalette:\n  text: bad\n")
	Init(home)
	if Current().Name != "Acme" {
		t.Fatalf("Current().Name = %q", Current().Name)
	}
	if len(Warnings()) != 1 {
		t.Fatalf("Warnings() = %v", Warnings())
	}
	c := Current()
	c.Logo[0] = "mutated"
	if Current().Logo[0] == "mutated" {
		t.Fatal("Current must return a copy")
	}
}
