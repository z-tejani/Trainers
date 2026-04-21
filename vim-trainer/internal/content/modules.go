package content

import (
	"strings"

	"vimtrainer/internal/engine"
)

func moduleSet() []Module {
	base := []Module{
		{ID: "onboarding", Title: "Onboarding", Summary: "Learn modes, insertion, Esc, and the command-line."},
		{ID: "motions", Title: "Motions", Summary: "Move by character, word, count, and file anchors.", PrerequisiteModules: []string{"onboarding"}},
		{ID: "operators", Title: "Operators", Summary: "Use operators with motions instead of hand-editing.", PrerequisiteModules: []string{"motions"}},
		{ID: "search", Title: "Search And Navigation", Summary: "Search, repeat matches, and jump with marks.", PrerequisiteModules: []string{"operators"}},
		{ID: "editing", Title: "Editing Primitives", Summary: "Edit, open lines, and use undo/redo effectively.", PrerequisiteModules: []string{"search"}},
		{ID: "textobjects", Title: "Text Objects", Summary: "Operate on words semantically instead of character-by-character.", PrerequisiteModules: []string{"editing"}},
		{ID: "commandline", Title: "Command-Line And Options", Summary: "Use :set, :noh, :help, :e, :w, and :Ex.", PrerequisiteModules: []string{"textobjects"}},
		{ID: "windows", Title: "Windows And Buffers", Summary: "Split windows and move among buffers without leaving Vim.", PrerequisiteModules: []string{"commandline"}},
		{ID: "macros", Title: "Macros And Repeat", Summary: "Use . and basic macros to automate repetitive edits.", PrerequisiteModules: []string{"windows"}},
		{ID: "vimscript", Title: "Vimscript Essentials", Summary: "Use :let, :echo, and mappings to personalize Vim.", PrerequisiteModules: []string{"macros"}},
		{ID: "neovim", Title: "Neovim Mode", Summary: "Learn Neovim-specific workflows like :checkhealth, :terminal, and :lua.", PrerequisiteModules: []string{"commandline"}},
	}
	return append(base, extraModules()...)
}

