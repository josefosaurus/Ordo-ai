package screens

import (
	"strings"

	"github.com/gentleman-programming/gentle-ai/v4/internal/model"
	"github.com/gentleman-programming/gentle-ai/v4/internal/tui/styles"
)

func PersonaOptions() []model.PersonaID {
	return []model.PersonaID{model.PersonaOrdo, model.PersonaGentleman, model.PersonaNeutral, model.PersonaCustom}
}

var personaDescriptions = map[model.PersonaID]string{
	model.PersonaOrdo:      "Team voice and rules you can edit with the persona command; English technical artifacts",
	model.PersonaGentleman: "Voseo conversation; English technical artifacts",
	// The legacy alias is remapped at normalization time and no longer offered
	// in the picker; the entry stays so the review screen can label persisted
	// state that has not been migrated yet.
	model.PersonaGentlemanNeutralArtifacts: "No regional conversation tone; English technical artifacts (legacy alias, remapped)",
	model.PersonaNeutral:                   "No regional conversation tone; English technical artifacts",
	model.PersonaCustom:                    "Do not install a managed persona; choose themes/logo on the next screens",
}

// personaLabels holds display names that differ from the persona ID. The ID
// itself is a protocol value (state.json, --persona) and stays unchanged.
var personaLabels = map[model.PersonaID]string{
	model.PersonaGentleman: "Mentor",
}

// PersonaLabel returns the name the persona picker shows for persona.
func PersonaLabel(persona model.PersonaID) string {
	if label, ok := personaLabels[persona]; ok {
		return label
	}
	return string(persona)
}

func RenderPersona(selected model.PersonaID, cursor int) string {
	var b strings.Builder

	b.WriteString(styles.TitleStyle.Render("Choose your Persona"))
	b.WriteString("\n\n")
	b.WriteString(styles.SubtextStyle.Render("How your agents talk to you. Code, docs, and commits stay in English."))
	b.WriteString("\n\n")

	for idx, persona := range PersonaOptions() {
		isSelected := persona == selected
		focused := idx == cursor
		b.WriteString(renderRadio(PersonaLabel(persona), isSelected, focused))
		b.WriteString(styles.SubtextStyle.Render("    " + personaDescriptions[persona]))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(renderOptions([]string{"Back"}, cursor-len(PersonaOptions())))
	b.WriteString("\n")
	b.WriteString(styles.HelpStyle.Render("j/k: navigate • enter: select • esc: back"))

	return b.String()
}
