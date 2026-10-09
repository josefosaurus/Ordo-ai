package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/gentleman-programming/gentle-ai/v4/internal/agents"
	"github.com/gentleman-programming/gentle-ai/v4/internal/components/filemerge"
	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
	opencodeconfig "github.com/gentleman-programming/gentle-ai/v4/internal/opencode"
)

type InjectionResult struct {
	Changed bool
	Files   []string
}

// OrdoThemeName is the theme ID Ordo selects and the base name of the theme
// files it installs.
const OrdoThemeName = "ordo"

var themeOverlayJSON = []byte("{\n  \"theme\": \"" + OrdoThemeName + "\"\n}\n")

type claudeTheme struct {
	Name      string            `json:"name"`
	Base      string            `json:"base"`
	Overrides map[string]string `json:"overrides"`
}

type openCodeTheme struct {
	Schema string            `json:"$schema"`
	Theme  map[string]string `json:"theme"`
}

const openCodeThemeSchema = "https://opencode.ai/theme.json"

func palette(pairs ...string) map[string]string {
	colors := make(map[string]string, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		colors[pairs[i]] = pairs[i+1]
	}
	return colors
}

// Rosé Pine (main, dark), matching internal/brand/default.yaml. The diff
// backgrounds are derived tints of pine and love over base, because the
// palette defines no diff colors of its own.
const (
	rpSurface = "#1f1d2e"
	rpOverlay = "#26233a"
	rpMuted   = "#6e6a86"
	rpSubtle  = "#908caa"
	rpText    = "#e0def4"
	rpLove    = "#eb6f92"
	rpGold    = "#f6c177"
	rpRose    = "#ebbcba"
	rpPine    = "#31748f"
	rpFoam    = "#9ccfd8"
	rpIris    = "#c4a7e7"
	rpHLLow   = "#21202e"
	rpHLMed   = "#403d52"

	rpAddedBg       = "#1f2e3f" // pine at 25% over base
	rpRemovedBg     = "#43293a" // love at 20% over base
	rpAddedWordBg   = "#274f64" // pine at 60% over base
	rpRemovedWordBg = "#773f55" // love at 45% over base
)

var ordoClaudeTheme = claudeTheme{
	Name: OrdoThemeName, Base: "dark",
	Overrides: palette(
		"claude", rpIris, "claudeShimmer", rpRose, "text", rpText, "inactive", rpSubtle, "subtle", rpMuted, "suggestion", rpRose,
		"permission", rpIris, "promptBorder", rpIris, "planMode", rpFoam, "autoAccept", rpRose, "bashBorder", rpGold,
		"remember", rpGold, "success", rpFoam, "merged", rpIris, "error", rpLove, "warning", rpGold,
		"diffAdded", rpAddedBg, "diffRemoved", rpRemovedBg, "diffAddedWord", rpAddedWordBg, "diffRemovedWord", rpRemovedWordBg,
		"userMessageBackground", rpSurface, "userMessageBackgroundHover", rpOverlay, "selectionBg", rpHLMed, "memoryBackgroundColor", rpHLLow, "bashMessageBackgroundColor", rpSurface,
	),
}

var ordoOpenCodeTheme = openCodeTheme{
	Schema: openCodeThemeSchema,
	Theme: palette(
		"background", "none", "backgroundPanel", rpSurface, "backgroundElement", rpOverlay, "text", rpText, "textMuted", rpSubtle,
		"primary", rpIris, "secondary", rpRose, "accent", rpGold, "error", rpLove, "warning", rpGold, "success", rpFoam, "info", rpFoam,
		"border", rpHLMed, "borderActive", rpIris, "borderSubtle", rpOverlay, "diffAdded", rpFoam, "diffRemoved", rpLove, "diffContext", rpMuted,
		"diffHunkHeader", rpSubtle, "diffHighlightAdded", rpFoam, "diffHighlightRemoved", rpLove, "diffAddedBg", rpAddedBg, "diffRemovedBg", rpRemovedBg, "diffContextBg", rpSurface,
		"diffLineNumber", rpMuted, "diffAddedLineNumberBg", rpAddedBg, "diffRemovedLineNumberBg", rpRemovedBg, "markdownText", rpText, "markdownHeading", rpIris,
		"markdownLink", rpFoam, "markdownLinkText", rpRose, "markdownCode", rpGold, "markdownBlockQuote", rpSubtle, "markdownEmph", rpRose, "markdownStrong", rpGold,
		"markdownHorizontalRule", rpMuted, "markdownListItem", rpIris, "markdownListEnumeration", rpFoam, "markdownImage", rpFoam, "markdownImageText", rpRose, "markdownCodeBlock", rpText,
		"syntaxComment", rpMuted, "syntaxKeyword", rpPine, "syntaxFunction", rpRose, "syntaxVariable", rpText, "syntaxString", rpGold, "syntaxNumber", rpGold, "syntaxType", rpFoam, "syntaxOperator", rpSubtle, "syntaxPunctuation", rpSubtle,
	),
}

