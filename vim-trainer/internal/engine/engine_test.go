package engine

import "testing"

func TestYankLineAndPaste(t *testing.T) {
	editor := NewEditor(Scenario{
		Buffers: []Buffer{{Name: "routes.txt", Lines: []string{"/health", "/users"}}},
	})

	editor.ProcessKey("y")
	editor.ProcessKey("y")
	editor.ProcessKey("p")

	got := editor.State().Buffers[0].Lines
	if len(got) != 3 || got[0] != "/health" || got[1] != "/health" || got[2] != "/users" {
		t.Fatalf("unexpected pasted lines: %#v", got)
	}
}

func TestSetCommonSearchOptions(t *testing.T) {
	editor := NewEditor(Scenario{
		Buffers: []Buffer{{Name: "search.txt", Lines: []string{"Deploy", "deploy"}}},
	})

	editor.mode = ModeCommand
	editor.executeCommand("set ignorecase smartcase incsearch")

	state := editor.State()
	if !state.Options.IgnoreCase || !state.Options.SmartCase || !state.Options.IncSearch {
		t.Fatalf("expected search options enabled, got %#v", state.Options)
	}
}

func TestVimscriptLetAndEcho(t *testing.T) {
	editor := NewEditor(Scenario{
		Buffers: []Buffer{{Name: ".vimrc", Lines: []string{"\" test"}}},
	})

	editor.mode = ModeCommand
	editor.executeCommand("let g:trainer_level = \"core\"")
	editor.executeCommand("echo g:trainer_level")

	state := editor.State()
	if state.Variables["g:trainer_level"] != "core" {
		t.Fatalf("variable not set: %#v", state.Variables)
	}
	if state.LastEcho != "core" {
		t.Fatalf("LastEcho = %q, want %q", state.LastEcho, "core")
	}
}

func TestVimscriptMappingLifecycle(t *testing.T) {
	editor := NewEditor(Scenario{
		Buffers: []Buffer{{Name: ".vimrc", Lines: []string{"\" map"}}},
	})

	editor.mode = ModeCommand
	editor.executeCommand("nnoremap <leader>w :w<CR>")
	if got := editor.State().NormalMappings["<leader>w"]; got != ":w<CR>" {
		t.Fatalf("mapping = %q", got)
	}
	editor.executeCommand("unmap <leader>w")
	if _, ok := editor.State().NormalMappings["<leader>w"]; ok {
		t.Fatal("mapping should have been removed")
	}
}

func TestSedFilterCommand(t *testing.T) {
	editor := NewEditor(Scenario{
		Buffers: []Buffer{{Name: "tasks.txt", Lines: []string{"TODO docs", "TODO tests", "ship"}}},
	})

	editor.mode = ModeCommand
	result := editor.executeCommand("%!sed 's/TODO/DONE/g'")
	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	got := editor.State().Buffers[0].Lines
	if got[0] != "DONE docs" || got[1] != "DONE tests" {
		t.Fatalf("sed filter not applied: %#v", got)
	}
}

func TestNeovimLuaAndTerminalCommands(t *testing.T) {
	editor := NewEditor(Scenario{
		Buffers: []Buffer{{Name: "init.lua", Lines: []string{"-- config"}}},
	})

	editor.mode = ModeCommand
	editor.executeCommand("lua vim.g.mapleader = \" \"")
	editor.executeCommand("lua print(vim.g.mapleader)")
	editor.executeCommand("terminal")

	state := editor.State()
	if state.Variables["g:mapleader"] != " " {
		t.Fatalf("mapleader not set: %#v", state.Variables)
	}
	if state.LastEcho != " " {
		t.Fatalf("LastEcho = %q, want space", state.LastEcho)
	}
	active := state.Windows[state.ActiveWindow]
	if state.Buffers[active.Buffer].Name != "terminal://zsh" {
		t.Fatalf("expected terminal buffer, got %q", state.Buffers[active.Buffer].Name)
	}
}

