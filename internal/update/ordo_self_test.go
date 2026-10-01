package update

import (
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/brand"
	"github.com/gentleman-programming/gentle-ai/v4/internal/system"
)

func TestShippedSelfToolPointsAtOrdoReleases(t *testing.T) {
	tool := shippedSelfTool
	if tool.Name != brand.Command || tool.Owner != brand.ReleaseOwner || tool.Repo != brand.ReleaseRepo {
		t.Fatalf("self tool identity = %s %s/%s, want %s %s/%s", tool.Name, tool.Owner, tool.Repo, brand.Command, brand.ReleaseOwner, brand.ReleaseRepo)
	}
	if tool.ArchiveName != brand.Command {
		t.Fatalf("ArchiveName = %q, want %q (goreleaser publishes %s_* archives)", tool.ArchiveName, brand.Command, brand.Command)
	}
	if tool.InstallMethod != InstallBinary || tool.GoImportPath != "" || tool.DetectCmd != nil {
		t.Fatalf("self tool = %+v; want signed binary only, no go-install path (it would fetch upstream), version from ldflags", tool)
	}
	if isGentleAIRepo(tool) {
		t.Fatal("the Ordo entry must not enable upstream's beta main-head channel")
	}
}

func TestOrdoHintNeverPointsAtUpstream(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		hint := updateHint(shippedSelfTool, system.PlatformProfile{OS: goos})
		if hint == "" || strings.Contains(hint, "Gentleman-Programming") || strings.Contains(hint, "go install") {
			t.Fatalf("%s hint = %q", goos, hint)
		}
	}
	if hint := updateHint(shippedSelfTool, system.PlatformProfile{OS: "linux"}); !strings.Contains(hint, brand.ReleaseOwner+"/"+brand.ReleaseRepo) {
		t.Fatalf("linux hint must name the Ordo repository: %q", hint)
	}
}
