package theme

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/agents"
	"github.com/gentleman-programming/gentle-ai/v4/internal/agents/claude"
	"github.com/gentleman-programming/gentle-ai/v4/internal/agents/opencode"
	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
)

func claudeAdapter() agents.Adapter   { return claude.NewAdapter() }
func opencodeAdapter() agents.Adapter { return opencode.NewAdapter() }

func TestInjectMergesThemeOverlayIntoAdapterSettings(t *testing.T) {
	home := t.TempDir()
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("MkdirAll(settings dir) error = %v", err)
	}
	if err := os.WriteFile(settingsPath, []byte("{\n  \"permissions\": {\n    \"allow\": [\"Bash(go test ./...)\"]\n  },\n  \"theme\": \"existing-theme\"\n}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(settings) error = %v", err)
	}

	first, err := Inject(home, claudeAdapter())
	if err != nil {
		t.Fatalf("Inject() first error = %v", err)
	}
	if !first.Changed {
		t.Fatalf("Inject() first changed = false")
	}

	second, err := Inject(home, claudeAdapter())
	if err != nil {
		t.Fatalf("Inject() second error = %v", err)
	}
	if second.Changed {
		t.Fatalf("Inject() second changed = true")
	}

	if len(first.Files) != 1 || first.Files[0] != settingsPath {
		t.Fatalf("files = %#v, want only %q", first.Files, settingsPath)
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(settings) error = %v", err)
	}
	var root struct {
		Permissions map[string][]string `json:"permissions"`
		Theme       string              `json:"theme"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("Unmarshal(settings) error = %v", err)
	}
	if root.Theme != "ordo" {
		t.Fatalf("theme = %q, want ordo", root.Theme)
	}
	if got := root.Permissions["allow"]; len(got) != 1 || got[0] != "Bash(go test ./...)" {
		t.Fatalf("permissions.allow = %#v, want preserved existing permission", got)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "themes", "ordo.json")); !os.IsNotExist(err) {
		t.Fatalf("Inject() should not write Claude custom theme file; stat error = %v", err)
	}
}

func TestInjectCreatesAdapterSettingsWhenMissing(t *testing.T) {
	home := t.TempDir()

	result, err := Inject(home, opencodeAdapter())
	if err != nil {
		t.Fatalf("Inject() error = %v", err)
	}

	settingsPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	if !result.Changed {
		t.Fatalf("Inject() changed = false")
	}
	if len(result.Files) != 1 || result.Files[0] != settingsPath {
		t.Fatalf("files = %#v, want only %q", result.Files, settingsPath)
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("ReadFile(settings) error = %v", err)
	}
	var root struct {
		Theme string `json:"theme"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("Unmarshal(settings) error = %v", err)
	}
	if root.Theme != "ordo" {
		t.Fatalf("theme = %q, want ordo", root.Theme)
	}
}

func TestInjectAtPathPreservesSelectedJSONCAndLeavesDecoy(t *testing.T) {
	root := t.TempDir()
	selected := filepath.Join(root, "opencode.jsonc")
	decoy := filepath.Join(root, "opencode.json")
	before := []byte("// selected\n{\"theme\":\"old\",\"custom\":true}\n")
	if err := os.WriteFile(selected, before, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(decoy, []byte(`{"theme":"mine"}`), 0600); err != nil {
		t.Fatal(err)
	}
	first, err := InjectAtPath(selected)
	if err != nil || !first.Changed || len(first.Files) != 1 || first.Files[0] != selected {
		t.Fatalf("first injection = %#v, %v", first, err)
	}
	second, err := InjectAtPath(selected)
	if err != nil || second.Changed {
		t.Fatalf("second injection = %#v, %v", second, err)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(selected)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("selected JSONC mode = %v, %v; want 0600", info, err)
		}
	}
	got, err := os.ReadFile(selected)
	if err != nil || !bytes.Contains(got, []byte(`"theme":"ordo"`)) || !bytes.Contains(got, []byte(`"custom":true`)) || !bytes.Contains(got, []byte("// selected")) {
		t.Fatalf("selected JSONC = %s, %v", got, err)
	}
	got, err = os.ReadFile(decoy)
	if err != nil || string(got) != `{"theme":"mine"}` {
		t.Fatalf("decoy JSON = %s, %v", got, err)
	}
}

