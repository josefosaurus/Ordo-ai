package upgrade

import (
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/system"
	"github.com/gentleman-programming/gentle-ai/v4/internal/update"
)

func TestOrdoSelfToolAlwaysUsesSignedBinary(t *testing.T) {
	orig := homebrewPackageInstalled
	homebrewPackageInstalled = func(string) bool { return false }
	t.Cleanup(func() { homebrewPackageInstalled = orig })

	for _, goos := range []string{"linux", "darwin", "windows"} {
		profile := system.PlatformProfile{OS: goos, GoAvailable: true}
		if got := effectiveMethod(shippedSelfTool, profile); got != update.InstallBinary {
			t.Fatalf("%s: effectiveMethod = %q, want %q", goos, got, update.InstallBinary)
		}
	}
}

func TestOrdoReleaseArchiveUsesOrdoPrefixUnderOrdoRepo(t *testing.T) {
	r := update.UpdateResult{Tool: shippedSelfTool, LatestVersion: "0.1.0"}
	name, url := releaseArchive(r, "darwin", "arm64")
	if name != "ordo_0.1.0_darwin_arm64.tar.gz" {
		t.Fatalf("archive = %q", name)
	}
	if want := "https://github.com/josefosaurus/Ordo-ai/releases/download/v0.1.0/ordo_0.1.0_darwin_arm64.tar.gz"; url != want {
		t.Fatalf("url = %q, want %q", url, want)
	}

	engram := update.UpdateResult{Tool: update.ToolInfo{Owner: "Gentleman-Programming", Repo: "engram"}, LatestVersion: "1.2.3"}
	if name, url := releaseArchive(engram, "linux", "amd64"); name != "engram_1.2.3_linux_amd64.tar.gz" ||
		url != "https://github.com/Gentleman-Programming/engram/releases/download/v1.2.3/engram_1.2.3_linux_amd64.tar.gz" {
		t.Fatalf("tools without ArchiveName must keep the repo prefix: %q %q", name, url)
	}
}