// legacyTheme is a theme asset that earlier releases installed under an
// upstream name. Ordo proves it wrote a file only when its bytes still hash to
// the digest of the exact content it shipped.
type legacyTheme struct {
	file   string
	sha256 string
}

var (
	legacyClaudeThemes = []legacyTheme{
		{file: "gentleman.json", sha256: "d4053c31da27ea340a48dde843b9ad8914acea557b13ff375ecbd8fc09c1f5fc"},
		{file: "gentleman-cute.json", sha256: "88b8ed78227fa1541a3cd48b121116a08bee4a5e5bbe78d321b65846d57de0e9"},
	}
	legacyOpenCodeThemes = []legacyTheme{
		{file: "gentleman.json", sha256: "1f09532ac91b066ad71d5940bce65ce58a134266ef010971d5f597e5d4891f52"},
		{file: "gentleman-cute.json", sha256: "b681e9860137bb481870d0b82d138f0cb95a52241424ea2f07e187e69f93c2c7"},
	}
	// Settings values earlier releases selected, per agent.
	legacyClaudeThemeNames   = []string{"gentleman", "Gentleman Cute"}
	legacyOpenCodeThemeNames = []string{"gentleman", "gentleman-cute"}
)

// IsManagedThemeName reports whether a settings theme value is one Ordo
// selects or selected in an earlier release, as opposed to a user choice.
func IsManagedThemeName(name string) bool {
	return name == OrdoThemeName || slices.Contains(legacyClaudeThemeNames, name) || slices.Contains(legacyOpenCodeThemeNames, name)
}

func Inject(homeDir string, adapter agents.Adapter) (InjectionResult, error) {
	return injectSettings(adapter.SettingsPath(homeDir))
}

// InjectAtPath writes the theme to a caller-selected settings file. OpenCode
// callers pass the effective JSON/JSONC path and get a refusal for symlinked,
// non-regular or locked files; other adapters use Inject and keep the shared
// writer behavior.
func InjectAtPath(settingsPath string) (InjectionResult, error) {
	if settingsPath == "" {
		return InjectionResult{}, nil
	}
	if err := filemerge.RefuseLockedSettingsFile(settingsPath); err != nil {
		return InjectionResult{}, err
	}
	return injectSettings(settingsPath)
}

func injectSettings(settingsPath string) (InjectionResult, error) {
	if settingsPath == "" {
		return InjectionResult{}, nil
	}

	writeResult, err := mergeJSONFile(settingsPath, themeOverlayJSON)
	if err != nil {
		return InjectionResult{}, err
	}

	return InjectionResult{Changed: writeResult.Changed, Files: []string{settingsPath}}, nil
}

// InjectVisualThemes writes the managed Ordo theme asset without selecting it
// in an agent's active settings. It also migrates the upstream theme state that
// earlier releases installed: a settings theme value naming a legacy managed
// theme switches to Ordo, and a legacy theme file is removed only while its
// bytes still match what Ordo wrote. Any other user choice is left alone.
func InjectVisualThemes(homeDir string, adapter agents.Adapter) (InjectionResult, error) {
	paths := VisualThemePaths(homeDir, adapter)
	if len(paths) == 0 {
		return InjectionResult{}, nil
	}

	var value any
	var legacy []legacyTheme
	var legacyNames []string
	switch adapter.Agent() {
	case model.AgentClaudeCode:
		value, legacy, legacyNames = ordoClaudeTheme, legacyClaudeThemes, legacyClaudeThemeNames
	case model.AgentOpenCode:
		value, legacy, legacyNames = ordoOpenCodeTheme, legacyOpenCodeThemes, legacyOpenCodeThemeNames
	}

	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return InjectionResult{}, fmt.Errorf("marshal visual theme %q: %w", filepath.Base(paths[0]), err)
	}
	content = append(content, '\n')
	writeResult, err := filemerge.WriteFileAtomic(paths[0], content, 0o644)
	if err != nil {
		return InjectionResult{}, err
	}
	result := InjectionResult{Changed: writeResult.Changed, Files: []string{paths[0]}}

	settingsPath := VisualThemeSettingsPath(homeDir, adapter)
	switched, err := migrateLegacyThemeSetting(settingsPath, adapter.Agent(), legacyNames)
	if err != nil {
		return InjectionResult{}, err
	}
	if switched {
		result.Changed = true
		result.Files = append(result.Files, settingsPath)
	}

	root := filepath.Dir(paths[0])
	for _, old := range legacy {
		path := filepath.Join(root, old.file)
		removed, err := removeOwnedLegacyTheme(path, old.sha256)
		if err != nil {
			return InjectionResult{}, err
		}
		if removed {
			result.Changed = true
			result.Files = append(result.Files, path)
		}
	}
	return result, nil
}

