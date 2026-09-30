package reviewassets_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/agents"
	"github.com/gentleman-programming/gentle-ai/v4/internal/components/reviewassets"
	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
)

// Captured by actual sdd.Inject in the disposable af4ce122 worktree, TestParityCapture.
// Review agents ship only to receipt-driven development runtimes: Cursor
// installs none, Kiro keeps Judgment Day, and Kimi keeps its main agent.
// Ordo: command renamed to ordo (the Judgment Day judges name `ordo review capture-result`).
var installedHashes = map[model.AgentID]map[string]string{
	model.AgentClaudeCode: {
		"jd-fix-agent.md": "a62bf9736226b81512cdedecbf5b6888714e6ae9dd6881b220504b859d35a218", "jd-judge-a.md": "3a780404d7d3b86fa07e5f7a3773f15b27d82db14e75f3906241672bf604c612", "jd-judge-b.md": "86f0eaad242f8d9c9805d4e3258cd3069b100d2af94eb3193e795356a1ae73a2", "review-readability.md": "3a15838d28ff2f02fca684e7036116917311f8bbe72c9d09b929015363d36737", "review-refuter.md": "fa58bacaa0af136963db25d25abe7fccad91a87f3454024f3d283339308976db", "review-reliability.md": "cd667908097d9d02d9c9ee0211b3040c4b4d507fbf2b21a78dd4bdf3d09e97ef", "review-resilience.md": "a4a186feb1b5e22b9edf09db9967416ddf3b41b613a268b7a9ee86f6cdc35986", "review-risk.md": "5c10ef801d1bddad5ee4f3e310b750b3dd5b98087c0f4ef1893f94c1d40c16b1",
	},
	model.AgentKimi: {
		"gentleman.yaml": "4fd319f06d3381954556e7828c96bfc0901c428c1f00bacc407d1f63342349b1",
	},
	model.AgentKiroIDE: {
		"jd-fix-agent.md": "f60d26fa25e810c7c6d4526005b886aaca65d82337a51430206b22b605a3a265", "jd-judge-a.md": "3687dae692278ac51fa5188773c82b6e149332e8ba8818bbb38e39af7ffef02a", "jd-judge-b.md": "dec7f6d8b38d445f4f25cc71b394154b899a17cfd9b172328e1627deb1bed262",
	},
}

func containsFile(files []string, path string) bool {
	for _, file := range files {
		if file == path {
			return true
		}
	}
	return false
}

func TestAllUnknownNativeAgentsLeaveLedgerAbsent(t *testing.T) {
	adapter, err := agents.NewAdapter(model.AgentKiroIDE)
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	initial, err := reviewassets.InstallNativeAgents(home, adapter, reviewassets.InstallOptions{})
	if err != nil || !initial.Changed {
		t.Fatalf("initial install: %+v, %v", initial, err)
	}
	ledger := filepath.Join(adapter.SubAgentsDir(home), reviewassets.OwnershipLedgerFilename)
	if err := os.Remove(ledger); err != nil {
		t.Fatal(err)
	}
	for _, guidance := range []string{"", "new guidance"} {
		result, err := reviewassets.InstallNativeAgents(home, adapter, reviewassets.InstallOptions{CodeGraphGuidanceMarkdown: guidance})
		if err != nil {
			t.Fatal(err)
		}
		if result.Changed || len(result.Files) != 0 || len(result.Skipped) != len(reviewassets.NativeAgentManifest[model.AgentKiroIDE]) {
			t.Fatalf("all-skipped install result = %+v", result)
		}
		if _, err := os.Lstat(ledger); !os.IsNotExist(err) {
			t.Fatalf("all-skipped ledger exists: %v", err)
		}
	}
}

func TestInstalledNativeAgentParity(t *testing.T) {
	count := 0
	for agent, expected := range installedHashes {
		t.Run(string(agent), func(t *testing.T) {
			home := t.TempDir()
			adapter, err := agents.NewAdapter(agent)
			if err != nil {
				t.Fatal(err)
			}
			opts := reviewassets.InstallOptions{CodeGraphGuidanceMarkdown: "Use CodeGraph before broad filesystem search."}
			result, err := reviewassets.InstallNativeAgents(home, adapter, opts)
			if err != nil {
				t.Fatal(err)
			}
			if !result.Changed || len(result.Files) != len(expected)+1 || !containsFile(result.Files, filepath.Join(adapter.SubAgentsDir(home), reviewassets.OwnershipLedgerFilename)) {
				t.Fatalf("unexpected install result: %+v", result)
			}
			entries, err := os.ReadDir(adapter.SubAgentsDir(home))
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != len(expected)+1 {
				t.Fatalf("installed %d entries, want %d agents plus ledger", len(entries), len(expected)+1)
			}
			for name, want := range expected {
				if strings.HasPrefix(name, "sdd-") {
					t.Fatal("SDD asset in retained fixture")
				}
				path := filepath.Join(adapter.SubAgentsDir(home), name)
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != want {
					t.Errorf("%s hash: got %s want %s", name, got, want)
				}
			}
			second, err := reviewassets.InstallNativeAgents(home, adapter, opts)
			if err != nil {
				t.Fatal(err)
			}
			if second.Changed || len(second.Files) != 0 {
				t.Fatalf("second install changed files: %+v", second)
			}
		})
		count += len(expected)
	}
	if count != 12 {
		t.Fatalf("fixture has %d paths, want 12", count)
	}
}