func TestInjectAtPathRejectsZeroPermissionSettingsWithoutChangingThem(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission bits are not supported on Windows")
	}

	path := filepath.Join(t.TempDir(), "opencode.jsonc")
	before := []byte("// private settings\n{\"theme\":\"old\"}\n")
	if err := os.WriteFile(path, before, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0); err != nil {
		t.Skipf("cannot set file mode 0000: %v", err)
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode().Perm() != 0 {
		t.Skipf("cannot verify file mode 0000: %v, %v", info, err)
	}

	result, err := InjectAtPath(path)
	if err == nil || !strings.Contains(err.Error(), "0000") || !strings.Contains(err.Error(), "permissions") {
		t.Errorf("InjectAtPath() = %#v, %v; want actionable mode 0000 error", result, err)
	}
	if result.Changed || len(result.Files) != 0 {
		t.Errorf("InjectAtPath() = %#v; want no changed files", result)
	}
	info, err = os.Lstat(path)
	if err != nil || info.Mode().Perm() != 0 {
		t.Fatalf("settings mode after injection = %v, %v; want 0000", info, err)
	}
	// Restore access only to inspect bytes, then leave the file locked down.
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	got, readErr := os.ReadFile(path)
	if err := os.Chmod(path, 0); err != nil {
		t.Fatal(err)
	}
	if readErr != nil || !bytes.Equal(got, before) {
		t.Fatalf("settings bytes after injection = %q, %v; want %q", got, readErr, before)
	}
}

func TestInjectAtPathRefusesSymlinkWithoutChangingLinkOrTarget(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "user.jsonc")
	selected := filepath.Join(dir, "opencode.jsonc")
	before := []byte("// private settings\n{\"theme\":\"old\"}\n")
	if err := os.WriteFile(target, before, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, selected); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	if result, err := InjectAtPath(selected); err == nil || !strings.Contains(err.Error(), "select a regular settings file") || result.Changed {
		t.Fatalf("InjectAtPath() = %#v, %v; want symlink refusal", result, err)
	}
	if link, err := os.Readlink(selected); err != nil || link != target {
		t.Fatalf("selected symlink changed: %q, %v", link, err)
	}
	got, err := os.ReadFile(target)
	if err != nil || !bytes.Equal(got, before) {
		t.Fatalf("target bytes = %q, %v; want %q", got, err, before)
	}
	if info, err := os.Stat(target); err != nil || (runtime.GOOS != "windows" && info.Mode().Perm() != 0o600) {
		t.Fatalf("target mode changed: %v, %v", info, err)
	}
}

// The selected-settings refusal is OpenCode-only; other agents keep the base
// writer behavior for a dotfiles-managed (symlinked) settings file.
func TestInjectNonOpenCodeSymlinkedSettingsKeepBaseWriterBehavior(t *testing.T) {
	home := t.TempDir()
	adapter := claudeAdapter()
	settings := adapter.SettingsPath(home)
	target := filepath.Join(home, "dotfiles", "settings.json")
	before := []byte("{\"theme\":\"old\"}\n")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, before, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, settings); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	result, err := Inject(home, adapter)
	if err == nil || !strings.Contains(err.Error(), "refusing to read symlink") || strings.Contains(err.Error(), "select a regular settings file") {
		t.Fatalf("Inject() = %#v, %v; want base writer symlink error, not the OpenCode refusal", result, err)
	}
	if link, err := os.Readlink(settings); err != nil || link != target {
		t.Fatalf("settings symlink changed: %q, %v", link, err)
	}
	if got, err := os.ReadFile(target); err != nil || !bytes.Equal(got, before) {
		t.Fatalf("target bytes = %q, %v; want %q", got, err, before)
	}
}

func TestInjectVisualThemesIsIdempotentForClaude(t *testing.T) {
	home := t.TempDir()

	first, err := InjectVisualThemes(home, claudeAdapter())
	if err != nil {
		t.Fatalf("InjectVisualThemes() first error = %v", err)
	}
	if !first.Changed {
		t.Fatalf("InjectVisualThemes() first changed = false")
	}

	second, err := InjectVisualThemes(home, claudeAdapter())
	if err != nil {
		t.Fatalf("InjectVisualThemes() second error = %v", err)
	}
	if second.Changed {
		t.Fatalf("InjectVisualThemes() second changed = true")
	}

	path := filepath.Join(home, ".claude", "themes", "ordo.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected Claude theme file %q: %v", path, err)
	}
}

