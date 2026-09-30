package opencodeplugin

import (
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/brand"
)

func TestGentleLogoPluginSourceFollowsBrand(t *testing.T) {
	t.Cleanup(func() { brand.Set(brand.Default()) })

	src := gentleLogoPluginSource()
	if !strings.Contains(src, brand.Default().Logo[0]) || !strings.Contains(src, `"✦ Ordo ✦"`) {
		t.Fatal("default plugin source must embed the Ordo logo and name")
	}
	if strings.Contains(src, "__BRAND_") || strings.Contains(src, "Gentle AI") {
		t.Fatalf("plugin source has unreplaced placeholders or upstream branding:\n%s", src)
	}

	b := brand.Default()
	b.Name = `Ac"me`
	b.Logo = []string{`LOGO "quoted" \ line`}
	brand.Set(b)
	src = gentleLogoPluginSource()
	for _, want := range []string{`["LOGO \"quoted\" \\ line"]`, `"✦ Ac\"me ✦"`, "term.width >= 24"} {
		if !strings.Contains(src, want) {
			t.Fatalf("plugin source missing %s:\n%s", want, src)
		}
	}
}
