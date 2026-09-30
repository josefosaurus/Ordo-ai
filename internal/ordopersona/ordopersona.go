// Package ordopersona owns the editable parts of the Ordo persona: voice,
// chat language, and team rules.
//
// Defaults are embedded (default.yaml). Each user may override any field in
// ~/.gentle-ai/persona.yaml. Invalid override fields fall back to the default
// with a warning. The rendered section is appended to the neutral persona,
// whose Persona Scope rules stay authoritative and are not editable.
package ordopersona

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

//go:embed default.yaml
var defaultYAML []byte

// OverrideFile is the per-user override file name inside ~/.gentle-ai.
const OverrideFile = "persona.yaml"

// Limits keep the rendered section short enough to live in every agent's
// system prompt.
const (
	MaxVoiceRunes    = 600
	MaxLanguageRunes = 40
	MaxRules         = 20
	MaxRuleRunes     = 300
)

// Persona is the resolved editable persona.
type Persona struct {
	Voice        string   `yaml:"voice,omitempty"`
	ChatLanguage string   `yaml:"chat_language,omitempty"`
	Rules        []string `yaml:"rules,omitempty"`
}

var defaultPersona = mustParseDefault()

func mustParseDefault() Persona {
	var p Persona
	if err := yaml.Unmarshal(defaultYAML, &p); err != nil {
		panic(fmt.Sprintf("ordopersona: invalid embedded default.yaml: %v", err))
	}
	return p
}

// Default returns a copy of the embedded persona.
func Default() Persona { return clone(defaultPersona) }

// OverridePath returns the per-user override path.
func OverridePath(homeDir string) string {
	return filepath.Join(homeDir, ".gentle-ai", OverrideFile)
}

// ReadOverride reads the raw override. A missing file is an empty override.
func ReadOverride(homeDir string) (Persona, error) {
	var p Persona
	data, err := os.ReadFile(OverridePath(homeDir))
	if errors.Is(err, os.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	if err := yaml.Unmarshal(data, &p); err != nil {
		return Persona{}, fmt.Errorf("parse %s: %w", OverridePath(homeDir), err)
	}
	return p, nil
}

// WriteOverride atomically replaces the override after validating it. An
// empty override removes the file.
func WriteOverride(homeDir string, o Persona) error {
	if _, problems := Merge(Default(), o); len(problems) > 0 {
		return fmt.Errorf("invalid %s", strings.Join(problems, "; invalid "))
	}
	path := OverridePath(homeDir)
	if o.Voice == "" && o.ChatLanguage == "" && len(o.Rules) == 0 {
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

// Load resolves the persona for homeDir. It never fails; problems come back
// as warnings.
func Load(homeDir string) (Persona, []string) {
	o, err := ReadOverride(homeDir)
	if err != nil {
		return Default(), []string{fmt.Sprintf("ignoring persona override: %v", err)}
	}
	p, problems := Merge(Default(), o)
	warnings := make([]string, 0, len(problems))
	for _, pr := range problems {
		warnings = append(warnings, "persona "+pr+"; using default")
	}
	return p, warnings
}

// Merge applies o over base field by field. Invalid fields keep the base
// value and are reported as "<field>: <problem>".
func Merge(base, o Persona) (Persona, []string) {
	p := clone(base)
	var problems []string
	report := func(field string, err error) {
		problems = append(problems, fmt.Sprintf("%s: %v", field, err))
	}
	if o.Voice != "" {
		if err := validateText(strings.TrimSpace(o.Voice), MaxVoiceRunes); err != nil {
			report("voice", err)
		} else {
			p.Voice = strings.TrimSpace(o.Voice)
		}
	}
	if o.ChatLanguage != "" {
		if err := validateText(strings.TrimSpace(o.ChatLanguage), MaxLanguageRunes); err != nil {
			report("chat_language", err)
		} else {
			p.ChatLanguage = strings.TrimSpace(o.ChatLanguage)
		}
	}
	if len(o.Rules) > 0 {
		if err := validateRules(o.Rules); err != nil {
			report("rules", err)
		} else {
			p.Rules = trimAll(o.Rules)
		}
	}
	return p, problems
}

// Render returns the Markdown section appended to the neutral persona.
// productName is the active brand name.
func Render(p Persona, productName string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## %s Team Profile\n\n", productName)
	b.WriteString("This section adds the team's voice and rules to the persona above. ")
	b.WriteString("The Persona Scope rules above still decide what the voice applies to: it styles chat replies only, never code, UI copy, comments, docs, or commits.\n\n")
	if p.Voice != "" {
		fmt.Fprintf(&b, "### Voice\n\n%s\n\n", p.Voice)
	}
	if p.ChatLanguage != "" {
		fmt.Fprintf(&b, "### Chat Language\n\nReply to the user in %s unless they ask for another language. This replaces \"match the user's current language\" for chat replies only; technical artifacts follow Persona Scope.\n\n", p.ChatLanguage)
	}
	if len(p.Rules) > 0 {
		b.WriteString("### Team Rules\n\n")
		for _, r := range p.Rules {
			fmt.Fprintf(&b, "- %s\n", r)
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func validateText(s string, maxRunes int) error {
	if s == "" {
		return errors.New("must not be blank")
	}
	if n := len([]rune(s)); n > maxRunes {
		return fmt.Errorf("is %d characters, max %d", n, maxRunes)
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return errors.New("must be a single line without control characters")
		}
	}
	// Managed persona blocks are delimited by HTML comment markers; user text
	// must never be able to open or close one.
	if strings.Contains(s, "<!--") || strings.Contains(s, "-->") {
		return errors.New("must not contain HTML comment markers")
	}
	return nil
}

func validateRules(rules []string) error {
	if len(rules) > MaxRules {
		return fmt.Errorf("has %d rules, max %d", len(rules), MaxRules)
	}
	for i, r := range rules {
		if err := validateText(strings.TrimSpace(r), MaxRuleRunes); err != nil {
			return fmt.Errorf("rule %d %w", i+1, err)
		}
	}
	return nil
}

func trimAll(in []string) []string {
	out := make([]string, len(in))
	for i, s := range in {
		out[i] = strings.TrimSpace(s)
	}
	return out
}

func clone(p Persona) Persona {
	p.Rules = append([]string(nil), p.Rules...)
	return p
}