func TestVisualModeDelete(t *testing.T) {
	editor := NewEditor(Scenario{
		Buffers: []Buffer{{Name: "todo.txt", Lines: []string{"TODO"}}},
	})

	editor.ProcessKey("v")
	editor.ProcessKey("e")
	editor.ProcessKey("d")

	if got := editor.State().Buffers[0].Lines[0]; got != "" {
		t.Fatalf("line = %q, want empty", got)
	}
}

func TestNamedRegisterYankAndPaste(t *testing.T) {
	editor := NewEditor(Scenario{
		Buffers: []Buffer{{Name: "copy.txt", Lines: []string{"alpha", "omega"}}},
	})

	editor.ProcessKey("\"")
	editor.ProcessKey("a")
	editor.ProcessKey("y")
	editor.ProcessKey("y")
	editor.ProcessKey("G")
	editor.ProcessKey("p")

	lines := editor.State().Buffers[0].Lines
	if len(lines) != 3 || lines[2] != "alpha" {
		t.Fatalf("unexpected lines: %#v", lines)
	}
}

func TestQuickfixVimgrepAndCnext(t *testing.T) {
	editor := NewEditor(Scenario{
		Buffers: []Buffer{{Name: "todos.txt", Lines: []string{"TODO alpha", "noop", "TODO beta"}}},
	})

	editor.mode = ModeCommand
	editor.executeCommand("vimgrep /TODO/ %")
	editor.executeCommand("cnext")

	if row := editor.State().Windows[0].Cursor.Row; row != 2 {
		t.Fatalf("cursor row = %d, want 2", row)
	}
}

func TestExplorerParentPath(t *testing.T) {
	editor := NewEditor(Scenario{
		Buffers: []Buffer{{Name: "notes.txt", Lines: []string{"x"}}},
	})

	editor.mode = ModeCommand
	editor.executeCommand("Explore /tmp")
	editor.ProcessKey("-")
	editor.ProcessKey("esc")

	state := editor.State()
	if state.ExplorerOpen {
		t.Fatal("explorer should be closed")
	}
	if state.ExplorerPath != "/" {
		t.Fatalf("ExplorerPath = %q, want /", state.ExplorerPath)
	}
}

func TestAdvancedTextObjects(t *testing.T) {
	editor := NewEditor(Scenario{
		Buffers: []Buffer{{Name: "cfg.txt", Lines: []string{"status = \"pending\"", "deploy(api, true)"}}},
		Windows: []Window{{Buffer: 0, Cursor: Position{Row: 0, Col: 11}}},
	})

	editor.ProcessKey("c")
	editor.ProcessKey("i")
	editor.ProcessKey("\"")
	editor.ProcessKey("d")
	editor.ProcessKey("o")
	editor.ProcessKey("n")
	editor.ProcessKey("e")
	editor.ProcessKey("esc")
	editor.ProcessKey("j")
	editor.ProcessKey("d")
	editor.ProcessKey("i")
	editor.ProcessKey("(")

	lines := editor.State().Buffers[0].Lines
	if lines[0] != "status = \"done\"" {
		t.Fatalf("line0 = %q", lines[0])
	}
	if lines[1] != "deploy()" {
		t.Fatalf("line1 = %q", lines[1])
	}
}

func TestJumpListBackAndForward(t *testing.T) {
	editor := NewEditor(Scenario{
		Buffers: []Buffer{{Name: "release.txt", Lines: []string{"build", "deploy staging", "deploy production"}}},
	})

	editor.mode = ModeCommand
	editor.executeCommand("vimgrep /deploy/ %")
	editor.executeCommand("cnext")
	editor.ProcessKey("ctrl+o")
	editor.ProcessKey("ctrl+i")

	if row := editor.State().Windows[0].Cursor.Row; row != 2 {
		t.Fatalf("cursor row = %d, want 2", row)
	}
}
