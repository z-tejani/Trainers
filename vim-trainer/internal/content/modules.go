package content

// moduleSet defines the curriculum's module structure and dependency
// graph. Modules stay in Go because the Module struct is small and
// changes infrequently; lessons live as data in lessons/*.json and are
// loaded at startup.
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

// lessonSet returns the full lesson catalog. With the data-file
// migration done, this just delegates to the JSON loader. Errors during
// load are non-fatal — malformed lessons are skipped, the rest still
// ship — so a single bad file can't take down the curriculum.
func lessonSet() []Lesson {
	lessons, _ := loadJSONLessons()
	return lessons
}
