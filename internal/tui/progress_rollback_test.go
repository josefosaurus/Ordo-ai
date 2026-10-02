package tui

import (
	"strings"
	"testing"

	"github.com/gentleman-programming/gentle-ai/v4/internal/pipeline"
	"github.com/gentleman-programming/gentle-ai/v4/internal/system"
)

// A failed apply rolls back earlier steps. The install screen must still
// reach 100% and offer "Press Enter to continue" instead of freezing.
func TestProgressFromFailedExecutionWithRollbackCompletes(t *testing.T) {
	result := pipeline.ExecutionResult{
		Apply: pipeline.StageResult{Steps: []pipeline.StepResult{
			{StepID: "component:skills", Status: pipeline.StepStatusSucceeded},
			{StepID: "component:engram", Status: pipeline.StepStatusFailed},
		}},
		Rollback: pipeline.StageResult{Steps: []pipeline.StepResult{
			{StepID: "component:skills", Status: pipeline.StepStatusRolledBack},
		}},
	}
	progress := ProgressFromExecution(result)
	if !progress.Done() || progress.Percent() != 100 {
		t.Fatalf("Percent = %d, Done = %v; a finished run with rollbacks must be done", progress.Percent(), progress.Done())
	}
	view := progress.ViewModel()
	if !view.Done {
		t.Fatal("view model must report done so the screen offers Enter")
	}
}

func TestInstallingRendersRolledBackSteps(t *testing.T) {
	result := pipeline.ExecutionResult{
		Apply:    pipeline.StageResult{Steps: []pipeline.StepResult{{StepID: "component:engram", Status: pipeline.StepStatusFailed}}},
		Rollback: pipeline.StageResult{Steps: []pipeline.StepResult{{StepID: "component:skills", Status: pipeline.StepStatusRolledBack}}},
	}
	m := NewModel(system.DetectionResult{}, "dev")
	m.Screen = ScreenInstalling
	m.Progress = ProgressFromExecution(result)
	out := m.View()
	for _, want := range []string{"component:skills (rolled back)", "Press Enter to continue"} {
		if !strings.Contains(out, want) {
			t.Fatalf("installing view missing %q:\n%s", want, out)
		}
	}
}
