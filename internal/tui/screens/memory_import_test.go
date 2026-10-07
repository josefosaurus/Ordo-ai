package screens

import (
	"strings"
	"testing"
)

func TestRenderMemoryImportPathHintListsFormats(t *testing.T) {
	out := RenderMemoryImport(MemoryImportView{Step: MemoryImportPath})
	for _, want := range []string{".md", ".csv", ".jsonl", ".json;"} {
		if !strings.Contains(out, want) {
			t.Errorf("path step hint missing %q:\n%s", want, out)
		}
	}
}