func lessonSet() []Lesson {
	base := []Lesson{
		{
			ID:              "onboarding-insert",
			ModuleID:        "onboarding",
			Title:           "Modal Editing Starts Here",
			Goal:            "Press i, type vim, then press Esc so the line becomes `name = vim`.",
			Explanation:     "Vim starts in Normal mode. You only enter Insert mode when you intend to type text.",
			Hints:           []string{"Use i to enter Insert mode.", "Type vim.", "Press Esc to return to Normal mode."},
			Skills:          []string{"modes.normal", "modes.insert", "escape"},
			CommandsLearned: []string{"i", "Esc"},
			CanonicalSolutions: []string{
				"i vim Esc",
			},
			FocusTokens: []string{"i", "esc"},
			Rule:        "Normal mode is for commands; Insert mode is for typing text.",
			CommonMistakes: map[string]string{
				"v": "You typed text while still in Normal mode. Press i first so Vim knows you want to insert text.",
			},
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "scratch.txt", Lines: []string{"name = "}}},
				Windows: []engine.Window{{Buffer: 0, Cursor: engine.Position{Row: 0, Col: 7}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Mode == engine.ModeNormal && state.Buffers[0].Lines[0] == "name = vim", "You switched into Insert mode, typed the text, and escaped back to Normal mode."
			},
		},
		{
			ID:              "onboarding-commandline",
			ModuleID:        "onboarding",
			Title:           "The Command-Line Prompt",
			Goal:            "Turn on line numbers with :set number.",
			Explanation:     "The : prompt is where Vim options and file commands live.",
			Hints:           []string{"Press : to open the prompt.", "Type set number and hit Enter."},
			Skills:          []string{"commandline.open", "options.number"},
			CommandsLearned: []string{":set number"},
			CanonicalSolutions: []string{
				":set number",
			},
			FocusTokens: []string{":set number"},
			Rule:        "Use the : command-line for editor commands and options.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "main.go", Lines: []string{"package main", "", "func main() {}"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Options.Number, "Absolute line numbers are now enabled."
			},
		},
		{
			ID:              "motions-hjkl",
			ModuleID:        "motions",
			Title:           "Stay On Home Row",
			Goal:            "Move from the start of alpha to the start of gamma using j then j.",
			Explanation:     "hjkl let you move without leaving the home row.",
			Hints:           []string{"j moves down one line.", "Use j twice."},
			Skills:          []string{"motions.hjkl"},
			CommandsLearned: []string{"h", "j", "k", "l"},
			CanonicalSolutions: []string{
				"jj",
			},
			FocusTokens: []string{"j"},
			Rule:        "Use home-row motions for deliberate, precise cursor movement.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "list.txt", Lines: []string{"alpha", "beta", "gamma"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Windows[state.ActiveWindow].Cursor == (engine.Position{Row: 2, Col: 0}), "j moved the cursor down line-by-line."
			},
		},
		{
			ID:              "motions-word",
			ModuleID:        "motions",
			Title:           "Move By Word",
			Goal:            "Jump from alpha to gamma using w twice.",
			Explanation:     "Word motions are where Vim starts to feel fast.",
			Hints:           []string{"w jumps to the start of the next word.", "Use a count like 2w if you prefer."},
			Skills:          []string{"motions.w", "motions.counts"},
			CommandsLearned: []string{"w", "2w"},
			CanonicalSolutions: []string{
				"ww",
				"2w",
			},
			FocusTokens: []string{"w", "2w"},
			Rule:        "Prefer semantic movement like word jumps over repeated single-character steps.",
			CommonMistakes: map[string]string{
				"ll": "Single-character movement works, but this lesson is training word motions. Use w to jump by words.",
			},
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "words.txt", Lines: []string{"alpha beta gamma delta"}}},
			},
			Check: func(state engine.State) (bool, string) {
				pos := state.Windows[state.ActiveWindow].Cursor
				return pos == (engine.Position{Row: 0, Col: 11}), "Word motions let you cross larger distances with fewer keys."
			},
		},
		{
			ID:              "motions-anchors",
			ModuleID:        "motions",
			Title:           "Counts And Anchors",
			Goal:            "Use 3j to move to the fourth line, then use $ to land on the end of that line.",
			Explanation:     "Counts multiply motions, and anchors jump to structural edges.",
			Hints:           []string{"Start with 3j.", "Then press $."},
			Skills:          []string{"motions.counts", "motions.lineend"},
			CommandsLearned: []string{"3j", "$"},
			CanonicalSolutions: []string{
				"3j$",
			},
			FocusTokens: []string{"3j", "$"},
			Rule:        "Counts and anchors reduce repeated movement and make navigation predictable.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "stack.txt", Lines: []string{"one", "two", "three", "deploy target"}}},
			},
			Check: func(state engine.State) (bool, string) {
				pos := state.Windows[state.ActiveWindow].Cursor
				return pos == (engine.Position{Row: 3, Col: 12}), "You used a count and a line anchor to land exactly where you wanted."
			},
		},
		{
			ID:              "motions-file",
			ModuleID:        "motions",
			Title:           "Jump Across The File",
			Goal:            "Go to the top with gg, then jump to the bottom with G.",
			Explanation:     "File-level jumps matter when you already know the rough location.",
			Hints:           []string{"Press g twice.", "Then press Shift-g."},
			Skills:          []string{"motions.gg", "motions.G"},
			CommandsLearned: []string{"gg", "G"},
			CanonicalSolutions: []string{
				"ggG",
			},
			FocusTokens: []string{"gg", "G"},
			Rule:        "Use gg and G when the destination is at a file boundary.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "files.txt", Lines: []string{"README.md", "main.go", "go.mod", "go.sum"}}},
				Windows: []engine.Window{{Buffer: 0, Cursor: engine.Position{Row: 2, Col: 0}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Windows[state.ActiveWindow].Cursor.Row == 3, "You moved between both file boundaries using Vim's structural jumps."
			},
		},
		{
			ID:              "operators-x",
			ModuleID:        "operators",
			Title:           "Delete A Single Character",
			Goal:            "Delete the ! from `build!` using x.",
			Explanation:     "x is the smallest delete command and a good bridge into operators.",
			Hints:           []string{"The cursor starts on !.", "Press x."},
			Skills:          []string{"operators.x"},
			CommandsLearned: []string{"x"},
			CanonicalSolutions: []string{
				"x",
			},
			FocusTokens: []string{"x"},
			Rule:        "Use x when you want to remove exactly the character under the cursor.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "cmd.txt", Lines: []string{"build!"}}},
				Windows: []engine.Window{{Buffer: 0, Cursor: engine.Position{Row: 0, Col: 5}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Buffers[0].Lines[0] == "build", "x removed the character under the cursor."
			},
		},
		{
			ID:              "operators-dw-dd",
			ModuleID:        "operators",
			Title:           "Delete With Motions",
			Goal:            "Delete TODO with dw, then delete the remaining line `review` with dd.",
			Explanation:     "Operators become powerful when they combine with motions.",
			Hints:           []string{"Start on TODO and use dw.", "Then use dd on the next line."},
			Skills:          []string{"operators.dw", "operators.dd"},
			CommandsLearned: []string{"dw", "dd"},
			CanonicalSolutions: []string{
				"dwjdd",
			},
			FocusTokens: []string{"dw", "dd"},
			Rule:        "An operator describes what to do; the motion describes where it applies.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "tasks.txt", Lines: []string{"TODO fix tests", "review", "ship"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return stringsJoin(state.Buffers[0].Lines) == "fix tests\nship", "You used an operator with both a motion and a line target."
			},
		},
		{
			ID:              "operators-cw-d$",
			ModuleID:        "operators",
			Title:           "Change And Delete To The End",
			Goal:            "Change draft to ready with cw, then on the next line delete to the end with d$.",
			Explanation:     "c is delete plus Insert mode, while d$ applies an operator to a line anchor.",
			Hints:           []string{"Start on draft and use cw ready Esc.", "Then move down and use d$ from the start of the line."},
			Skills:          []string{"operators.cw", "operators.dline"},
			CommandsLearned: []string{"cw", "d$"},
			CanonicalSolutions: []string{
				"cwready Esc jd$",
			},
			FocusTokens: []string{"cw", "d$"},
			Rule:        "Use c when you want to replace text and d with a motion when you want a precise range.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "status.txt", Lines: []string{"status = draft", "delete the rest"}}},
				Windows: []engine.Window{{Buffer: 0, Cursor: engine.Position{Row: 0, Col: 9}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Mode == engine.ModeNormal && stringsJoin(state.Buffers[0].Lines) == "status = ready\n", "You changed one word and deleted the rest of the next line."
			},
		},
		{
			ID:              "search-repeat",
			ModuleID:        "search",
			Title:           "Search And Repeat",
			Goal:            "Search for deploy, jump to the next match with n, then back to the previous match with N.",
			Explanation:     "Search is often the fastest motion in a real file, and n/N make it fluid.",
			Hints:           []string{"Use /deploy.", "Then press n.", "Then press N."},
			Skills:          []string{"search.forward", "search.repeat"},
			CommandsLearned: []string{"/deploy", "n", "N"},
			CanonicalSolutions: []string{
				"/deploy Enter n N",
			},
			FocusTokens: []string{"/deploy", "n", "N"},
			Rule:        "Search establishes a target pattern; n and N traverse that pattern in each direction.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "release.txt", Lines: []string{"build test", "deploy staging", "deploy production"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.LastSearch == "deploy" && state.Windows[state.ActiveWindow].Cursor == (engine.Position{Row: 1, Col: 0}), "You searched forward, moved to the next match, then reversed direction with N."
			},
		},
		{
			ID:              "search-star-noh",
			ModuleID:        "search",
			Title:           "Search Current Word",
			Goal:            "Use * on deploy, then clear highlights with :noh.",
			Explanation:     "* turns the word under the cursor into your search target.",
			Hints:           []string{"The cursor starts on deploy.", "Press *.", "Then use :noh."},
			Skills:          []string{"search.star", "options.noh"},
			CommandsLearned: []string{"*", ":noh"},
			CanonicalSolutions: []string{
				"* :noh",
			},
			FocusTokens: []string{"*", ":noh"},
			Rule:        "Use * for quick word-based navigation and :noh when highlights are no longer useful.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "ops.txt", Lines: []string{"deploy staging", "deploy production"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.LastSearch == "deploy" && !state.Options.HLSearch, "You searched for the word under the cursor and then cleared highlights."
			},
		},
		{
			ID:              "search-marks",
			ModuleID:        "search",
			Title:           "Marks Remember Places",
			Goal:            "Set mark a on beta with ma, move to the last line, then jump back with 'a.",
			Explanation:     "Marks are a lightweight way to bookmark locations while you work.",
			Hints:           []string{"Press ma on beta.", "Move down twice with j.", "Jump back with 'a."},
			Skills:          []string{"marks.set", "marks.jump"},
			CommandsLearned: []string{"ma", "'a"},
			CanonicalSolutions: []string{
				"majj'a",
			},
			FocusTokens: []string{"ma", "'a"},
			Rule:        "Set marks before leaving a place you know you will need again.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "notes.txt", Lines: []string{"alpha", "beta", "gamma", "delta"}}},
				Windows: []engine.Window{{Buffer: 0, Cursor: engine.Position{Row: 1, Col: 1}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Windows[state.ActiveWindow].Cursor == (engine.Position{Row: 1, Col: 0}), "You bookmarked a location with a mark and returned to it."
			},
		},
		{
			ID:              "editing-openlines",
			ModuleID:        "editing",
			Title:           "Open New Lines",
			Goal:            "Use o to open a line below, type second, press Esc, then use O to open a line above and type zero.",
			Explanation:     "o and O create new structure without manual cursor gymnastics.",
			Hints:           []string{"Press o, type second, Esc.", "Then press O, type zero, Esc."},
			Skills:          []string{"editing.o", "editing.O"},
			CommandsLearned: []string{"o", "O"},
			CanonicalSolutions: []string{
				"osecond Esc Ozero Esc",
			},
			FocusTokens: []string{"o", "O"},
			Rule:        "Use o/O when you want new lines without first entering Insert mode manually.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "list.txt", Lines: []string{"first"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Mode == engine.ModeNormal && stringsJoin(state.Buffers[0].Lines) == "zero\nfirst\nsecond", "You created new lines above and below without leaving the command flow."
			},
		},
		{
			ID:              "editing-undo-redo",
			ModuleID:        "editing",
			Title:           "Undo And Redo",
			Goal:            "Delete the ! with x, undo it with u, then redo it with Ctrl-r.",
			Explanation:     "Confident Vim use depends on trusting undo and redo.",
			Hints:           []string{"Use x.", "Then u.", "Then Ctrl-r."},
			Skills:          []string{"editing.undo", "editing.redo"},
			CommandsLearned: []string{"u", "Ctrl-r"},
			CanonicalSolutions: []string{
				"x u Ctrl-r",
			},
			FocusTokens: []string{"x", "u", "<C-r>"},
			Rule:        "Fast editing gets safer when you can confidently reverse and restore changes.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "ping.txt", Lines: []string{"pong!"}}},
				Windows: []engine.Window{{Buffer: 0, Cursor: engine.Position{Row: 0, Col: 4}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Buffers[0].Lines[0] == "pong", "You deleted the character, undid the change, and then redid it."
			},
		},
		{
			ID:              "editing-yank-paste",
			ModuleID:        "editing",
			Title:           "Yank And Paste",
			Goal:            "Use yy to yank the current line, then press p to paste a copy below it.",
			Explanation:     "Yank and paste are essential for fast structural edits.",
			Hints:           []string{"Press y twice to yank the line.", "Press p to paste it below."},
			Skills:          []string{"editing.yank", "editing.paste"},
			CommandsLearned: []string{"yy", "p"},
			CanonicalSolutions: []string{
				"yyp",
			},
			FocusTokens: []string{"yy", "p"},
			Rule:        "Use yy to copy a line and p to place it without leaving Normal mode.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "routes.txt", Lines: []string{"/health", "/users"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return stringsJoin(state.Buffers[0].Lines) == "/health\n/health\n/users", "You yanked a full line and pasted it directly below."
			},
		},
		{
			ID:              "textobjects-diw",
			ModuleID:        "textobjects",
			Title:           "Inner Word",
			Goal:            "Delete just the word middle with diw so the line becomes `alpha  gamma`.",
			Explanation:     "iw means inner word: the word itself without surrounding whitespace.",
			Hints:           []string{"Place the cursor on middle.", "Use d then i then w."},
			Skills:          []string{"textobjects.iw", "textobjects.diw"},
			CommandsLearned: []string{"iw", "diw"},
			CanonicalSolutions: []string{
				"diw",
			},
			FocusTokens: []string{"diw"},
			Rule:        "Text objects let you target syntactic units instead of counting characters.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "words.txt", Lines: []string{"alpha middle gamma"}}},
				Windows: []engine.Window{{Buffer: 0, Cursor: engine.Position{Row: 0, Col: 7}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Buffers[0].Lines[0] == "alpha  gamma", "diw removed only the inner word, not the adjacent spaces."
			},
		},
		{
			ID:              "textobjects-caw",
			ModuleID:        "textobjects",
			Title:           "A Word Includes Space",
			Goal:            "Change beta to core with caw, type core, and press Esc so the line becomes `alpha core gamma`.",
			Explanation:     "aw includes surrounding space, which is useful when you want a clean replacement.",
			Hints:           []string{"Use caw on beta.", "Type core and press Esc."},
			Skills:          []string{"textobjects.aw", "textobjects.caw"},
			CommandsLearned: []string{"aw", "caw"},
			CanonicalSolutions: []string{
				"cawcore Esc",
			},
			FocusTokens: []string{"caw"},
			Rule:        "Use aw when the replacement should cleanly absorb the surrounding spacing.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "phrase.txt", Lines: []string{"alpha beta gamma"}}},
				Windows: []engine.Window{{Buffer: 0, Cursor: engine.Position{Row: 0, Col: 6}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Mode == engine.ModeNormal && state.Buffers[0].Lines[0] == "alpha core gamma", "caw replaced the word together with its surrounding spacing."
			},
		},
		{
			ID:              "commandline-options",
			ModuleID:        "commandline",
			Title:           "Layer Vim Options",
			Goal:            "Turn on relative numbers with :set relativenumber, then turn on search highlights with :set hlsearch.",
			Explanation:     "Command-line options change how Vim behaves around your editing flow.",
			Hints:           []string{"Use :set relativenumber.", "Then use :set hlsearch."},
			Skills:          []string{"options.relativenumber", "options.hlsearch"},
			CommandsLearned: []string{":set relativenumber", ":set hlsearch"},
			CanonicalSolutions: []string{
				":set relativenumber",
				":set hlsearch",
			},
			FocusTokens: []string{":set relativenumber", ":set hlsearch"},
			Rule:        "Use :set to shape the editor around the navigation style you want.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "main.go", Lines: []string{"package main", "", "func main() {}"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Options.RelativeNumber && state.Options.HLSearch, "You enabled both relative numbers and highlighted search matches."
			},
		},
		{
			ID:              "commandline-search-tuning",
			ModuleID:        "commandline",
			Title:           "Search Tuning Options",
			Goal:            "Enable smarter search defaults with :set ignorecase smartcase incsearch.",
			Explanation:     "These three options make searching feel better in everyday coding sessions.",
			Hints:           []string{"Use a single :set command with all three options.", "You can also run them as separate :set commands."},
			Skills:          []string{"options.ignorecase", "options.smartcase", "options.incsearch"},
			CommandsLearned: []string{":set ignorecase smartcase incsearch"},
			CanonicalSolutions: []string{
				":set ignorecase smartcase incsearch",
			},
			FocusTokens: []string{":set ignorecase smartcase incsearch"},
			Rule:        "Tune search behavior to reduce friction before it slows you down.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "search.txt", Lines: []string{"Deploy", "deploy"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Options.IgnoreCase && state.Options.SmartCase && state.Options.IncSearch, "Search options are now configured for common workflows."
			},
		},
		{
			ID:              "commandline-help-source",
			ModuleID:        "commandline",
			Title:           "Help And Source",
			Goal:            "Open help for :set with :help :set, then source your config with :source ~/.vimrc.",
			Explanation:     "Knowing how to access docs and reload config saves time long-term.",
			Hints:           []string{"Use :help :set first.", "Then use :source ~/.vimrc."},
			Skills:          []string{"commandline.help", "commandline.source"},
			CommandsLearned: []string{":help", ":source"},
			CanonicalSolutions: []string{
				":help :set",
				":source ~/.vimrc",
			},
			FocusTokens: []string{":help :set", ":source ~/.vimrc"},
			Rule:        "Use :help while learning and :source to apply config changes quickly.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "vimrc", Lines: []string{"set number"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return historyHas(state.CommandHistory, ":help :set") && historyHas(state.CommandHistory, ":source ~/.vimrc"), "You used built-in help and simulated a config reload."
			},
		},
		{
			ID:              "commandline-sed-filter",
			ModuleID:        "commandline",
			Title:           "Run A Sed Filter From Vim",
			Goal:            "Transform TODO to DONE across the whole buffer using :%!sed 's/TODO/DONE/g'.",
			Explanation:     ":%!sed pipes the full buffer through sed, which is useful for bulk transformations.",
			Hints:           []string{"Use :%!sed 's/TODO/DONE/g'.", "Run it once and inspect the buffer state."},
			Skills:          []string{"commandline.filter", "commandline.sed"},
			CommandsLearned: []string{":%!sed"},
			CanonicalSolutions: []string{
				":%!sed 's/TODO/DONE/g'",
			},
			FocusTokens: []string{":%!sed 's/TODO/DONE/g'"},
			Rule:        "Use :%!<cmd> when you want an external filter to rewrite the current buffer.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "tasks.txt", Lines: []string{"TODO docs", "TODO tests", "ship"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return stringsJoin(state.Buffers[0].Lines) == "DONE docs\nDONE tests\nship", "You applied a sed filter to the full buffer from inside Vim."
			},
		},
		{
			ID:              "commandline-files",
			ModuleID:        "commandline",
			Title:           "Open Files And Explorer",
			Goal:            "Open README.md with :e README.md, then open the explorer with :Ex.",
			Explanation:     "The command-line is also where many file-level workflows begin.",
			Hints:           []string{"Use :e README.md.", "Then use :Ex."},
			Skills:          []string{"files.edit", "files.explorer"},
			CommandsLearned: []string{":e", ":Ex"},
			CanonicalSolutions: []string{
				":e README.md",
				":Ex",
			},
			FocusTokens: []string{":e README.md", ":Ex"},
			Rule:        "Use :e to move between buffers and :Ex when you want a built-in file browser.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "main.go", Lines: []string{"package main"}}, {Name: "README.md", Lines: []string{"# Docs"}}},
			},
			Check: func(state engine.State) (bool, string) {
				active := state.Windows[state.ActiveWindow]
				return state.Buffers[active.Buffer].Name == "README.md" && state.ExplorerOpen, "You switched to another buffer and opened the explorer."
			},
		},
		{
			ID:              "windows-splits",
			ModuleID:        "windows",
			Title:           "Split And Cycle Windows",
			Goal:            "Open a split with :split, then cycle to the other window with Ctrl-w w.",
			Explanation:     "Windows let you see multiple views without leaving Vim.",
			Hints:           []string{"Use :split.", "Then press Ctrl-w followed by w."},
			Skills:          []string{"windows.split", "windows.cycle"},
			CommandsLearned: []string{":split", "Ctrl-w w"},
			CanonicalSolutions: []string{
				":split",
				"Ctrl-w w",
			},
			FocusTokens: []string{":split", "<C-w>w"},
			Rule:        "Window management is a command layer on top of the same buffer data.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "notes.txt", Lines: []string{"alpha", "beta"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return len(state.Windows) == 2 && state.ActiveWindow == 1, "You created a second window and moved focus to it."
			},
		},
		{
			ID:              "windows-buffers",
			ModuleID:        "windows",
			Title:           "Cycle Buffers",
			Goal:            "Open config.lua with :e config.lua, then move to the next buffer with :bn.",
			Explanation:     "Buffers are open files; windows are views onto those buffers.",
			Hints:           []string{"Use :e config.lua.", "Then use :bn."},
			Skills:          []string{"buffers.open", "buffers.next"},
			CommandsLearned: []string{":e config.lua", ":bn"},
			CanonicalSolutions: []string{
				":e config.lua",
				":bn",
			},
			FocusTokens: []string{":e config.lua", ":bn"},
			Rule:        "Use buffer commands when you want to move among open files without changing the window layout.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "main.go", Lines: []string{"package main"}}, {Name: "README.md", Lines: []string{"# Docs"}}, {Name: "config.lua", Lines: []string{"vim.o.number = true"}}},
			},
			Check: func(state engine.State) (bool, string) {
				active := state.Windows[state.ActiveWindow]
				return state.Buffers[active.Buffer].Name == "main.go", "You opened another buffer and then cycled forward through the buffer list."
			},
		},
		{
			ID:              "macros-repeat",
			ModuleID:        "macros",
			Title:           "Repeat The Last Change",
			Goal:            "Delete the first # with x, move down with j, then repeat that delete with .",
			Explanation:     ". repeats your last change, which is often faster than recording a full macro.",
			Hints:           []string{"Press x on the first #.", "Move down with j.", "Press . to repeat the delete."},
			Skills:          []string{"repeat.dot"},
			CommandsLearned: []string{"."},
			CanonicalSolutions: []string{
				"xj.",
			},
			FocusTokens: []string{"x", "."},
			Rule:        "Reach for . whenever you want to repeat the same edit in a nearby context.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "todo.txt", Lines: []string{"# one", "# two"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return stringsJoin(state.Buffers[0].Lines) == " one\n two", "You used . to repeat the previous delete without retyping it."
			},
		},
		{
			ID:              "macros-basic",
			ModuleID:        "macros",
			Title:           "Record And Replay A Macro",
			Goal:            "Record x into register a with qaxq, move down with j, then replay it with @a.",
			Explanation:     "Macros let you record a sequence of keys and replay it later.",
			Hints:           []string{"Press q then a to start recording.", "Press x to delete the first #, then q to stop.", "Move down and use @a."},
			Skills:          []string{"macros.record", "macros.play"},
			CommandsLearned: []string{"qa...q", "@a"},
			CanonicalSolutions: []string{
				"qaxqj@a",
			},
			FocusTokens: []string{"qa", "@a"},
			Rule:        "Record macros when the same multi-key transformation will happen again.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "todo.txt", Lines: []string{"# alpha", "# beta"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return stringsJoin(state.Buffers[0].Lines) == " alpha\n beta", "You recorded a macro once and replayed it on the next line."
			},
		},
		{
			ID:              "vimscript-let-echo",
			ModuleID:        "vimscript",
			Title:           "Variables And Echo",
			Goal:            "Set g:trainer_level to \"core\" with :let, then print it with :echo g:trainer_level.",
			Explanation:     "Vimscript starts with variables and quick feedback using :echo.",
			Hints:           []string{"Use :let g:trainer_level = \"core\".", "Then run :echo g:trainer_level."},
			Skills:          []string{"vimscript.let", "vimscript.echo"},
			CommandsLearned: []string{":let", ":echo"},
			CanonicalSolutions: []string{
				":let g:trainer_level = \"core\"",
				":echo g:trainer_level",
			},
			FocusTokens: []string{":let g:trainer_level = \"core\"", ":echo g:trainer_level"},
			Rule:        "Define a variable with :let and inspect it with :echo while iterating on config.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: ".vimrc", Lines: []string{"\" training config"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Variables["g:trainer_level"] == "core" && state.LastEcho == "core", "You stored a Vimscript variable and read it back successfully."
			},
		},
		{
			ID:              "vimscript-mapping",
			ModuleID:        "vimscript",
			Title:           "Create A Normal-Mode Mapping",
			Goal:            "Add a save mapping with :nnoremap <leader>w :w<CR>, then remove it with :unmap <leader>w.",
			Explanation:     "Mappings are one of the highest-leverage Vim customizations.",
			Hints:           []string{"Create the mapping first with :nnoremap.", "Then remove it with :unmap."},
			Skills:          []string{"vimscript.nnoremap", "vimscript.unmap"},
			CommandsLearned: []string{":nnoremap", ":unmap"},
			CanonicalSolutions: []string{
				":nnoremap <leader>w :w<CR>",
				":unmap <leader>w",
			},
			FocusTokens: []string{":nnoremap <leader>w :w<CR>", ":unmap <leader>w"},
			Rule:        "Use non-recursive mappings (`nnoremap`) for predictable key behavior.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: ".vimrc", Lines: []string{"\" map section"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return historyHas(state.CommandHistory, ":nnoremap <leader>w :w<CR>") && historyHas(state.CommandHistory, ":unmap <leader>w") && !mappingExists(state.NormalMappings, "<leader>w"), "You created and removed a mapping, which is a common Vimscript workflow."
			},
		},
		{
			ID:              "neovim-checkhealth",
			ModuleID:        "neovim",
			Title:           "Check Neovim Health",
			Goal:            "Run :checkhealth to validate your Neovim environment.",
			Explanation:     ":checkhealth is the first command to run when plugins or integrations feel broken.",
			Hints:           []string{"Open command-line mode and run :checkhealth."},
			Skills:          []string{"neovim.checkhealth"},
			CommandsLearned: []string{":checkhealth"},
			CanonicalSolutions: []string{
				":checkhealth",
			},
			FocusTokens: []string{":checkhealth"},
			Rule:        "Start Neovim troubleshooting with health checks before editing config blindly.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "init.lua", Lines: []string{"vim.o.number = true"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return historyHas(state.CommandHistory, ":checkhealth"), "You ran Neovim's built-in health checks."
			},
		},
		{
			ID:              "neovim-terminal",
			ModuleID:        "neovim",
			Title:           "Use The Built-In Terminal",
			Goal:            "Open Neovim's integrated terminal with :terminal.",
			Explanation:     "Neovim's terminal lets you run shell commands without leaving the editor.",
			Hints:           []string{"Run :terminal and notice the terminal:// buffer."},
			Skills:          []string{"neovim.terminal"},
			CommandsLearned: []string{":terminal"},
			CanonicalSolutions: []string{
				":terminal",
			},
			FocusTokens: []string{":terminal"},
			Rule:        "Integrated terminal workflows reduce context switching for build/test loops.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "main.go", Lines: []string{"package main"}}},
			},
			Check: func(state engine.State) (bool, string) {
				active := state.Windows[state.ActiveWindow]
				return strings.HasPrefix(state.Buffers[active.Buffer].Name, "terminal://"), "You opened a terminal buffer from inside Neovim."
			},
		},
		{
			ID:              "neovim-lua-basics",
			ModuleID:        "neovim",
			Title:           "Lua Config Basics",
			Goal:            "Set `vim.g.mapleader` with :lua vim.g.mapleader = \" \" then confirm it with :lua print(vim.g.mapleader).",
			Explanation:     "Neovim config is Lua-first, so `:lua` is a core command to understand.",
			Hints:           []string{"Assign vim.g.mapleader first.", "Then print it with :lua print(...)."},
			Skills:          []string{"neovim.lua", "neovim.lua.globals"},
			CommandsLearned: []string{":lua"},
			CanonicalSolutions: []string{
				":lua vim.g.mapleader = \" \"",
				":lua print(vim.g.mapleader)",
			},
			FocusTokens: []string{":lua vim.g.mapleader = \" \"", ":lua print(vim.g.mapleader)"},
			Rule:        "Use :lua for quick config experiments before editing larger Lua files.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "init.lua", Lines: []string{"-- Neovim config"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Variables["g:mapleader"] == " " && state.LastEcho == " ", "You set and read a Neovim Lua global successfully."
			},
		},
	}
	return append(base, extraLessons()...)
}

func stringsJoin(lines []string) string {
	out := ""
	for i, line := range lines {
		if i > 0 {
			out += "\n"
		}
		out += line
	}
	return out
}

func historyHas(history []string, token string) bool {
	for _, item := range history {
		if item == token {
			return true
		}
	}
	return false
}

func mappingExists(mappings map[string]string, lhs string) bool {
	if mappings == nil {
		return false
	}
	_, ok := mappings[lhs]
	return ok
}
