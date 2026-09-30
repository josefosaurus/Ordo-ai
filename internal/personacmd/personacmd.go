// Package personacmd implements `ordo persona`: show, set, and reset the
// per-user Ordo persona override (see internal/ordopersona).
package personacmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/gentleman-programming/gentle-ai/v4/internal/brand"
	"github.com/gentleman-programming/gentle-ai/v4/internal/ordopersona"
)

const usage = `Customize how your agents talk to you with the ordo persona: voice, chat
language, and team rules. Changes are stored per user in ~/.gentle-ai/persona.yaml
and reach your agents on the next ordo sync.

USAGE
  ordo persona show
  ordo persona set voice <text>
  ordo persona set language <text>     e.g. Spanish; empty default matches the user
  ordo persona add-rule <text>
  ordo persona remove-rule <number>    number as listed by show
  ordo persona reset [voice|language|rules]

Code, UI copy, docs, and commits always stay in English; the voice styles chat only.
`

const applyHint = "run ordo sync to apply it to your agents (persona must be ordo)"

// Run dispatches a persona subcommand for the user whose home is homeDir.
func Run(args []string, homeDir string, stdout io.Writer) error {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stdout, usage)
		return nil
	}
	switch args[0] {
	case "help", "--help", "-h":
		_, _ = fmt.Fprint(stdout, usage)
		return nil
	case "show":
		return show(homeDir, stdout)
	case "set":
		if len(args) != 3 {
			return errors.New("usage: ordo persona set <voice|language> <text> (see ordo persona help)")
		}
		return set(homeDir, args[1], args[2], stdout)
	case "add-rule":
		if len(args) != 2 {
			return errors.New("usage: ordo persona add-rule <text> (see ordo persona help)")
		}
		return addRule(homeDir, args[1], stdout)
	case "remove-rule":
		if len(args) != 2 {
			return errors.New("usage: ordo persona remove-rule <number> (see ordo persona help)")
		}
		return removeRule(homeDir, args[1], stdout)
	case "reset":
		if len(args) > 2 {
			return errors.New("usage: ordo persona reset [voice|language|rules] (see ordo persona help)")
		}
		field := ""
		if len(args) == 2 {
			field = args[1]
		}
		return reset(homeDir, field, stdout)
	default:
		return fmt.Errorf("unknown persona command %q (see ordo persona help)", args[0])
	}
}

func show(homeDir string, stdout io.Writer) error {
	p, warnings := ordopersona.Load(homeDir)
	language := p.ChatLanguage
	if language == "" {
		language = "match the user's language"
	}
	_, _ = fmt.Fprintf(stdout, "voice     %s\n", p.Voice)
	_, _ = fmt.Fprintf(stdout, "language  %s\n", language)
	_, _ = fmt.Fprintln(stdout, "rules")
	for i, r := range p.Rules {
		_, _ = fmt.Fprintf(stdout, "  %d. %s\n", i+1, r)
	}
	path := ordopersona.OverridePath(homeDir)
	if _, err := os.Stat(path); err == nil {
		_, _ = fmt.Fprintf(stdout, "\noverride  %s\n", path)
	} else {
		_, _ = fmt.Fprintln(stdout, "\noverride  none (using defaults)")
	}
	for _, w := range warnings {
		_, _ = fmt.Fprintln(stdout, "warning: "+w)
	}
	_, _ = fmt.Fprintf(stdout, "\n--- what agents receive ---\n%s", ordopersona.Render(p, brand.Current().Name))
	return nil
}

func readOverride(homeDir string) (ordopersona.Persona, error) {
	o, err := ordopersona.ReadOverride(homeDir)
	if err != nil {
		return o, fmt.Errorf("%w; fix the file or run ordo persona reset", err)
	}
	return o, nil
}

func write(homeDir string, o ordopersona.Persona, done string, stdout io.Writer) error {
	if err := ordopersona.WriteOverride(homeDir, o); err != nil {
		return fmt.Errorf("persona not changed: %w", err)
	}
	_, _ = fmt.Fprintf(stdout, "persona %s; %s\n", done, applyHint)
	return nil
}

func set(homeDir, field, value string, stdout io.Writer) error {
	o, err := readOverride(homeDir)
	if err != nil {
		return err
	}
	switch field {
	case "voice":
		o.Voice = value
	case "language":
		o.ChatLanguage = value
	default:
		return fmt.Errorf("unknown persona field %q (use voice or language; see ordo persona help)", field)
	}
	return write(homeDir, o, field+" updated", stdout)
}

func addRule(homeDir, rule string, stdout io.Writer) error {
	o, err := readOverride(homeDir)
	if err != nil {
		return err
	}
	if len(o.Rules) == 0 {
		// Start from the effective rules so adding one keeps the defaults.
		o.Rules = ordopersona.Default().Rules
	}
	o.Rules = append(o.Rules, rule)
	return write(homeDir, o, "rule added", stdout)
}

func removeRule(homeDir, number string, stdout io.Writer) error {
	o, err := readOverride(homeDir)
	if err != nil {
		return err
	}
	rules := o.Rules
	if len(rules) == 0 {
		rules = ordopersona.Default().Rules
	}
	n, err := strconv.Atoi(strings.TrimSpace(number))
	if err != nil || n < 1 || n > len(rules) {
		return fmt.Errorf("rule number must be between 1 and %d (see ordo persona show)", len(rules))
	}
	remaining := append(append([]string(nil), rules[:n-1]...), rules[n:]...)
	if len(remaining) == 0 {
		return errors.New("cannot remove the last rule; the persona needs at least one (use ordo persona reset rules to restore defaults)")
	}
	o.Rules = remaining
	return write(homeDir, o, "rule removed", stdout)
}

func reset(homeDir, field string, stdout io.Writer) error {
	if field == "" {
		return write(homeDir, ordopersona.Persona{}, "reset to defaults", stdout)
	}
	o, err := readOverride(homeDir)
	if err != nil {
		return fmt.Errorf("%w; run ordo persona reset to remove the whole file", err)
	}
	switch field {
	case "voice":
		o.Voice = ""
	case "language":
		o.ChatLanguage = ""
	case "rules":
		o.Rules = nil
	default:
		return fmt.Errorf("unknown persona field %q (use voice, language, or rules; see ordo persona help)", field)
	}
	return write(homeDir, o, field+" reset to default", stdout)
}
