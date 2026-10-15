package pi

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestGentlePiConfigDirNormalizesTildeAndRelativeValues(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	t.Chdir(cwd)
	// t.TempDir may sit behind a symlink (macOS /var -> /private/var); compare
	// against the cwd the process actually resolves.
	resolvedCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}

	for _, tt := range []struct {
		name       string
		configDir  string
		configHome string
		want       string
	}{
		{"config dir bare tilde is home", "~", "", home},
		{"config dir tilde expands against home", "~/bundled/gentle-ai", "", filepath.Join(home, "bundled", "gentle-ai")},
		{"config dir relative resolves against cwd", filepath.Join("rel", "gentle-ai"), "", filepath.Join(resolvedCwd, "rel", "gentle-ai")},
		{"config dir absolute is cleaned", filepath.Join(home, "a", "..", "b"), "", filepath.Join(home, "b")},
		{"config home tilde expands against home", "", "~/cfg", filepath.Join(home, "cfg", "gentle-ai")},
		{"config home relative resolves against cwd", "", "cfg", filepath.Join(resolvedCwd, "cfg", "gentle-ai")},
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

func TestGentlePiConfigDirFallsBackWhenCwdIsUnavailable(t *testing.T) {
	home := t.TempDir()
	original := resolveAbsPath
	resolveAbsPath = func(string) (string, error) { return "", errors.New("getwd failed") }
	t.Cleanup(func() { resolveAbsPath = original })

	t.Setenv(GentlePiConfigDirEnv, "rel")
	t.Setenv(GentlePiConfigHomeEnv, "")
	want := filepath.Join(home, ".pi", "gentle-ai")
	if got := GentlePiConfigDir(home); got != want {
		t.Fatalf("GentlePiConfigDir() = %q, want default %q", got, want)
	}
}

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
