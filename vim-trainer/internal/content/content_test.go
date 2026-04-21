package content

import (
	"testing"

	"vimtrainer/internal/engine"
)

func TestCatalogHasModulesAndLessons(t *testing.T) {
	catalog := NewCatalog()
	if len(catalog.Modules()) < 5 {
		t.Fatal("expected multiple modules")
	}
	if len(catalog.Lessons()) < 10 {
		t.Fatal("expected multiple lessons")
	}
}

func TestFeedbackExplainsOffTargetMove(t *testing.T) {
	catalog := NewCatalog()
	lesson, ok := catalog.Lesson("motions-word")
	if !ok {
		t.Fatal("lesson not found")
	}
	state := engine.State{
		LastResult: engine.ActionResult{
			Token:       "l",
			Description: "moved right",
			Completed:   true,
		},
	}
	feedback := catalog.Evaluate(lesson, state)
	if feedback.Body == "" {
		t.Fatal("expected explanatory feedback")
	}
}
