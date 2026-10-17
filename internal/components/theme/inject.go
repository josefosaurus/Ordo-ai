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

// Theme files earlier releases installed under upstream names, per agent.
// Install and sync leave them in place; uninstall removes them.
var (
	legacyClaudeThemeFiles   = []string{"gentleman.json", "gentleman-cute.json"}
	legacyOpenCodeThemeFiles = []string{"gentleman.json", "gentleman-cute.json"}
	// Settings values earlier releases selected, per agent.
	legacyClaudeThemeNames   = []string{"gentleman", "Gentleman Cute"}
	legacyOpenCodeThemeNames = []string{"gentleman", "gentleman-cute"}
)

// IsManagedThemeName reports whether a settings theme value is one Ordo
// selects or selected in an earlier release, as opposed to a user choice.
func IsManagedThemeName(name string) bool {
	return name == OrdoThemeName || slices.Contains(legacyClaudeThemeNames, name) || slices.Contains(legacyOpenCodeThemeNames, name)
}

// SelectsTheme reports whether selecting the Ordo theme makes sense for an
// adapter: only agents Ordo ships a theme asset for (visualThemeDir) can load
// it. Other agents own their active theme, so selection is skipped there.
func SelectsTheme(adapter agents.Adapter) bool {
	return shipsOrdoTheme(adapter.Agent())
}

// Inject selects the Ordo theme in the adapter settings. It is a no-op for an
// adapter SelectsTheme rejects, so their own theme choice is left alone.
func Inject(homeDir string, adapter agents.Adapter) (InjectionResult, error) {
	if !SelectsTheme(adapter) {
		return InjectionResult{}, nil
	}
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
// in an agent's active settings. It also migrates a settings theme value that
// names a legacy managed theme to Ordo; any other user choice is left alone.
// Legacy theme files stay in place until uninstall, so a selection this step
// cannot migrate (or a project-scoped one it never reads) never dangles.
func InjectVisualThemes(homeDir string, adapter agents.Adapter) (InjectionResult, error) {
	paths := VisualThemePaths(homeDir, adapter)
	if len(paths) == 0 {
		return InjectionResult{}, nil
	}

	var value any
	var legacyNames []string
	switch adapter.Agent() {
	case model.AgentClaudeCode:
		value, legacyNames = ordoClaudeTheme, legacyClaudeThemeNames
	case model.AgentOpenCode:
		value, legacyNames = ordoOpenCodeTheme, legacyOpenCodeThemeNames
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
	migration := migrateLegacyThemeSetting(settingsPath, adapter.Agent(), legacyNames)
	if migration == themeSettingSwitched {
		result.Changed = true
		result.Files = append(result.Files, settingsPath)
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
// installed for an adapter. Install and sync leave them in place; uninstall
// removes them like the current asset.
func LegacyVisualThemePaths(homeDir string, adapter agents.Adapter) []string {
	root := visualThemeDir(homeDir, adapter)
	if root == "" {
		return nil
	}
	var legacy []string
	switch adapter.Agent() {
	case model.AgentClaudeCode:
		legacy = legacyClaudeThemeFiles
	case model.AgentOpenCode:
		legacy = legacyOpenCodeThemeFiles
	}
	paths := make([]string, 0, len(legacy))
	for _, file := range legacy {
		paths = append(paths, filepath.Join(root, file))
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

// visualThemeDir places theme assets under the adapter config root, like the
// other OpenCode assets (plugins, skills, AGENTS.md), which ignore
// OPENCODE_CONFIG_DIR. Only the theme selection follows the effective settings
// file, as every other settings writer does (VisualThemeSettingsPath).
func visualThemeDir(homeDir string, adapter agents.Adapter) string {
	if !shipsOrdoTheme(adapter.Agent()) {
		return ""
	}
	return filepath.Join(adapter.GlobalConfigDir(homeDir), "themes")
}

// shipsOrdoTheme is the single source for which agents get an Ordo theme
// asset (visualThemeDir) and therefore an Ordo theme selection (SelectsTheme).
func shipsOrdoTheme(agent model.AgentID) bool {
	return agent == model.AgentClaudeCode || agent == model.AgentOpenCode
}

// themeSettingMigration is the outcome of migrateLegacyThemeSetting.
type themeSettingMigration int

const (
	// themeSettingNotLegacy: settings are missing or readable and select no
	// legacy managed theme.
	themeSettingNotLegacy themeSettingMigration = iota
	// themeSettingSwitched: a legacy selection now selects Ordo.
	themeSettingSwitched
	// themeSettingSkipped: the settings could not be read, parsed or written
	// (including a refused OpenCode file) and were left untouched.
	themeSettingSkipped
)

// migrateLegacyThemeSetting selects the Ordo theme only when the settings file
// currently selects one of the legacy managed names. Settings it cannot read,
// parse or rewrite are left untouched and reported as skipped: the visual theme
// step never depended on settings before this migration, so a broken or locked
// file must not abort it.
func migrateLegacyThemeSetting(settingsPath string, agent model.AgentID, legacyNames []string) themeSettingMigration {
	if settingsPath == "" {
		return themeSettingNotLegacy
	}
	raw, err := osReadFile(settingsPath)
	if err != nil {
		return themeSettingSkipped
	}
	if raw == nil {
		return themeSettingNotLegacy
	}
	root, err := filemerge.UnmarshalJSONObject(raw)
	if err != nil {
		return themeSettingSkipped
	}
	current, _ := root["theme"].(string)
	if !slices.Contains(legacyNames, current) {
		return themeSettingNotLegacy
	}
	if agent == model.AgentOpenCode {
		if err := filemerge.RefuseLockedSettingsFile(settingsPath); err != nil {
			return themeSettingSkipped
		}
	}
	if _, err := mergeJSONFile(settingsPath, themeOverlayJSON); err != nil {
		return themeSettingSkipped
	}
	return themeSettingSwitched
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
