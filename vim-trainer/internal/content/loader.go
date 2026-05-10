package content

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"vimtrainer/internal/engine"
)

// Lesson data lives next to this file under lessons/*.json. New lessons
// can be added without touching Go code as long as their Check expression
// fits the declarative CheckSpec grammar below — or, for one-off logic,
// the lesson registers a Go closure under its ID via RegisterCheck.
//
//go:embed lessons/*.json
var lessonsFS embed.FS

// checkRegistry holds custom Go check closures keyed by lesson ID. Lessons
// can opt into a closure by setting "check": {"type": "custom"} in JSON
// and calling RegisterCheck("lesson-id", fn) from a Go file.
var checkRegistry = map[string]func(engine.State) (bool, string){}

// RegisterCheck wires a custom Go closure into the data-driven loader for
// a given lesson ID. Use sparingly — most lessons should express their
// success criterion via CheckSpec instead.
func RegisterCheck(lessonID string, fn func(engine.State) (bool, string)) {
	checkRegistry[lessonID] = fn
}

// CheckSpec is the declarative success criterion a JSON lesson can carry.
// The grammar is intentionally rich enough that every shipped lesson can
// express its check without dropping to a Go closure — see the `Type`
// switch in buildCheck for the full list.
type CheckSpec struct {
	Type        string      `json:"type"`
	Buffer      []string    `json:"buffer,omitempty"`
	BufferIndex int         `json:"buffer_index,omitempty"`
	Row         *int        `json:"row,omitempty"`
	Col         *int        `json:"col,omitempty"`
	Mode        string      `json:"mode,omitempty"`
	Commands    []string    `json:"commands,omitempty"`
	Variable    string      `json:"variable,omitempty"`
	Value       string      `json:"value,omitempty"`
	Option      string      `json:"option,omitempty"`
	BoolValue   *bool       `json:"bool_value,omitempty"`
	Mapping     string      `json:"mapping,omitempty"`
	BufferName  string      `json:"buffer_name,omitempty"`
	Prefix      string      `json:"prefix,omitempty"`
	Count       *int        `json:"count,omitempty"`
	Children    []CheckSpec `json:"children,omitempty"`
	SuccessText string      `json:"success_text,omitempty"`
}

// lessonFile is the JSON shape of a lesson on disk. Fields map 1:1 to
// content.Lesson except for Initial (which uses a friendlier schema) and
// Check (which is a CheckSpec instead of a closure).
type lessonFile struct {
	ID                 string            `json:"id"`
	ModuleID           string            `json:"module_id"`
	Title              string            `json:"title"`
	Goal               string            `json:"goal"`
	Explanation        string            `json:"explanation"`
	Hints              []string          `json:"hints"`
	Skills             []string          `json:"skills"`
	Prerequisites      []string          `json:"prerequisites"`
	CommandsLearned    []string          `json:"commands_learned"`
	CanonicalSolutions []string          `json:"canonical_solutions"`
	FocusTokens        []string          `json:"focus_tokens"`
	Rule               string            `json:"rule"`
	CommonMistakes     map[string]string `json:"common_mistakes"`
	OptimalKeys        int               `json:"optimal_keys,omitempty"`
	TimeTargetMS       int               `json:"time_target_ms,omitempty"`
	Initial            scenarioFile      `json:"initial"`
	Check              CheckSpec         `json:"check"`
}

type scenarioFile struct {
	Buffers      []bufferFile `json:"buffers"`
	Windows      []windowFile `json:"windows,omitempty"`
	ActiveWindow int          `json:"active_window,omitempty"`
	StartingMode string       `json:"starting_mode,omitempty"`
}

type bufferFile struct {
	Name  string   `json:"name"`
	Lines []string `json:"lines"`
}

type windowFile struct {
	Buffer int `json:"buffer"`
	Row    int `json:"row"`
	Col    int `json:"col"`
}

