package content

import "vimtrainer/internal/engine"

func extraModules() []Module {
	return []Module{
		{ID: "visual", Title: "Visual Mode", Summary: "Use visual selections for quick region-based edits.", PrerequisiteModules: []string{"textobjects"}},
		{ID: "textobjects-advanced", Title: "Advanced Text Objects", Summary: "Operate inside quotes, parentheses, and paragraphs.", PrerequisiteModules: []string{"visual"}},
		{ID: "jumps", Title: "Jumps And Change List", Summary: "Navigate with jumplist and changelist primitives.", PrerequisiteModules: []string{"search"}},
		{ID: "registers", Title: "Registers", Summary: "Use named registers for reusable yanks and edits.", PrerequisiteModules: []string{"operators"}},
		{ID: "quickfix", Title: "Quickfix Workflow", Summary: "Search globally and iterate findings with quickfix.", PrerequisiteModules: []string{"search"}},
		{ID: "netrw", Title: "Explorer And Netrw Basics", Summary: "Navigate the built-in explorer with command-line flows.", PrerequisiteModules: []string{"windows"}},
		{ID: "ecosystem", Title: "Plugin Manager Literacy", Summary: "Understand common plugin manager and tooling commands.", PrerequisiteModules: []string{"neovim"}},
		{ID: "lsp", Title: "LSP Essentials", Summary: "Use LSP discovery and definition jumps in daily editing.", PrerequisiteModules: []string{"ecosystem"}},
		{ID: "telescope", Title: "Telescope Navigation", Summary: "Use Telescope pickers for files and buffer switching.", PrerequisiteModules: []string{"lsp"}},
		{ID: "safety", Title: "Recovery And Safety", Summary: "Learn save-all, recovery, and force-quit safety flows.", PrerequisiteModules: []string{"commandline"}},
		{ID: "performance", Title: "Performance And Debugging", Summary: "Use profile and script inspection commands.", PrerequisiteModules: []string{"safety"}},
	}
}

