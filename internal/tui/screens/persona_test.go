package screens

import (
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
)

func TestPersonaLabel(t *testing.T) {
	tests := []struct {
		name    string
		persona model.PersonaID
		want    string
	}{
		{name: "gentleman shows as Mentor", persona: model.PersonaGentleman, want: "Mentor"},
		{name: "ordo keeps its id", persona: model.PersonaOrdo, want: "ordo"},
		{name: "neutral keeps its id", persona: model.PersonaNeutral, want: "neutral"},
		{name: "custom keeps its id", persona: model.PersonaCustom, want: "custom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PersonaLabel(tt.persona); got != tt.want {
				t.Fatalf("PersonaLabel(%q) = %q, want %q", tt.persona, got, tt.want)
			}
		})
	}
}

func TestRenderPersonaShowsMentorInsteadOfGentlemanID(t *testing.T) {
	out := RenderPersona(model.PersonaGentleman, 1)
	if !strings.Contains(out, "Mentor") {
		t.Fatalf("RenderPersona missing the Mentor label; output:\n%s", out)
	}
	if strings.Contains(strings.ToLower(out), "gentleman") {
		t.Fatalf("RenderPersona still shows the gentleman id; output:\n%s", out)
	}
}