func TestInjectVisualThemesSkipsUnsupportedAdapter(t *testing.T) {
	home := t.TempDir()
	adapter, _ := agents.NewAdapter(model.AgentGeminiCLI)

	result, err := InjectVisualThemes(home, adapter)
	if err != nil {
		t.Fatalf("InjectVisualThemes() error = %v", err)
	}
	if result.Changed || len(result.Files) != 0 {
		t.Fatalf("InjectVisualThemes() = %#v, want no-op for unsupported adapter", result)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "themes", "ordo.json")); !os.IsNotExist(err) {
		t.Fatalf("InjectVisualThemes() should not write Claude files for Gemini; stat error = %v", err)
	}
}

func TestInjectVisualThemesWritesOnlyOrdoTheme(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	for _, tt := range []struct {
		adapter agents.Adapter
		want    string
	}{
		{claudeAdapter(), filepath.Join(home, ".claude", "themes", "ordo.json")},
		{opencodeAdapter(), filepath.Join(xdg, "opencode", "themes", "ordo.json")},
	} {
		result, err := InjectVisualThemes(home, tt.adapter)
		if err != nil {
			t.Fatalf("InjectVisualThemes(%s) error = %v", tt.adapter.Agent(), err)
		}
		if len(result.Files) != 1 || result.Files[0] != tt.want {
			t.Fatalf("%s files = %#v, want only %q", tt.adapter.Agent(), result.Files, tt.want)
		}
		if paths := VisualThemePaths(home, tt.adapter); len(paths) != 1 || paths[0] != tt.want {
			t.Fatalf("%s VisualThemePaths = %#v, want only %q", tt.adapter.Agent(), paths, tt.want)
		}
		for _, legacy := range LegacyVisualThemePaths(home, tt.adapter) {
			if _, err := os.Stat(legacy); !os.IsNotExist(err) {
				t.Fatalf("%s legacy theme %q must not be written: %v", tt.adapter.Agent(), legacy, err)
			}
		}
	}

	var claudeRoot claudeTheme
	data, err := os.ReadFile(filepath.Join(home, ".claude", "themes", "ordo.json"))
	if err != nil || json.Unmarshal(data, &claudeRoot) != nil {
		t.Fatalf("read Claude Ordo theme: %v", err)
	}
	if claudeRoot.Name != "ordo" || claudeRoot.Base != "dark" {
		t.Fatalf("theme identity = %q/%q, want ordo/dark", claudeRoot.Name, claudeRoot.Base)
	}
	// Claude accepts only its own override keys; the Ordo theme reuses the
	// key set the previous managed theme proved valid.
	var legacyCute claudeTheme
	if err := json.Unmarshal([]byte(legacyCuteClaudeFixture), &legacyCute); err != nil {
		t.Fatal(err)
	}
	if len(claudeRoot.Overrides) != len(legacyCute.Overrides) {
		t.Fatalf("Ordo overrides = %d keys, want %d", len(claudeRoot.Overrides), len(legacyCute.Overrides))
	}
	for key := range legacyCute.Overrides {
		if _, ok := claudeRoot.Overrides[key]; !ok {
			t.Fatalf("Ordo Claude theme missing override %q", key)
		}
	}
}

