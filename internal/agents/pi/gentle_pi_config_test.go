package pi

import (
	"path/filepath"
	"testing"
)

func TestGentlePiConfigDir(t *testing.T) {
	home := t.TempDir()
	exactDir := filepath.Join(t.TempDir(), "bundled", "config", "gentle-ai")
	configHome := filepath.Join(t.TempDir(), "bundled", "config")

	for _, tt := range []struct {
		name       string
		configDir  string
		configHome string
		want       string
	}{
		{"default stays under home", "", "", filepath.Join(home, ".pi", "gentle-ai")},
		{"blank overrides are ignored", "  ", "\t", filepath.Join(home, ".pi", "gentle-ai")},
		{"config home appends gentle-ai", "", configHome, filepath.Join(configHome, "gentle-ai")},
		{"config dir is the folder itself", exactDir, "", exactDir},
		{"config dir outranks config home", exactDir, configHome, exactDir},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(GentlePiConfigDirEnv, tt.configDir)
			t.Setenv(GentlePiConfigHomeEnv, tt.configHome)
			if got := GentlePiConfigDir(home); got != tt.want {
				t.Fatalf("GentlePiConfigDir() = %q, want %q", got, tt.want)
			}
		})
	}
}
