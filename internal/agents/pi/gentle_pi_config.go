package pi

import (
	"os"
	"path/filepath"
	"strings"
)

// GentlePiConfigDirEnv names the exact gentle-pi config folder that holds
// persona.json and background-subagents.json. Gentle Shell exports it to the
// Ordo processes it starts so a bundled install writes where gentle-pi reads.
const GentlePiConfigDirEnv = "GENTLE_AI_PI_CONFIG_DIR"

// GentlePiConfigHomeEnv names gentle-pi's config base directory; the managed
// files live in its "gentle-ai" child.
const GentlePiConfigHomeEnv = "GENTLE_PI_CONFIG_HOME"

// GentlePiConfigDir returns the global folder gentle-pi reads its managed
// config from: GENTLE_AI_PI_CONFIG_DIR as the folder itself, else
// GENTLE_PI_CONFIG_HOME/gentle-ai, else homeDir/.pi/gentle-ai. Blank values
// are ignored. Each override is normalized so the result never depends on how
// a caller later joins it: "~" and "~/..." expand against homeDir, absolute
// paths are cleaned, and relative paths resolve against the process cwd. If
// cwd resolution fails the override is ignored. Workspace-scoped config never
// goes through this resolver.
func GentlePiConfigDir(homeDir string) string {
	if dir, ok := normalizeGentlePiConfigOverride(os.Getenv(GentlePiConfigDirEnv), homeDir); ok {
		return dir
	}
	if base, ok := normalizeGentlePiConfigOverride(os.Getenv(GentlePiConfigHomeEnv), homeDir); ok {
		return filepath.Join(base, "gentle-ai")
	}
	return filepath.Join(ConfigPath(homeDir), "gentle-ai")
}

// normalizeGentlePiConfigOverride turns one env override into an absolute,
// cleaned path. It reports false for a blank value or when a relative value
// cannot be resolved, so the caller falls through to the next source.
func normalizeGentlePiConfigOverride(value, homeDir string) (string, bool) {
	value = strings.TrimSpace(value)
	switch {
	case value == "":
		return "", false
	case value == "~":
		return filepath.Clean(homeDir), true
	case strings.HasPrefix(value, "~/"):
		return filepath.Join(homeDir, strings.TrimPrefix(value, "~/")), true
	case filepath.IsAbs(value):
		return filepath.Clean(value), true
	default:
		abs, err := resolveAbsPath(value)
		if err != nil {
			return "", false
		}
		return abs, true
	}
}
