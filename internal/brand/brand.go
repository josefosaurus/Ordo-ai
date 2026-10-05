// Package brand owns the user-facing identity shown by the CLI and TUI:
// display name, tagline, logo, and color palette.
//
// The compiled-in defaults (default.yaml) are the Ordo brand. Each user may
// override any field except the attribution in ~/.gentle-ai/brand.yaml.
// Invalid override fields fall back to the default with a warning; a broken
// override never blocks the CLI.
package brand

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/rivo/uniseg"
	"gopkg.in/yaml.v3"
)

//go:embed default.yaml
var defaultYAML []byte

// Command is the executable name users and agents type to run the CLI. It
// is fixed at build time and, unlike the rest of the brand, is not
// per-user editable: printed continuations, agent hooks, and plugins must
// all name the binary that is actually installed. The state directory
// (~/.gentle-ai), GENTLE_AI_* variables, and gentle-ai.* schema identifiers
// are protocol names and deliberately do not follow it.
const Command = "ordo"

// ReleaseOwner and ReleaseRepo identify the GitHub repository that publishes
// signed Ordo releases. Self-update downloads from it and requires the
// release signature's trusted comment to name exactly this repository.
const (
	ReleaseOwner = "josefosaurus"
	ReleaseRepo  = "Ordo-ai"
)

// OverrideFile is the per-user override file name inside ~/.gentle-ai.
const OverrideFile = "brand.yaml"

// Limits keep overrides renderable in an ordinary terminal.
const (
	MaxNameRunes     = 40
	MaxTaglineRunes  = 80
	MaxLogoLines     = 24
	MaxLogoWidth     = 80
	MaxGradientStops = 12
)

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Palette maps UI roles to hex colors.
type Palette struct {
	Primary      string   `yaml:"primary,omitempty"`
	Accent       string   `yaml:"accent,omitempty"`
	Text         string   `yaml:"text,omitempty"`
	Muted        string   `yaml:"muted,omitempty"`
	Border       string   `yaml:"border,omitempty"`
	Success      string   `yaml:"success,omitempty"`
	Error        string   `yaml:"error,omitempty"`
	Warning      string   `yaml:"warning,omitempty"`
	Highlight    string   `yaml:"highlight,omitempty"`
	LogoGradient []string `yaml:"logo_gradient,omitempty,flow"`
}

// Brand is the resolved identity.
type Brand struct {
	Name        string   `yaml:"name"`
	Tagline     string   `yaml:"tagline"`
	Attribution string   `yaml:"attribution"`
	Logo        []string `yaml:"logo"`
	Palette     Palette  `yaml:"palette"`
}

// Override is what a user may set. Empty fields keep the default. There is
// deliberately no attribution field.
type Override struct {
	Name    string   `yaml:"name,omitempty"`
	Tagline string   `yaml:"tagline,omitempty"`
	Logo    []string `yaml:"logo,omitempty"`
	Palette Palette  `yaml:"palette,omitempty"`
}

// Headline renders "<name> <version>[ — <tagline>] · <attribution>".
func (b Brand) Headline(version string) string {
	line := b.Name
	if version != "" {
		line += " " + version
	}
	if b.Tagline != "" {
		line += " — " + b.Tagline
	}
	return line + " · " + b.Attribution
}

var defaultBrand = mustParseDefault()

func mustParseDefault() Brand {
	var b Brand
	if err := yaml.Unmarshal(defaultYAML, &b); err != nil {
		panic(fmt.Sprintf("brand: invalid embedded default.yaml: %v", err))
	}
	return b
}

// Default returns a copy of the compiled-in brand.
func Default() Brand { return clone(defaultBrand) }

// OverridePath returns the per-user override path.
func OverridePath(homeDir string) string {
	return filepath.Join(homeDir, ".gentle-ai", OverrideFile)
}

// ReadOverride reads the raw override. A missing file is an empty override.
func ReadOverride(homeDir string) (Override, error) {
	var o Override
	data, err := os.ReadFile(OverridePath(homeDir))
	if errors.Is(err, os.ErrNotExist) {
		return o, nil
	}
	if err != nil {
		return o, err
	}
	if err := yaml.Unmarshal(data, &o); err != nil {
		return Override{}, fmt.Errorf("parse %s: %w", OverridePath(homeDir), err)
	}
	return o, nil
}

