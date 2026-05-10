package content

// extraModules contributes the second-tier modules (visual, advanced text
// objects, jumps, registers, quickfix, netrw, ecosystem, lsp, telescope,
// safety, performance, refactor, indent, insert tricks, ex ranges) that
// build on the base modules. Lessons for these modules live as data in
// lessons/*.json.
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
		{ID: "refactor", Title: "Refactoring Workflows", Summary: "Combine search, gn, dot, macros, and quickfix to refactor at speed.", PrerequisiteModules: []string{"macros", "quickfix"}},
		{ID: "indent", Title: "Indent And Format", Summary: "Use >>, <<, and visual >/< to manage indentation.", PrerequisiteModules: []string{"operators"}},
		{ID: "inserttricks", Title: "Insert-Mode Tricks", Summary: "Stay in Insert mode and still wield <C-o>, <C-w>, <C-u>.", PrerequisiteModules: []string{"editing"}},
		{ID: "exranges", Title: "Ex Ranges", Summary: "Operate on slices of the buffer with :N,M ranges.", PrerequisiteModules: []string{"commandline"}},
		{ID: "tabs", Title: "Tab Pages", Summary: "Use :tabnew, gt / gT, and :tabclose for layout sets.", PrerequisiteModules: []string{"windows"}},
		{ID: "format", Title: "Format And Reflow", Summary: "Reindent code with =ip and reflow prose with gqip.", PrerequisiteModules: []string{"indent"}},
		{ID: "windowsmove", Title: "Window Movement", Summary: "Rearrange splits with Ctrl-W H/J/K/L and Ctrl-W o.", PrerequisiteModules: []string{"windows"}},
		{ID: "macrosadv", Title: "Macros (Advanced)", Summary: "Replay last macro with @@, append with qA, run macros over ranges with :normal.", PrerequisiteModules: []string{"macros"}},
		{ID: "marksadv", Title: "Marks (Advanced)", Summary: "Use auto marks ('., '^, '<, '>) and global marks (mA-mZ) across buffers.", PrerequisiteModules: []string{"search"}},
		{ID: "registersadv", Title: "Registers (Advanced)", Summary: "Black hole \"_, system clipboard \"+, the numbered ring \"1-\"9, and special registers.", PrerequisiteModules: []string{"registers"}},
		{ID: "shellintegration", Title: "Shell Integration", Summary: "Run :!cmd, read output with :r !, and drive :make.", PrerequisiteModules: []string{"commandline"}},
		{ID: "configwalkthrough", Title: "Config Walkthrough", Summary: "Set leader, indent, splits, completion, clipboard, undo, and more.", PrerequisiteModules: []string{"commandline"}},
		{ID: "exdeep", Title: "Ex Deep", Summary: "Use :sort, :retab, :args, :argdo, :bufdo, :execute, :normal, <C-^>.", PrerequisiteModules: []string{"exranges"}},
		{ID: "mappingsdeep", Title: "Mappings Deep", Summary: "Per-mode mappings (inoremap, vnoremap, cnoremap, tnoremap, omap) and :command!.", PrerequisiteModules: []string{"vimscript"}},
		{ID: "foldsdeep", Title: "Folds", Summary: "Manual folds with zf/zo/zc/za, plus zR / zM / zj / zk navigation.", PrerequisiteModules: []string{"editing"}},
		{ID: "spelldiff", Title: "Spell And Diff", Summary: "Toggle :set spell, navigate misspells with ]s, and run :diffthis.", PrerequisiteModules: []string{"commandline"}},
		{ID: "powermoves", Title: "Misc Power Moves", Summary: "<C-a>, <C-x>, J, gJ, ~, g~iw, gUiw, guiw, g??, gi.", PrerequisiteModules: []string{"editing"}},
		{ID: "autocmds", Title: "Autocommands", Summary: ":autocmd, :augroup, :doautocmd — events the editor reacts to.", PrerequisiteModules: []string{"vimscript"}},
		{ID: "sessions", Title: "Sessions And Views", Summary: ":mksession, :source, :mkview, :loadview, :wshada.", PrerequisiteModules: []string{"commandline"}},
		{ID: "uisurfaces", Title: "UI Surfaces", Summary: "statusline, tabline, winbar, conceal, sign column, popups.", PrerequisiteModules: []string{"windows"}},
		{ID: "helpcmdline", Title: "Help And Cmdline History", Summary: "q:, q/, :helpgrep, <C-]> tag jump.", PrerequisiteModules: []string{"commandline"}},
		{ID: "ecosystem-deep", Title: "Plugin Ecosystem (Deep)", Summary: "Treesitter, DAP, snippets — recognize the commands.", PrerequisiteModules: []string{"ecosystem"}},
	}
}