const (
	ordoClaudeFixture   = `{"name":"ordo","base":"dark","overrides":{"claude":"#c4a7e7","claudeShimmer":"#ebbcba","text":"#e0def4","inactive":"#908caa","subtle":"#6e6a86","suggestion":"#ebbcba","permission":"#c4a7e7","promptBorder":"#c4a7e7","planMode":"#9ccfd8","autoAccept":"#ebbcba","bashBorder":"#f6c177","remember":"#f6c177","success":"#9ccfd8","merged":"#c4a7e7","error":"#eb6f92","warning":"#f6c177","diffAdded":"#1f2e3f","diffRemoved":"#43293a","diffAddedWord":"#274f64","diffRemovedWord":"#773f55","userMessageBackground":"#1f1d2e","userMessageBackgroundHover":"#26233a","selectionBg":"#403d52","memoryBackgroundColor":"#21202e","bashMessageBackgroundColor":"#1f1d2e"}}`
	ordoOpenCodeFixture = `{"$schema":"https://opencode.ai/theme.json","theme":{"background":"none","backgroundPanel":"#1f1d2e","backgroundElement":"#26233a","text":"#e0def4","textMuted":"#908caa","primary":"#c4a7e7","secondary":"#ebbcba","accent":"#f6c177","error":"#eb6f92","warning":"#f6c177","success":"#9ccfd8","info":"#9ccfd8","border":"#403d52","borderActive":"#c4a7e7","borderSubtle":"#26233a","diffAdded":"#9ccfd8","diffRemoved":"#eb6f92","diffContext":"#6e6a86","diffHunkHeader":"#908caa","diffHighlightAdded":"#9ccfd8","diffHighlightRemoved":"#eb6f92","diffAddedBg":"#1f2e3f","diffRemovedBg":"#43293a","diffContextBg":"#1f1d2e","diffLineNumber":"#6e6a86","diffAddedLineNumberBg":"#1f2e3f","diffRemovedLineNumberBg":"#43293a","markdownText":"#e0def4","markdownHeading":"#c4a7e7","markdownLink":"#9ccfd8","markdownLinkText":"#ebbcba","markdownCode":"#f6c177","markdownBlockQuote":"#908caa","markdownEmph":"#ebbcba","markdownStrong":"#f6c177","markdownHorizontalRule":"#6e6a86","markdownListItem":"#c4a7e7","markdownListEnumeration":"#9ccfd8","markdownImage":"#9ccfd8","markdownImageText":"#ebbcba","markdownCodeBlock":"#e0def4","syntaxComment":"#6e6a86","syntaxKeyword":"#31748f","syntaxFunction":"#ebbcba","syntaxVariable":"#e0def4","syntaxString":"#f6c177","syntaxNumber":"#f6c177","syntaxType":"#9ccfd8","syntaxOperator":"#908caa","syntaxPunctuation":"#908caa"}}`

	// Exact theme assets earlier releases installed. Migration deletes a
	// legacy file only when its bytes still match one of these.
	legacyGentlemanClaudeFixture   = `{"name":"gentleman","base":"dark","overrides":{"diffAdded":"#3F4A2D","diffRemoved":"#5C3838","diffAddedWord":"#76946A","diffRemovedWord":"#C34043","chromeYellow":"#DCA561","briefLabelYou":"#DCA561","rainbow_yellow":"#DCA561","yellow_FOR_SUBAGENTS_ONLY":"#DCA561"}}`
	legacyCuteClaudeFixture        = `{"name":"Gentleman Cute","base":"dark","overrides":{"claude":"#F095C8","claudeShimmer":"#FFB1DD","text":"#F6EFF3","inactive":"#A78E9B","subtle":"#76616B","suggestion":"#FFB1DD","permission":"#F095C8","promptBorder":"#F095C8","planMode":"#A9C7EE","autoAccept":"#FF81CC","bashBorder":"#E0C27A","remember":"#E0C27A","success":"#B4E7C7","merged":"#B4E7C7","error":"#FF718F","warning":"#F2B86D","diffAdded":"#1A2420","diffRemoved":"#2D151F","diffAddedWord":"#2D5A45","diffRemovedWord":"#7A2948","userMessageBackground":"#241822","userMessageBackgroundHover":"#342230","selectionBg":"#563040","memoryBackgroundColor":"#1A1218","bashMessageBackgroundColor":"#151316"}}`
	legacyGentlemanOpenCodeFixture = `{"$schema":"https://opencode.ai/theme.json","theme":{"background":"none","backgroundPanel":"#06080f","backgroundElement":"#06080f","text":"#F3F6F9","textMuted":"#5C6170","primary":"#7FB4CA","secondary":"#A3B5D6","accent":"#E0C15A","error":"#CB7C94","warning":"#DEBA87","success":"#B7CC85","info":"#7FB4CA","border":"#313342","borderActive":"#7FB4CA","borderSubtle":"#232A40","diffAdded":"#B7CC85","diffRemoved":"#CB7C94","diffContext":"#5C6170","diffHunkHeader":"#8394A3","diffHighlightAdded":"#D1E8A9","diffHighlightRemoved":"#DE8FA8","diffAddedBg":"#1a2e1a","diffRemovedBg":"#2e1a1a","diffContextBg":"#0d0f14","diffLineNumber":"#8394A3","diffAddedLineNumberBg":"#1a2e1a","diffRemovedLineNumberBg":"#2e1a1a","markdownText":"#F3F6F9","markdownHeading":"#B5B2D0","markdownLink":"#7FB4CA","markdownLinkText":"#79B8EA","markdownCode":"#B7CC85","markdownBlockQuote":"#DEBA87","markdownEmph":"#7CB9DD","markdownStrong":"#DEBA87","markdownHorizontalRule":"#5C6170","markdownListItem":"#7FB4CA","markdownListEnumeration":"#A3B5D6","markdownImage":"#7FB4CA","markdownImageText":"#79B8EA","markdownCodeBlock":"#F3F6F9","syntaxComment":"#8394A3","syntaxKeyword":"#C99AD6","syntaxFunction":"#B99BF2","syntaxVariable":"#F3F6F9","syntaxString":"#DFBD76","syntaxNumber":"#A4DAA7","syntaxType":"#8FB8DD","syntaxOperator":"#DEBA87","syntaxPunctuation":"#96A2B0"}}`
	legacyCuteOpenCodeFixture      = `{"$schema":"https://opencode.ai/theme.json","theme":{"background":"none","backgroundPanel":"#1A1218","backgroundElement":"#241822","text":"#F6EFF3","textMuted":"#A78E9B","primary":"#F095C8","secondary":"#D7A0B8","accent":"#F095C8","error":"#FF718F","warning":"#F2B86D","success":"#B4E7C7","info":"#D7A0B8","border":"#342230","borderActive":"#FFB1DD","borderSubtle":"#241822","diffAdded":"#B4E7C7","diffRemoved":"#FF718F","diffContext":"#A78E9B","diffHunkHeader":"#D7A0B8","diffHighlightAdded":"#B4E7C7","diffHighlightRemoved":"#FF718F","diffAddedBg":"#1A2420","diffRemovedBg":"#261019","diffContextBg":"#1A1218","diffLineNumber":"#76616B","diffAddedLineNumberBg":"#1A2420","diffRemovedLineNumberBg":"#261019","markdownText":"#F6EFF3","markdownHeading":"#E0C27A","markdownLink":"#F095C8","markdownLinkText":"#F095C8","markdownCode":"#E0C27A","markdownBlockQuote":"#A78E9B","markdownEmph":"#D7A0B8","markdownStrong":"#E0C27A","markdownHorizontalRule":"#A78E9B","markdownListItem":"#F095C8","markdownListEnumeration":"#D7A0B8","markdownImage":"#F095C8","markdownImageText":"#F095C8","markdownCodeBlock":"#F6EFF3","syntaxComment":"#A78E9B","syntaxKeyword":"#F095C8","syntaxFunction":"#A9C7EE","syntaxVariable":"#F6EFF3","syntaxString":"#B4E7C7","syntaxNumber":"#F2B86D","syntaxType":"#E0C27A","syntaxOperator":"#C4DAF6","syntaxPunctuation":"#A78E9B"}}`
)