// WriteOverride atomically replaces the override file after validating it.
// An empty override removes the file.
func WriteOverride(homeDir string, o Override) error {
	if _, problems := Merge(Default(), o); len(problems) > 0 {
		return fmt.Errorf("invalid %s", strings.Join(problems, "; invalid "))
	}
	path := OverridePath(homeDir)
	if isEmpty(o) {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	data, err := yaml.Marshal(o)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), OverrideFile+".*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Load resolves the brand for homeDir: defaults merged with the user's
// override. It never fails; problems come back as warnings.
func Load(homeDir string) (Brand, []string) {
	o, err := ReadOverride(homeDir)
	if err != nil {
		return Default(), []string{fmt.Sprintf("ignoring brand override: %v", err)}
	}
	b, problems := Merge(Default(), o)
	warnings := make([]string, 0, len(problems))
	for _, p := range problems {
		warnings = append(warnings, "brand "+p+"; using default")
	}
	return b, warnings
}

// Merge applies o over base field by field. Invalid fields keep the base
// value and are reported as "<field>: <problem>".
func Merge(base Brand, o Override) (Brand, []string) {
	b := clone(base)
	var warnings []string
	warn := func(field string, err error) {
		warnings = append(warnings, fmt.Sprintf("%s: %v", field, err))
	}

	if o.Name != "" {
		if err := validateText(strings.TrimSpace(o.Name), MaxNameRunes); err != nil {
			warn("name", err)
		} else {
			b.Name = strings.TrimSpace(o.Name)
		}
	}
	if o.Tagline != "" {
		if err := validateText(strings.TrimSpace(o.Tagline), MaxTaglineRunes); err != nil {
			warn("tagline", err)
		} else {
			b.Tagline = strings.TrimSpace(o.Tagline)
		}
	}
	if len(o.Logo) > 0 {
		if err := validateLogo(o.Logo); err != nil {
			warn("logo", err)
		} else {
			b.Logo = append([]string(nil), o.Logo...)
		}
	}

	colors := []struct {
		name     string
		src, dst *string
	}{
		{"palette.primary", &o.Palette.Primary, &b.Palette.Primary},
		{"palette.accent", &o.Palette.Accent, &b.Palette.Accent},
		{"palette.text", &o.Palette.Text, &b.Palette.Text},
		{"palette.muted", &o.Palette.Muted, &b.Palette.Muted},
		{"palette.border", &o.Palette.Border, &b.Palette.Border},
		{"palette.success", &o.Palette.Success, &b.Palette.Success},
		{"palette.error", &o.Palette.Error, &b.Palette.Error},
		{"palette.warning", &o.Palette.Warning, &b.Palette.Warning},
		{"palette.highlight", &o.Palette.Highlight, &b.Palette.Highlight},
	}
	for _, c := range colors {
		if *c.src == "" {
			continue
		}
		if !hexColor.MatchString(*c.src) {
			warn(c.name, fmt.Errorf("%q is not a #RRGGBB color", *c.src))
			continue
		}
		*c.dst = *c.src
	}
	if len(o.Palette.LogoGradient) > 0 {
		if err := validateGradient(o.Palette.LogoGradient); err != nil {
			warn("palette.logo_gradient", err)
		} else {
			b.Palette.LogoGradient = append([]string(nil), o.Palette.LogoGradient...)
		}
	}
	return b, warnings
}

func validateText(s string, maxRunes int) error {
	if s == "" {
		return errors.New("must not be blank")
	}
	if n := len([]rune(s)); n > maxRunes {
		return fmt.Errorf("is %d characters, max %d", n, maxRunes)
	}
	if hasControl(s) {
		return errors.New("must not contain control characters")
	}
	return nil
}

func validateLogo(lines []string) error {
	if len(lines) > MaxLogoLines {
		return fmt.Errorf("has %d lines, max %d", len(lines), MaxLogoLines)
	}
	for i, line := range lines {
		if hasControl(line) {
			return fmt.Errorf("line %d contains control characters", i+1)
		}
		if w := uniseg.StringWidth(line); w > MaxLogoWidth {
			return fmt.Errorf("line %d is %d columns wide, max %d", i+1, w, MaxLogoWidth)
		}
	}
	return nil
}

func validateGradient(stops []string) error {
	if len(stops) > MaxGradientStops {
		return fmt.Errorf("has %d colors, max %d", len(stops), MaxGradientStops)
	}
	for _, c := range stops {
		if !hexColor.MatchString(c) {
			return fmt.Errorf("%q is not a #RRGGBB color", c)
		}
	}
	return nil
}

// hasControl rejects terminal escapes and other control characters that
// could rewrite the user's terminal when rendered.
func hasControl(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

func isEmpty(o Override) bool {
	return o.Name == "" && o.Tagline == "" && len(o.Logo) == 0 &&
		o.Palette.Primary == "" && o.Palette.Accent == "" && o.Palette.Text == "" &&
		o.Palette.Muted == "" && o.Palette.Border == "" && o.Palette.Success == "" &&
		o.Palette.Error == "" && o.Palette.Warning == "" && o.Palette.Highlight == "" &&
		len(o.Palette.LogoGradient) == 0
}

func clone(b Brand) Brand {
	b.Logo = append([]string(nil), b.Logo...)
	b.Palette.LogoGradient = append([]string(nil), b.Palette.LogoGradient...)
	return b
}

var (
	mu       sync.RWMutex
	current  = Default()
	warnings []string
)

// Init loads the user's brand for the rest of the process. Until Init runs,
// Current returns the defaults, so tests and library callers never read the
// real home directory implicitly.
func Init(homeDir string) {
	b, w := Load(homeDir)
	mu.Lock()
	current, warnings = b, w
	mu.Unlock()
}

// Current returns the active brand.
func Current() Brand {
	mu.RLock()
	defer mu.RUnlock()
	return clone(current)
}

// Warnings returns the problems found by the last Init.
func Warnings() []string {
	mu.RLock()
	defer mu.RUnlock()
	return append([]string(nil), warnings...)
}

// Set replaces the active brand. Intended for tests.
func Set(b Brand) {
	mu.Lock()
	current, warnings = clone(b), nil
	mu.Unlock()
}
