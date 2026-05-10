package content

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

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
	// OptimalKeys is the par keystroke count for the lesson. When > 0 the
	// summary screen reports efficiency relative to this target. When 0 the
	// lesson is treated as "no efficiency target" and only completion is
	// reported.
	OptimalKeys int
	// TimeTarget is the par solve time. When > 0, the summary reports
	// time-vs-target. When 0, time is shown but not graded.
	TimeTarget time.Duration
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
	for i := range lessons {
		// Derive a default keystroke par from the shortest canonical
		// solution if the lesson did not pin one explicitly. This means
		// every lesson gets an efficiency target without per-lesson
		// hand-curation.
		if lessons[i].OptimalKeys == 0 {
			lessons[i].OptimalKeys = parKeysFromCanonical(lessons[i].CanonicalSolutions)
		}
		byID[lessons[i].ID] = lessons[i]
	}
	return &Catalog{modules: modules, lessons: lessons, byID: byID}
}

// parKeysFromCanonical estimates the keystroke par for a lesson by counting
// atoms in the shortest canonical solution. The heuristic treats
// space-separated tokens as atoms: named keys ("Esc", "Enter", "Tab",
// "ctrl+r", "<C-r>") count as 1; plain text counts its rune length; ex /
// search prompts add one extra Enter.
func parKeysFromCanonical(sols []string) int {
	best := 0
	for _, sol := range sols {
		n := countSolutionAtoms(sol)
		if n > 0 && (best == 0 || n < best) {
			best = n
		}
	}
	return best
}

func countSolutionAtoms(sol string) int {
	fields := strings.Fields(sol)
	if len(fields) == 0 {
		return 0
	}
	hasEnterAtom := false
	for _, f := range fields {
		lc := strings.ToLower(f)
		if lc == "enter" || lc == "<cr>" || lc == "cr" {
			hasEnterAtom = true
			break
		}
	}
	n := 0
	needsEnter := false
	for _, f := range fields {
		lc := strings.ToLower(f)
		switch {
		case lc == "esc" || lc == "enter" || lc == "tab" || lc == "cr" || lc == "<cr>":
			n++
		case strings.HasPrefix(lc, "ctrl+"):
			n++
		case strings.HasPrefix(f, "<") && strings.HasSuffix(f, ">") && len(f) > 2:
			n++
		default:
			n += len([]rune(f))
			if strings.HasPrefix(f, ":") || strings.HasPrefix(f, "/") || strings.HasPrefix(f, "?") {
				needsEnter = true
			}
		}
	}
	if needsEnter && !hasEnterAtom {
		n++
	}
	return n
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

// ContentHash returns a stable short hash of the learner-facing parts of a
// lesson: title, goal, explanation, hints, canonical solutions, focus
// tokens, and the initial scenario. Used by the progress layer to flag a
// completed lesson as "stale" if its content changed since the user
// completed it (Q1=a, informational only — completion status is not
// reset).
//
// Engine internals like the Check closure are intentionally NOT part of
// the hash; they aren't surfaced to the learner and almost always co-vary
// with content changes anyway.
func (l Lesson) ContentHash() string {
	h := sha1.New()
	fmt.Fprintln(h, l.ID)
	fmt.Fprintln(h, l.Title)
	fmt.Fprintln(h, l.Goal)
	fmt.Fprintln(h, l.Explanation)
	fmt.Fprintln(h, l.Rule)
	for _, hint := range l.Hints {
		fmt.Fprintln(h, "hint:", hint)
	}
	for _, sol := range l.CanonicalSolutions {
		fmt.Fprintln(h, "sol:", sol)
	}
	for _, tok := range l.FocusTokens {
		fmt.Fprintln(h, "focus:", tok)
	}
	for _, cmd := range l.CommandsLearned {
		fmt.Fprintln(h, "cmd:", cmd)
	}
	for _, skill := range l.Skills {
		fmt.Fprintln(h, "skill:", skill)
	}
	for i, buf := range l.Initial.Buffers {
		fmt.Fprintf(h, "buf[%d]:%s\n", i, buf.Name)
		for j, line := range buf.Lines {
			fmt.Fprintf(h, "  line[%d]:%s\n", j, line)
		}
	}
	for i, win := range l.Initial.Windows {
		fmt.Fprintf(h, "win[%d]:buffer=%d row=%d col=%d\n", i, win.Buffer, win.Cursor.Row, win.Cursor.Col)
	}
	fmt.Fprintf(h, "active:%d\n", l.Initial.ActiveWindow)
	fmt.Fprintf(h, "mode:%s\n", l.Initial.StartingMode)
	// 12 hex chars is plenty for change-detection while staying short
	// in JSON dumps.
	return hex.EncodeToString(h.Sum(nil))[:12]
}