// loadJSONLessons reads every embedded lesson JSON file and returns the
// resulting Lesson values. Files may contain a single lesson object or a
// JSON array of lesson objects (catalog files); both shapes are handled.
//
// To preserve lesson ordering across files, the loader sorts files
// lexically by filename and within an array file preserves insertion
// order. Catalog filenames typically use a numeric prefix (e.g.
// `01-onboarding.json`, `02-motions.json`, ...) to control the sequence.
//
// Errors during loading are aggregated; one bad file doesn't take the
// rest down with it.
func loadJSONLessons() ([]Lesson, error) {
	entries, err := fs.ReadDir(lessonsFS, "lessons")
	if err != nil {
		return nil, nil
	}
	// Sort files lexically so curriculum order is deterministic and
	// authorable via filename prefixes.
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	var lessons []Lesson
	var loadErrors []string
	for _, name := range names {
		data, err := fs.ReadFile(lessonsFS, "lessons/"+name)
		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("%s: read: %v", name, err))
			continue
		}
		trimmed := strings.TrimLeftFunc(string(data), func(r rune) bool {
			return r == ' ' || r == '\t' || r == '\n' || r == '\r'
		})
		if strings.HasPrefix(trimmed, "[") {
			var lfs []lessonFile
			if err := json.Unmarshal(data, &lfs); err != nil {
				loadErrors = append(loadErrors, fmt.Sprintf("%s: parse: %v", name, err))
				continue
			}
			for i, lf := range lfs {
				lesson, err := lf.toLesson()
				if err != nil {
					loadErrors = append(loadErrors, fmt.Sprintf("%s[%d]: build: %v", name, i, err))
					continue
				}
				lessons = append(lessons, lesson)
			}
			continue
		}
		var lf lessonFile
		if err := json.Unmarshal(data, &lf); err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("%s: parse: %v", name, err))
			continue
		}
		lesson, err := lf.toLesson()
		if err != nil {
			loadErrors = append(loadErrors, fmt.Sprintf("%s: build: %v", name, err))
			continue
		}
		lessons = append(lessons, lesson)
	}
	if len(loadErrors) > 0 {
		return lessons, fmt.Errorf("lesson load errors: %s", strings.Join(loadErrors, "; "))
	}
	return lessons, nil
}

func (lf lessonFile) toLesson() (Lesson, error) {
	if lf.ID == "" {
		return Lesson{}, fmt.Errorf("lesson has no id")
	}
	scenario, err := lf.Initial.toScenario()
	if err != nil {
		return Lesson{}, fmt.Errorf("scenario: %w", err)
	}
	check, err := buildCheck(lf.ID, lf.Check)
	if err != nil {
		return Lesson{}, fmt.Errorf("check: %w", err)
	}
	timeTarget := time.Duration(lf.TimeTargetMS) * time.Millisecond
	return Lesson{
		ID:                 lf.ID,
		ModuleID:           lf.ModuleID,
		Title:              lf.Title,
		Goal:               lf.Goal,
		Explanation:        lf.Explanation,
		Hints:              lf.Hints,
		Skills:             lf.Skills,
		Prerequisites:      lf.Prerequisites,
		CommandsLearned:    lf.CommandsLearned,
		CanonicalSolutions: lf.CanonicalSolutions,
		FocusTokens:        lf.FocusTokens,
		Rule:               lf.Rule,
		CommonMistakes:     lf.CommonMistakes,
		Initial:            scenario,
		Check:              check,
		OptimalKeys:        lf.OptimalKeys,
		TimeTarget:         timeTarget,
	}, nil
}

func (sf scenarioFile) toScenario() (engine.Scenario, error) {
	buffers := make([]engine.Buffer, 0, len(sf.Buffers))
	for _, bf := range sf.Buffers {
		lines := bf.Lines
		if len(lines) == 0 {
			lines = []string{""}
		}
		buffers = append(buffers, engine.Buffer{Name: bf.Name, Lines: lines})
	}
	windows := make([]engine.Window, 0, len(sf.Windows))
	for _, wf := range sf.Windows {
		windows = append(windows, engine.Window{Buffer: wf.Buffer, Cursor: engine.Position{Row: wf.Row, Col: wf.Col}})
	}
	scenario := engine.Scenario{
		Buffers:      buffers,
		Windows:      windows,
		ActiveWindow: sf.ActiveWindow,
	}
	switch strings.ToLower(sf.StartingMode) {
	case "", "normal":
		// default
	case "insert":
		scenario.StartingMode = engine.ModeInsert
	case "visual":
		scenario.StartingMode = engine.ModeVisual
	case "command":
		scenario.StartingMode = engine.ModeCommand
	case "search":
		scenario.StartingMode = engine.ModeSearch
	default:
		return engine.Scenario{}, fmt.Errorf("unknown starting_mode %q", sf.StartingMode)
	}
	return scenario, nil
}

