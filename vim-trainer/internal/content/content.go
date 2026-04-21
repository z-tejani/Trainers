package content

import (
	"fmt"
	"sort"
	"strings"

	"vimtrainer/internal/engine"
)

type Module struct {
	ID                  string
	Title               string
	Summary             string
	PrerequisiteModules []string
}

type Lesson struct {
	ID                 string
	ModuleID           string
	Title              string
	Goal               string
	Explanation        string
	Hints              []string
	Skills             []string
	Prerequisites      []string
	CommandsLearned    []string
	CanonicalSolutions []string
	FocusTokens        []string
	Rule               string
	CommonMistakes     map[string]string
	Initial            engine.Scenario
	Check              func(engine.State) (bool, string)
}

type Feedback struct {
	Title       string
	Body        string
	Rule        string
	Completed   bool
	NextHint    string
	Interpreted string
}

type Catalog struct {
	modules []Module
	lessons []Lesson
	byID    map[string]Lesson
}

func NewCatalog() *Catalog {
	modules := moduleSet()
	lessons := lessonSet()
	byID := make(map[string]Lesson, len(lessons))
	for _, lesson := range lessons {
		byID[lesson.ID] = lesson
	}
	return &Catalog{modules: modules, lessons: lessons, byID: byID}
}

func (c *Catalog) Modules() []Module {
	out := make([]Module, len(c.modules))
	copy(out, c.modules)
	return out
}

func (c *Catalog) Lessons() []Lesson {
	out := make([]Lesson, len(c.lessons))
	copy(out, c.lessons)
	return out
}

func (c *Catalog) Lesson(id string) (Lesson, bool) {
	lesson, ok := c.byID[id]
	return lesson, ok
}

func (c *Catalog) LessonsForModule(moduleID string) []Lesson {
	var lessons []Lesson
	for _, lesson := range c.lessons {
		if lesson.ModuleID == moduleID {
			lessons = append(lessons, lesson)
		}
	}
	return lessons
}

func (c *Catalog) Module(id string) (Module, bool) {
	for _, module := range c.modules {
		if module.ID == id {
			return module, true
		}
	}
	return Module{}, false
}

func (c *Catalog) LessonIndex(id string) int {
	for i, lesson := range c.lessons {
		if lesson.ID == id {
			return i
		}
	}
	return -1
}

func (c *Catalog) NextLessonID(currentID string) string {
	idx := c.LessonIndex(currentID)
	if idx < 0 || idx+1 >= len(c.lessons) {
		return ""
	}
	return c.lessons[idx+1].ID
}

func (c *Catalog) Evaluate(lesson Lesson, state engine.State) Feedback {
	completed, successText := safeCheck(lesson, state)
	if completed {
		hint := ""
		if len(lesson.Hints) > 0 {
			hint = lesson.Hints[len(lesson.Hints)-1]
		}
		return Feedback{
			Title:       "Lesson complete",
			Body:        successText,
			Rule:        lesson.Rule,
			Completed:   true,
			NextHint:    hint,
			Interpreted: state.LastResult.Token,
		}
	}

	body := state.LastResult.Description
	if body == "" {
		body = "Use Vim keys to move toward the goal."
	}
	if state.LastResult.Error != "" {
		body = fmt.Sprintf("Vim interpreted that as %s. %s", tokenOrPlaceholder(state.LastResult.Token), state.LastResult.Description)
	}
	if override, ok := lesson.CommonMistakes[state.LastResult.Token]; ok {
		body = override
	} else if state.LastResult.Completed && state.LastResult.Token != "" && !tokenMatches(lesson.FocusTokens, state.LastResult.Token) {
		body = fmt.Sprintf("Vim interpreted that as %s. This lesson is focusing on %s.", tokenOrPlaceholder(state.LastResult.Token), strings.Join(lesson.FocusTokens, ", "))
	}
	hint := ""
	if len(lesson.Hints) > 0 {
		hint = lesson.Hints[0]
	}
	return Feedback{
		Title:       "Keep going",
		Body:        body,
		Rule:        lesson.Rule,
		NextHint:    hint,
		Interpreted: state.LastResult.Token,
	}
}

func safeCheck(lesson Lesson, state engine.State) (completed bool, successText string) {
	if lesson.Check == nil {
		return false, ""
	}
	defer func() {
		if recover() != nil {
			completed = false
			successText = ""
		}
	}()
	return lesson.Check(state)
}

func (c *Catalog) Skills() []string {
	seen := map[string]struct{}{}
	for _, lesson := range c.lessons {
		for _, skill := range lesson.Skills {
			seen[skill] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for skill := range seen {
		out = append(out, skill)
	}
	sort.Strings(out)
	return out
}

func tokenMatches(tokens []string, got string) bool {
	if got == "" {
		return false
	}
	for _, token := range tokens {
		if token == got {
			return true
		}
	}
	return false
}

func tokenOrPlaceholder(token string) string {
	if token == "" {
		return "nothing yet"
	}
	return token
}
