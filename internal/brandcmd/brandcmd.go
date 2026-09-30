// Package brandcmd implements `gentle-ai brand`: show, set, and reset the
// per-user brand override (see internal/brand).
package brandcmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gentleman-programming/gentle-ai/v4/internal/brand"
	"github.com/gentleman-programming/gentle-ai/v4/internal/tui/styles"
)

// maxLogoFileBytes bounds `brand set logo <file>` reads.
const maxLogoFileBytes = 64 * 1024

const usage = `Customize the name, tagline, logo, and colors shown by the CLI and TUI.
Changes are stored per user in ~/.gentle-ai/brand.yaml.

USAGE
  gentle-ai brand show
  gentle-ai brand set name <text>
  gentle-ai brand set tagline <text>
  gentle-ai brand set logo <file>          plain-text logo, one line per row
  gentle-ai brand set color.<role> <#RRGGBB>
  gentle-ai brand set gradient <#RRGGBB,#RRGGBB,...>
  gentle-ai brand reset [name|tagline|logo|color.<role>|gradient]

COLOR ROLES
  primary accent text muted border success error warning highlight
`

var colorRoles = []string{"primary", "accent", "text", "muted", "border", "success", "error", "warning", "highlight"}

// Run dispatches a brand subcommand for the user whose home is homeDir.
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
			return errors.New("usage: gentle-ai brand set <field> <value> (see gentle-ai brand help)")
		}
		return set(homeDir, args[1], args[2], stdout)
	case "reset":
		if len(args) > 2 {
			return errors.New("usage: gentle-ai brand reset [field] (see gentle-ai brand help)")
		}
		field := ""
		if len(args) == 2 {
			field = args[1]
		}
		return reset(homeDir, field, stdout)
	default:
		return fmt.Errorf("unknown brand command %q (see gentle-ai brand help)", args[0])
	}
}

func show(homeDir string, stdout io.Writer) error {
	b, warnings := brand.Load(homeDir)
	styles.Apply(b.Palette)
	brand.Set(b)

	_, _ = fmt.Fprintln(stdout, styles.RenderLogo())
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintf(stdout, "name         %s\n", b.Name)
	_, _ = fmt.Fprintf(stdout, "tagline      %s\n", b.Tagline)
	_, _ = fmt.Fprintf(stdout, "attribution  %s (not editable)\n", b.Attribution)
	_, _ = fmt.Fprintln(stdout, "colors")
	for _, role := range colorRoles {
		c := *colorField(&b.Palette, role)
		swatch := styles.UnselectedStyle.Foreground(lipgloss.Color(c)).Render("■")
		_, _ = fmt.Fprintf(stdout, "  %-10s %s %s\n", role, c, swatch)
	}
	_, _ = fmt.Fprintf(stdout, "  %-10s %s\n", "gradient", strings.Join(b.Palette.LogoGradient, ","))

	path := brand.OverridePath(homeDir)
	if _, err := os.Stat(path); err == nil {
		_, _ = fmt.Fprintf(stdout, "\noverride     %s\n", path)
	} else {
		_, _ = fmt.Fprintln(stdout, "\noverride     none (using defaults)")
	}
	for _, w := range warnings {
		_, _ = fmt.Fprintln(stdout, "warning: "+w)
	}
	return nil
}

func set(homeDir, field, value string, stdout io.Writer) error {
	o, err := brand.ReadOverride(homeDir)
	if err != nil {
		return fmt.Errorf("%w; fix the file or run gentle-ai brand reset", err)
	}
	switch {
	case field == "name":
		o.Name = value
	case field == "tagline":
		o.Tagline = value
	case field == "logo":
		lines, err := readLogo(value)
		if err != nil {
			return err
		}
		o.Logo = lines
	case field == "gradient":
		o.Palette.LogoGradient = splitList(value)
	case strings.HasPrefix(field, "color."):
		dst := colorField(&o.Palette, strings.TrimPrefix(field, "color."))
		if dst == nil {
			return fmt.Errorf("unknown color role %q (roles: %s)", strings.TrimPrefix(field, "color."), strings.Join(colorRoles, ", "))
		}
		*dst = value
	default:
		return fmt.Errorf("unknown brand field %q (see gentle-ai brand help)", field)
	}
	if err := brand.WriteOverride(homeDir, o); err != nil {
		return fmt.Errorf("brand not changed: %w", err)
	}
	_, _ = fmt.Fprintf(stdout, "brand %s updated\n", field)
	return nil
}

func reset(homeDir, field string, stdout io.Writer) error {
	if field == "" {
		if err := brand.WriteOverride(homeDir, brand.Override{}); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(stdout, "brand reset to defaults")
		return nil
	}
	o, err := brand.ReadOverride(homeDir)
	if err != nil {
		return fmt.Errorf("%w; run gentle-ai brand reset to remove the whole file", err)
	}
	switch {
	case field == "name":
		o.Name = ""
	case field == "tagline":
		o.Tagline = ""
	case field == "logo":
		o.Logo = nil
	case field == "gradient":
		o.Palette.LogoGradient = nil
	case strings.HasPrefix(field, "color."):
		dst := colorField(&o.Palette, strings.TrimPrefix(field, "color."))
		if dst == nil {
			return fmt.Errorf("unknown color role %q (roles: %s)", strings.TrimPrefix(field, "color."), strings.Join(colorRoles, ", "))
		}
		*dst = ""
	default:
		return fmt.Errorf("unknown brand field %q (see gentle-ai brand help)", field)
	}
	if err := brand.WriteOverride(homeDir, o); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "brand %s reset to default\n", field)
	return nil
}

func readLogo(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxLogoFileBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxLogoFileBytes {
		return nil, fmt.Errorf("logo file %s is larger than %d bytes", path, maxLogoFileBytes)
	}
	text := strings.TrimRight(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("logo file %s is empty", path)
	}
	return strings.Split(text, "\n"), nil
}

func splitList(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

func colorField(p *brand.Palette, role string) *string {
	switch role {
	case "primary":
		return &p.Primary
	case "accent":
		return &p.Accent
	case "text":
		return &p.Text
	case "muted":
		return &p.Muted
	case "border":
		return &p.Border
	case "success":
		return &p.Success
	case "error":
		return &p.Error
	case "warning":
		return &p.Warning
	case "highlight":
		return &p.Highlight
	}
	return nil
}
