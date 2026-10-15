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
// are ignored. Workspace-scoped config never goes through this resolver.
func GentlePiConfigDir(homeDir string) string {
	if dir := strings.TrimSpace(os.Getenv(GentlePiConfigDirEnv)); dir != "" {
		return dir
	}
	if base := strings.TrimSpace(os.Getenv(GentlePiConfigHomeEnv)); base != "" {
		return filepath.Join(base, "gentle-ai")
	}
	return filepath.Join(ConfigPath(homeDir), "gentle-ai")
}