func TestInjectVisualThemesWritesExpectedAssets(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	for _, adapter := range []agents.Adapter{claudeAdapter(), opencodeAdapter()} {
		settings := adapter.SettingsPath(home)
		if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(settings, []byte(`{"theme":"user-selected"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		first, err := InjectVisualThemes(home, adapter)
		if err != nil || !first.Changed {
			t.Fatalf("first injection = %#v, %v", first, err)
		}
		if second, err := InjectVisualThemes(home, adapter); err != nil || second.Changed {
			t.Fatalf("second injection = %#v, %v", second, err)
		}
		if got, err := os.ReadFile(settings); err != nil || string(got) != `{"theme":"user-selected"}` {
			t.Fatalf("settings changed = %q, %v", got, err)
		}
	}
	for _, tt := range []struct {
		path, fixture string
		value         any
	}{
		{filepath.Join(home, ".claude", "themes", "ordo.json"), ordoClaudeFixture, &claudeTheme{}},
		{filepath.Join(xdg, "opencode", "themes", "ordo.json"), ordoOpenCodeFixture, &openCodeTheme{}},
	} {
		got, err := os.ReadFile(tt.path)
		if err != nil || !bytes.Equal(got, indentedJSON(t, tt.fixture, tt.value)) {
			t.Fatalf("%s bytes mismatch: %v\n%s", tt.path, err, got)
		}
	}
	for _, key := range []string{"background", "backgroundFullscreen", "backgroundUser", "backgroundHover", "backgroundSelection", "backgroundMemory", "backgroundBash", "shimmer"} {
		if _, exists := ordoClaudeTheme.Overrides[key]; exists {
			t.Fatalf("Claude Ordo theme contains invalid override %q", key)
		}
	}
}

// legacyThemeFixture writes the four upstream theme files exactly as earlier
// releases installed them and returns their paths.
func legacyThemeFixture(t *testing.T, home string) []string {
	t.Helper()
	var paths []string
	for _, tt := range []struct {
		adapter  agents.Adapter
		fixtures []string
		newValue func() any
	}{
		{claudeAdapter(), []string{legacyGentlemanClaudeFixture, legacyCuteClaudeFixture}, func() any { return &claudeTheme{} }},
		{opencodeAdapter(), []string{legacyGentlemanOpenCodeFixture, legacyCuteOpenCodeFixture}, func() any { return &openCodeTheme{} }},
	} {
		legacy := LegacyVisualThemePaths(home, tt.adapter)
		if len(legacy) != 2 {
			t.Fatalf("%s legacy paths = %#v, want 2", tt.adapter.Agent(), legacy)
		}
		for i, path := range legacy {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, indentedJSON(t, tt.fixtures[i], tt.newValue()), 0o644); err != nil {
				t.Fatal(err)
			}
			paths = append(paths, path)
		}
	}
	return paths
}

func writeSettings(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func settingsTheme(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	var root struct {
		Theme string `json:"theme"`
	}
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("Unmarshal(%s) error = %v", path, err)
	}
	return root.Theme
}

func TestInjectVisualThemesMigratesOwnedLegacyThemes(t *testing.T) {
	for _, tt := range []struct {
		name                 string
		claudeTheme, ocTheme string
	}{
		{name: "gentleman selected", claudeTheme: "gentleman", ocTheme: "gentleman"},
		{name: "cute selected", claudeTheme: "Gentleman Cute", ocTheme: "gentleman-cute"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			home, xdg := t.TempDir(), t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("XDG_CONFIG_HOME", xdg)
			t.Setenv("OPENCODE_CONFIG_DIR", "")
			legacy := legacyThemeFixture(t, home)
			claudeSettings := claudeAdapter().SettingsPath(home)
			ocSettings := opencodeAdapter().SettingsPath(home)
			writeSettings(t, claudeSettings, `{"theme":"`+tt.claudeTheme+`","keep":true}`)
			writeSettings(t, ocSettings, `{"theme":"`+tt.ocTheme+`","keep":true}`)

			for _, adapter := range []agents.Adapter{claudeAdapter(), opencodeAdapter()} {
				first, err := InjectVisualThemes(home, adapter)
				if err != nil || !first.Changed {
					t.Fatalf("%s migration = %#v, %v", adapter.Agent(), first, err)
				}
				if second, err := InjectVisualThemes(home, adapter); err != nil || second.Changed {
					t.Fatalf("%s second run = %#v, %v; want no-op", adapter.Agent(), second, err)
				}
				if _, err := os.Stat(VisualThemePaths(home, adapter)[0]); err != nil {
					t.Fatalf("%s Ordo theme missing: %v", adapter.Agent(), err)
				}
			}
			for _, path := range legacy {
				if _, err := os.Stat(path); !os.IsNotExist(err) {
					t.Fatalf("owned legacy theme %q should be removed: %v", path, err)
				}
			}
			for _, path := range []string{claudeSettings, ocSettings} {
				if got := settingsTheme(t, path); got != "ordo" {
					t.Fatalf("%s theme = %q, want ordo", path, got)
				}
				if data, _ := os.ReadFile(path); !bytes.Contains(data, []byte(`"keep"`)) {
					t.Fatalf("%s lost unrelated settings: %s", path, data)
				}
			}
		})
	}
}

func TestInjectVisualThemesKeepsUnprovenLegacyFilesButSwitchesSetting(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	legacy := legacyThemeFixture(t, home)
	edited := []byte(`{"name":"gentleman","base":"dark","overrides":{"text":"#ffffff"}}` + "\n")
	if err := os.WriteFile(legacy[0], edited, 0o644); err != nil {
		t.Fatal(err)
	}
	settings := claudeAdapter().SettingsPath(home)
	writeSettings(t, settings, `{"theme":"gentleman"}`)

	if _, err := InjectVisualThemes(home, claudeAdapter()); err != nil {
		t.Fatalf("InjectVisualThemes() error = %v", err)
	}
	if got, err := os.ReadFile(legacy[0]); err != nil || !bytes.Equal(got, edited) {
		t.Fatalf("user-edited legacy theme changed: %q, %v", got, err)
	}
	if _, err := os.Stat(legacy[1]); !os.IsNotExist(err) {
		t.Fatalf("unchanged legacy theme %q should be removed: %v", legacy[1], err)
	}
	if got := settingsTheme(t, settings); got != "ordo" {
		t.Fatalf("theme = %q, want ordo", got)
	}
}

func TestInjectVisualThemesLeavesUserThemeAndMissingSettingsAlone(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	// A user theme that merely resembles a legacy name is not managed.
	settings := claudeAdapter().SettingsPath(home)
	writeSettings(t, settings, `{"theme":"Gentleman"}`)
	if _, err := InjectVisualThemes(home, claudeAdapter()); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(settings); string(got) != `{"theme":"Gentleman"}` {
		t.Fatalf("user theme changed: %s", got)
	}

	if _, err := InjectVisualThemes(home, opencodeAdapter()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(opencodeAdapter().SettingsPath(home)); !os.IsNotExist(err) {
		t.Fatalf("missing OpenCode settings must not be created: %v", err)
	}
}

// Before the migration existed, the visual theme step never read settings, so
// a settings file it cannot parse must not abort it: only the legacy selection
// switch is skipped, and the broken file is left byte-for-byte untouched.
func TestInjectVisualThemesSkipsSettingsMigrationForMalformedSettings(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	legacy := legacyThemeFixture(t, home)
	broken := `{"theme": "gentleman", ` // truncated JSON
	for _, adapter := range []agents.Adapter{claudeAdapter(), opencodeAdapter()} {
		settings := VisualThemeSettingsPath(home, adapter)
		writeSettings(t, settings, broken)
		result, err := InjectVisualThemes(home, adapter)
		if err != nil {
			t.Fatalf("%s InjectVisualThemes() error = %v; want malformed settings tolerated", adapter.Agent(), err)
		}
		if !result.Changed {
			t.Fatalf("%s result = %#v; want Ordo theme written", adapter.Agent(), result)
		}
		if _, err := os.Stat(VisualThemePaths(home, adapter)[0]); err != nil {
			t.Fatalf("%s Ordo theme missing: %v", adapter.Agent(), err)
		}
		if got, err := os.ReadFile(settings); err != nil || string(got) != broken {
			t.Fatalf("%s malformed settings changed: %q, %v", adapter.Agent(), got, err)
		}
	}
	// The broken file may still select a legacy theme, so its files stay.
	for _, path := range legacy {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("legacy theme %q must be kept while the selection cannot migrate: %v", path, err)
		}
	}
}

// assertLegacyKeptAfterSkippedMigration runs the visual theme step against
// settings whose legacy selection cannot be migrated and checks the step
// succeeds, writes the Ordo theme, leaves settings untouched and keeps the
// owned legacy theme files the selection may still point at.
func assertLegacyKeptAfterSkippedMigration(t *testing.T, home string, adapter agents.Adapter, settings string, legacy []string) {
	t.Helper()
	before, _ := os.ReadFile(settings)
	result, err := InjectVisualThemes(home, adapter)
	if err != nil {
		t.Fatalf("%s InjectVisualThemes() error = %v; want skipped migration", adapter.Agent(), err)
	}
	if len(result.Files) != 1 || result.Files[0] != VisualThemePaths(home, adapter)[0] {
		t.Fatalf("%s files = %#v; want only the Ordo theme", adapter.Agent(), result.Files)
	}
	if _, err := os.Stat(VisualThemePaths(home, adapter)[0]); err != nil {
		t.Fatalf("%s Ordo theme missing: %v", adapter.Agent(), err)
	}
	if after, _ := os.ReadFile(settings); !bytes.Equal(after, before) {
		t.Fatalf("%s settings changed: %q -> %q", adapter.Agent(), before, after)
	}
	for _, path := range legacy {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s legacy theme %q must be kept: %v", adapter.Agent(), path, err)
		}
	}
}

func TestInjectVisualThemesKeepsLegacyFilesWhenSettingsLocked(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	legacyThemeFixture(t, home)
	for _, adapter := range []agents.Adapter{opencodeAdapter(), claudeAdapter()} {
		// A symlinked settings file is refused for OpenCode and fails the
		// base writer for Claude; either way the selection cannot migrate.
		settings := VisualThemeSettingsPath(home, adapter)
		target := filepath.Join(home, "dotfiles", string(adapter.Agent())+".json")
		writeSettings(t, target, `{"theme":"gentleman"}`)
		if err := os.MkdirAll(filepath.Dir(settings), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, settings); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		assertLegacyKeptAfterSkippedMigration(t, home, adapter, target, LegacyVisualThemePaths(home, adapter))
	}
}

// With no settings file, or a readable one that selects no legacy theme,
// nothing can point at the owned legacy files, so they are removed.
func TestInjectVisualThemesRemovesOwnedLegacyFilesWhenSelectionNotLegacy(t *testing.T) {
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	legacy := legacyThemeFixture(t, home)
	writeSettings(t, claudeAdapter().SettingsPath(home), `{"theme":"dark"}`)
	for _, adapter := range []agents.Adapter{claudeAdapter(), opencodeAdapter()} {
		if _, err := InjectVisualThemes(home, adapter); err != nil {
			t.Fatal(err)
		}
	}
	for _, path := range legacy {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("owned legacy theme %q should be removed: %v", path, err)
		}
	}
	if got := settingsTheme(t, claudeAdapter().SettingsPath(home)); got != "dark" {
		t.Fatalf("user theme = %q, want dark", got)
	}
}

func TestInjectVisualThemesKeepsLegacyFilesWhenSettingsUnreadable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permission bits are not supported on Windows")
	}
	home, xdg := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("OPENCODE_CONFIG_DIR", "")
	legacyThemeFixture(t, home)
	adapter := claudeAdapter()
	settings := VisualThemeSettingsPath(home, adapter)
	writeSettings(t, settings, `{"theme":"gentleman"}`)
	if err := os.Chmod(settings, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(settings, 0o644) })
	if _, err := os.ReadFile(settings); err == nil {
		t.Skip("running with privileges that ignore file mode 0000")
	}
	assertLegacyKeptAfterSkippedMigration(t, home, adapter, settings, LegacyVisualThemePaths(home, adapter))
}

// OpenCode theme assets live under the adapter config root, like its plugins
// and skills, while the theme selection lives in the effective settings file
// that also honors an absolute OPENCODE_CONFIG_DIR. Every caller resolves both
// through the same two helpers.
func TestOpenCodeVisualThemePathsWithConfigDirOverride(t *testing.T) {
	home, xdg, override := t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("OPENCODE_CONFIG_DIR", override)
	adapter := opencodeAdapter()

	themesDir := filepath.Join(adapter.GlobalConfigDir(home), "themes")
	if got := VisualThemePaths(home, adapter); len(got) != 1 || got[0] != filepath.Join(themesDir, "ordo.json") {
		t.Fatalf("VisualThemePaths = %#v, want ordo.json under %q", got, themesDir)
	}
	for _, path := range LegacyVisualThemePaths(home, adapter) {
		if filepath.Dir(path) != themesDir {
			t.Fatalf("legacy path %q not under %q", path, themesDir)
		}
	}
	settings := filepath.Join(override, "opencode.json")
	if got := VisualThemeSettingsPath(home, adapter); got != settings {
		t.Fatalf("VisualThemeSettingsPath = %q, want %q", got, settings)
	}

	writeSettings(t, settings, `{"theme":"gentleman-cute"}`)
	if _, err := InjectVisualThemes(home, adapter); err != nil {
		t.Fatal(err)
	}
	if got := settingsTheme(t, settings); got != "ordo" {
		t.Fatalf("override settings theme = %q, want ordo", got)
	}
	if _, err := os.Stat(filepath.Join(themesDir, "ordo.json")); err != nil {
		t.Fatalf("Ordo theme not under adapter config root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(override, "themes")); !os.IsNotExist(err) {
		t.Fatalf("theme assets must not be written under OPENCODE_CONFIG_DIR: %v", err)
	}
}

func TestIsManagedThemeName(t *testing.T) {
	for name, want := range map[string]bool{
		"ordo": true, "gentleman": true, "Gentleman Cute": true, "gentleman-cute": true,
		"": false, "dark": false, "Gentleman": false, "custom:ordo": false,
	} {
		if got := IsManagedThemeName(name); got != want {
			t.Errorf("IsManagedThemeName(%q) = %v, want %v", name, got, want)
		}
	}
}

func indentedJSON(t *testing.T, fixture string, value any) []byte {
	t.Helper()
	if err := json.Unmarshal([]byte(fixture), value); err != nil {
		t.Fatal(err)
	}
	result, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(result, '\n')
}