// VisualThemePaths returns the installer-owned visual theme assets for an adapter.
func VisualThemePaths(homeDir string, adapter agents.Adapter) []string {
	root := visualThemeDir(homeDir, adapter)
	if root == "" {
		return nil
	}
	return []string{filepath.Join(root, OrdoThemeName+".json")}
}

// LegacyVisualThemePaths returns the upstream theme assets earlier releases
// installed for an adapter. Install removes them only when it can prove Ordo
// wrote them; uninstall removes them like the current asset.
func LegacyVisualThemePaths(homeDir string, adapter agents.Adapter) []string {
	root := visualThemeDir(homeDir, adapter)
	if root == "" {
		return nil
	}
	var legacy []legacyTheme
	switch adapter.Agent() {
	case model.AgentClaudeCode:
		legacy = legacyClaudeThemes
	case model.AgentOpenCode:
		legacy = legacyOpenCodeThemes
	}
	paths := make([]string, 0, len(legacy))
	for _, old := range legacy {
		paths = append(paths, filepath.Join(root, old.file))
	}
	return paths
}

// VisualThemeSettingsPath returns the settings file whose legacy theme value
// InjectVisualThemes migrates, so callers can back it up first.
func VisualThemeSettingsPath(homeDir string, adapter agents.Adapter) string {
	switch adapter.Agent() {
	case model.AgentClaudeCode:
		return adapter.SettingsPath(homeDir)
	case model.AgentOpenCode:
		return opencodeconfig.EffectiveSettingsPath(homeDir, "")
	default:
		return ""
	}
}

func visualThemeDir(homeDir string, adapter agents.Adapter) string {
	switch adapter.Agent() {
	case model.AgentClaudeCode:
		return filepath.Join(adapter.GlobalConfigDir(homeDir), "themes")
	case model.AgentOpenCode:
		return filepath.Join(filepath.Dir(adapter.SettingsPath(homeDir)), "themes")
	default:
		return ""
	}
}

// migrateLegacyThemeSetting selects the Ordo theme only when the settings file
// currently selects one of the legacy managed names.
func migrateLegacyThemeSetting(settingsPath string, agent model.AgentID, legacyNames []string) (bool, error) {
	if settingsPath == "" {
		return false, nil
	}
	raw, err := osReadFile(settingsPath)
	if err != nil || raw == nil {
		return false, err
	}
	root, err := filemerge.UnmarshalJSONObject(raw)
	if err != nil {
		return false, fmt.Errorf("parse settings %q: %w", settingsPath, err)
	}
	current, _ := root["theme"].(string)
	if !slices.Contains(legacyNames, current) {
		return false, nil
	}
	if agent == model.AgentOpenCode {
		if err := filemerge.RefuseLockedSettingsFile(settingsPath); err != nil {
			return false, err
		}
	}
	writeResult, err := mergeJSONFile(settingsPath, themeOverlayJSON)
	if err != nil {
		return false, err
	}
	return writeResult.Changed, nil
}

// removeOwnedLegacyTheme deletes a legacy theme file only when it is a regular
// file whose bytes match the digest of the content Ordo shipped.
func removeOwnedLegacyTheme(path, wantSHA256 string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stat legacy theme %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return false, nil
	}
	digest, _, err := filemerge.FileDigest(path)
	if err != nil {
		return false, fmt.Errorf("hash legacy theme %q: %w", path, err)
	}
	if digest != wantSHA256 {
		return false, nil
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("remove legacy theme %q: %w", path, err)
	}
	return true, nil
}

func mergeJSONFile(path string, overlay []byte) (filemerge.WriteResult, error) {
	mode := filemerge.ExistingFileMode(path, 0o644)

	baseJSON, err := osReadFile(path)
	if err != nil {
		return filemerge.WriteResult{}, err
	}

	merged, err := filemerge.MergeJSONObjectsForPath(path, baseJSON, overlay)
	if err != nil {
		return filemerge.WriteResult{}, err
	}

	return filemerge.WriteFileAtomic(path, merged, mode)
}

var osReadFile = func(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read json file %q: %w", path, err)
	}

	return content, nil
}
