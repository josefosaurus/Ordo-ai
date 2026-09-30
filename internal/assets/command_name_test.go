package assets

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/brand"
)

// staleCommandInvocation matches the retired `gentle-ai <verb>` invocation
// and a plugin spawning the retired executable by name. Paths (.gentle-ai/),
// managed markers (gentle-ai:), schema ids (gentle-ai.), and agent/skill ids
// (gentle-ai-explore) are protocol names and deliberately do not match.
var staleCommandInvocation = regexp.MustCompile(`(?:^|[^./\w-])gentle-ai (?:[a-z<\[]|--)|["']gentle-ai["']`)

// TestShippedAssetsInvokeTheBrandCommand keeps every prompt, skill, and plugin
// an agent receives on the installed executable name: an agent told to run
// the retired name hits "command not found".
func TestShippedAssetsInvokeTheBrandCommand(t *testing.T) {
	if brand.Command == "gentle-ai" {
		t.Skip("the brand command is the retired name")
	}
	err := fs.WalkDir(FS, ".", func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil || entry.IsDir() {
			return walkErr
		}
		// The bench skill documents this repository's own gentle-ai-bench harness.
		if strings.HasPrefix(path, "skills/gentle-ai-bench/") {
			return nil
		}
		content, err := fs.ReadFile(FS, path)
		if err != nil {
			return err
		}
		for index, line := range strings.Split(string(content), "\n") {
			if staleCommandInvocation.MatchString(line) {
				t.Errorf("%s:%d invokes the retired command name instead of %q: %s", path, index+1, brand.Command, strings.TrimSpace(line))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