func extraLessons() []Lesson {
	return []Lesson{
		{
			ID:              "visual-basic-delete",
			ModuleID:        "visual",
			Title:           "Visual Character Delete",
			Goal:            "Delete `TODO` using visual mode with `v`, `e`, then `d`.",
			Explanation:     "Visual mode is ideal when you can see the range and want an immediate operation.",
			Hints:           []string{"Press v to enter visual mode.", "Move to the end of TODO with e.", "Press d to delete the selected text."},
			Skills:          []string{"visual.char", "visual.delete"},
			CommandsLearned: []string{"v", "e", "d"},
			CanonicalSolutions: []string{
				"ved",
			},
			FocusTokens: []string{"v", "e", "vd"},
			Rule:        "Use visual mode when selecting and operating on a region is clearer than building an operator motion.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "todo.txt", Lines: []string{"TODO"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Buffers[0].Lines[0] == "", "You selected and deleted a visual range."
			},
		},
		{
			ID:              "visual-line-yank-paste",
			ModuleID:        "visual",
			Title:           "Visual Line Yank And Paste",
			Goal:            "Yank the middle line with `V y`, move down, then paste with `p`.",
			Explanation:     "Linewise visual mode is useful for copy-and-reorder tasks.",
			Hints:           []string{"Use V to enter visual-line mode.", "Press y to yank the line.", "Move down with j and paste with p."},
			Skills:          []string{"visual.line", "operators.yank", "operators.paste"},
			CommandsLearned: []string{"V", "y", "p"},
			CanonicalSolutions: []string{
				"Vyjp",
			},
			FocusTokens: []string{"V", "vy", "p"},
			Rule:        "Linewise yanks are useful when structure matters more than exact columns.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "order.txt", Lines: []string{"alpha", "beta", "gamma"}}},
				Windows: []engine.Window{{Buffer: 0, Cursor: engine.Position{Row: 1, Col: 0}}},
			},
			Check: func(state engine.State) (bool, string) {
				lines := state.Buffers[0].Lines
				return len(lines) == 4 && lines[3] == "beta", "You yanked a full line in visual mode and pasted it."
			},
		},
		{
			ID:              "textobjects-quotes",
			ModuleID:        "textobjects-advanced",
			Title:           "Change Inside Quotes",
			Goal:            "Replace `pending` with `done` using `ci\"`.",
			Explanation:     "Quote text objects let you edit values without touching the surrounding syntax.",
			Hints:           []string{"Place the cursor inside the quoted text.", "Use c i \" then type done and Esc."},
			Skills:          []string{"textobjects.quote", "operators.change"},
			CommandsLearned: []string{"ci\""},
			CanonicalSolutions: []string{
				"ci\"done Esc",
			},
			FocusTokens: []string{"ci\""},
			Rule:        "Text objects are safer than manual cursor movement for structured edits.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "config.lua", Lines: []string{"status = \"pending\""}}},
				Windows: []engine.Window{{Buffer: 0, Cursor: engine.Position{Row: 0, Col: 11}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Mode == engine.ModeNormal && state.Buffers[0].Lines[0] == "status = \"done\"", "You changed only the text inside the quotes."
			},
		},
		{
			ID:              "textobjects-parens",
			ModuleID:        "textobjects-advanced",
			Title:           "Delete Inside Parentheses",
			Goal:            "Use `di(` to remove function arguments from `deploy(api, true)`.",
			Explanation:     "Parenthesis text objects are core for editing function calls and signatures.",
			Hints:           []string{"Place the cursor between ( and ).", "Use d i (."},
			Skills:          []string{"textobjects.paren", "operators.delete"},
			CommandsLearned: []string{"di("},
			CanonicalSolutions: []string{
				"di(",
			},
			FocusTokens: []string{"di("},
			Rule:        "Use inner objects (`i`) when you want to preserve delimiters.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "call.go", Lines: []string{"deploy(api, true)"}}},
				Windows: []engine.Window{{Buffer: 0, Cursor: engine.Position{Row: 0, Col: 8}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Buffers[0].Lines[0] == "deploy()", "You removed only the content inside the parentheses."
			},
		},
		{
			ID:              "textobjects-paragraph",
			ModuleID:        "textobjects-advanced",
			Title:           "Delete Around Paragraph",
			Goal:            "Use `dap` to remove the first paragraph and the separator line.",
			Explanation:     "Paragraph objects are useful for prose and commit message cleanup.",
			Hints:           []string{"Start in the first paragraph.", "Use d a p."},
			Skills:          []string{"textobjects.paragraph", "operators.delete"},
			CommandsLearned: []string{"dap"},
			CanonicalSolutions: []string{
				"dap",
			},
			FocusTokens: []string{"dap"},
			Rule:        "Around objects (`a`) include surrounding separators when appropriate.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "notes.md", Lines: []string{"alpha", "beta", "", "tail"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return len(state.Buffers[0].Lines) == 1 && state.Buffers[0].Lines[0] == "tail", "You deleted a paragraph as a structural unit."
			},
		},
		{
			ID:              "jumps-jumplist",
			ModuleID:        "jumps",
			Title:           "Jumplist Back And Forward",
			Goal:            "Search `deploy`, move to next hit, then use Ctrl-o and Ctrl-i to move through jump history.",
			Explanation:     "Jumplist navigation lets you explore aggressively and still return safely.",
			Hints:           []string{"Use /deploy and Enter.", "Use n to move forward.", "Then Ctrl-o and Ctrl-i."},
			Skills:          []string{"jumps.list", "search.repeat"},
			CommandsLearned: []string{"/", "n", "Ctrl-o", "Ctrl-i"},
			CanonicalSolutions: []string{
				"/deploy Enter n ctrl+o ctrl+i",
			},
			FocusTokens: []string{"<C-o>", "<C-i>"},
			Rule:        "Jump history is a navigation safety net for large files.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "release.txt", Lines: []string{"build", "deploy staging", "deploy production"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Windows[state.ActiveWindow].Cursor.Row == 2, "You used jumplist navigation to move back and forward through jumps."
			},
		},
		{
			ID:              "jumps-changelist",
			ModuleID:        "jumps",
			Title:           "Change List Navigation",
			Goal:            "Make edits on two lines, then use `g;` to jump to an older change.",
			Explanation:     "Changelist jumps are useful when revisiting recent edits quickly.",
			Hints:           []string{"Delete one char with x, move down, delete another with x.", "Then use g;."},
			Skills:          []string{"jumps.changelist"},
			CommandsLearned: []string{"x", "g;"},
			CanonicalSolutions: []string{
				"xjxg;",
			},
			FocusTokens: []string{"g;"},
			Rule:        "Use changelist jumps when reviewing or finishing recent edits.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "changes.txt", Lines: []string{"ab", "cd"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Windows[state.ActiveWindow].Cursor.Row == 0, "You jumped back to an older change location."
			},
		},
		{
			ID:              "registers-named",
			ModuleID:        "registers",
			Title:           "Named Register Reuse",
			Goal:            "Yank the first line into register a, jump to the bottom, then paste it.",
			Explanation:     "Named registers let you keep reusable text while performing other edits.",
			Hints:           []string{"Use \"a then yy.", "Jump to bottom with G.", "Paste with p."},
			Skills:          []string{"registers.named", "operators.yank", "operators.paste"},
			CommandsLearned: []string{"\"ayy", "G", "p"},
			CanonicalSolutions: []string{
				"\"ayyGp",
			},
			FocusTokens: []string{"\"a", "yy", "p"},
			Rule:        "Use named registers when you need to preserve text beyond one operation.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "copy.txt", Lines: []string{"alpha", "omega"}}},
			},
			Check: func(state engine.State) (bool, string) {
				lines := state.Buffers[0].Lines
				return len(lines) == 3 && lines[2] == "alpha", "You stored text in a named register and reused it later."
			},
		},
		{
			ID:              "quickfix-vimgrep",
			ModuleID:        "quickfix",
			Title:           "Quickfix Iterate Matches",
			Goal:            "Run :vimgrep /TODO/ % and jump to the next hit with :cnext.",
			Explanation:     "Quickfix turns global matches into an actionable navigation list.",
			Hints:           []string{"Start with :vimgrep /TODO/ %.", "Then run :cnext."},
			Skills:          []string{"quickfix.vimgrep", "quickfix.next"},
			CommandsLearned: []string{":vimgrep", ":cnext"},
			CanonicalSolutions: []string{
				":vimgrep /TODO/ %",
				":cnext",
			},
			FocusTokens: []string{":vimgrep /TODO/ %", ":cnext"},
			Rule:        "Use quickfix when you need to process many search hits efficiently.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "todos.txt", Lines: []string{"TODO alpha", "noop", "TODO beta"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return state.Windows[state.ActiveWindow].Cursor.Row == 2, "You moved through a quickfix result set."
			},
		},
		{
			ID:              "netrw-parent-nav",
			ModuleID:        "netrw",
			Title:           "Explorer Parent Navigation",
			Goal:            "Open explorer at /tmp, move to parent with -, then exit with Esc.",
			Explanation:     "Netrw is a lightweight built-in explorer, useful even without plugins.",
			Hints:           []string{"Use :Explore /tmp.", "Press - to go up.", "Press Esc to close explorer."},
			Skills:          []string{"netrw.open", "netrw.parent", "netrw.exit"},
			CommandsLearned: []string{":Explore", "-", "Esc"},
			CanonicalSolutions: []string{
				":Explore /tmp",
				"-",
				"Esc",
			},
			FocusTokens: []string{":Explore /tmp", "-", "esc"},
			Rule:        "Know the built-in explorer so you can navigate files even in minimal setups.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "project.txt", Lines: []string{"root"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return !state.ExplorerOpen && state.ExplorerPath == "/" && state.Mode == engine.ModeNormal, "You opened explorer, navigated to parent, and exited cleanly."
			},
		},
		{
			ID:              "ecosystem-plugin-commands",
			ModuleID:        "ecosystem",
			Title:           "Plugin Workflow Literacy",
			Goal:            "Run :Lazy sync, then :Mason, then :TSUpdate.",
			Explanation:     "You do not need to memorize one plugin manager, but you should recognize common maintenance flows.",
			Hints:           []string{"Run :Lazy sync.", "Then :Mason.", "Then :TSUpdate."},
			Skills:          []string{"plugins.lazy", "plugins.mason", "plugins.treesitter"},
			CommandsLearned: []string{":Lazy sync", ":Mason", ":TSUpdate"},
			CanonicalSolutions: []string{
				":Lazy sync",
				":Mason",
				":TSUpdate",
			},
			FocusTokens: []string{":Lazy sync", ":Mason", ":TSUpdate"},
			Rule:        "Operational literacy matters: know what maintenance commands look like and when to run them.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "init.lua", Lines: []string{"-- plugins"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return historyHas(state.CommandHistory, ":Lazy sync") && historyHas(state.CommandHistory, ":Mason") && historyHas(state.CommandHistory, ":TSUpdate"), "You ran the core plugin maintenance commands."
			},
		},
		{
			ID:              "lsp-definition-jump",
			ModuleID:        "lsp",
			Title:           "LSP Info And Definition Jump",
			Goal:            "Run :LspInfo, then jump to deploy definition with gd.",
			Explanation:     "LSP workflows are anchored by two habits: inspect client state and jump to definitions quickly.",
			Hints:           []string{"Run :LspInfo first.", "Move cursor over deploy and use gd."},
			Skills:          []string{"lsp.info", "lsp.definition"},
			CommandsLearned: []string{":LspInfo", "gd"},
			CanonicalSolutions: []string{
				":LspInfo",
				"gd",
			},
			FocusTokens: []string{":LspInfo", "gd"},
			Rule:        "Use :LspInfo for diagnosis and gd for core navigation.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "main.go", Lines: []string{"deploy()", "func deploy() {}", "func other() {}"}}},
				Windows: []engine.Window{{Buffer: 0, Cursor: engine.Position{Row: 0, Col: 0}}},
			},
			Check: func(state engine.State) (bool, string) {
				return historyHas(state.CommandHistory, ":LspInfo") && state.Windows[state.ActiveWindow].Cursor.Row == 1, "You inspected LSP status and jumped to a definition."
			},
		},
		{
			ID:              "telescope-pickers",
			ModuleID:        "telescope",
			Title:           "Telescope Picker Basics",
			Goal:            "Open :Telescope find_files, exit with Esc, then run :Telescope buffers.",
			Explanation:     "Telescope pickers are fast entry points for file, text, and buffer navigation.",
			Hints:           []string{"Start with :Telescope find_files.", "Press Esc to close picker/explorer mode.", "Run :Telescope buffers."},
			Skills:          []string{"telescope.files", "telescope.buffers"},
			CommandsLearned: []string{":Telescope find_files", ":Telescope buffers"},
			CanonicalSolutions: []string{
				":Telescope find_files",
				"Esc",
				":Telescope buffers",
			},
			FocusTokens: []string{":Telescope find_files", ":Telescope buffers"},
			Rule:        "Treat pickers as command interfaces for navigation, not just UI widgets.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "app.lua", Lines: []string{"print('hi')"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return historyHas(state.CommandHistory, ":Telescope find_files") && historyHas(state.CommandHistory, ":Telescope buffers"), "You used two common Telescope pickers in sequence."
			},
		},
		{
			ID:              "safety-recovery-flow",
			ModuleID:        "safety",
			Title:           "Safety And Recovery Commands",
			Goal:            "Run :wa, then :qa!, then :recover.",
			Explanation:     "Safe editing includes understanding global write, force quit, and recovery flows.",
			Hints:           []string{"Use :wa first.", "Then :qa!.", "Then run :recover."},
			Skills:          []string{"safety.writeall", "safety.forcequit", "safety.recover"},
			CommandsLearned: []string{":wa", ":qa!", ":recover"},
			CanonicalSolutions: []string{
				":wa",
				":qa!",
				":recover",
			},
			FocusTokens: []string{":wa", ":qa!", ":recover"},
			Rule:        "Know the safety commands before you need them in a stressful situation.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "notes.txt", Lines: []string{"draft"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return historyHas(state.CommandHistory, ":wa") && historyHas(state.CommandHistory, ":qa!") && historyHas(state.CommandHistory, ":recover"), "You practiced the core safety and recovery commands."
			},
		},
		{
			ID:              "performance-profile",
			ModuleID:        "performance",
			Title:           "Profile And Inspect",
			Goal:            "Start profiling, stop profiling, then inspect script load order with :scriptnames.",
			Explanation:     "Performance debugging starts with collecting profile data and inspecting runtime context.",
			Hints:           []string{"Use :profile start /tmp/vim.prof.", "Then :profile stop.", "Then :scriptnames."},
			Skills:          []string{"debug.profile", "debug.scriptnames"},
			CommandsLearned: []string{":profile start", ":profile stop", ":scriptnames"},
			CanonicalSolutions: []string{
				":profile start /tmp/vim.prof",
				":profile stop",
				":scriptnames",
			},
			FocusTokens: []string{":profile start /tmp/vim.prof", ":profile stop", ":scriptnames"},
			Rule:        "Measure first, then inspect runtime context before making performance changes.",
			Initial: engine.Scenario{
				Buffers: []engine.Buffer{{Name: "init.vim", Lines: []string{"set number"}}},
			},
			Check: func(state engine.State) (bool, string) {
				return !state.ProfileActive && historyHas(state.CommandHistory, ":scriptnames"), "You started and stopped profiling, then inspected script loading."
			},
		},
	}
}
