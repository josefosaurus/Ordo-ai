package brandcmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/brand"
	"github.com/gentleman-programming/gentle-ai/v4/internal/tui/styles"
)

func run(t *testing.T, home string, args ...string) (string, error) {
	t.Helper()
	t.Cleanup(func() {
		brand.Set(brand.Default())
		styles.Apply(brand.Default().Palette)
	})
	var out bytes.Buffer
	err := Run(args, home, &out)
	return out.String(), err
}

func TestSetShowReset(t *testing.T) {
	home := t.TempDir()
	if _, err := run(t, home, "set", "name", "Acme"); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, home, "set", "color.primary", "#ff0000"); err != nil {
		t.Fatal(err)
	}
	logo := filepath.Join(t.TempDir(), "logo.txt")
	if err := os.WriteFile(logo, []byte("ACME\r\n----\n\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, home, "set", "logo", logo); err != nil {
		t.Fatal(err)
	}

	b, warnings := brand.Load(home)
	if len(warnings) != 0 || b.Name != "Acme" || b.Palette.Primary != "#ff0000" {
		t.Fatalf("brand after set = %+v, %v", b, warnings)
	}
	if strings.Join(b.Logo, "|") != "ACME|----" {
		t.Fatalf("logo = %q", b.Logo)
	}

	out, err := run(t, home, "show")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"ACME", "name         Acme", "#ff0000", brand.OverridePath(home)} {
		if !strings.Contains(out, want) {
			t.Fatalf("show output missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "attribution") {
		t.Fatalf("show output must not list an attribution:\n%s", out)
	}

	if _, err := run(t, home, "reset", "name"); err != nil {
		t.Fatal(err)
	}
	if b, _ := brand.Load(home); b.Name != "Ordo" || b.Palette.Primary != "#ff0000" {
		t.Fatalf("reset name must keep other fields: %+v", b)
	}
	if _, err := run(t, home, "reset"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(brand.OverridePath(home)); !os.IsNotExist(err) {
		t.Fatal("full reset must remove the override file")
	}
}

func TestSetRejectsInvalidValuesWithoutWriting(t *testing.T) {
	home := t.TempDir()
	cases := [][]string{
		{"set", "color.primary", "red"},
		{"set", "color.nope", "#ffffff"},
		{"set", "nope", "x"},
		{"set", "name", strings.Repeat("x", brand.MaxNameRunes+1)},
		{"set", "gradient", "#fff,#000"},
		{"set", "logo", filepath.Join(home, "missing.txt")},
		{"set", "name"},
		{"frobnicate"},
	}
	for _, args := range cases {
		if _, err := run(t, home, args...); err == nil {
			t.Fatalf("Run(%q) succeeded, want error", args)
		}
	}
	if _, err := os.Stat(brand.OverridePath(home)); !os.IsNotExist(err) {
		t.Fatal("invalid input must not create an override file")
	}
}

func TestShowReportsBrokenOverrideAndResetRecovers(t *testing.T) {
	home := t.TempDir()
	path := brand.OverridePath(home)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("name: [broken\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, home, "show")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "name         Ordo") || !strings.Contains(out, "warning:") {
		t.Fatalf("show must fall back to defaults with a warning:\n%s", out)
	}
	if _, err := run(t, home, "set", "name", "Acme"); err == nil {
		t.Fatal("set on a broken override must refuse instead of overwriting it")
	}
	if _, err := run(t, home, "reset"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("reset must remove the broken override")
	}
}

func TestHelp(t *testing.T) {
	out, err := run(t, t.TempDir())
	if err != nil || !strings.Contains(out, "ordo brand set name") {
		t.Fatalf("help = %q, %v", out, err)
	}
}