func buildCheck(lessonID string, spec CheckSpec) (func(engine.State) (bool, string), error) {
	if spec.Type == "" {
		return nil, fmt.Errorf("check has no type")
	}
	switch spec.Type {
	case "custom":
		fn, ok := checkRegistry[lessonID]
		if !ok {
			return nil, fmt.Errorf("custom check requested but no closure registered for %q", lessonID)
		}
		return fn, nil
	case "buffer_equals":
		want := append([]string{}, spec.Buffer...)
		mode := spec.Mode
		bufferIdx := spec.BufferIndex
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			if bufferIdx < 0 || bufferIdx >= len(state.Buffers) {
				return false, ""
			}
			lines := state.Buffers[bufferIdx].Lines
			if len(lines) != len(want) {
				return false, ""
			}
			for i, line := range lines {
				if line != want[i] {
					return false, ""
				}
			}
			if mode != "" && string(state.Mode) != strings.ToUpper(mode) {
				return false, ""
			}
			return true, successText
		}, nil
	case "cursor_at":
		row, col := 0, 0
		if spec.Row != nil {
			row = *spec.Row
		}
		if spec.Col != nil {
			col = *spec.Col
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			if state.ActiveWindow >= len(state.Windows) {
				return false, ""
			}
			cur := state.Windows[state.ActiveWindow].Cursor
			return cur.Row == row && cur.Col == col, successText
		}, nil
	case "cursor_row_is":
		row := 0
		if spec.Row != nil {
			row = *spec.Row
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			if state.ActiveWindow >= len(state.Windows) {
				return false, ""
			}
			return state.Windows[state.ActiveWindow].Cursor.Row == row, successText
		}, nil
	case "mode_is":
		want := strings.ToUpper(spec.Mode)
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return string(state.Mode) == want, successText
		}, nil
	case "last_search_is":
		want := spec.Value
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.LastSearch == want, successText
		}, nil
	case "last_echo_is":
		want := spec.Value
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.LastEcho == want, successText
		}, nil
	case "mapping_absent":
		lhs := spec.Mapping
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			_, exists := state.NormalMappings[lhs]
			return !exists, successText
		}, nil
	case "mapping_exists":
		lhs := spec.Mapping
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			_, exists := state.NormalMappings[lhs]
			return exists, successText
		}, nil
	case "active_buffer_name_is":
		want := spec.BufferName
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			if state.ActiveWindow >= len(state.Windows) {
				return false, ""
			}
			bufIdx := state.Windows[state.ActiveWindow].Buffer
			if bufIdx < 0 || bufIdx >= len(state.Buffers) {
				return false, ""
			}
			return state.Buffers[bufIdx].Name == want, successText
		}, nil
	case "active_buffer_name_has_prefix":
		prefix := spec.Prefix
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			if state.ActiveWindow >= len(state.Windows) {
				return false, ""
			}
			bufIdx := state.Windows[state.ActiveWindow].Buffer
			if bufIdx < 0 || bufIdx >= len(state.Buffers) {
				return false, ""
			}
			return strings.HasPrefix(state.Buffers[bufIdx].Name, prefix), successText
		}, nil
	case "window_count_is":
		want := 1
		if spec.Count != nil {
			want = *spec.Count
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return len(state.Windows) == want, successText
		}, nil
	case "active_window_is":
		want := 0
		if spec.Count != nil {
			want = *spec.Count
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.ActiveWindow == want, successText
		}, nil
	case "explorer_open_is":
		want := false
		if spec.BoolValue != nil {
			want = *spec.BoolValue
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.ExplorerOpen == want, successText
		}, nil
	case "explorer_path_is":
		want := spec.Value
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.ExplorerPath == want, successText
		}, nil
	case "confirm_active_is":
		want := false
		if spec.BoolValue != nil {
			want = *spec.BoolValue
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.ConfirmActive == want, successText
		}, nil
	case "profile_active_is":
		want := false
		if spec.BoolValue != nil {
			want = *spec.BoolValue
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.ProfileActive == want, successText
		}, nil
	case "tab_count_is":
		want := 1
		if spec.Count != nil {
			want = *spec.Count
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.TabCount == want, successText
		}, nil
	case "active_tab_is":
		want := 0
		if spec.Count != nil {
			want = *spec.Count
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.ActiveTab == want, successText
		}, nil
	case "active_window_buffer_is":
		want := 0
		if spec.Count != nil {
			want = *spec.Count
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			if state.ActiveWindow >= len(state.Windows) {
				return false, ""
			}
			return state.Windows[state.ActiveWindow].Buffer == want, successText
		}, nil
	case "autocmd_count_is":
		want := 0
		if spec.Count != nil {
			want = *spec.Count
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.AutocmdCount == want, successText
		}, nil
	case "session_count_is":
		want := 0
		if spec.Count != nil {
			want = *spec.Count
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.SessionCount == want, successText
		}, nil
	case "view_count_is":
		want := 0
		if spec.Count != nil {
			want = *spec.Count
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.ViewCount == want, successText
		}, nil
	case "sign_count_is":
		want := 0
		if spec.Count != nil {
			want = *spec.Count
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.SignCount == want, successText
		}, nil
	case "statusline_is":
		want := spec.Value
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.Statusline == want, successText
		}, nil
	case "conceal_level_is":
		want := 0
		if spec.Count != nil {
			want = *spec.Count
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.ConcealLevel == want, successText
		}, nil
	case "fold_count_is":
		want := 0
		if spec.Count != nil {
			want = *spec.Count
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.FoldCount == want, successText
		}, nil
	case "fold_closed_count_is":
		want := 0
		if spec.Count != nil {
			want = *spec.Count
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.FoldClosedCount == want, successText
		}, nil
	case "mark_set":
		// Verifies a mark exists at all (any position).
		ch := spec.Mapping
		if ch == "" {
			return nil, fmt.Errorf("mark_set requires `mapping` to name the mark")
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			r := []rune(ch)
			if len(r) == 0 {
				return false, ""
			}
			_, ok := state.Marks[r[0]]
			return ok, successText
		}, nil
	case "textwidth_is":
		want := 0
		if spec.Count != nil {
			want = *spec.Count
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			return state.TextWidth == want, successText
		}, nil
	case "option_int_is":
		opt := strings.ToLower(spec.Option)
		want := 0
		if spec.Count != nil {
			want = *spec.Count
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			got := 0
			switch opt {
			case "tabstop":
				got = state.Options.TabStop
			case "shiftwidth":
				got = state.Options.ShiftWidth
			case "scrolloff":
				got = state.Options.ScrollOff
			case "updatetime":
				got = state.Options.UpdateTime
			case "timeoutlen":
				got = state.Options.TimeoutLen
			case "textwidth":
				got = state.TextWidth
			default:
				return false, ""
			}
			return got == want, successText
		}, nil
	case "option_string_is":
		opt := strings.ToLower(spec.Option)
		want := spec.Value
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			got := ""
			switch opt {
			case "clipboard":
				got = state.Options.Clipboard
			case "mouse":
				got = state.Options.Mouse
			case "spelllang":
				got = state.Options.SpellLang
			case "foldmethod":
				got = state.Options.FoldMethod
			case "listchars":
				got = state.Options.ListChars
			case "completeopt":
				got = state.Options.CompleteOpt
			case "wildmode":
				got = state.Options.WildMode
			case "colorscheme":
				got = state.Options.ColorScheme
			default:
				return false, ""
			}
			return got == want, successText
		}, nil
	case "bool_option_is":
		// Any of the new bool options (P2 expansion).
		opt := strings.ToLower(spec.Option)
		want := false
		if spec.BoolValue != nil {
			want = *spec.BoolValue
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			got := false
			switch opt {
			case "expandtab":
				got = state.Options.ExpandTab
			case "undofile":
				got = state.Options.UndoFile
			case "splitbelow":
				got = state.Options.SplitBelow
			case "splitright":
				got = state.Options.SplitRight
			case "termguicolors":
				got = state.Options.TermGUIColors
			case "spell":
				got = state.Options.Spell
			case "cursorline":
				got = state.Options.CursorLine
			case "list":
				got = state.Options.List
			case "wildmenu":
				got = state.Options.WildMenu
			case "lazyredraw":
				got = state.Options.LazyRedraw
			case "backup":
				got = state.Options.BackupEnabled
			case "swapfile":
				got = state.Options.SwapFile
			case "autoread":
				got = state.Options.AutoRead
			default:
				return false, ""
			}
			return got == want, successText
		}, nil
	case "command_includes":
		want := append([]string{}, spec.Commands...)
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			for _, token := range want {
				found := false
				for _, h := range state.CommandHistory {
					if h == token {
						found = true
						break
					}
				}
				if !found {
					return false, ""
				}
			}
			return true, successText
		}, nil
	case "option_is":
		opt := strings.ToLower(spec.Option)
		want := false
		if spec.BoolValue != nil {
			want = *spec.BoolValue
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			got := false
			switch opt {
			case "number":
				got = state.Options.Number
			case "relativenumber":
				got = state.Options.RelativeNumber
			case "hlsearch":
				got = state.Options.HLSearch
			case "ignorecase":
				got = state.Options.IgnoreCase
			case "smartcase":
				got = state.Options.SmartCase
			case "incsearch":
				got = state.Options.IncSearch
			case "wrap":
				got = state.Options.Wrap
			default:
				return false, ""
			}
			return got == want, successText
		}, nil
	case "variable_equals":
		name := spec.Variable
		want := spec.Value
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			got, ok := state.Variables[name]
			if !ok {
				return false, ""
			}
			return got == want, successText
		}, nil
	case "all_of":
		children := make([]func(engine.State) (bool, string), 0, len(spec.Children))
		for _, child := range spec.Children {
			fn, err := buildCheck(lessonID, child)
			if err != nil {
				return nil, err
			}
			children = append(children, fn)
		}
		successText := spec.SuccessText
		return func(state engine.State) (bool, string) {
			text := successText
			for _, fn := range children {
				ok, msg := fn(state)
				if !ok {
					return false, ""
				}
				if text == "" && msg != "" {
					text = msg
				}
			}
			return true, text
		}, nil
	default:
		return nil, fmt.Errorf("unsupported check type %q", spec.Type)
	}
}
