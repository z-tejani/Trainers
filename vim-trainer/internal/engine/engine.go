package engine

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"vimtrainer/internal/config"
)

type Mode string

const (
	ModeNormal   Mode = "NORMAL"
	ModeInsert   Mode = "INSERT"
	ModeVisual   Mode = "VISUAL"
	ModeReplace  Mode = "REPLACE"
	ModeCommand  Mode = "COMMAND"
	ModeSearch   Mode = "SEARCH"
	ModeExplorer Mode = "EXPLORER"
	// ModeConfirm is entered while a :s/.../gc substitute is awaiting
	// per-match user confirmation. Keys y / n / a / q advance the flow.
	ModeConfirm Mode = "CONFIRM"
)

type Position struct {
	Row int
	Col int
}

type Buffer struct {
	Name  string
	Lines []string
}

type Window struct {
	Buffer int
	Cursor Position
}

type Options struct {
	Number         bool
	RelativeNumber bool
	HLSearch       bool
	IgnoreCase     bool
	SmartCase      bool
	IncSearch      bool
	Wrap           bool
	// P2 additions. The trainer doesn't always *act* on these (rendering
	// doesn't honor scrolloff, the editor doesn't actually swap files
	// based on autoread), but storing them lets curriculum lessons
	// teach the syntax and verify settings.
	TabStop        int
	ShiftWidth     int
	ExpandTab      bool
	ScrollOff      int
	UndoFile       bool
	Clipboard      string
	SplitBelow     bool
	SplitRight     bool
	TermGUIColors  bool
	Mouse          string
	Spell          bool
	SpellLang      string
	FoldMethod     string
	CursorLine     bool
	List           bool
	ListChars      string
	BackupEnabled  bool
	SwapFile       bool
	AutoRead       bool
	LazyRedraw     bool
	UpdateTime     int
	TimeoutLen     int
	CompleteOpt    string
	WildMenu       bool
	WildMode       string
	ColorScheme    string
}

type Scenario struct {
	Buffers        []Buffer
	Windows        []Window
	ActiveWindow   int
	Options        Options
	ExplorerOpen   bool
	StartingMode   Mode
	Marks          map[rune]Position
	CurrentBuffer  string
	CurrentCursor  Position
	InitialMessage string
}

type ActionResult struct {
	Key         string
	Token       string
	Description string
	Error       string
	Changed     bool
	Completed   bool
}

type QuickfixItem struct {
	Buffer int
	Pos    Position
	Text   string
}

type State struct {
	Mode              Mode
	Options           Options
	ExplorerOpen      bool
	Buffers           []Buffer
	Windows           []Window
	ActiveWindow      int
	CommandBuffer     string
	CommandHistory    []string
	LastSearch        string
	LastSearchDir     int
	LastResult        ActionResult
	PendingCount      string
	PendingPrefix     string
	PendingOperator   rune
	PendingTextObject rune
	PendingRegister   bool
	ActiveRegister    rune
	RecordingRegister rune
	Marks             map[rune]Position
	LastChange        []string
	LastEcho          string
	Variables         map[string]string
	NormalMappings    map[string]string
	ExplorerPath      string
	Quickfix          []QuickfixItem
	QuickfixIndex     int
	QuickfixOpen      bool
	ProfileActive     bool
	// ConfirmActive is true while a :s/.../gc substitute is awaiting input.
	// ConfirmPrompt holds the user-facing question for the current match.
	// ConfirmContext is the full source line; ConfirmMatchStart /
	// ConfirmMatchEnd are rune-indexed bounds of the match within it,
	// for the UI to highlight.
	ConfirmActive     bool
	ConfirmPrompt     string
	ConfirmContext    string
	ConfirmMatchStart int
	ConfirmMatchEnd   int
	// QuitRequested is set when the user runs :q / :wq / :qa / :qa! / :quit
	// from the engine. The TUI checks this after every keystroke and exits
	// when set, mirroring how those commands behave in real Vim.
	QuitRequested bool
	// TabCount and ActiveTab expose the engine's tab-page state. The
	// active tab's windows are the live state surfaced via Windows /
	// ActiveWindow above; other tabs hide behind the engine.
	TabCount  int
	ActiveTab int
	// TextWidth is the wrap target for `gq`. 0 means "use the default
	// (78)"; matches Vim's `set textwidth=N` semantics.
	TextWidth int
	// FoldCount reports how many manual folds the active buffer has.
	// FoldClosedCount tracks how many of those are currently closed.
	FoldCount       int
	FoldClosedCount int
	// P3 surface — counts and current values for declarative checks.
	AutocmdCount   int
	SessionCount   int
	ViewCount      int
	SignCount      int
	PopupCount     int
	Statusline     string
	Tabline        string
	Winbar         string
	ConcealLevel   int
}

type snapshot struct {
	Buffers       []Buffer
	Windows       []Window
	ActiveWindow  int
	Options       Options
	ExplorerOpen  bool
	Mode          Mode
	LastSearch    string
	LastSearchDir int
	Marks         map[rune]Position
}

type Editor struct {
	mode              Mode
	options           Options
	buffers           []Buffer
	windows           []Window
	activeWindow      int
	explorerOpen      bool
	commandBuffer     string
	commandHistory    []string
	lastSearch        string
	lastSearchDir     int
	lastResult        ActionResult
	pendingCount      string
	pendingPrefix     string
	pendingOperator   rune
	pendingTextObject rune
	pendingRegister   bool
	activeRegister    rune
	pendingMarkSet    bool
	pendingMarkJump   rune
	pendingCtrlW      bool
	pendingMacroAt     bool
	pendingMacroQ      bool
	lastMacroRegister  rune
	pendingInsertCtrlO bool
	pendingInsertCtrlR bool
	pendingReplace     bool
	pendingGn         bool
	pendingGN         bool
	pendingSurround   rune // 0, 'd' (ds), 'c' (cs), 'y' (ys)
	pendingSurroundCS rune // for cs<old><new>: stores <old>
	pendingSurroundYS rune // for ys<motion><char>: motion target like 'i','a'
	pendingSurroundYO rune // for ys<motion><char>: object char like 'w','"'
	marks             map[rune]Position
	undo              []snapshot
	redo              []snapshot
	macroRegisters    map[rune][]string
	textRegisters     map[rune]yankData
	yankBuffer        yankData
	recordingRegister rune
	currentMacro      []string
	lastChange        []string
	changeBaseline    *snapshot
	changeTokens      []string
	replaying         bool
	lastEcho          string
	variables         map[string]string
	normalMappings    map[string]string
	visualStart       Position
	visualLine        bool
	visualBlock       bool
	jumpList          []jumpEntry
	jumpIndex         int
	changeList        []jumpEntry
	changeIndex       int
	quickfix          []QuickfixItem
	quickfixIndex     int
	quickfixOpen      bool
	quickfixLists     [][]QuickfixItem // ring of recent lists for :cnewer / :colder
	quickfixListIdx   int
	explorerPath      string
	profileActive     bool
	confirm           *substituteConfirmState
	quitRequested     bool
	// markBuffers tracks the buffer index a given mark belongs to. The
	// trainer's existing `marks` map only stores Position; uppercase
	// marks (A–Z) need to remember their buffer too.
	markBuffers map[rune]int
	// args is the argument list seeded by :args file ... — used by
	// :next, :prev, :argdo.
	args     []string
	argIndex int
	// alternateBuffer is the buffer index that <C-^> switches to.
	alternateBuffer int
	// folds tracks per-buffer manual fold ranges.
	folds map[int][]foldRange
	// errorformat is the parser config for :make output.
	errorFormat string
	// lastShellCmd records the most recent :!cmd argument so :!! can
	// repeat it.
	lastShellCmd string
	// makePrg is the configured `:set makeprg=…`. The trainer doesn't
	// actually invoke it — fake output simulates a build run.
	makePrg string
	// Per-mode mapping tables. Storage only — the trainer doesn't yet
	// replay these on key events (replaying mappings is its own can of
	// worms with `<expr>` etc.). Keeping them as data is enough to teach
	// `:inoremap` / `:vnoremap` / `:cnoremap` / `:tnoremap` / `:omap`
	// muscle memory and to power `:command!`.
	insertMappings   map[string]string
	visualMappings   map[string]string
	cmdlineMappings  map[string]string
	terminalMappings map[string]string
	operatorMappings map[string]string
	userCommands     map[string]string
	// P3 storage (no live behavior — these exist so the curriculum can
	// teach the syntax and verify settings round-trip).
	autocmds      []autocmdEntry
	currentAuGroup string
	sessions      map[string]bool // recorded :mksession paths
	views         map[string]bool // recorded :mkview paths
	signs         []signEntry
	popups        []popupEntry
	statusline    string
	tabline       string
	winbar        string
	concealLevel  int
	// tabs holds inactive tab pages. The active tab's windows /
	// activeWindow live in the editor's main fields; tabs[activeTab] is
	// kept in sync only at switch time.
	tabs      []tabSnapshot
	activeTab int
	textWidth int
}

// foldRange is a single manual fold's [Start, End] line span plus its
// open/closed state.
type foldRange struct {
	Start  int
	End    int
	Closed bool
}

// autocmdEntry records one :autocmd directive. The trainer doesn't fire
// these on real events; storage lets curriculum teach the syntax and
// :doautocmd lets a lesson trigger one explicitly.
type autocmdEntry struct {
	Group   string
	Event   string
	Pattern string
	Command string
}

// signEntry records a :sign place directive.
type signEntry struct {
	Name   string
	Buffer int
	Line   int
}

// popupEntry records a popup_create / nvim_open_win invocation. The
// trainer does not render real popups; this is curriculum vocabulary.
type popupEntry struct {
	ID    int
	Title string
	Body  string
}

// tabSnapshot saves the windows + activeWindow of an inactive tab page.
type tabSnapshot struct {
	windows      []Window
	activeWindow int
}

type confirmMatch struct {
	Row   int
	Start int // rune-indexed
	End   int // rune-indexed, exclusive
}

type substituteConfirmState struct {
	re             *regexp.Regexp
	replacement    string // already in Go regexp form
	matches        []confirmMatch
	index          int
	replacedCount  int
	originalToken  string
}

type yankData struct {
	Text     string
	Linewise bool
}

type jumpEntry struct {
	Buffer int
	Pos    Position
}

func NewEditor(s Scenario) *Editor {
	buffers := cloneBuffers(s.Buffers)
	if len(buffers) == 0 {
		buffers = []Buffer{{Name: "buffer.txt", Lines: []string{""}}}
	}
	for i := range buffers {
		if len(buffers[i].Lines) == 0 {
			buffers[i].Lines = []string{""}
		}
	}

	windows := cloneWindows(s.Windows)
	if len(windows) == 0 {
		active := 0
		if s.CurrentBuffer != "" {
			for i, buf := range buffers {
				if buf.Name == s.CurrentBuffer {
					active = i
					break
				}
			}
		}
		windows = []Window{{Buffer: active, Cursor: s.CurrentCursor}}
	}

	e := &Editor{
		mode:           ModeNormal,
		options:        s.Options,
		buffers:        buffers,
		windows:        windows,
		activeWindow:   clamp(0, s.ActiveWindow, len(windows)-1),
		explorerOpen:   s.ExplorerOpen,
		lastSearchDir:  1,
		marks:          map[rune]Position{},
		macroRegisters: map[rune][]string{},
		textRegisters:  map[rune]yankData{},
		variables:      map[string]string{},
		normalMappings: map[string]string{},
		explorerPath:   "~",
		markBuffers:    map[rune]int{},
	}
	if s.StartingMode != "" {
		e.mode = s.StartingMode
	}
	for mark, pos := range s.Marks {
		e.marks[mark] = pos
	}
	e.normalizeWindows()
	// Seed the tabs slice with one placeholder so every tab index is
	// valid; the active tab's state always lives in the live windows /
	// activeWindow fields, never in tabs[activeTab].
	e.tabs = []tabSnapshot{{}}
	e.activeTab = 0
	cur := e.currentCursor()
	e.jumpList = []jumpEntry{{Buffer: e.active().Buffer, Pos: cur}}
	e.jumpIndex = 0
	e.changeIndex = -1
	e.textRegisters['"'] = yankData{}
	if s.InitialMessage != "" {
		e.lastResult.Description = s.InitialMessage
	}
	return e
}

// DebugSummary returns a stable, human-readable single-line dump of the
// engine's pending command-state. The UI's debug overlay calls this
// instead of poking individual fields, so the engine's internal layout
// can change without breaking the view.
func (e *Editor) DebugSummary() string {
	return fmt.Sprintf(
		"token=%q count=%q pending=%q op=%q textobj=%q recording=%q last_change=%v",
		e.lastResult.Token,
		e.pendingCount,
		e.pendingPrefix,
		string(e.pendingOperator),
		string(e.pendingTextObject),
		string(e.recordingRegister),
		e.lastChange,
	)
}

func (e *Editor) State() State {
	cmdHistory := make([]string, len(e.commandHistory))
	copy(cmdHistory, e.commandHistory)
	marks := map[rune]Position{}
	for k, v := range e.marks {
		marks[k] = v
	}
	lastChange := make([]string, len(e.lastChange))
	copy(lastChange, e.lastChange)
	variables := map[string]string{}
	for k, v := range e.variables {
		variables[k] = v
	}
	mappings := map[string]string{}
	for k, v := range e.normalMappings {
		mappings[k] = v
	}
	quickfix := make([]QuickfixItem, len(e.quickfix))
	copy(quickfix, e.quickfix)
	return State{
		Mode:              e.mode,
		Options:           e.options,
		ExplorerOpen:      e.explorerOpen,
		Buffers:           cloneBuffers(e.buffers),
		Windows:           cloneWindows(e.windows),
		ActiveWindow:      e.activeWindow,
		CommandBuffer:     e.commandBuffer,
		CommandHistory:    cmdHistory,
		LastSearch:        e.lastSearch,
		LastSearchDir:     e.lastSearchDir,
		LastResult:        e.lastResult,
		PendingCount:      e.pendingCount,
		PendingPrefix:     e.pendingPrefix,
		PendingOperator:   e.pendingOperator,
		PendingTextObject: e.pendingTextObject,
		PendingRegister:   e.pendingRegister,
		ActiveRegister:    e.activeRegister,
		RecordingRegister: e.recordingRegister,
		Marks:             marks,
		LastChange:        lastChange,
		LastEcho:          e.lastEcho,
		Variables:         variables,
		NormalMappings:    mappings,
		ExplorerPath:      e.explorerPath,
		Quickfix:          quickfix,
		QuickfixIndex:     e.quickfixIndex,
		QuickfixOpen:      e.quickfixOpen,
		ProfileActive:     e.profileActive,
		ConfirmActive:     e.confirm != nil,
		ConfirmPrompt:     e.confirmPrompt(),
		ConfirmContext:    e.confirmContext(),
		ConfirmMatchStart: e.confirmMatchStart(),
		ConfirmMatchEnd:   e.confirmMatchEnd(),
		QuitRequested:     e.quitRequested,
		TabCount:          len(e.tabs),
		ActiveTab:         e.activeTab,
		TextWidth:         e.textWidth,
		FoldCount:         e.foldCountForActive(),
		FoldClosedCount:   e.foldClosedForActive(),
		AutocmdCount:      len(e.autocmds),
		SessionCount:      len(e.sessions),
		ViewCount:         len(e.views),
		SignCount:         len(e.signs),
		PopupCount:        len(e.popups),
		Statusline:        e.statusline,
		Tabline:           e.tabline,
		Winbar:            e.winbar,
		ConcealLevel:      e.concealLevel,
	}
}

func (e *Editor) foldCountForActive() int {
	if e.folds == nil {
		return 0
	}
	if e.activeWindow >= len(e.windows) {
		return 0
	}
	return len(e.folds[e.windows[e.activeWindow].Buffer])
}

func (e *Editor) foldClosedForActive() int {
	if e.folds == nil {
		return 0
	}
	if e.activeWindow >= len(e.windows) {
		return 0
	}
	n := 0
	for _, f := range e.folds[e.windows[e.activeWindow].Buffer] {
		if f.Closed {
			n++
		}
	}
	return n
}

func (e *Editor) ProcessKey(key string) ActionResult {
	// Snapshot pendingMacroQ BEFORE dispatching: if this keystroke is the
	// register-name following `q`, the dispatcher resets the flag and
	// shouldRecordKey would otherwise see false and record the register
	// name into the macro itself.
	wasPendingMacroQ := e.pendingMacroQ
	res := e.processKey(key)
	if e.recordingRegister != 0 && !e.replaying && shouldRecordKey(key, wasPendingMacroQ, res.Token) {
		e.currentMacro = append(e.currentMacro, key)
	}
	e.lastResult = res
	return res
}

func (e *Editor) processKey(key string) ActionResult {
	switch e.mode {
	case ModeInsert:
		return e.handleInsert(key)
	case ModeVisual:
		return e.handleVisual(key)
	case ModeReplace:
		return e.handleReplace(key)
	case ModeCommand:
		return e.handlePrompt(key, true)
	case ModeSearch:
		return e.handlePrompt(key, false)
	case ModeExplorer:
		return e.handleExplorer(key)
	case ModeConfirm:
		return e.handleConfirm(key)
	default:
		return e.handleNormal(key)
	}
}

func (e *Editor) ExecuteKeys(keys ...string) ActionResult {
	var res ActionResult
	for _, key := range keys {
		res = e.ProcessKey(key)
	}
	return res
}

func (e *Editor) handleExplorer(key string) ActionResult {
	switch key {
	case "esc":
		e.explorerOpen = false
		e.mode = ModeNormal
		return ActionResult{Key: key, Token: "esc", Description: "closed explorer", Completed: true}
	case "-":
		e.explorerPath = parentPath(e.explorerPath)
		return ActionResult{Key: key, Token: "-", Description: "moved to parent directory in explorer", Completed: true}
	case "enter":
		// The trainer's explorer doesn't model a real cursor over the
		// listing; pressing Enter behaves as "open the first non-current
		// buffer" so it stays useful without any lesson-specific
		// hardcoded filename.
		current := e.windows[e.activeWindow].Buffer
		target := -1
		for i := range e.buffers {
			if i != current {
				target = i
				break
			}
		}
		if target >= 0 {
			e.windows[e.activeWindow].Buffer = target
			e.windows[e.activeWindow].Cursor = e.clampCursor(target, Position{})
		}
		e.mode = ModeNormal
		e.explorerOpen = false
		return ActionResult{Key: key, Token: "enter", Description: "exited explorer; switched to first other buffer if available", Completed: true}
	default:
		return ActionResult{Key: key, Token: key, Description: "explorer is open; use -, Enter, or Esc", Completed: true}
	}
}

func (e *Editor) handlePrompt(key string, command bool) ActionResult {
	switch key {
	case "esc":
		e.mode = ModeNormal
		e.commandBuffer = ""
		return ActionResult{Key: key, Token: "esc", Description: "canceled prompt", Completed: true}
	case "backspace":
		runes := []rune(e.commandBuffer)
		if len(runes) > 0 {
			e.commandBuffer = string(runes[:len(runes)-1])
		}
		return ActionResult{Key: key, Token: "backspace", Description: "deleted prompt character"}
	case "enter":
		text := strings.TrimSpace(e.commandBuffer)
		e.commandBuffer = ""
		e.mode = ModeNormal
		if command {
			return e.executeCommand(text)
		}
		return e.executeSearch(text, 1)
	default:
		if len([]rune(key)) == 1 {
			e.commandBuffer += key
			return ActionResult{Key: key, Token: key, Description: "typed prompt character"}
		}
		return ActionResult{Key: key, Token: key, Error: "unsupported prompt key", Description: "only text, Backspace, Enter, and Esc work in prompts", Completed: true}
	}
}

func (e *Editor) handleInsert(key string) ActionResult {
	// Ctrl-O accepts one normal-mode command, then bounces back to Insert.
	// We model this by toggling a one-shot flag and routing the next key
	// through handleNormal.
	if e.pendingInsertCtrlO {
		e.pendingInsertCtrlO = false
		// Switch to Normal, run one command, return to Insert. The
		// one-shot exists outside the change recording so it does not
		// pollute lastChange replays for `.`.
		e.mode = ModeNormal
		res := e.handleNormal(key)
		// If the handled key kept the editor in another mode (e.g.
		// command/search prompt), we leave it there — Vim's <C-o> behaves
		// the same way (entering : drops you onto the cmdline, not back
		// to Insert until the cmdline finishes).
		if e.mode == ModeNormal {
			e.mode = ModeInsert
		}
		res.Description = "ran one normal command then returned to insert"
		return res
	}
	// <C-r><reg> pastes the contents of the named register at the cursor
	// without leaving Insert mode.
	if e.pendingInsertCtrlR {
		e.pendingInsertCtrlR = false
		if len([]rune(key)) != 1 {
			return ActionResult{Key: key, Token: "<C-r>", Error: "register name must be one character", Description: "use <C-r> followed by a register name", Completed: true}
		}
		e.activeRegister = []rune(key)[0]
		data := e.resolveReadRegister()
		if data.Text == "" {
			return ActionResult{Key: key, Token: "<C-r>" + key, Description: "register was empty", Completed: true}
		}
		for _, r := range data.Text {
			if r == '\n' {
				e.insertNewlineAtCursor()
				continue
			}
			e.insertRune(r)
		}
		e.changeTokens = append(e.changeTokens, "<C-r>"+key)
		return ActionResult{Key: key, Token: "<C-r>" + key, Description: fmt.Sprintf("pasted register %s in insert mode", key), Changed: true}
	}
	switch key {
	case "esc":
		e.mode = ModeNormal
		// Record the esc as part of the change so that `.` replays the
		// full insert + exit and ends back in Normal mode. Without this,
		// dot-repeat after `cw`, `cgn`, `o`, etc. leaves the editor in
		// Insert mode on the second invocation.
		e.changeTokens = append(e.changeTokens, "esc")
		// '^ remembers the cursor position when Insert mode last ended.
		if e.marks == nil {
			e.marks = map[rune]Position{}
		}
		e.marks['^'] = e.currentCursor()
		e.finishChange("insert")
		return ActionResult{Key: key, Token: "esc", Description: "returned to normal mode", Completed: true}
	case "backspace":
		if e.backspaceInsert() {
			e.changeTokens = append(e.changeTokens, "backspace")
			return ActionResult{Key: key, Token: "backspace", Description: "deleted inserted character", Changed: true}
		}
		return ActionResult{Key: key, Token: "backspace", Description: "nothing to delete"}
	case "ctrl+o":
		e.pendingInsertCtrlO = true
		return ActionResult{Key: key, Token: "<C-o>", Description: "next key runs as a single normal-mode command"}
	case "ctrl+w":
		// Delete the word fragment to the left of the cursor. Mirrors how
		// most shells and Vim's insert mode treat <C-w>.
		if e.deleteWordBackwardInsert() {
			e.changeTokens = append(e.changeTokens, "<C-w>")
			return ActionResult{Key: key, Token: "<C-w>", Description: "deleted previous word", Changed: true}
		}
		return ActionResult{Key: key, Token: "<C-w>", Description: "nothing to delete"}
	case "ctrl+u":
		// Delete from the cursor to the start of the line.
		if e.deleteToLineStartInsert() {
			e.changeTokens = append(e.changeTokens, "<C-u>")
			return ActionResult{Key: key, Token: "<C-u>", Description: "deleted to start of line", Changed: true}
		}
		return ActionResult{Key: key, Token: "<C-u>", Description: "already at start of line"}
	case "ctrl+r":
		e.pendingInsertCtrlR = true
		return ActionResult{Key: key, Token: "<C-r>", Description: "next key picks a register to paste"}
	case "ctrl+n":
		// Buffer-keyword completion. We pop the next match in the buffer
		// in front of the cursor and insert its remaining characters.
		// This is a simplified stand-in for Vim's full completion menu.
		if word := e.completeFromBufferWords(true); word != "" {
			for _, r := range word {
				e.insertRune(r)
			}
			e.changeTokens = append(e.changeTokens, "<C-n>")
			return ActionResult{Key: key, Token: "<C-n>", Description: "completed keyword from buffer", Changed: true}
		}
		return ActionResult{Key: key, Token: "<C-n>", Description: "no completion match found"}
	case "ctrl+p":
		if word := e.completeFromBufferWords(false); word != "" {
			for _, r := range word {
				e.insertRune(r)
			}
			e.changeTokens = append(e.changeTokens, "<C-p>")
			return ActionResult{Key: key, Token: "<C-p>", Description: "completed keyword from buffer (previous)", Changed: true}
		}
		return ActionResult{Key: key, Token: "<C-p>", Description: "no completion match found"}
	default:
		if len([]rune(key)) == 1 {
			e.insertRune([]rune(key)[0])
			e.changeTokens = append(e.changeTokens, key)
			return ActionResult{Key: key, Token: key, Description: fmt.Sprintf("inserted %q", key), Changed: true}
		}
		return ActionResult{Key: key, Token: key, Error: "unsupported insert-mode key", Description: "type text or press Esc", Completed: true}
	}
}

// completeFromBufferWords scans the active buffer for words that begin
// with the same prefix as the partial word at the cursor and returns
// the *remaining* characters of the next match. forward=true picks the
// next match (Ctrl-N); forward=false picks the previous (Ctrl-P).
//
// This is a deliberately small stand-in for Vim's full Ctrl-X
// completion menu — enough fidelity to teach the keybinding without
// shipping a popup menu.
func (e *Editor) completeFromBufferWords(forward bool) string {
	cur := e.active()
	runes := e.lineRunes()
	end := cur.Cursor.Col
	start := end
	for start > 0 && isWordRune(runes[start-1]) {
		start--
	}
	if start == end {
		return ""
	}
	prefix := string(runes[start:end])
	// Collect every word-like token in the buffer that starts with
	// `prefix` and is strictly longer.
	seen := map[string]struct{}{}
	var matches []string
	for _, line := range e.currentBuffer().Lines {
		lineRunes := []rune(line)
		i := 0
		for i < len(lineRunes) {
			for i < len(lineRunes) && !isWordRune(lineRunes[i]) {
				i++
			}
			j := i
			for j < len(lineRunes) && isWordRune(lineRunes[j]) {
				j++
			}
			if j > i {
				word := string(lineRunes[i:j])
				if word != prefix && strings.HasPrefix(word, prefix) {
					if _, dup := seen[word]; !dup {
						seen[word] = struct{}{}
						matches = append(matches, word)
					}
				}
			}
			i = j
			if i < len(lineRunes) {
				i++
			}
		}
	}
	if len(matches) == 0 {
		return ""
	}
	idx := 0
	if !forward {
		idx = len(matches) - 1
	}
	choice := matches[idx]
	return strings.TrimPrefix(choice, prefix)
}

// insertNewlineAtCursor splits the current line at the cursor; the right
// half becomes the next line. Used by <C-r> when the register contents
// contain a literal newline.
func (e *Editor) insertNewlineAtCursor() {
	cur := e.active()
	runes := e.lineRunes()
	col := cur.Cursor.Col
	if col > len(runes) {
		col = len(runes)
	}
	left := string(runes[:col])
	right := string(runes[col:])
	lines := e.currentBuffer().Lines
	row := cur.Cursor.Row
	updated := append([]string{}, lines[:row]...)
	updated = append(updated, left, right)
	updated = append(updated, lines[row+1:]...)
	e.currentBuffer().Lines = updated
	cur.Cursor = Position{Row: row + 1, Col: 0}
}

// deleteWordBackwardInsert removes the word fragment immediately to the
// left of the cursor. Trailing whitespace before the word is also consumed
// so repeated <C-w> presses keep deleting whole words.
func (e *Editor) deleteWordBackwardInsert() bool {
	cur := e.active()
	runes := e.lineRunes()
	if cur.Cursor.Col == 0 {
		return false
	}
	end := cur.Cursor.Col
	i := end - 1
	// Eat any trailing whitespace so "foo " + <C-w> nukes "foo " not just
	// the trailing space.
	for i >= 0 && unicode.IsSpace(runes[i]) {
		i--
	}
	for i >= 0 && isWordRune(runes[i]) {
		i--
	}
	start := i + 1
	if start >= end {
		return false
	}
	runes = append(runes[:start], runes[end:]...)
	e.setCurrentLine(string(runes))
	cur.Cursor.Col = start
	return true
}

// deleteToLineStartInsert removes from the cursor (exclusive) back to the
// start of the line. Mirrors Vim's <C-u> in Insert mode.
func (e *Editor) deleteToLineStartInsert() bool {
	cur := e.active()
	runes := e.lineRunes()
	if cur.Cursor.Col == 0 {
		return false
	}
	runes = append([]rune{}, runes[cur.Cursor.Col:]...)
	e.setCurrentLine(string(runes))
	cur.Cursor.Col = 0
	return true
}

func (e *Editor) handleVisual(key string) ActionResult {
	switch key {
	case "esc":
		// Stamp '< / '> with the bounds of the just-finished selection
		// before tearing the selection down — that's how `gv` or `'<`
		// recover the last visual region in real Vim.
		e.recordVisualMarks()
		e.mode = ModeNormal
		e.visualLine = false
		e.visualBlock = false
		return ActionResult{Key: key, Token: "esc", Description: "exited visual mode", Completed: true}
	case "h":
		e.moveLeft()
		return ActionResult{Key: key, Token: "h", Description: "expanded visual selection left", Completed: true}
	case "j":
		e.moveDown()
		return ActionResult{Key: key, Token: "j", Description: "expanded visual selection down", Completed: true}
	case "k":
		e.moveUp()
		return ActionResult{Key: key, Token: "k", Description: "expanded visual selection up", Completed: true}
	case "l":
		e.moveRight()
		return ActionResult{Key: key, Token: "l", Description: "expanded visual selection right", Completed: true}
	case "w":
		e.moveWordForward()
		return ActionResult{Key: key, Token: "w", Description: "expanded visual selection by word", Completed: true}
	case "b":
		e.moveWordBackward()
		return ActionResult{Key: key, Token: "b", Description: "expanded visual selection backward by word", Completed: true}
	case "e":
		e.moveWordEnd()
		return ActionResult{Key: key, Token: "e", Description: "expanded visual selection to end of word", Completed: true}
	case "0":
		e.moveLineStart()
		return ActionResult{Key: key, Token: "0", Description: "expanded visual selection to line start", Completed: true}
	case "$":
		e.moveLineEnd()
		return ActionResult{Key: key, Token: "$", Description: "expanded visual selection to line end", Completed: true}
	case "d":
		token := "d"
		e.beginChange(token)
		deleted, ok := e.deleteVisualSelection()
		if !ok {
			e.cancelChange()
			return ActionResult{Key: key, Token: token, Error: "empty visual selection", Description: "expand the visual selection before deleting", Completed: true}
		}
		e.recordVisualMarks()
		e.mode = ModeNormal
		e.visualLine = false
		e.visualBlock = false
		e.writeTextRegister(deleted, true)
		e.finishChange("vd")
		return ActionResult{Key: key, Token: "vd", Description: "deleted visual selection", Changed: true, Completed: true}
	case "y":
		yanked, ok := e.yankVisualSelection()
		if !ok {
			return ActionResult{Key: key, Token: "y", Error: "empty visual selection", Description: "expand the visual selection before yanking", Completed: true}
		}
		e.recordVisualMarks()
		e.mode = ModeNormal
		e.visualLine = false
		e.visualBlock = false
		e.writeTextRegister(yanked, false)
		e.recordHistory("vy")
		return ActionResult{Key: key, Token: "vy", Description: "yanked visual selection", Completed: true}
	case "p":
		token := "vp"
		e.beginChange(token)
		if !e.replaceVisualSelectionWithPaste() {
			e.cancelChange()
			return ActionResult{Key: key, Token: token, Description: "nothing to paste", Completed: true}
		}
		e.recordVisualMarks()
		e.mode = ModeNormal
		e.visualLine = false
		e.visualBlock = false
		e.finishChange(token)
		return ActionResult{Key: key, Token: token, Description: "replaced selection with register text", Changed: true, Completed: true}
	case ">":
		sr, _, er, _, _ := e.visualRange()
		token := "v>"
		e.beginChange(token)
		e.indentLines(sr, er, +1)
		e.recordVisualMarks()
		e.mode = ModeNormal
		e.visualLine = false
		e.visualBlock = false
		e.finishChange(token)
		return ActionResult{Key: key, Token: token, Description: "indented selected lines", Changed: true, Completed: true}
	case "<":
		sr, _, er, _, _ := e.visualRange()
		token := "v<"
		e.beginChange(token)
		e.indentLines(sr, er, -1)
		e.recordVisualMarks()
		e.mode = ModeNormal
		e.visualLine = false
		e.visualBlock = false
		e.finishChange(token)
		return ActionResult{Key: key, Token: token, Description: "outdented selected lines", Changed: true, Completed: true}
	default:
		return ActionResult{Key: key, Token: key, Error: "unsupported visual-mode key", Description: "the trainer supports h/j/k/l/w/b/e/0/$ plus d, y, p, >, <, and Esc in visual mode", Completed: true}
	}
}

func (e *Editor) handleNormal(key string) ActionResult {
	if e.pendingReplace {
		e.pendingReplace = false
		if len([]rune(key)) != 1 {
			return ActionResult{Key: key, Token: "r", Error: "replace needs one character", Description: "press r followed by a single replacement character", Completed: true}
		}
		token := e.composeToken("r"+key, true)
		e.beginChange(token)
		if !e.replaceCharUnderCursor([]rune(key)[0]) {
			e.cancelChange()
			return ActionResult{Key: key, Token: token, Error: "nothing to replace", Description: "place the cursor on a character before using r", Completed: true}
		}
		e.finishChange(token)
		return ActionResult{Key: key, Token: token, Description: fmt.Sprintf("replaced character with %q", key), Changed: true, Completed: true}
	}
	if e.pendingSurround != 0 {
		op := e.pendingSurround
		switch op {
		case 'd':
			e.pendingSurround = 0
			if len([]rune(key)) != 1 {
				return ActionResult{Key: key, Token: "ds", Error: "ds needs a surround character", Description: "use ds followed by \", ', (, [, {, or <", Completed: true}
			}
			token := "ds" + key
			e.beginChange(token)
			if !e.deleteSurround([]rune(key)[0]) {
				e.cancelChange()
				return ActionResult{Key: key, Token: token, Error: "no matching surround", Description: "the cursor is not inside a matching pair", Completed: true}
			}
			e.finishChange(token)
			return ActionResult{Key: key, Token: token, Description: "deleted surrounding pair", Changed: true, Completed: true}
		case 'c':
			if e.pendingSurroundCS == 0 {
				if len([]rune(key)) != 1 {
					return ActionResult{Key: key, Token: "cs", Error: "cs needs two surround characters", Description: "use cs<old><new>", Completed: true}
				}
				e.pendingSurroundCS = []rune(key)[0]
				return ActionResult{Key: key, Token: "cs" + key, Description: "waiting for new surround character"}
			}
			old := e.pendingSurroundCS
			e.pendingSurroundCS = 0
			e.pendingSurround = 0
			if len([]rune(key)) != 1 {
				return ActionResult{Key: key, Token: "cs", Error: "cs needs two surround characters", Description: "use cs<old><new>", Completed: true}
			}
			token := "cs" + string(old) + key
			e.beginChange(token)
			if !e.changeSurround(old, []rune(key)[0]) {
				e.cancelChange()
				return ActionResult{Key: key, Token: token, Error: "no matching surround", Description: "the cursor is not inside a matching pair for that character", Completed: true}
			}
			e.finishChange(token)
			return ActionResult{Key: key, Token: token, Description: "changed surrounding pair", Changed: true, Completed: true}
		case 'y':
			if e.pendingSurroundYS == 0 {
				if key == "i" || key == "a" {
					e.pendingSurroundYS = []rune(key)[0]
					return ActionResult{Key: key, Token: "ys" + key, Description: "waiting for text object target"}
				}
				if key == "s" {
					// yss<char> wraps current line
					e.pendingSurroundYS = 's'
					return ActionResult{Key: key, Token: "yss", Description: "waiting for surround character to wrap line"}
				}
				e.pendingSurround = 0
				return ActionResult{Key: key, Token: "ys", Error: "unsupported ys motion", Description: "use ysiw<char>, ysaw<char>, or yss<char>", Completed: true}
			}
			if e.pendingSurroundYS == 's' {
				// yss<char>
				e.pendingSurroundYS = 0
				e.pendingSurround = 0
				if len([]rune(key)) != 1 {
					return ActionResult{Key: key, Token: "yss", Error: "need a surround character", Description: "use yss followed by \", ', (, [, {, or <", Completed: true}
				}
				token := "yss" + key
				e.beginChange(token)
				if !e.surroundLine([]rune(key)[0]) {
					e.cancelChange()
					return ActionResult{Key: key, Token: token, Error: "could not wrap line", Description: "yss could not surround the current line", Completed: true}
				}
				e.finishChange(token)
				return ActionResult{Key: key, Token: token, Description: "wrapped line with surround", Changed: true, Completed: true}
			}
			if e.pendingSurroundYO == 0 {
				if len([]rune(key)) != 1 {
					return ActionResult{Key: key, Token: "ys", Error: "need a text object", Description: "use ysiw<char> or ysaw<char>", Completed: true}
				}
				e.pendingSurroundYO = []rune(key)[0]
				return ActionResult{Key: key, Token: "ys" + string(e.pendingSurroundYS) + key, Description: "waiting for surround character"}
			}
			motion := e.pendingSurroundYS
			object := e.pendingSurroundYO
			e.pendingSurroundYS = 0
			e.pendingSurroundYO = 0
			e.pendingSurround = 0
			if len([]rune(key)) != 1 {
				return ActionResult{Key: key, Token: "ys", Error: "need a surround character", Description: "use ysiw<char> or ysaw<char>", Completed: true}
			}
			token := "ys" + string(motion) + string(object) + key
			e.beginChange(token)
			if !e.surroundTextObject(motion, object, []rune(key)[0]) {
				e.cancelChange()
				return ActionResult{Key: key, Token: token, Error: "no text object target", Description: "place the cursor inside a supported text object", Completed: true}
			}
			e.finishChange(token)
			return ActionResult{Key: key, Token: token, Description: "added surround around text object", Changed: true, Completed: true}
		}
	}
	if e.pendingMacroQ {
		e.pendingMacroQ = false
		// q: / q/ / q? open the cmdline / search history as a buffer.
		// We synthesize a buffer named `cmdline-history://...` and
		// switch to it. Real Vim makes this buffer editable; the
		// trainer treats it as read-only display.
		if key == ":" || key == "/" || key == "?" {
			label := map[string]string{":": "cmdline", "/": "search", "?": "search"}[key]
			name := "history://" + label
			var lines []string
			for i := len(e.commandHistory) - 1; i >= 0 && len(lines) < 50; i-- {
				h := e.commandHistory[i]
				if key == ":" && strings.HasPrefix(h, ":") {
					lines = append(lines, strings.TrimPrefix(h, ":"))
				}
				if (key == "/" || key == "?") && (strings.HasPrefix(h, "/") || strings.HasPrefix(h, "?")) {
					lines = append(lines, h[1:])
				}
			}
			if len(lines) == 0 {
				lines = []string{"(no " + label + " history yet)"}
			}
			// Reverse so newest is at bottom.
			for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
				lines[i], lines[j] = lines[j], lines[i]
			}
			// Replace any existing history buffer or append fresh.
			found := -1
			for i, b := range e.buffers {
				if b.Name == name {
					found = i
					break
				}
			}
			if found >= 0 {
				e.buffers[found].Lines = lines
				e.active().Buffer = found
			} else {
				e.buffers = append(e.buffers, Buffer{Name: name, Lines: lines})
				e.active().Buffer = len(e.buffers) - 1
			}
			e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: len(lines) - 1, Col: 0})
			token := "q" + key
			e.recordHistory(token)
			return ActionResult{Key: key, Token: token, Description: "opened " + label + " history buffer", Completed: true}
		}
		if len([]rune(key)) == 1 {
			r := []rune(key)[0]
			e.recordingRegister = r
			if unicode.IsUpper(r) {
				e.currentMacro = append([]string{}, e.macroRegisters[unicode.ToLower(r)]...)
			} else {
				e.currentMacro = nil
			}
			return ActionResult{Key: key, Token: "q" + key, Description: fmt.Sprintf("recording macro in register %s", key), Completed: true}
		}
		return ActionResult{Key: key, Token: "q", Error: "macro register must be a letter", Description: "press q followed by a register like a", Completed: true}
	}
	if e.pendingMacroAt {
		e.pendingMacroAt = false
		// Determine the count first so 5@a / 100@a both work.
		count := 1
		if e.pendingCount != "" {
			if n, err := strconv.Atoi(e.pendingCount); err == nil && n > 0 {
				count = n
			}
			e.pendingCount = ""
		}
		if len([]rune(key)) == 1 {
			reg := []rune(key)[0]
			// @@ replays the most recently used macro, mirroring real Vim.
			if reg == '@' {
				if e.lastMacroRegister == 0 {
					return ActionResult{Key: key, Token: "@@", Error: "no previous macro", Description: "run @<reg> at least once before @@", Completed: true}
				}
				reg = e.lastMacroRegister
			}
			tokens := e.macroRegisters[reg]
			if len(tokens) == 0 {
				return ActionResult{Key: key, Token: "@" + key, Error: "macro register is empty", Description: fmt.Sprintf("register %s has no recorded macro", key), Completed: true}
			}
			e.lastMacroRegister = reg
			var res ActionResult
			for i := 0; i < count; i++ {
				res = e.replay(tokens, "@"+string(reg))
			}
			if count > 1 {
				res.Token = strconv.Itoa(count) + res.Token
				res.Description = fmt.Sprintf("replayed macro %d times", count)
			}
			return res
		}
		return ActionResult{Key: key, Token: "@", Error: "macro register must be a letter", Description: "press @ followed by a register like a", Completed: true}
	}
	if e.pendingRegister {
		e.pendingRegister = false
		if len([]rune(key)) != 1 {
			return ActionResult{Key: key, Token: "\"", Error: "register name must be one character", Description: "use \" followed by a register like a or 0", Completed: true}
		}
		reg := []rune(key)[0]
		validSpecial := reg == '"' || reg == '-' || reg == '_' || reg == '+' || reg == '*' ||
			reg == '/' || reg == ':' || reg == '.' || reg == '%' || reg == '#' || reg == '='
		if unicode.IsLetter(reg) || unicode.IsDigit(reg) || validSpecial {
			e.activeRegister = reg
			token := "\"" + key
			e.recordHistory(token)
			return ActionResult{Key: key, Token: token, Description: fmt.Sprintf("using register %s for the next operation", key), Completed: true}
		}
		return ActionResult{Key: key, Token: "\"" + key, Error: "unsupported register", Description: "supported register names: letters a–z (and A–Z append), digits 0–9, and special \" - _ + * / : . % # =", Completed: true}
	}
	if e.pendingMarkSet {
		e.pendingMarkSet = false
		if len([]rune(key)) == 1 {
			r := []rune(key)[0]
			e.marks[r] = e.currentCursor()
			// Capital marks (A–Z) are global: they remember the buffer
			// they were set in, so 'A jumps across buffers.
			if unicode.IsUpper(r) {
				e.markBuffers[r] = e.active().Buffer
			}
			token := "m" + key
			e.recordHistory(token)
			return ActionResult{Key: key, Token: token, Description: fmt.Sprintf("set mark %s", key), Completed: true}
		}
		return ActionResult{Key: key, Token: "m", Error: "marks require a letter", Description: "press m followed by a mark name like a or A", Completed: true}
	}
	if e.pendingMarkJump != 0 {
		jumpKind := e.pendingMarkJump
		e.pendingMarkJump = 0
		if len([]rune(key)) == 1 {
			reg := []rune(key)[0]
			pos, ok := e.marks[reg]
			if !ok {
				return ActionResult{Key: key, Token: string(jumpKind) + key, Error: "mark not set", Description: fmt.Sprintf("mark %s has not been created yet", key), Completed: true}
			}
			win := e.active()
			// Uppercase marks know their home buffer; switch to it
			// before applying the cursor.
			if unicode.IsUpper(reg) {
				if buf, ok := e.markBuffers[reg]; ok && buf >= 0 && buf < len(e.buffers) {
					win.Buffer = buf
				}
			}
			if jumpKind == '\'' {
				pos.Col = 0
			}
			e.pushJump()
			win.Cursor = e.clampCursor(win.Buffer, pos)
			token := string(jumpKind) + key
			e.recordHistory(token)
			return ActionResult{Key: key, Token: token, Description: fmt.Sprintf("jumped to mark %s", key), Completed: true}
		}
		return ActionResult{Key: key, Token: string(jumpKind), Error: "mark jump requires a letter", Description: "press 'a or `a to jump to a mark", Completed: true}
	}
	if e.pendingCtrlW {
		e.pendingCtrlW = false
		switch key {
		case "w":
			if len(e.windows) > 1 {
				e.activeWindow = (e.activeWindow + 1) % len(e.windows)
				e.recordHistory("<C-w>w")
				return ActionResult{Key: key, Token: "<C-w>w", Description: "moved to the next window", Completed: true}
			}
			return ActionResult{Key: key, Token: "<C-w>w", Description: "there is only one window", Completed: true}
		case "H", "J", "K", "L":
			if !e.moveActiveWindow([]rune(key)[0]) {
				return ActionResult{Key: key, Token: "<C-w>" + key, Description: "no movement available in that direction", Completed: true}
			}
			token := "<C-w>" + key
			e.recordHistory(token)
			return ActionResult{Key: key, Token: token, Description: "rearranged windows", Completed: true}
		case "=":
			// Trainer doesn't model window sizes; accept the equalize
			// command as a no-op so layout-management lessons still flow.
			e.recordHistory("<C-w>=")
			return ActionResult{Key: key, Token: "<C-w>=", Description: "equalized window sizes (sizes are not modeled)", Completed: true}
		case "c":
			// Close the active window. If only one remains, no-op.
			if len(e.windows) <= 1 {
				return ActionResult{Key: key, Token: "<C-w>c", Description: "only one window — nothing to close", Completed: true}
			}
			e.windows = append(e.windows[:e.activeWindow], e.windows[e.activeWindow+1:]...)
			if e.activeWindow >= len(e.windows) {
				e.activeWindow = len(e.windows) - 1
			}
			e.recordHistory("<C-w>c")
			return ActionResult{Key: key, Token: "<C-w>c", Description: "closed the current window", Completed: true}
		case "o":
			// Only-window: drop every window except the active one.
			if len(e.windows) <= 1 {
				return ActionResult{Key: key, Token: "<C-w>o", Description: "already the only window", Completed: true}
			}
			active := e.windows[e.activeWindow]
			e.windows = []Window{active}
			e.activeWindow = 0
			e.recordHistory("<C-w>o")
			return ActionResult{Key: key, Token: "<C-w>o", Description: "kept only the current window", Completed: true}
		case "T":
			// Move current window into a new tab.
			if len(e.windows) <= 1 {
				e.openNewTab()
				e.recordHistory("<C-w>T")
				return ActionResult{Key: key, Token: "<C-w>T", Description: "opened the buffer in a new tab", Completed: true}
			}
			active := e.windows[e.activeWindow]
			e.windows = append(e.windows[:e.activeWindow], e.windows[e.activeWindow+1:]...)
			if e.activeWindow >= len(e.windows) {
				e.activeWindow = len(e.windows) - 1
			}
			e.snapshotActiveTab()
			e.tabs = append(e.tabs, tabSnapshot{windows: []Window{active}, activeWindow: 0})
			e.activeTab = len(e.tabs) - 1
			e.restoreTab(e.activeTab)
			e.recordHistory("<C-w>T")
			return ActionResult{Key: key, Token: "<C-w>T", Description: "moved window to a new tab", Completed: true}
		default:
			return ActionResult{Key: key, Token: "<C-w>" + key, Error: "unsupported window command", Description: "trainer supports <C-w>w/H/J/K/L/=/c/o/T", Completed: true}
		}
	}
	if e.pendingPrefix == "g" {
		e.pendingPrefix = ""
		switch key {
		case "g":
			e.pushJump()
			e.applyCountToMotion(func(count int) { e.moveFileTop() }, "gg")
			return ActionResult{Key: key, Token: "gg", Description: "jumped to the top of the file", Completed: true}
		case "n", "N":
			direction := 1
			if key == "N" {
				direction = -1
			}
			token := "g" + key
			if e.lastSearch == "" {
				e.pendingOperator = 0
				return ActionResult{Key: key, Token: token, Error: "no previous search", Description: "search with / or * before using gn", Completed: true}
			}
			op := e.pendingOperator
			e.pendingOperator = 0
			sr, sc, er, ec, ok := e.findMatchRange(e.lastSearch, direction)
			if !ok {
				return ActionResult{Key: key, Token: token, Error: "pattern not found", Description: "no further matches for the last search", Completed: true}
			}
			if op == 0 {
				// Plain motion: move cursor to end of match (Vim selects, but we just move).
				e.pushJump()
				e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: er, Col: max(0, ec-1)})
				e.recordHistory(token)
				return ActionResult{Key: key, Token: token, Description: "moved to next search match", Completed: true}
			}
			fullToken := string(op) + token
			e.beginChange(fullToken)
			data := yankData{}
			switch op {
			case 'd', 'c':
				data = e.deleteRange(sr, sc, er, ec, false)
			case 'y':
				data = e.extractRange(sr, sc, er, ec, false)
			}
			if data.Text == "" {
				e.cancelChange()
				return ActionResult{Key: key, Token: fullToken, Error: "empty match", Description: "matched range was empty", Completed: true}
			}
			e.writeTextRegister(data, op != 'y')
			if op == 'c' {
				e.mode = ModeInsert
				e.changeTokens = []string{string(op), "g", key}
				return ActionResult{Key: key, Token: fullToken, Description: "changed match and entered insert mode", Changed: true, Completed: true}
			}
			e.finishChange(fullToken)
			return ActionResult{Key: key, Token: fullToken, Description: "applied operator to next search match", Changed: op != 'y', Completed: true}
		case "d":
			if !e.goToDefinition() {
				return ActionResult{Key: key, Token: "gd", Error: "definition not found", Description: "no obvious definition match found for the symbol under cursor", Completed: true}
			}
			e.recordHistory("gd")
			return ActionResult{Key: key, Token: "gd", Description: "jumped to a definition match", Completed: true}
		case "q":
			e.pendingPrefix = "gq"
			return ActionResult{Key: key, Token: "gq", Description: "waiting for reflow target (gqq, gqip, gqap)"}
		case "J":
			// gJ joins lines without inserting a separator space.
			token := "gJ"
			e.beginChange(token)
			if !e.joinLines(false) {
				e.cancelChange()
				return ActionResult{Key: key, Token: token, Description: "no line below to join", Completed: true}
			}
			e.finishChange(token)
			return ActionResult{Key: key, Token: token, Description: "joined without space", Changed: true, Completed: true}
		case "i":
			// gi resumes Insert mode at the position '^ remembers.
			pos, ok := e.marks['^']
			if !ok {
				pos = e.currentCursor()
			}
			token := "gi"
			e.beginChange(token)
			e.active().Cursor = e.clampCursor(e.active().Buffer, pos)
			e.mode = ModeInsert
			e.changeTokens = []string{"gi"}
			return ActionResult{Key: key, Token: token, Description: "resumed insert at last insert position", Completed: true}
		case "~":
			e.pendingPrefix = "g~"
			return ActionResult{Key: key, Token: "g~", Description: "waiting for case-toggle motion (g~iw, g~~)"}
		case "U":
			e.pendingPrefix = "gU"
			return ActionResult{Key: key, Token: "gU", Description: "waiting for uppercase motion (gUiw, gUU)"}
		case "u":
			e.pendingPrefix = "gu"
			return ActionResult{Key: key, Token: "gu", Description: "waiting for lowercase motion (guiw, guu)"}
		case "?":
			e.pendingPrefix = "g?"
			return ActionResult{Key: key, Token: "g?", Description: "waiting for rot13 motion (g?? for line)"}
		case "t":
			if len(e.tabs) <= 1 {
				return ActionResult{Key: key, Token: "gt", Description: "only one tab page", Completed: true}
			}
			e.switchTab(e.activeTab + 1)
			e.recordHistory("gt")
			return ActionResult{Key: key, Token: "gt", Description: "moved to the next tab", Completed: true}
		case "T":
			if len(e.tabs) <= 1 {
				return ActionResult{Key: key, Token: "gT", Description: "only one tab page", Completed: true}
			}
			e.switchTab(e.activeTab - 1)
			e.recordHistory("gT")
			return ActionResult{Key: key, Token: "gT", Description: "moved to the previous tab", Completed: true}
		case ";":
			if !e.changeOlder() {
				return ActionResult{Key: key, Token: "g;", Description: "already at the oldest change location", Completed: true}
			}
			e.recordHistory("g;")
			return ActionResult{Key: key, Token: "g;", Description: "jumped to older change", Completed: true}
		case ",":
			if !e.changeNewer() {
				return ActionResult{Key: key, Token: "g,", Description: "already at the newest change location", Completed: true}
			}
			e.recordHistory("g,")
			return ActionResult{Key: key, Token: "g,", Description: "jumped to newer change", Completed: true}
		default:
			return ActionResult{Key: key, Token: "g" + key, Error: "unsupported g command", Description: "the trainer currently supports gg, gd, g;, and g,", Completed: true}
		}
	}
	if e.pendingTextObject != 0 {
		objType := e.pendingTextObject
		e.pendingTextObject = 0
		token := e.composeToken(string(e.pendingOperator)+string(objType)+key, true)
		switch e.pendingOperator {
		case 'd':
			e.beginChange(token)
			deleted, ok := e.applyTextObjectOperation('d', objType, firstRune(key))
			e.pendingOperator = 0
			if !ok || (deleted.Text == "" && !deleted.Linewise) {
				e.cancelChange()
				return ActionResult{Key: key, Token: token, Error: "no text object target", Description: "place the cursor inside a supported text object", Completed: true}
			}
			e.writeTextRegister(deleted, true)
			e.finishChange(token)
			return ActionResult{Key: key, Token: token, Description: "deleted text object", Changed: true, Completed: true}
		case 'c':
			e.beginChange(token)
			deleted, ok := e.applyTextObjectOperation('c', objType, firstRune(key))
			e.pendingOperator = 0
			if !ok || (deleted.Text == "" && !deleted.Linewise) {
				e.cancelChange()
				return ActionResult{Key: key, Token: token, Error: "no text object target", Description: "place the cursor inside a supported text object", Completed: true}
			}
			e.writeTextRegister(deleted, true)
			e.mode = ModeInsert
			e.changeTokens = []string{string('c'), string(objType), key}
			return ActionResult{Key: key, Token: token, Description: "changed text object and entered insert mode", Changed: true, Completed: true}
		case 'y':
			deleted, ok := e.applyTextObjectOperation('y', objType, firstRune(key))
			e.pendingOperator = 0
			if !ok || (deleted.Text == "" && !deleted.Linewise) {
				return ActionResult{Key: key, Token: token, Error: "no text object target", Description: "place the cursor inside a supported text object", Completed: true}
			}
			e.writeTextRegister(deleted, false)
			e.recordHistory(token)
			return ActionResult{Key: key, Token: token, Description: "yanked text object", Completed: true}
		case '=':
			// =ip / =ap: reindent the lines spanned by the text object.
			e.pendingOperator = 0
			sr, _, er, _, _, ok := e.textObjectRange(objType, firstRune(key))
			if !ok {
				return ActionResult{Key: key, Token: token, Error: "no text object target", Description: "place the cursor inside a supported text object", Completed: true}
			}
			fullToken := "=" + string(objType) + key
			e.beginChange(fullToken)
			e.reindentRange(sr, er)
			e.finishChange(fullToken)
			return ActionResult{Key: key, Token: fullToken, Description: "reindented text object", Changed: true, Completed: true}
		case 'Q':
			// gqip / gqap: reflow the lines spanned by the text object.
			e.pendingOperator = 0
			sr, _, er, _, _, ok := e.textObjectRange(objType, firstRune(key))
			if !ok {
				return ActionResult{Key: key, Token: token, Error: "no text object target", Description: "place the cursor inside a supported text object", Completed: true}
			}
			fullToken := "gq" + string(objType) + key
			e.beginChange(fullToken)
			e.reflowRange(sr, er)
			e.finishChange(fullToken)
			return ActionResult{Key: key, Token: fullToken, Description: "reflowed text object", Changed: true, Completed: true}
		case '~', 'U', 'u', '?':
			// g~iw / gUiw / guiw / g?iw — case transform across a
			// text-object range.
			caseOp := e.pendingOperator
			e.pendingOperator = 0
			sr, sc, er, ec, _, ok := e.textObjectRange(objType, firstRune(key))
			if !ok {
				return ActionResult{Key: key, Token: token, Error: "no text object target", Description: "place the cursor inside a supported text object", Completed: true}
			}
			fullToken := "g" + string(caseOp) + string(objType) + key
			e.beginChange(fullToken)
			e.applyCaseTransform(byte(caseOp), sr, sc, er, ec)
			e.finishChange(fullToken)
			return ActionResult{Key: key, Token: fullToken, Description: "case-transformed text object", Changed: true, Completed: true}
		default:
			e.pendingOperator = 0
			return ActionResult{Key: key, Token: token, Error: "text objects need d, c, y, =, or gq", Description: "use a text object with an operator like diw, ci\", yap, =ip, gqip", Completed: true}
		}
	}
	// g~ / gU / gu / g? — case operators with motion. The duplicate form
	// (g~~, gUU, guu, g??) operates on the current line. iw/aw/etc. defer
	// to the text-object pipeline.
	if strings.HasPrefix(e.pendingPrefix, "g") && len(e.pendingPrefix) == 2 {
		op := e.pendingPrefix[1] // '~', 'U', 'u', '?'
		if op == '~' || op == 'U' || op == 'u' || op == '?' {
			e.pendingPrefix = ""
			// Duplicate keystroke: g~~, gUU, guu, g?? — operate on the
			// current line.
			if (op == '~' && key == "~") || (op == 'U' && key == "U") || (op == 'u' && key == "u") || (op == '?' && key == "?") {
				row := e.currentCursor().Row
				token := "g" + string(op) + key
				e.beginChange(token)
				e.applyCaseTransform(op, row, 0, row, len([]rune(e.currentBuffer().Lines[row])))
				e.finishChange(token)
				return ActionResult{Key: key, Token: token, Description: "case-transformed current line", Changed: true, Completed: true}
			}
			// Otherwise expect a text-object motion. Buffer the
			// pendingTextObject and a sentinel operator.
			if key == "i" || key == "a" {
				e.pendingTextObject = []rune(key)[0]
				e.pendingOperator = caseOpSentinel(op)
				return ActionResult{Key: key, Token: "g" + string(op) + key, Description: "waiting for a text object"}
			}
			// Single-motion forms aren't supported (e.g. g~w would
			// require a motion-to-range translator we don't have for
			// every motion). Surface a clean error.
			return ActionResult{Key: key, Token: "g" + string(op) + key, Error: "unsupported case target", Description: "use the iw/aw/ip/ap forms or the duplicate (g~~, gUU, guu, g??)", Completed: true}
		}
	}
	if e.pendingPrefix == "]" || e.pendingPrefix == "[" {
		direction := e.pendingPrefix
		e.pendingPrefix = ""
		token := direction + key
		switch key {
		case "s":
			return ActionResult{Key: key, Token: token, Description: "moved to " + map[string]string{"]": "next", "[": "previous"}[direction] + " misspelled word (simulated)", Completed: true}
		case "c":
			return ActionResult{Key: key, Token: token, Description: "moved to " + map[string]string{"]": "next", "[": "previous"}[direction] + " diff hunk (simulated)", Completed: true}
		default:
			return ActionResult{Key: key, Token: token, Error: "unsupported bracket target", Description: "trainer supports ]s/[s and ]c/[c", Completed: true}
		}
	}
	if e.pendingPrefix == "z" {
		e.pendingPrefix = ""
		switch key {
		case "=":
			// z= asks for spell suggestions. The trainer doesn't ship a
			// dictionary, so it just acknowledges the request.
			return ActionResult{Key: key, Token: "z=", Description: "spell suggestions requested (no dictionary in trainer)", Completed: true}
		case "g":
			return ActionResult{Key: key, Token: "zg", Description: "added word to spell-good list (simulated)", Completed: true}
		case "w":
			return ActionResult{Key: key, Token: "zw", Description: "added word to spell-bad list (simulated)", Completed: true}
		case "f":
			// zf{motion} — start a manual fold operator. Simplest:
			// zfap creates a fold around the current paragraph; zf
			// alone awaits a motion (we treat it like vap for now).
			e.pendingPrefix = "zf"
			return ActionResult{Key: key, Token: "zf", Description: "fold operator pending — use ap, ip, or motion"}
		case "o":
			e.toggleFoldUnderCursor(false, true)
			e.recordHistory("zo")
			return ActionResult{Key: key, Token: "zo", Description: "opened fold", Completed: true}
		case "c":
			e.toggleFoldUnderCursor(true, false)
			e.recordHistory("zc")
			return ActionResult{Key: key, Token: "zc", Description: "closed fold", Completed: true}
		case "a":
			e.toggleFoldUnderCursor(false, false)
			e.recordHistory("za")
			return ActionResult{Key: key, Token: "za", Description: "toggled fold", Completed: true}
		case "R":
			e.openAllFolds()
			e.recordHistory("zR")
			return ActionResult{Key: key, Token: "zR", Description: "opened every fold", Completed: true}
		case "M":
			e.closeAllFolds()
			e.recordHistory("zM")
			return ActionResult{Key: key, Token: "zM", Description: "closed every fold", Completed: true}
		case "j":
			if !e.jumpFold(+1) {
				return ActionResult{Key: key, Token: "zj", Description: "no fold below cursor", Completed: true}
			}
			e.recordHistory("zj")
			return ActionResult{Key: key, Token: "zj", Description: "jumped to next fold", Completed: true}
		case "k":
			if !e.jumpFold(-1) {
				return ActionResult{Key: key, Token: "zk", Description: "no fold above cursor", Completed: true}
			}
			e.recordHistory("zk")
			return ActionResult{Key: key, Token: "zk", Description: "jumped to previous fold", Completed: true}
		case "d":
			e.deleteFoldUnderCursor()
			e.recordHistory("zd")
			return ActionResult{Key: key, Token: "zd", Description: "deleted fold under cursor", Completed: true}
		case "E":
			e.folds = nil
			e.recordHistory("zE")
			return ActionResult{Key: key, Token: "zE", Description: "deleted every fold", Completed: true}
		default:
			return ActionResult{Key: key, Token: "z" + key, Error: "unsupported fold command", Description: "trainer supports zf, zo, zc, za, zR, zM, zj, zk, zd, zE", Completed: true}
		}
	}
	if e.pendingPrefix == "zf" {
		e.pendingPrefix = ""
		// Accept ap / ip directly; otherwise treat as cancellation.
		objType := rune(0)
		target := rune(0)
		switch key {
		case "a", "i":
			objType = []rune(key)[0]
			e.pendingPrefix = "zf" + key
			return ActionResult{Key: key, Token: "zf" + key, Description: "fold target awaiting (p for paragraph)"}
		}
		_ = objType
		_ = target
		return ActionResult{Key: key, Token: "zf" + key, Error: "unsupported fold target", Description: "use zfap or zfip", Completed: true}
	}
	if strings.HasPrefix(e.pendingPrefix, "zf") && len(e.pendingPrefix) == 3 {
		// e.g. pendingPrefix = "zfa" or "zfi"
		objType := []rune(e.pendingPrefix)[2]
		e.pendingPrefix = ""
		sr, _, er, _, _, ok := e.textObjectRange(objType, firstRune(key))
		if !ok {
			return ActionResult{Key: key, Token: "zf" + string(objType) + key, Error: "no text object target", Description: "use zfap or zfip on a paragraph", Completed: true}
		}
		e.addFold(sr, er)
		token := "zf" + string(objType) + key
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: fmt.Sprintf("created a manual fold over %d line(s)", er-sr+1), Completed: true}
	}
	// = is the reindent operator. == reindents the current line; =ip /
	// =ap reindents a paragraph. The trainer "reindent" simply aligns
	// the target lines to the indent of the first non-empty line, which
	// is faithful enough for curated lesson scenarios.
	if e.pendingPrefix == "=" {
		e.pendingPrefix = ""
		switch key {
		case "=":
			row := e.currentCursor().Row
			token := e.composeToken("==", true)
			e.beginChange(token)
			e.reindentRange(row, row)
			e.finishChange(token)
			return ActionResult{Key: key, Token: token, Description: "reindented current line", Changed: true, Completed: true}
		case "i", "a":
			e.pendingTextObject = []rune(key)[0]
			e.pendingOperator = '='
			return ActionResult{Key: key, Token: "=" + key, Description: "waiting for a text object (e.g. =ip)"}
		default:
			return ActionResult{Key: key, Token: "=" + key, Error: "unsupported reindent target", Description: "use ==, =ip, or =ap", Completed: true}
		}
	}
	// gq formats / wraps a text-object range to textwidth. gqgq and gqq
	// reformat the current line; gqip / gqap a paragraph.
	if e.pendingPrefix == "gq" {
		e.pendingPrefix = ""
		switch key {
		case "q":
			row := e.currentCursor().Row
			token := e.composeToken("gqq", true)
			e.beginChange(token)
			e.reflowRange(row, row)
			e.finishChange(token)
			return ActionResult{Key: key, Token: token, Description: "reflowed current line to textwidth", Changed: true, Completed: true}
		case "i", "a":
			e.pendingTextObject = []rune(key)[0]
			e.pendingOperator = 'Q' // sentinel routed by applyTextObjectOperation
			return ActionResult{Key: key, Token: "gq" + key, Description: "waiting for a text object (e.g. gqip)"}
		default:
			return ActionResult{Key: key, Token: "gq" + key, Error: "unsupported reflow target", Description: "use gqq, gqip, or gqap", Completed: true}
		}
	}
	// >> and << indent / outdent the current line. We model them by
	// promoting > and < to operator-level keys that wait for a duplicate.
	if e.pendingPrefix == ">" {
		e.pendingPrefix = ""
		if key == ">" {
			token := e.composeToken(">>", true)
			e.beginChange(token)
			e.indentLines(e.currentCursor().Row, e.currentCursor().Row, +1)
			e.finishChange(token)
			return ActionResult{Key: key, Token: token, Description: "indented current line", Changed: true, Completed: true}
		}
		return ActionResult{Key: key, Token: ">" + key, Error: "unsupported indent target", Description: "use >> to indent the current line", Completed: true}
	}
	if e.pendingPrefix == "<" {
		e.pendingPrefix = ""
		if key == "<" {
			token := e.composeToken("<<", true)
			e.beginChange(token)
			e.indentLines(e.currentCursor().Row, e.currentCursor().Row, -1)
			e.finishChange(token)
			return ActionResult{Key: key, Token: token, Description: "outdented current line", Changed: true, Completed: true}
		}
		return ActionResult{Key: key, Token: "<" + key, Error: "unsupported outdent target", Description: "use << to outdent the current line", Completed: true}
	}

	if e.pendingOperator != 0 {
		op := e.pendingOperator
		// Surround integration: ds, cs, ys (vim-surround style)
		if key == "s" {
			e.pendingOperator = 0
			e.pendingSurround = op
			e.pendingSurroundCS = 0
			e.pendingSurroundYS = 0
			e.pendingSurroundYO = 0
			return ActionResult{Key: key, Token: string(op) + "s", Description: "waiting for surround target"}
		}
		// gn / gN as motion target for an operator
		if key == "gn" || key == "gN" {
			// Composed via the gPrefix path below; not used here.
		}
		switch key {
		case "g":
			// Defer to g-prefix handler which will dispatch gn/gN with the operator still pending.
			e.pendingPrefix = "g"
			return ActionResult{Key: key, Token: string(op) + "g", Description: "waiting for second g-key (e.g. gn)"}
		case "d":
			e.pendingOperator = 0
			token := e.composeToken("dd", true)
			e.beginChange(token)
			line := e.currentBuffer().Lines[e.currentCursor().Row]
			e.deleteLine()
			e.writeTextRegister(yankData{Text: line, Linewise: true}, true)
			e.finishChange(token)
			return ActionResult{Key: key, Token: token, Description: "deleted the current line", Changed: true, Completed: true}
		case "w":
			e.pendingOperator = 0
			token := e.composeToken(string(op)+"w", op != 'y')
			if op == 'y' {
				if !e.yankWord() {
					return ActionResult{Key: key, Token: token, Error: "no word under cursor", Description: "place the cursor on a word before yanking", Completed: true}
				}
				e.recordHistory(token)
				return ActionResult{Key: key, Token: token, Description: "yanked to the next word", Completed: true}
			}
			e.beginChange(token)
			if op == 'd' {
				deleted, ok := e.deleteWord()
				if !ok {
					e.cancelChange()
					return ActionResult{Key: key, Token: token, Error: "no word under cursor", Description: "place the cursor on a word before deleting", Completed: true}
				}
				e.writeTextRegister(deleted, true)
				e.finishChange(token)
				return ActionResult{Key: key, Token: token, Description: "deleted to the next word", Changed: true, Completed: true}
			}
			deleted, ok := e.changeWord()
			if !ok {
				e.cancelChange()
				return ActionResult{Key: key, Token: token, Error: "no word under cursor", Description: "place the cursor on a word before changing", Completed: true}
			}
			e.writeTextRegister(deleted, true)
			e.mode = ModeInsert
			e.changeTokens = []string{string(op), "w"}
			return ActionResult{Key: key, Token: token, Description: "changed the current word and entered insert mode", Changed: true, Completed: true}
		case "$":
			e.pendingOperator = 0
			token := e.composeToken(string(op)+"$", op != 'y')
			if op == 'y' {
				if !e.yankToLineEnd() {
					return ActionResult{Key: key, Token: token, Description: "nothing to yank", Completed: true}
				}
				e.recordHistory(token)
				return ActionResult{Key: key, Token: token, Description: "yanked to the end of the line", Completed: true}
			}
			e.beginChange(token)
			if op == 'd' {
				deleted, ok := e.deleteToLineEnd()
				if !ok {
					e.cancelChange()
					return ActionResult{Key: key, Token: token, Description: "nothing to delete", Completed: true}
				}
				e.writeTextRegister(deleted, true)
				e.finishChange(token)
				return ActionResult{Key: key, Token: token, Description: "deleted to the end of the line", Changed: true, Completed: true}
			}
			deleted, ok := e.changeToLineEnd()
			if !ok {
				e.cancelChange()
				return ActionResult{Key: key, Token: token, Description: "nothing to change", Completed: true}
			}
			e.writeTextRegister(deleted, true)
			e.mode = ModeInsert
			e.changeTokens = []string{string(op), "$"}
			return ActionResult{Key: key, Token: token, Description: "changed to the end of the line and entered insert mode", Changed: true, Completed: true}
		case "i", "a":
			e.pendingTextObject = []rune(key)[0]
			return ActionResult{Key: key, Token: string(op) + key, Description: "waiting for a text object like w"}
		case "y":
			e.pendingOperator = 0
			token := e.composeToken("yy", false)
			e.yankLine()
			e.recordHistory(token)
			return ActionResult{Key: key, Token: token, Description: "yanked the current line", Completed: true}
		default:
			e.pendingOperator = 0
			return ActionResult{Key: key, Token: string(op) + key, Error: "unsupported operator target", Description: "the trainer supports dd, dw, d$, cw, c$, yy, yw, y$, diw, daw, ciw, and caw", Completed: true}
		}
	}

	if unicode.IsDigit(firstRune(key)) && key != "0" {
		e.pendingCount += key
		return ActionResult{Key: key, Token: e.pendingCount, Description: "started a count"}
	}
	if key == "0" && e.pendingCount != "" {
		e.pendingCount += key
		return ActionResult{Key: key, Token: e.pendingCount, Description: "extended the count"}
	}

	switch key {
	case "0":
		e.moveLineStart()
		token := e.composeToken("0", false)
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: "moved to the start of the line", Completed: true}
	case "h":
		count := e.consumeCount()
		for i := 0; i < count; i++ {
			e.moveLeft()
		}
		token := withCount(count, "h")
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: "moved left", Completed: true}
	case "j":
		count := e.consumeCount()
		for i := 0; i < count; i++ {
			e.moveDown()
		}
		token := withCount(count, "j")
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: "moved down", Completed: true}
	case "k":
		count := e.consumeCount()
		for i := 0; i < count; i++ {
			e.moveUp()
		}
		token := withCount(count, "k")
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: "moved up", Completed: true}
	case "l":
		count := e.consumeCount()
		for i := 0; i < count; i++ {
			e.moveRight()
		}
		token := withCount(count, "l")
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: "moved right", Completed: true}
	case "w":
		count := e.consumeCount()
		for i := 0; i < count; i++ {
			e.moveWordForward()
		}
		token := withCount(count, "w")
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: "jumped forward by word", Completed: true}
	case "b":
		count := e.consumeCount()
		for i := 0; i < count; i++ {
			e.moveWordBackward()
		}
		token := withCount(count, "b")
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: "jumped backward by word", Completed: true}
	case "e":
		count := e.consumeCount()
		for i := 0; i < count; i++ {
			e.moveWordEnd()
		}
		token := withCount(count, "e")
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: "moved to the end of the word", Completed: true}
	case "$":
		e.moveLineEnd()
		token := e.composeToken("$", false)
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: "moved to the end of the line", Completed: true}
	case "g":
		e.pendingPrefix = "g"
		return ActionResult{Key: key, Token: "g", Description: "waiting for another g"}
	case "z":
		e.pendingPrefix = "z"
		return ActionResult{Key: key, Token: "z", Description: "waiting for a fold command (zf, zo, zc, za, zR, zM, zj, zk)"}
	case ">":
		e.pendingPrefix = ">"
		return ActionResult{Key: key, Token: ">", Description: "waiting for indent target (>>)"}
	case "<":
		e.pendingPrefix = "<"
		return ActionResult{Key: key, Token: "<", Description: "waiting for outdent target (<<)"}
	case "=":
		e.pendingPrefix = "="
		return ActionResult{Key: key, Token: "=", Description: "waiting for reindent target (==, =ip)"}
	case "G":
		e.pushJump()
		e.moveFileBottom()
		token := e.composeToken("G", false)
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: "jumped to the bottom of the file", Completed: true}
	case "d":
		e.pendingOperator = 'd'
		return ActionResult{Key: key, Token: "d", Description: "delete operator pending"}
	case "c":
		e.pendingOperator = 'c'
		return ActionResult{Key: key, Token: "c", Description: "change operator pending"}
	case "y":
		e.pendingOperator = 'y'
		return ActionResult{Key: key, Token: "y", Description: "yank operator pending"}
	case "x":
		token := e.composeToken("x", true)
		e.beginChange(token)
		deleted, ok := e.deleteChar()
		if ok {
			e.writeTextRegister(deleted, true)
			e.finishChange(token)
			return ActionResult{Key: key, Token: token, Description: "deleted the character under the cursor", Changed: true, Completed: true}
		}
		e.cancelChange()
		return ActionResult{Key: key, Token: token, Description: "nothing to delete", Completed: true}
	case "v":
		e.mode = ModeVisual
		e.visualLine = false
		e.visualBlock = false
		e.visualStart = e.currentCursor()
		e.recordHistory("v")
		return ActionResult{Key: key, Token: "v", Description: "entered visual mode", Completed: true}
	case "V":
		e.mode = ModeVisual
		e.visualLine = true
		e.visualBlock = false
		e.visualStart = e.currentCursor()
		e.recordHistory("V")
		return ActionResult{Key: key, Token: "V", Description: "entered visual-line mode", Completed: true}
	case "ctrl+v":
		e.mode = ModeVisual
		e.visualLine = false
		e.visualBlock = true
		e.visualStart = e.currentCursor()
		e.recordHistory("<C-v>")
		return ActionResult{Key: key, Token: "<C-v>", Description: "entered visual-block mode", Completed: true}
	case "r":
		e.pendingReplace = true
		return ActionResult{Key: key, Token: "r", Description: "waiting for a replacement character"}
	case "R":
		e.mode = ModeReplace
		e.beginChange("R")
		e.changeTokens = []string{"R"}
		e.recordHistory("R")
		return ActionResult{Key: key, Token: "R", Description: "entered replace mode", Completed: true}
	case "i":
		token := e.composeToken("i", true)
		e.beginChange(token)
		e.mode = ModeInsert
		e.changeTokens = []string{"i"}
		return ActionResult{Key: key, Token: token, Description: "entered insert mode", Completed: true}
	case "a":
		token := e.composeToken("a", true)
		e.beginChange(token)
		e.moveRightAppend()
		e.mode = ModeInsert
		e.changeTokens = []string{"a"}
		return ActionResult{Key: key, Token: token, Description: "entered append mode", Completed: true}
	case "A":
		// Append at end of line + insert.
		token := e.composeToken("A", true)
		e.beginChange(token)
		runes := e.lineRunes()
		e.active().Cursor.Col = len(runes)
		e.mode = ModeInsert
		e.changeTokens = []string{"A"}
		return ActionResult{Key: key, Token: token, Description: "entered append-at-end mode", Completed: true}
	case "I":
		// Insert at first non-blank + insert.
		token := e.composeToken("I", true)
		e.beginChange(token)
		runes := e.lineRunes()
		col := 0
		for col < len(runes) && (runes[col] == ' ' || runes[col] == '\t') {
			col++
		}
		e.active().Cursor.Col = col
		e.mode = ModeInsert
		e.changeTokens = []string{"I"}
		return ActionResult{Key: key, Token: token, Description: "entered insert-at-first-nonblank mode", Completed: true}
	case "o":
		token := e.composeToken("o", true)
		e.beginChange(token)
		e.openLineBelow()
		e.mode = ModeInsert
		e.changeTokens = []string{"o"}
		return ActionResult{Key: key, Token: token, Description: "opened a line below", Changed: true, Completed: true}
	case "O":
		token := e.composeToken("O", true)
		e.beginChange(token)
		e.openLineAbove()
		e.mode = ModeInsert
		e.changeTokens = []string{"O"}
		return ActionResult{Key: key, Token: token, Description: "opened a line above", Changed: true, Completed: true}
	case "u":
		if e.undoChange() {
			e.recordHistory("u")
			return ActionResult{Key: key, Token: "u", Description: "undid the last change", Completed: true}
		}
		return ActionResult{Key: key, Token: "u", Description: "nothing to undo", Completed: true}
	case "ctrl+r":
		if e.redoChange() {
			e.recordHistory("<C-r>")
			return ActionResult{Key: key, Token: "<C-r>", Description: "redid the last undone change", Completed: true}
		}
		return ActionResult{Key: key, Token: "<C-r>", Description: "nothing to redo", Completed: true}
	case "ctrl+o":
		if !e.jumpOlder() {
			return ActionResult{Key: key, Token: "<C-o>", Description: "already at the oldest jump", Completed: true}
		}
		e.recordHistory("<C-o>")
		return ActionResult{Key: key, Token: "<C-o>", Description: "jumped backward in jumplist", Completed: true}
	case "ctrl+i", "tab":
		if !e.jumpNewer() {
			return ActionResult{Key: key, Token: "<C-i>", Description: "already at the newest jump", Completed: true}
		}
		e.recordHistory("<C-i>")
		return ActionResult{Key: key, Token: "<C-i>", Description: "jumped forward in jumplist", Completed: true}
	case ":":
		e.mode = ModeCommand
		e.commandBuffer = ""
		return ActionResult{Key: key, Token: ":", Description: "opened the command-line"}
	case "/":
		e.mode = ModeSearch
		e.commandBuffer = ""
		return ActionResult{Key: key, Token: "/", Description: "opened the search prompt"}
	case "n":
		if e.lastSearch == "" {
			return ActionResult{Key: key, Token: "n", Error: "no previous search", Description: "search with / or * first", Completed: true}
		}
		return e.executeSearch(e.lastSearch, e.lastSearchDir)
	case "N":
		if e.lastSearch == "" {
			return ActionResult{Key: key, Token: "N", Error: "no previous search", Description: "search with / or * first", Completed: true}
		}
		return e.executeSearch(e.lastSearch, -e.lastSearchDir)
	case "*":
		word := e.wordUnderCursor()
		if word == "" {
			return ActionResult{Key: key, Token: "*", Error: "no word under cursor", Description: "move onto a word before using *", Completed: true}
		}
		e.options.HLSearch = true
		return e.executeSearch(word, 1)
	case "m":
		e.pendingMarkSet = true
		return ActionResult{Key: key, Token: "m", Description: "waiting for a mark name"}
	case "'", "`":
		e.pendingMarkJump = firstRune(key)
		return ActionResult{Key: key, Token: key, Description: "waiting for a mark name"}
	case "q":
		if e.recordingRegister != 0 {
			// Capital recordings store under the lowercase letter so
			// `@a` plays back the appended sequence.
			target := e.recordingRegister
			if unicode.IsUpper(target) {
				target = unicode.ToLower(target)
			}
			e.macroRegisters[target] = append([]string{}, e.currentMacro...)
			token := "q"
			e.recordingRegister = 0
			e.currentMacro = nil
			e.recordHistory(token)
			return ActionResult{Key: key, Token: token, Description: "stopped macro recording", Completed: true}
		}
		e.pendingMacroQ = true
		return ActionResult{Key: key, Token: "q", Description: "waiting for a macro register"}
	case "\"":
		e.pendingRegister = true
		return ActionResult{Key: key, Token: "\"", Description: "waiting for a register name"}
	case "@":
		e.pendingMacroAt = true
		return ActionResult{Key: key, Token: "@", Description: "waiting for a macro register"}
	case ".":
		if len(e.lastChange) == 0 {
			return ActionResult{Key: key, Token: ".", Description: "there is no repeatable change yet", Completed: true}
		}
		return e.replay(e.lastChange, ".")
	case "ctrl+w":
		e.pendingCtrlW = true
		return ActionResult{Key: key, Token: "<C-w>", Description: "waiting for a window command"}
	case "]":
		e.pendingPrefix = "]"
		return ActionResult{Key: key, Token: "]", Description: "waiting for ] target (s for next misspell, c for next diff hunk)"}
	case "[":
		e.pendingPrefix = "["
		return ActionResult{Key: key, Token: "[", Description: "waiting for [ target (s for previous misspell, c for previous diff hunk)"}
	case "ctrl+]":
		// <C-]> tag jump — the trainer reuses goToDefinition because
		// real tag-stack handling needs an external tags file.
		if !e.goToDefinition() {
			return ActionResult{Key: key, Token: "<C-]>", Error: "tag not found", Description: "no obvious tag match for the symbol under cursor", Completed: true}
		}
		e.recordHistory("<C-]>")
		return ActionResult{Key: key, Token: "<C-]>", Description: "jumped via tag", Completed: true}
	case "ctrl+t":
		// <C-T> pops the tag stack — we map it to jumpOlder, which
		// roughly matches the user-facing semantic ("go back").
		if !e.jumpOlder() {
			return ActionResult{Key: key, Token: "<C-t>", Description: "tag stack empty", Completed: true}
		}
		e.recordHistory("<C-t>")
		return ActionResult{Key: key, Token: "<C-t>", Description: "popped tag stack", Completed: true}
	case "ctrl+a":
		if !e.adjustNumberUnderCursor(+1) {
			return ActionResult{Key: key, Token: "<C-a>", Description: "no number to increment", Completed: true}
		}
		token := e.composeToken("<C-a>", true)
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: "incremented number under cursor", Changed: true, Completed: true}
	case "ctrl+x":
		if !e.adjustNumberUnderCursor(-1) {
			return ActionResult{Key: key, Token: "<C-x>", Description: "no number to decrement", Completed: true}
		}
		token := e.composeToken("<C-x>", true)
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: "decremented number under cursor", Changed: true, Completed: true}
	case "J":
		// Join the current line with the next, separated by a space.
		token := e.composeToken("J", true)
		e.beginChange(token)
		if !e.joinLines(true) {
			e.cancelChange()
			return ActionResult{Key: key, Token: token, Description: "no line below to join", Completed: true}
		}
		e.finishChange(token)
		return ActionResult{Key: key, Token: token, Description: "joined with next line", Changed: true, Completed: true}
	case "~":
		// Toggle case of the char under the cursor and advance.
		if !e.toggleCaseUnderCursor() {
			return ActionResult{Key: key, Token: "~", Description: "no character to toggle", Completed: true}
		}
		token := e.composeToken("~", true)
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: "toggled case", Changed: true, Completed: true}
	case "ctrl+^", "ctrl+6":
		// <C-^> swaps to the alternate buffer ("the file you were just in").
		if e.alternateBuffer < 0 || e.alternateBuffer >= len(e.buffers) || e.alternateBuffer == e.active().Buffer {
			return ActionResult{Key: key, Token: "<C-^>", Description: "no alternate buffer to switch to", Completed: true}
		}
		prev := e.active().Buffer
		e.active().Buffer = e.alternateBuffer
		e.active().Cursor = e.clampCursor(e.alternateBuffer, Position{})
		e.alternateBuffer = prev
		e.recordHistory("<C-^>")
		return ActionResult{Key: key, Token: "<C-^>", Description: "swapped to alternate buffer", Completed: true}
	case "p":
		if !e.pasteAfter() {
			return ActionResult{Key: key, Token: "p", Description: "nothing to paste", Completed: true}
		}
		token := e.composeToken("p", true)
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: "pasted yanked text", Changed: true, Completed: true}
	case "enter":
		// In Vim Normal mode, <CR> moves down one line (to the first non-
		// blank, but moveDown is close enough for the trainer's purposes).
		count := e.consumeCount()
		for i := 0; i < count; i++ {
			e.moveDown()
		}
		token := withCount(count, "<CR>")
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: "moved down one line", Completed: true}
	case "esc":
		e.pendingCount = ""
		e.pendingPrefix = ""
		e.pendingOperator = 0
		e.pendingTextObject = 0
		e.pendingRegister = false
		e.activeRegister = 0
		e.pendingMarkSet = false
		e.pendingMarkJump = 0
		e.pendingCtrlW = false
		e.pendingMacroAt = false
		e.pendingMacroQ = false
		return ActionResult{Key: key, Token: "esc", Description: "cleared pending command state", Completed: true}
	default:
		return ActionResult{Key: key, Token: key, Error: "unsupported normal-mode key", Description: "that key is not part of the trainer's supported Vim surface yet", Completed: true}
	}
}

func (e *Editor) executeCommand(cmd string) ActionResult {
	if cmd == "" {
		return ActionResult{Key: "enter", Token: ":", Description: "empty command", Completed: true}
	}
	token := ":" + cmd
	e.recordHistory(token)
	// Range-aware commands (:s, :g, :v) must be detected before splitting on
	// whitespace because their syntax is :[range]s/pat/rep/flags.
	if sub, ok := parseSubstituteCommand(cmd); ok {
		return e.applySubstitute(sub, token)
	}
	if g, ok := parseGlobalCommand(cmd); ok {
		return e.applyGlobal(g, token)
	}
	if n, ok := parseNormalCommand(cmd); ok {
		return e.applyNormal(n, token)
	}
	if ex, ok := parseExecuteCommand(cmd); ok {
		return e.applyExecute(ex, token)
	}
	// `:!cmd` runs a shell command. The trainer doesn't fork a real
	// subprocess — that would surprise learners and create a security
	// surface. We fake the output deterministically so lessons remain
	// reproducible.
	if strings.HasPrefix(cmd, "!") {
		shell := strings.TrimSpace(cmd[1:])
		if shell == "!" {
			// :!! — re-run the most recent shell command.
			shell = strings.TrimPrefix(e.lastShellCmd, "!")
			if shell == "" {
				return ActionResult{Key: "enter", Token: token, Error: "no previous :! command", Description: "run :!cmd at least once before :!!", Completed: true}
			}
		} else {
			e.lastShellCmd = shell
		}
		out := e.synthesizeShellOutput(shell)
		e.lastEcho = strings.Join(out, "\n")
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf(":!%s → %d line(s) of (simulated) output", shell, len(out)), Completed: true}
	}
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return ActionResult{Key: "enter", Token: token, Description: "empty command", Completed: true}
	}

	switch fields[0] {
	case "set":
		if len(fields) == 1 {
			return ActionResult{Key: "enter", Token: token, Error: "missing option", Description: "use :set followed by an option like number", Completed: true}
		}
		for _, field := range fields[1:] {
			switch field {
			case "number", "nu":
				e.options.Number = true
			case "nonumber", "nonu":
				e.options.Number = false
			case "relativenumber", "rnu":
				e.options.RelativeNumber = true
			case "norelativenumber", "nornu":
				e.options.RelativeNumber = false
			case "hlsearch", "hls":
				e.options.HLSearch = true
			case "nohlsearch", "nohls":
				e.options.HLSearch = false
			case "ignorecase", "ic":
				e.options.IgnoreCase = true
			case "noignorecase", "noic":
				e.options.IgnoreCase = false
			case "smartcase", "scs":
				e.options.SmartCase = true
			case "nosmartcase", "noscs":
				e.options.SmartCase = false
			case "incsearch", "is":
				e.options.IncSearch = true
			case "noincsearch", "nois":
				e.options.IncSearch = false
			case "wrap":
				e.options.Wrap = true
			case "nowrap":
				e.options.Wrap = false
			case "expandtab", "et":
				e.options.ExpandTab = true
			case "noexpandtab", "noet":
				e.options.ExpandTab = false
			case "undofile":
				e.options.UndoFile = true
			case "noundofile":
				e.options.UndoFile = false
			case "splitbelow", "sb":
				e.options.SplitBelow = true
			case "nosplitbelow", "nosb":
				e.options.SplitBelow = false
			case "splitright", "spr":
				e.options.SplitRight = true
			case "nosplitright", "nospr":
				e.options.SplitRight = false
			case "termguicolors", "tgc":
				e.options.TermGUIColors = true
			case "notermguicolors", "notgc":
				e.options.TermGUIColors = false
			case "spell":
				e.options.Spell = true
			case "nospell":
				e.options.Spell = false
			case "cursorline", "cul":
				e.options.CursorLine = true
			case "nocursorline", "nocul":
				e.options.CursorLine = false
			case "list":
				e.options.List = true
			case "nolist":
				e.options.List = false
			case "wildmenu":
				e.options.WildMenu = true
			case "nowildmenu":
				e.options.WildMenu = false
			case "lazyredraw", "lz":
				e.options.LazyRedraw = true
			case "nolazyredraw", "nolz":
				e.options.LazyRedraw = false
			case "backup", "bk":
				e.options.BackupEnabled = true
			case "nobackup", "nobk":
				e.options.BackupEnabled = false
			case "swapfile", "swf":
				e.options.SwapFile = true
			case "noswapfile", "noswf":
				e.options.SwapFile = false
			case "autoread", "ar":
				e.options.AutoRead = true
			case "noautoread", "noar":
				e.options.AutoRead = false
			default:
				// Numeric / string options of the form key=value.
				if eq := strings.Index(field, "="); eq > 0 {
					key := field[:eq]
					val := field[eq+1:]
					switch key {
					case "statusline", "stl":
						e.statusline = val
						continue
					case "tabline", "tbl":
						e.tabline = val
						continue
					case "winbar", "wbr":
						e.winbar = val
						continue
					case "conceallevel", "cole":
						if n, err := strconv.Atoi(val); err == nil {
							e.concealLevel = n
							continue
						}
					case "tabstop", "ts":
						if n, err := strconv.Atoi(val); err == nil {
							e.options.TabStop = n
							continue
						}
					case "shiftwidth", "sw":
						if n, err := strconv.Atoi(val); err == nil {
							e.options.ShiftWidth = n
							continue
						}
					case "scrolloff", "so":
						if n, err := strconv.Atoi(val); err == nil {
							e.options.ScrollOff = n
							continue
						}
					case "updatetime", "ut":
						if n, err := strconv.Atoi(val); err == nil {
							e.options.UpdateTime = n
							continue
						}
					case "timeoutlen", "tm":
						if n, err := strconv.Atoi(val); err == nil {
							e.options.TimeoutLen = n
							continue
						}
					case "clipboard", "cb":
						e.options.Clipboard = val
						continue
					case "mouse":
						e.options.Mouse = val
						continue
					case "spelllang", "spl":
						e.options.SpellLang = val
						continue
					case "foldmethod", "fdm":
						e.options.FoldMethod = val
						continue
					case "listchars", "lcs":
						e.options.ListChars = val
						continue
					case "completeopt", "cot":
						e.options.CompleteOpt = val
						continue
					case "wildmode", "wim":
						e.options.WildMode = val
						continue
					case "colorscheme":
						e.options.ColorScheme = val
						continue
					}
				}
				if strings.HasPrefix(field, "makeprg=") {
					e.makePrg = strings.TrimPrefix(field, "makeprg=")
					continue
				}
				if strings.HasPrefix(field, "errorformat=") || strings.HasPrefix(field, "efm=") {
					if strings.HasPrefix(field, "errorformat=") {
						e.errorFormat = strings.TrimPrefix(field, "errorformat=")
					} else {
						e.errorFormat = strings.TrimPrefix(field, "efm=")
					}
					continue
				}
				if strings.HasPrefix(field, "textwidth=") || strings.HasPrefix(field, "tw=") {
					prefix := "textwidth="
					if strings.HasPrefix(field, "tw=") {
						prefix = "tw="
					}
					n, err := strconv.Atoi(strings.TrimPrefix(field, prefix))
					if err != nil || n < 0 {
						return ActionResult{Key: "enter", Token: token, Error: "invalid textwidth", Description: "use :set textwidth=N (N >= 0)", Completed: true}
					}
					e.textWidth = n
					continue
				}
				return ActionResult{Key: "enter", Token: token, Error: "unsupported option", Description: "supported options include number, relativenumber, hlsearch, ignorecase, smartcase, incsearch, wrap, and textwidth", Completed: true}
			}
		}
		return ActionResult{Key: "enter", Token: token, Description: "updated editor options", Completed: true}
	case "noh", "nohlsearch":
		e.options.HLSearch = false
		return ActionResult{Key: "enter", Token: token, Description: "cleared search highlighting", Completed: true}
	case "w":
		return ActionResult{Key: "enter", Token: token, Description: "wrote the buffer (simulated)", Completed: true}
	case "q", "quit", "exit", "x":
		e.quitRequested = true
		return ActionResult{Key: "enter", Token: token, Description: "quit requested", Completed: true}
	case "q!", "quit!":
		e.quitRequested = true
		return ActionResult{Key: "enter", Token: token, Description: "force-quit requested", Completed: true}
	case "wq", "wq!", "xa", "xall":
		e.quitRequested = true
		return ActionResult{Key: "enter", Token: token, Description: "write and quit requested", Completed: true}
	case "wa", "wall":
		return ActionResult{Key: "enter", Token: token, Description: "wrote all buffers (simulated)", Completed: true}
	case "qa", "qall":
		e.quitRequested = true
		return ActionResult{Key: "enter", Token: token, Description: "quit all requested", Completed: true}
	case "qa!", "qall!":
		e.quitRequested = true
		return ActionResult{Key: "enter", Token: token, Description: "force-quit all requested", Completed: true}
	case "recover":
		return ActionResult{Key: "enter", Token: token, Description: "recovery flow opened (simulated)", Completed: true}
	case "e":
		if len(fields) < 2 {
			return ActionResult{Key: "enter", Token: token, Error: "missing buffer name", Description: "use :e followed by a file name", Completed: true}
		}
		e.openOrSwitchBuffer(fields[1])
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("opened %s", fields[1]), Completed: true}
	case "Ex", "Explore", "Sexplore", "Lexplore":
		e.explorerOpen = true
		e.mode = ModeExplorer
		if len(fields) > 1 {
			e.explorerPath = fields[1]
		}
		return ActionResult{Key: "enter", Token: token, Description: "opened the file explorer", Completed: true}
	case "split", "vsplit", "new", "vnew":
		e.openSplit()
		return ActionResult{Key: "enter", Token: token, Description: "opened a split window", Completed: true}
	case "tabnew", "tabedit":
		e.openNewTab()
		if len(fields) > 1 {
			e.openOrSwitchBuffer(fields[1])
		}
		return ActionResult{Key: "enter", Token: token, Description: "opened a new tab page", Completed: true}
	case "tabnext", "tabn":
		if len(e.tabs) <= 1 {
			return ActionResult{Key: "enter", Token: token, Description: "only one tab page", Completed: true}
		}
		e.switchTab(e.activeTab + 1)
		return ActionResult{Key: "enter", Token: token, Description: "moved to the next tab", Completed: true}
	case "tabprevious", "tabprev", "tabp", "tabNext":
		if len(e.tabs) <= 1 {
			return ActionResult{Key: "enter", Token: token, Description: "only one tab page", Completed: true}
		}
		e.switchTab(e.activeTab - 1)
		return ActionResult{Key: "enter", Token: token, Description: "moved to the previous tab", Completed: true}
	case "tabfirst", "tabrewind":
		e.switchTab(0)
		return ActionResult{Key: "enter", Token: token, Description: "moved to the first tab", Completed: true}
	case "tablast":
		e.switchTab(len(e.tabs) - 1)
		return ActionResult{Key: "enter", Token: token, Description: "moved to the last tab", Completed: true}
	case "tabclose", "tabc":
		if !e.closeActiveTab() {
			return ActionResult{Key: "enter", Token: token, Error: "last tab page", Description: "cannot close the last tab; use :q to quit", Completed: true}
		}
		return ActionResult{Key: "enter", Token: token, Description: "closed the current tab", Completed: true}
	case "tabonly":
		if len(e.tabs) <= 1 {
			return ActionResult{Key: "enter", Token: token, Description: "already the only tab", Completed: true}
		}
		e.snapshotActiveTab()
		current := e.tabs[e.activeTab]
		e.tabs = []tabSnapshot{current}
		e.activeTab = 0
		return ActionResult{Key: "enter", Token: token, Description: "kept only the current tab", Completed: true}
	case "tabs":
		e.lastEcho = fmt.Sprintf("%d tab page(s); active = %d", len(e.tabs), e.activeTab+1)
		return ActionResult{Key: "enter", Token: token, Description: e.lastEcho, Completed: true}
	case "sort":
		// :sort [u][n] [/pattern/] — sort lines (whole buffer for now).
		flags := ""
		if len(fields) > 1 {
			flags = fields[1]
		}
		e.sortLines(0, len(e.currentBuffer().Lines)-1, flags)
		return ActionResult{Key: "enter", Token: token, Description: "sorted buffer", Changed: true, Completed: true}
	case "retab":
		// :retab — replace tabs with config.IndentUnit and vice-versa.
		e.retabBuffer()
		return ActionResult{Key: "enter", Token: token, Description: "retabbed buffer", Changed: true, Completed: true}
	case "args":
		if len(fields) > 1 {
			e.args = append([]string{}, fields[1:]...)
			e.argIndex = 0
			return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("argument list: %d file(s)", len(e.args)), Completed: true}
		}
		if len(e.args) == 0 {
			return ActionResult{Key: "enter", Token: token, Description: "argument list is empty", Completed: true}
		}
		e.lastEcho = strings.Join(e.args, " ")
		return ActionResult{Key: "enter", Token: token, Description: e.lastEcho, Completed: true}
	case "next", "n":
		if len(e.args) == 0 {
			return ActionResult{Key: "enter", Token: token, Error: "argument list empty", Description: "use :args to populate", Completed: true}
		}
		e.argIndex = (e.argIndex + 1) % len(e.args)
		e.openOrSwitchBuffer(e.args[e.argIndex])
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("opened %s", e.args[e.argIndex]), Completed: true}
	case "prev", "N", "previous":
		if len(e.args) == 0 {
			return ActionResult{Key: "enter", Token: token, Error: "argument list empty", Description: "use :args to populate", Completed: true}
		}
		e.argIndex--
		if e.argIndex < 0 {
			e.argIndex = len(e.args) - 1
		}
		e.openOrSwitchBuffer(e.args[e.argIndex])
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("opened %s", e.args[e.argIndex]), Completed: true}
	case "argdo":
		if len(e.args) == 0 {
			return ActionResult{Key: "enter", Token: token, Error: "argument list empty", Description: "use :args to populate first", Completed: true}
		}
		body := strings.TrimSpace(strings.TrimPrefix(cmd, "argdo"))
		if body == "" {
			return ActionResult{Key: "enter", Token: token, Error: "missing command", Description: "use :argdo <cmd>", Completed: true}
		}
		ran := 0
		for _, name := range e.args {
			e.openOrSwitchBuffer(name)
			_ = e.executeCommand(body)
			ran++
		}
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("argdo ran across %d file(s)", ran), Changed: true, Completed: true}
	case "bufdo":
		body := strings.TrimSpace(strings.TrimPrefix(cmd, "bufdo"))
		if body == "" {
			return ActionResult{Key: "enter", Token: token, Error: "missing command", Description: "use :bufdo <cmd>", Completed: true}
		}
		ran := 0
		for i := range e.buffers {
			e.active().Buffer = i
			e.active().Cursor = e.clampCursor(i, Position{})
			_ = e.executeCommand(body)
			ran++
		}
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("bufdo ran across %d buffer(s)", ran), Changed: true, Completed: true}
	case "resize", "res":
		// Sizes are not modeled; accept the command for muscle-memory.
		return ActionResult{Key: "enter", Token: token, Description: "accepted resize (sizes are not modeled in the trainer)", Completed: true}
	case "vertical":
		return ActionResult{Key: "enter", Token: token, Description: "accepted vertical command (sizes are not modeled)", Completed: true}
	case "close":
		if len(e.windows) <= 1 {
			return ActionResult{Key: "enter", Token: token, Description: "only one window — nothing to close", Completed: true}
		}
		e.windows = append(e.windows[:e.activeWindow], e.windows[e.activeWindow+1:]...)
		if e.activeWindow >= len(e.windows) {
			e.activeWindow = len(e.windows) - 1
		}
		return ActionResult{Key: "enter", Token: token, Description: "closed the current window", Completed: true}
	case "only":
		if len(e.windows) <= 1 {
			return ActionResult{Key: "enter", Token: token, Description: "already the only window", Completed: true}
		}
		active := e.windows[e.activeWindow]
		e.windows = []Window{active}
		e.activeWindow = 0
		return ActionResult{Key: "enter", Token: token, Description: "kept only the current window", Completed: true}
	case "bn", "bnext":
		e.bufferNext()
		return ActionResult{Key: "enter", Token: token, Description: "switched to the next buffer", Completed: true}
	case "bp", "bprevious":
		e.bufferPrev()
		return ActionResult{Key: "enter", Token: token, Description: "switched to the previous buffer", Completed: true}
	case "copen":
		e.quickfixOpen = true
		return ActionResult{Key: "enter", Token: token, Description: "opened quickfix list", Completed: true}
	case "cclose":
		e.quickfixOpen = false
		return ActionResult{Key: "enter", Token: token, Description: "closed quickfix list", Completed: true}
	case "cn", "cnext":
		if !e.quickfixMove(1) {
			return ActionResult{Key: "enter", Token: token, Error: "quickfix is empty", Description: "run :vimgrep first to populate quickfix entries", Completed: true}
		}
		return ActionResult{Key: "enter", Token: token, Description: "jumped to next quickfix item", Completed: true}
	case "cp", "cprev":
		if !e.quickfixMove(-1) {
			return ActionResult{Key: "enter", Token: token, Error: "quickfix is empty", Description: "run :vimgrep first to populate quickfix entries", Completed: true}
		}
		return ActionResult{Key: "enter", Token: token, Description: "jumped to previous quickfix item", Completed: true}
	case "cfirst":
		if !e.quickfixSet(0) {
			return ActionResult{Key: "enter", Token: token, Error: "quickfix is empty", Description: "run :vimgrep first to populate quickfix entries", Completed: true}
		}
		return ActionResult{Key: "enter", Token: token, Description: "jumped to first quickfix item", Completed: true}
	case "clast":
		if !e.quickfixSet(len(e.quickfix) - 1) {
			return ActionResult{Key: "enter", Token: token, Error: "quickfix is empty", Description: "run :vimgrep first to populate quickfix entries", Completed: true}
		}
		return ActionResult{Key: "enter", Token: token, Description: "jumped to last quickfix item", Completed: true}
	case "cnewer":
		if !e.quickfixHistoryStep(+1) {
			return ActionResult{Key: "enter", Token: token, Error: "no newer quickfix list", Description: "you're already on the newest list", Completed: true}
		}
		return ActionResult{Key: "enter", Token: token, Description: "switched to newer quickfix list", Completed: true}
	case "colder":
		if !e.quickfixHistoryStep(-1) {
			return ActionResult{Key: "enter", Token: token, Error: "no older quickfix list", Description: "you're already on the oldest list", Completed: true}
		}
		return ActionResult{Key: "enter", Token: token, Description: "switched to older quickfix list", Completed: true}
	case "chistory":
		e.lastEcho = fmt.Sprintf("quickfix history: %d list(s)", len(e.quickfixLists)+1)
		return ActionResult{Key: "enter", Token: token, Description: e.lastEcho, Completed: true}
	case "cdo":
		body := strings.TrimSpace(strings.TrimPrefix(cmd, "cdo"))
		if body == "" {
			return ActionResult{Key: "enter", Token: token, Error: "missing command", Description: "use :cdo <ex command> — runs the command on every quickfix entry", Completed: true}
		}
		return e.runCDO(body, token)
	case "cfdo":
		body := strings.TrimSpace(strings.TrimPrefix(cmd, "cfdo"))
		if body == "" {
			return ActionResult{Key: "enter", Token: token, Error: "missing command", Description: "use :cfdo <ex command> — runs the command once per matched file", Completed: true}
		}
		return e.runCFDO(body, token)
	case "vimgrep":
		pattern := parseVimGrepPattern(cmd)
		if pattern == "" {
			return ActionResult{Key: "enter", Token: token, Error: "missing grep pattern", Description: "use :vimgrep /pattern/ % to populate quickfix", Completed: true}
		}
		count := e.populateQuickfix(pattern)
		if count == 0 {
			return ActionResult{Key: "enter", Token: token, Error: "no matches", Description: "vimgrep found no matches for that pattern", Completed: true}
		}
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("vimgrep found %d quickfix matches", count), Completed: true}
	case "help":
		topic := ""
		if len(fields) > 1 {
			topic = fields[1]
		}
		if topic == "" {
			topic = "index"
		}
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("opened help for %s", topic), Completed: true}
	case "source":
		target := ""
		if len(fields) > 1 {
			target = fields[1]
		}
		if target == "" {
			return ActionResult{Key: "enter", Token: token, Error: "missing source file", Description: "use :source followed by a path", Completed: true}
		}
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("sourced %s (simulated)", target), Completed: true}
	case "checkhealth":
		return ActionResult{Key: "enter", Token: token, Description: "ran :checkhealth (simulated)", Completed: true}
	case "LspInfo", "LspStart", "LspStop", "LspRestart":
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("ran %s (simulated)", fields[0]), Completed: true}
	case "Lazy", "PackerSync", "Mason", "TSInstall", "TSUpdate", "TSUpdateSync":
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("ran %s workflow (simulated)", fields[0]), Completed: true}
	case "Telescope":
		if len(fields) < 2 {
			return ActionResult{Key: "enter", Token: token, Error: "missing picker", Description: "use :Telescope find_files, live_grep, or buffers", Completed: true}
		}
		switch fields[1] {
		case "find_files":
			e.explorerOpen = true
			e.mode = ModeExplorer
			return ActionResult{Key: "enter", Token: token, Description: "opened Telescope find_files picker (simulated)", Completed: true}
		case "live_grep":
			return ActionResult{Key: "enter", Token: token, Description: "opened Telescope live_grep picker (simulated)", Completed: true}
		case "buffers":
			return ActionResult{Key: "enter", Token: token, Description: "opened Telescope buffers picker (simulated)", Completed: true}
		default:
			return ActionResult{Key: "enter", Token: token, Error: "unsupported Telescope picker", Description: "supported pickers are find_files, live_grep, and buffers", Completed: true}
		}
	case "terminal", "term":
		e.openTerminalBuffer()
		return ActionResult{Key: "enter", Token: token, Description: "opened terminal buffer", Completed: true}
	case "lua":
		script := strings.TrimSpace(strings.TrimPrefix(cmd, "lua"))
		out, err := e.applyLua(script)
		if err != nil {
			return ActionResult{Key: "enter", Token: token, Error: "invalid lua snippet", Description: err.Error(), Completed: true}
		}
		if out != "" {
			e.lastEcho = out
			return ActionResult{Key: "enter", Token: token, Description: "lua output: " + out, Completed: true}
		}
		return ActionResult{Key: "enter", Token: token, Description: "executed lua snippet", Completed: true}
	case "let":
		name, value, ok := parseLetCommand(cmd)
		if !ok {
			return ActionResult{Key: "enter", Token: token, Error: "invalid let expression", Description: "use :let name = value", Completed: true}
		}
		e.variables[name] = value
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("set %s = %s", name, value), Completed: true}
	case "echo", "echom":
		expr := strings.TrimSpace(strings.TrimPrefix(cmd, fields[0]))
		out, ok := e.evaluateExpression(expr)
		if !ok {
			return ActionResult{Key: "enter", Token: token, Error: "cannot evaluate expression", Description: "echo supports quoted text and previously defined variables", Completed: true}
		}
		e.lastEcho = out
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("echo: %s", out), Completed: true}
	case "nnoremap", "noremap", "nmap", "map":
		lhs, rhs, ok := parseMapCommand(cmd)
		if !ok {
			return ActionResult{Key: "enter", Token: token, Error: "invalid map syntax", Description: "use :nnoremap <lhs> <rhs>", Completed: true}
		}
		e.normalMappings[lhs] = rhs
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("mapped %s -> %s", lhs, rhs), Completed: true}
	case "inoremap", "imap":
		lhs, rhs, ok := parseMapCommand(cmd)
		if !ok {
			return ActionResult{Key: "enter", Token: token, Error: "invalid map syntax", Description: "use :inoremap <lhs> <rhs>", Completed: true}
		}
		if e.insertMappings == nil {
			e.insertMappings = map[string]string{}
		}
		e.insertMappings[lhs] = rhs
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("(insert) mapped %s -> %s", lhs, rhs), Completed: true}
	case "vnoremap", "vmap", "xnoremap", "xmap":
		lhs, rhs, ok := parseMapCommand(cmd)
		if !ok {
			return ActionResult{Key: "enter", Token: token, Error: "invalid map syntax", Description: "use :vnoremap <lhs> <rhs>", Completed: true}
		}
		if e.visualMappings == nil {
			e.visualMappings = map[string]string{}
		}
		e.visualMappings[lhs] = rhs
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("(visual) mapped %s -> %s", lhs, rhs), Completed: true}
	case "cnoremap", "cmap":
		lhs, rhs, ok := parseMapCommand(cmd)
		if !ok {
			return ActionResult{Key: "enter", Token: token, Error: "invalid map syntax", Description: "use :cnoremap <lhs> <rhs>", Completed: true}
		}
		if e.cmdlineMappings == nil {
			e.cmdlineMappings = map[string]string{}
		}
		e.cmdlineMappings[lhs] = rhs
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("(cmdline) mapped %s -> %s", lhs, rhs), Completed: true}
	case "tnoremap", "tmap":
		lhs, rhs, ok := parseMapCommand(cmd)
		if !ok {
			return ActionResult{Key: "enter", Token: token, Error: "invalid map syntax", Description: "use :tnoremap <lhs> <rhs>", Completed: true}
		}
		if e.terminalMappings == nil {
			e.terminalMappings = map[string]string{}
		}
		e.terminalMappings[lhs] = rhs
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("(terminal) mapped %s -> %s", lhs, rhs), Completed: true}
	case "onoremap", "omap":
		lhs, rhs, ok := parseMapCommand(cmd)
		if !ok {
			return ActionResult{Key: "enter", Token: token, Error: "invalid map syntax", Description: "use :onoremap <lhs> <rhs>", Completed: true}
		}
		if e.operatorMappings == nil {
			e.operatorMappings = map[string]string{}
		}
		e.operatorMappings[lhs] = rhs
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("(operator-pending) mapped %s -> %s", lhs, rhs), Completed: true}
	case "nunmap", "unmap":
		if len(fields) < 2 {
			return ActionResult{Key: "enter", Token: token, Error: "missing lhs", Description: "use :unmap <lhs>", Completed: true}
		}
		delete(e.normalMappings, fields[1])
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("removed mapping for %s", fields[1]), Completed: true}
	case "iunmap":
		if len(fields) < 2 {
			return ActionResult{Key: "enter", Token: token, Error: "missing lhs", Description: "use :iunmap <lhs>", Completed: true}
		}
		delete(e.insertMappings, fields[1])
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("removed insert mapping for %s", fields[1]), Completed: true}
	case "command", "command!":
		// :command! Name <body> — register a user command. The trainer
		// stores it; full execution would require ex-mode evaluation
		// which is out of scope. Re-running the command name treats it
		// as a no-op success so curriculum can demonstrate registration.
		spec := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(cmd, fields[0]), " "))
		if spec == "" {
			return ActionResult{Key: "enter", Token: token, Error: "missing definition", Description: "use :command! Name body", Completed: true}
		}
		if e.userCommands == nil {
			e.userCommands = map[string]string{}
		}
		// Strip any -nargs / -range / -bang flags (keywords starting with -).
		parts := strings.Fields(spec)
		i := 0
		for i < len(parts) && strings.HasPrefix(parts[i], "-") {
			i++
		}
		if i >= len(parts) {
			return ActionResult{Key: "enter", Token: token, Error: "missing command name", Description: "use :command! Name body", Completed: true}
		}
		name := parts[i]
		body := strings.TrimSpace(strings.Join(parts[i+1:], " "))
		e.userCommands[name] = body
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("registered :%s -> %s", name, body), Completed: true}
	case "registers", "reg":
		e.lastEcho = e.registerDump()
		return ActionResult{Key: "enter", Token: token, Description: "listed register contents", Completed: true}
	case "marks":
		e.lastEcho = e.marksDump()
		return ActionResult{Key: "enter", Token: token, Description: "listed marks", Completed: true}
	case "delmarks", "delm":
		// :delmarks a-c | :delmarks! to drop everything.
		if len(fields) > 1 && (fields[1] == "!" || fields[0] == "delmarks!" || fields[0] == "delm!") {
			e.marks = map[rune]Position{}
			e.markBuffers = map[rune]int{}
			return ActionResult{Key: "enter", Token: token, Description: "cleared every mark", Completed: true}
		}
		if len(fields) < 2 {
			return ActionResult{Key: "enter", Token: token, Error: "missing mark list", Description: "use :delmarks a or :delmarks a-c or :delmarks!", Completed: true}
		}
		dropped := 0
		for _, ch := range expandMarkList(strings.Join(fields[1:], "")) {
			if _, ok := e.marks[ch]; ok {
				delete(e.marks, ch)
				delete(e.markBuffers, ch)
				dropped++
			}
		}
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("dropped %d mark(s)", dropped), Completed: true}
	case "delmarks!", "delm!":
		e.marks = map[rune]Position{}
		e.markBuffers = map[rune]int{}
		return ActionResult{Key: "enter", Token: token, Description: "cleared every mark", Completed: true}
	case "jumps":
		e.lastEcho = fmt.Sprintf("jumps: %d items", len(e.jumpList))
		return ActionResult{Key: "enter", Token: token, Description: e.lastEcho, Completed: true}
	case "changes":
		e.lastEcho = fmt.Sprintf("changes: %d items", len(e.changeList))
		return ActionResult{Key: "enter", Token: token, Description: e.lastEcho, Completed: true}
	case "messages":
		return ActionResult{Key: "enter", Token: token, Description: "message history opened (simulated)", Completed: true}
	case "scriptnames":
		return ActionResult{Key: "enter", Token: token, Description: "listed sourced scripts (simulated)", Completed: true}
	case "verbose":
		return ActionResult{Key: "enter", Token: token, Description: "ran verbose command (simulated)", Completed: true}
	case "profile":
		if len(fields) > 1 && fields[1] == "start" {
			e.profileActive = true
			return ActionResult{Key: "enter", Token: token, Description: "started profiling (simulated)", Completed: true}
		}
		if len(fields) > 1 && (fields[1] == "stop" || fields[1] == "pause") {
			e.profileActive = false
			return ActionResult{Key: "enter", Token: token, Description: "stopped profiling (simulated)", Completed: true}
		}
		return ActionResult{Key: "enter", Token: token, Description: "profile command accepted (simulated)", Completed: true}
	case "checktime":
		return ActionResult{Key: "enter", Token: token, Description: "checked file timestamps (simulated)", Completed: true}
	case "make":
		// :make [args] runs the configured makeprg and parses its output
		// into the quickfix list. The trainer doesn't fork a real
		// subprocess; it produces a plausible fake-but-faithful set of
		// errorformat-shaped lines so curriculum can teach the workflow.
		count := e.populateQuickfix("ERROR")
		if count == 0 {
			// Seed a synthetic error so the workflow still has something
			// to navigate. Lines look like "main.go:10:1: error: …".
			cur := e.active().Buffer
			if cur < len(e.buffers) {
				e.quickfix = []QuickfixItem{{Buffer: cur, Pos: Position{Row: 0, Col: 0}, Text: e.buffers[cur].Name + ":1:1: error: simulated build failure"}}
				e.quickfixIndex = 0
				count = 1
			}
		}
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf(":make populated quickfix with %d entry/entries", count), Completed: true}
	case "colorscheme", "colo":
		if len(fields) < 2 {
			return ActionResult{Key: "enter", Token: token, Error: "missing colorscheme name", Description: "use :colorscheme habamax|tokyonight|...", Completed: true}
		}
		e.options.ColorScheme = fields[1]
		return ActionResult{Key: "enter", Token: token, Description: "colorscheme set to " + fields[1], Completed: true}
	case "autocmd", "au":
		// :autocmd Event Pattern Cmd  — register
		// :autocmd!                   — clear (or :autocmd! Event)
		// :autocmd Event              — list
		if len(fields) >= 2 && (fields[0] == "autocmd!" || fields[0] == "au!") {
			e.autocmds = nil
			return ActionResult{Key: "enter", Token: token, Description: "cleared all autocmds", Completed: true}
		}
		if len(fields) < 4 {
			if len(fields) == 1 {
				e.lastEcho = fmt.Sprintf("%d autocmd(s) registered", len(e.autocmds))
				return ActionResult{Key: "enter", Token: token, Description: e.lastEcho, Completed: true}
			}
			return ActionResult{Key: "enter", Token: token, Error: "incomplete autocmd", Description: "use :autocmd Event Pattern Cmd", Completed: true}
		}
		body := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(cmd, fields[0]), " "))
		// Re-split body into Event, Pattern, then the rest as Cmd.
		parts := strings.SplitN(body, " ", 3)
		if len(parts) < 3 {
			return ActionResult{Key: "enter", Token: token, Error: "incomplete autocmd", Description: "use :autocmd Event Pattern Cmd", Completed: true}
		}
		e.autocmds = append(e.autocmds, autocmdEntry{
			Group:   e.currentAuGroup,
			Event:   parts[0],
			Pattern: parts[1],
			Command: parts[2],
		})
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("registered autocmd %s %s -> %s", parts[0], parts[1], parts[2]), Completed: true}
	case "autocmd!", "au!":
		e.autocmds = nil
		return ActionResult{Key: "enter", Token: token, Description: "cleared all autocmds", Completed: true}
	case "augroup":
		if len(fields) < 2 {
			return ActionResult{Key: "enter", Token: token, Error: "missing group name", Description: "use :augroup MyGroup or :augroup END", Completed: true}
		}
		if fields[1] == "END" || fields[1] == "end" {
			e.currentAuGroup = ""
			return ActionResult{Key: "enter", Token: token, Description: "ended augroup", Completed: true}
		}
		e.currentAuGroup = fields[1]
		return ActionResult{Key: "enter", Token: token, Description: "entered augroup " + fields[1], Completed: true}
	case "doautocmd":
		if len(fields) < 2 {
			return ActionResult{Key: "enter", Token: token, Error: "missing event", Description: "use :doautocmd Event", Completed: true}
		}
		// Match-and-acknowledge; we don't actually run the stored Cmd
		// (that would require an ex evaluator).
		matched := 0
		for _, ac := range e.autocmds {
			if ac.Event == fields[1] {
				matched++
			}
		}
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf(":doautocmd %s matched %d autocmd(s) (simulated, no body run)", fields[1], matched), Completed: true}
	case "mksession":
		path := "Session.vim"
		if len(fields) > 1 {
			path = fields[1]
		}
		if e.sessions == nil {
			e.sessions = map[string]bool{}
		}
		e.sessions[path] = true
		return ActionResult{Key: "enter", Token: token, Description: "wrote session to " + path + " (simulated)", Completed: true}
	case "mkview":
		path := "View"
		if len(fields) > 1 {
			path = fields[1]
		}
		if e.views == nil {
			e.views = map[string]bool{}
		}
		e.views[path] = true
		return ActionResult{Key: "enter", Token: token, Description: "wrote view " + path + " (simulated)", Completed: true}
	case "loadview":
		return ActionResult{Key: "enter", Token: token, Description: "loaded view (simulated)", Completed: true}
	case "wshada", "rshada":
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("ran %s (simulated)", fields[0]), Completed: true}
	case "sign":
		// :sign define name | :sign place id line=L name=N buffer=B | :sign list
		if len(fields) >= 2 && fields[1] == "place" {
			entry := signEntry{Buffer: e.active().Buffer}
			for _, kv := range fields[2:] {
				parts := strings.SplitN(kv, "=", 2)
				if len(parts) != 2 {
					continue
				}
				switch parts[0] {
				case "line":
					if n, err := strconv.Atoi(parts[1]); err == nil {
						entry.Line = n - 1
					}
				case "name":
					entry.Name = parts[1]
				case "buffer":
					if n, err := strconv.Atoi(parts[1]); err == nil && n > 0 && n <= len(e.buffers) {
						entry.Buffer = n - 1
					}
				}
			}
			e.signs = append(e.signs, entry)
			return ActionResult{Key: "enter", Token: token, Description: "placed sign", Completed: true}
		}
		if len(fields) >= 2 && fields[1] == "list" {
			e.lastEcho = fmt.Sprintf("%d sign(s) placed", len(e.signs))
			return ActionResult{Key: "enter", Token: token, Description: e.lastEcho, Completed: true}
		}
		return ActionResult{Key: "enter", Token: token, Description: ":sign accepted (simulated)", Completed: true}
	case "helpgrep":
		// :helpgrep populates quickfix with synthetic help-tag entries.
		if len(fields) < 2 {
			return ActionResult{Key: "enter", Token: token, Error: "missing pattern", Description: "use :helpgrep PATTERN", Completed: true}
		}
		pattern := fields[1]
		e.quickfix = []QuickfixItem{
			{Buffer: e.active().Buffer, Pos: Position{Row: 0, Col: 0}, Text: fmt.Sprintf("doc/help.txt:1: simulated help match for %q", pattern)},
		}
		e.quickfixIndex = 0
		return ActionResult{Key: "enter", Token: token, Description: ":helpgrep populated quickfix (simulated)", Completed: true}
	case "TSPlaygroundToggle", "TSBufToggle", "TSConfigInfo", "TSModuleInfo":
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("ran %s (simulated)", fields[0]), Completed: true}
	case "Snippets", "LuaSnipUnlinkCurrent", "LuaSnipListAvailable":
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("ran %s (simulated)", fields[0]), Completed: true}
	case "DapContinue", "DapStepOver", "DapStepInto", "DapStepOut", "DapToggleBreakpoint", "DapToggleRepl":
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("ran %s (simulated)", fields[0]), Completed: true}
	case "diffthis":
		return ActionResult{Key: "enter", Token: token, Description: "marked window for diff (simulated)", Completed: true}
	case "diffoff":
		return ActionResult{Key: "enter", Token: token, Description: "left diff mode (simulated)", Completed: true}
	case "diffupdate":
		return ActionResult{Key: "enter", Token: token, Description: "refreshed diff (simulated)", Completed: true}
	case "compiler":
		if len(fields) < 2 {
			return ActionResult{Key: "enter", Token: token, Error: "missing compiler name", Description: "use :compiler go|eslint|...", Completed: true}
		}
		// Compiler plugins set makeprg + errorformat; the trainer just
		// stores the name so config-walkthrough lessons can demonstrate
		// the wiring.
		e.lastEcho = "compiler set: " + fields[1]
		return ActionResult{Key: "enter", Token: token, Description: e.lastEcho, Completed: true}
	case "r", "read":
		if len(fields) < 2 {
			return ActionResult{Key: "enter", Token: token, Error: "missing argument", Description: "use :r file or :r !cmd", Completed: true}
		}
		if strings.HasPrefix(fields[1], "!") || (fields[1] == "!" && len(fields) > 2) {
			// :r !cmd — read fake command output below the cursor.
			cmdline := strings.TrimSpace(strings.TrimPrefix(strings.Join(fields[1:], " "), "!"))
			synth := e.synthesizeShellOutput(cmdline)
			e.insertLinesBelow(synth)
			return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("read %d line(s) from :!%s", len(synth), cmdline), Changed: true, Completed: true}
		}
		// :r file — synthesize a read by appending a fake line below.
		e.insertLinesBelow([]string{"// (simulated content of " + fields[1] + ")"})
		return ActionResult{Key: "enter", Token: token, Description: "read file (simulated)", Changed: true, Completed: true}
	case "%!sed":
		if len(fields) < 2 {
			return ActionResult{Key: "enter", Token: token, Error: "missing sed script", Description: "use :%!sed 's/old/new/g' to transform the current buffer", Completed: true}
		}
		script := strings.TrimSpace(strings.TrimPrefix(cmd, "%!sed"))
		changed, err := e.applySedScript(script)
		if err != nil {
			return ActionResult{Key: "enter", Token: token, Error: "invalid sed script", Description: err.Error(), Completed: true}
		}
		if !changed {
			return ActionResult{Key: "enter", Token: token, Description: "sed script ran but made no changes", Completed: true}
		}
		return ActionResult{Key: "enter", Token: token, Description: "applied sed filter to current buffer", Changed: true, Completed: true}
	default:
		return ActionResult{Key: "enter", Token: token, Error: "unsupported command", Description: "the trainer supports common :set/file commands, quickfix (:vimgrep/:cnext), explorer, lua/vimscript basics, plugin/LSP/Telescope commands, and :%!sed", Completed: true}
	}
}

func (e *Editor) executeSearch(query string, direction int) ActionResult {
	if query == "" {
		return ActionResult{Key: "enter", Token: "/", Description: "empty search", Completed: true}
	}
	if direction == 0 {
		direction = 1
	}
	e.lastSearch = query
	e.lastSearchDir = direction
	e.options.HLSearch = true

	startWindow := e.active()
	buf := e.currentBuffer()
	startRow := startWindow.Cursor.Row
	startCol := startWindow.Cursor.Col
	origin := startWindow.Cursor
	originBuffer := startWindow.Buffer
	if direction > 0 {
		startCol++
	}
	e.pushJumpAt(originBuffer, origin)
	found := e.findInBuffer(buf, query, startRow, startCol, direction)
	token := "/" + query
	if direction < 0 {
		token = "?" + query
	}
	e.recordHistory(token)
	if !found {
		return ActionResult{Key: "enter", Token: token, Error: "pattern not found", Description: fmt.Sprintf("Vim searched for %q but there was no match", query), Completed: true}
	}
	if direction < 0 {
		token = "N"
	}
	return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("found %q", query), Completed: true}
}

func (e *Editor) replay(tokens []string, token string) ActionResult {
	backup := e.replaying
	e.replaying = true
	var res ActionResult
	for _, key := range tokens {
		res = e.processKey(key)
	}
	e.replaying = backup
	res.Token = token
	res.Description = "replayed the recorded change"
	e.recordHistory(token)
	return res
}

func (e *Editor) beginChange(initialToken string) {
	snap := e.makeSnapshot()
	e.changeBaseline = &snap
	e.changeTokens = []string{initialToken}
}

func (e *Editor) finishChange(token string) {
	if e.changeBaseline == nil {
		return
	}
	if !snapshotsEqual(*e.changeBaseline, e.makeSnapshot()) {
		e.undo = append(e.undo, *e.changeBaseline)
		e.redo = nil
		e.lastChange = append([]string{}, e.changeTokens...)
		cur := e.currentCursor()
		e.recordChangeLocation(e.active().Buffer, cur)
		// Auto marks: '. = last change, '[ / '] = bounds of the most
		// recent change. The trainer doesn't track full ranges; we
		// approximate by marking the change cursor for both '[ and '].
		if e.marks == nil {
			e.marks = map[rune]Position{}
		}
		e.marks['.'] = cur
		e.marks['['] = cur
		e.marks[']'] = cur
		e.recordHistory(token)
	}
	e.changeBaseline = nil
	e.changeTokens = nil
}

func (e *Editor) cancelChange() {
	e.changeBaseline = nil
	e.changeTokens = nil
}

func (e *Editor) undoChange() bool {
	if len(e.undo) == 0 {
		return false
	}
	current := e.makeSnapshot()
	prev := e.undo[len(e.undo)-1]
	e.undo = e.undo[:len(e.undo)-1]
	e.redo = append(e.redo, current)
	e.restoreSnapshot(prev)
	return true
}

func (e *Editor) redoChange() bool {
	if len(e.redo) == 0 {
		return false
	}
	current := e.makeSnapshot()
	next := e.redo[len(e.redo)-1]
	e.redo = e.redo[:len(e.redo)-1]
	e.undo = append(e.undo, current)
	e.restoreSnapshot(next)
	return true
}

func (e *Editor) makeSnapshot() snapshot {
	marks := map[rune]Position{}
	for k, v := range e.marks {
		marks[k] = v
	}
	return snapshot{
		Buffers:       cloneBuffers(e.buffers),
		Windows:       cloneWindows(e.windows),
		ActiveWindow:  e.activeWindow,
		Options:       e.options,
		ExplorerOpen:  e.explorerOpen,
		Mode:          e.mode,
		LastSearch:    e.lastSearch,
		LastSearchDir: e.lastSearchDir,
		Marks:         marks,
	}
}

func (e *Editor) restoreSnapshot(s snapshot) {
	e.buffers = cloneBuffers(s.Buffers)
	e.windows = cloneWindows(s.Windows)
	e.activeWindow = s.ActiveWindow
	e.options = s.Options
	e.explorerOpen = s.ExplorerOpen
	e.mode = s.Mode
	e.lastSearch = s.LastSearch
	e.lastSearchDir = s.LastSearchDir
	e.marks = map[rune]Position{}
	for k, v := range s.Marks {
		e.marks[k] = v
	}
	e.normalizeWindows()
}

func snapshotsEqual(a, b snapshot) bool {
	if a.ActiveWindow != b.ActiveWindow || a.Options != b.Options || a.ExplorerOpen != b.ExplorerOpen || a.Mode != b.Mode || a.LastSearch != b.LastSearch || a.LastSearchDir != b.LastSearchDir {
		return false
	}
	if len(a.Buffers) != len(b.Buffers) || len(a.Windows) != len(b.Windows) || len(a.Marks) != len(b.Marks) {
		return false
	}
	for i := range a.Buffers {
		if a.Buffers[i].Name != b.Buffers[i].Name || strings.Join(a.Buffers[i].Lines, "\n") != strings.Join(b.Buffers[i].Lines, "\n") {
			return false
		}
	}
	for i := range a.Windows {
		if a.Windows[i] != b.Windows[i] {
			return false
		}
	}
	for k, v := range a.Marks {
		if b.Marks[k] != v {
			return false
		}
	}
	return true
}

func (e *Editor) active() *Window {
	return &e.windows[e.activeWindow]
}

func (e *Editor) currentBuffer() *Buffer {
	return &e.buffers[e.active().Buffer]
}

func (e *Editor) currentCursor() Position {
	return e.active().Cursor
}

func (e *Editor) clampCursor(bufferIdx int, pos Position) Position {
	if bufferIdx < 0 || bufferIdx >= len(e.buffers) {
		return Position{}
	}
	lines := e.buffers[bufferIdx].Lines
	if len(lines) == 0 {
		return Position{}
	}
	pos.Row = clamp(0, pos.Row, len(lines)-1)
	runes := []rune(lines[pos.Row])
	pos.Col = clamp(0, pos.Col, len(runes))
	if len(runes) > 0 && pos.Col == len(runes) && e.mode == ModeNormal {
		pos.Col--
	}
	return pos
}

func (e *Editor) normalizeWindows() {
	if len(e.windows) == 0 {
		e.windows = []Window{{Buffer: 0, Cursor: Position{}}}
	}
	e.activeWindow = clamp(0, e.activeWindow, len(e.windows)-1)
	for i := range e.windows {
		if e.windows[i].Buffer < 0 || e.windows[i].Buffer >= len(e.buffers) {
			e.windows[i].Buffer = 0
		}
		e.windows[i].Cursor = e.clampCursor(e.windows[i].Buffer, e.windows[i].Cursor)
	}
}

func (e *Editor) lineRunes() []rune {
	cur := e.currentCursor()
	return []rune(e.currentBuffer().Lines[cur.Row])
}

func (e *Editor) setCurrentLine(value string) {
	cur := e.currentCursor()
	e.currentBuffer().Lines[cur.Row] = value
	e.active().Cursor = e.clampCursor(e.active().Buffer, e.active().Cursor)
}

func (e *Editor) moveLeft() {
	cur := e.active()
	if cur.Cursor.Col > 0 {
		cur.Cursor.Col--
	}
}

func (e *Editor) moveRight() {
	cur := e.active()
	runes := e.lineRunes()
	if len(runes) == 0 {
		return
	}
	if cur.Cursor.Col < len(runes)-1 {
		cur.Cursor.Col++
	}
}

func (e *Editor) moveRightAppend() {
	cur := e.active()
	runes := e.lineRunes()
	if cur.Cursor.Col < len(runes) {
		cur.Cursor.Col++
	}
}

func (e *Editor) moveUp() {
	cur := e.active()
	if cur.Cursor.Row > 0 {
		cur.Cursor.Row--
		cur.Cursor = e.clampCursor(cur.Buffer, cur.Cursor)
	}
}

func (e *Editor) moveDown() {
	cur := e.active()
	if cur.Cursor.Row < len(e.currentBuffer().Lines)-1 {
		cur.Cursor.Row++
		cur.Cursor = e.clampCursor(cur.Buffer, cur.Cursor)
	}
}

func (e *Editor) moveLineStart() {
	e.active().Cursor.Col = 0
}

func (e *Editor) moveLineEnd() {
	runes := e.lineRunes()
	if len(runes) == 0 {
		e.active().Cursor.Col = 0
		return
	}
	e.active().Cursor.Col = len(runes) - 1
}

func (e *Editor) moveFileTop() {
	e.active().Cursor.Row = 0
	e.active().Cursor = e.clampCursor(e.active().Buffer, e.active().Cursor)
}

func (e *Editor) moveFileBottom() {
	e.active().Cursor.Row = len(e.currentBuffer().Lines) - 1
	e.active().Cursor = e.clampCursor(e.active().Buffer, e.active().Cursor)
}

func (e *Editor) moveWordForward() {
	cur := e.active()
	runes := e.lineRunes()
	if len(runes) == 0 {
		return
	}
	i := clamp(0, cur.Cursor.Col, len(runes)-1)
	for i < len(runes) && isWordRune(runes[i]) {
		i++
	}
	for i < len(runes) && !isWordRune(runes[i]) {
		i++
	}
	if i < len(runes) {
		cur.Cursor.Col = i
	}
}

func (e *Editor) moveWordBackward() {
	cur := e.active()
	runes := e.lineRunes()
	if len(runes) == 0 {
		return
	}
	i := clamp(0, cur.Cursor.Col, len(runes)-1)
	if i > 0 {
		i--
	}
	for i >= 0 && !isWordRune(runes[i]) {
		i--
	}
	for i >= 0 && isWordRune(runes[i]) {
		i--
	}
	cur.Cursor.Col = max(0, i+1)
}

func (e *Editor) moveWordEnd() {
	cur := e.active()
	runes := e.lineRunes()
	if len(runes) == 0 {
		return
	}
	i := clamp(0, cur.Cursor.Col, len(runes)-1)
	if !isWordRune(runes[i]) {
		for i < len(runes) && !isWordRune(runes[i]) {
			i++
		}
	}
	for i < len(runes) && isWordRune(runes[i]) {
		i++
	}
	if i > 0 {
		cur.Cursor.Col = i - 1
	}
}

func (e *Editor) wordBounds(includeSpaces bool) (int, int, bool) {
	runes := e.lineRunes()
	cur := e.currentCursor()
	if len(runes) == 0 || cur.Col >= len(runes) {
		return 0, 0, false
	}
	start := cur.Col
	if !isWordRune(runes[start]) {
		for start < len(runes) && !isWordRune(runes[start]) {
			start++
		}
	}
	if start >= len(runes) {
		return 0, 0, false
	}
	end := start
	for end < len(runes) && isWordRune(runes[end]) {
		end++
	}
	if includeSpaces {
		for end < len(runes) && unicode.IsSpace(runes[end]) {
			end++
		}
	}
	return start, end, true
}

func (e *Editor) wordUnderCursor() string {
	runes := e.lineRunes()
	cur := e.currentCursor()
	if len(runes) == 0 || cur.Col >= len(runes) {
		return ""
	}
	if !isWordRune(runes[cur.Col]) {
		return ""
	}
	start, end, ok := e.wordBounds(false)
	if !ok {
		return ""
	}
	return string(runes[start:end])
}

func (e *Editor) deleteChar() (yankData, bool) {
	runes := e.lineRunes()
	cur := e.currentCursor()
	if len(runes) == 0 || cur.Col >= len(runes) {
		return yankData{}, false
	}
	removed := runes[cur.Col]
	runes = append(runes[:cur.Col], runes[cur.Col+1:]...)
	e.setCurrentLine(string(runes))
	if cur.Col >= len(runes) && cur.Col > 0 {
		e.active().Cursor.Col--
	}
	return yankData{Text: string(removed), Linewise: false}, true
}

func (e *Editor) deleteWord() (yankData, bool) {
	runes := e.lineRunes()
	start, end, ok := e.wordBounds(true)
	if !ok {
		return yankData{}, false
	}
	deleted := string(runes[start:end])
	runes = append(runes[:start], runes[end:]...)
	e.setCurrentLine(string(runes))
	e.active().Cursor.Col = min(start, len([]rune(e.currentBuffer().Lines[e.currentCursor().Row])))
	return yankData{Text: deleted, Linewise: false}, true
}

func (e *Editor) changeWord() (yankData, bool) {
	return e.deleteWord()
}

func (e *Editor) deleteToLineEnd() (yankData, bool) {
	runes := e.lineRunes()
	cur := e.currentCursor()
	if cur.Col >= len(runes) {
		return yankData{}, false
	}
	deleted := string(runes[cur.Col:])
	runes = runes[:cur.Col]
	e.setCurrentLine(string(runes))
	e.active().Cursor = e.clampCursor(e.active().Buffer, e.active().Cursor)
	return yankData{Text: deleted, Linewise: false}, true
}

func (e *Editor) changeToLineEnd() (yankData, bool) {
	return e.deleteToLineEnd()
}

func (e *Editor) yankLine() {
	line := e.currentBuffer().Lines[e.currentCursor().Row]
	e.writeTextRegister(yankData{
		Text:     line,
		Linewise: true,
	}, false)
}

func (e *Editor) yankWord() bool {
	runes := e.lineRunes()
	start, end, ok := e.wordBounds(true)
	if !ok {
		return false
	}
	e.writeTextRegister(yankData{
		Text:     string(runes[start:end]),
		Linewise: false,
	}, false)
	return true
}

func (e *Editor) yankToLineEnd() bool {
	runes := e.lineRunes()
	cur := e.currentCursor()
	if cur.Col >= len(runes) {
		return false
	}
	e.writeTextRegister(yankData{
		Text:     string(runes[cur.Col:]),
		Linewise: false,
	}, false)
	return true
}

func (e *Editor) pasteAfter() bool {
	data := e.resolveReadRegister()
	if data.Text == "" {
		return false
	}
	if data.Linewise {
		cur := e.active()
		row := cur.Cursor.Row + 1
		lines := e.currentBuffer().Lines
		lines = append(lines[:row], append([]string{data.Text}, lines[row:]...)...)
		e.currentBuffer().Lines = lines
		cur.Cursor = Position{Row: row, Col: 0}
		return true
	}

	cur := e.active()
	runes := e.lineRunes()
	insertAt := cur.Cursor.Col + 1
	if len(runes) == 0 {
		insertAt = 0
	}
	if insertAt > len(runes) {
		insertAt = len(runes)
	}
	insertRunes := []rune(data.Text)
	runes = append(runes[:insertAt], append(insertRunes, runes[insertAt:]...)...)
	e.setCurrentLine(string(runes))
	if len(insertRunes) > 0 {
		cur.Cursor.Col = insertAt + len(insertRunes) - 1
	}
	return true
}

func (e *Editor) deleteLine() {
	cur := e.active()
	lines := e.currentBuffer().Lines
	lines = append(lines[:cur.Cursor.Row], lines[cur.Cursor.Row+1:]...)
	if len(lines) == 0 {
		lines = []string{""}
	}
	e.currentBuffer().Lines = lines
	if cur.Cursor.Row >= len(lines) {
		cur.Cursor.Row = len(lines) - 1
	}
	cur.Cursor = e.clampCursor(cur.Buffer, cur.Cursor)
}

// indentLines shifts each line in the inclusive [startRow, endRow] range
// by `direction` units of the trainer's shift width (config.IndentUnit).
// +1 indents, -1 outdents.
func (e *Editor) indentLines(startRow, endRow, direction int) {
	if direction == 0 {
		return
	}
	lines := e.currentBuffer().Lines
	if startRow < 0 {
		startRow = 0
	}
	if endRow >= len(lines) {
		endRow = len(lines) - 1
	}
	for r := startRow; r <= endRow; r++ {
		line := lines[r]
		if direction > 0 {
			lines[r] = config.IndentUnit + line
		} else {
			// Strip up to one IndentUnit's worth of leading whitespace.
			stripped := line
			for i := 0; i < len(config.IndentUnit); i++ {
				if len(stripped) > 0 && (stripped[0] == ' ' || stripped[0] == '\t') {
					stripped = stripped[1:]
				}
			}
			lines[r] = stripped
		}
	}
	e.currentBuffer().Lines = lines
	e.active().Cursor = e.clampCursor(e.active().Buffer, e.active().Cursor)
}

// reindentRange aligns every line in [startRow, endRow] (inclusive) to
// the indent of the first non-empty line in that range. This is a
// pragmatic stand-in for real `=` reindent (which delegates to a
// language-aware formatter). For curated trainer scenarios it produces
// the expected "everything lines up under the first line" result.
func (e *Editor) reindentRange(startRow, endRow int) {
	lines := e.currentBuffer().Lines
	if startRow < 0 {
		startRow = 0
	}
	if endRow >= len(lines) {
		endRow = len(lines) - 1
	}
	indent := ""
	for r := startRow; r <= endRow; r++ {
		if strings.TrimSpace(lines[r]) == "" {
			continue
		}
		indent = leadingWhitespace(lines[r])
		break
	}
	for r := startRow; r <= endRow; r++ {
		body := strings.TrimLeft(lines[r], " \t")
		if body == "" {
			lines[r] = ""
			continue
		}
		lines[r] = indent + body
	}
	e.currentBuffer().Lines = lines
	e.active().Cursor = e.clampCursor(e.active().Buffer, e.active().Cursor)
}

// reflowRange wraps text in [startRow, endRow] to the editor's text
// width (or 78 if unset). Words are split on whitespace; the resulting
// lines retain the original block's leading indent. Empty lines mark
// paragraph boundaries and are preserved.
func (e *Editor) reflowRange(startRow, endRow int) {
	lines := e.currentBuffer().Lines
	if startRow < 0 {
		startRow = 0
	}
	if endRow >= len(lines) {
		endRow = len(lines) - 1
	}
	if startRow > endRow {
		return
	}
	width := e.textWidth
	if width <= 0 {
		width = 78
	}
	indent := leadingWhitespace(lines[startRow])
	// Collect words across the range, preserving paragraph breaks.
	var paragraphs [][]string
	current := []string{}
	for r := startRow; r <= endRow; r++ {
		stripped := strings.TrimSpace(lines[r])
		if stripped == "" {
			if len(current) > 0 {
				paragraphs = append(paragraphs, current)
				current = nil
			}
			paragraphs = append(paragraphs, nil) // blank line marker
			continue
		}
		current = append(current, strings.Fields(stripped)...)
	}
	if len(current) > 0 {
		paragraphs = append(paragraphs, current)
	}
	// Render each paragraph at <= width chars, prefixing the indent.
	var rendered []string
	for _, words := range paragraphs {
		if words == nil {
			rendered = append(rendered, "")
			continue
		}
		var line strings.Builder
		line.WriteString(indent)
		first := true
		for _, w := range words {
			if first {
				line.WriteString(w)
				first = false
				continue
			}
			if line.Len()+1+len(w) > width {
				rendered = append(rendered, line.String())
				line.Reset()
				line.WriteString(indent)
				line.WriteString(w)
				continue
			}
			line.WriteByte(' ')
			line.WriteString(w)
		}
		if line.Len() > 0 {
			rendered = append(rendered, line.String())
		}
	}
	// Splice the rendered lines back into the buffer.
	updated := append([]string{}, lines[:startRow]...)
	updated = append(updated, rendered...)
	updated = append(updated, lines[endRow+1:]...)
	if len(updated) == 0 {
		updated = []string{""}
	}
	e.currentBuffer().Lines = updated
	e.active().Cursor = e.clampCursor(e.active().Buffer, e.active().Cursor)
}

// marksDump renders the engine's marks in :marks-style "name row col"
// rows for the :marks ex command. Order is stable.
func (e *Editor) marksDump() string {
	if len(e.marks) == 0 {
		return "no marks"
	}
	var keys []rune
	for k := range e.marks {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	var rows []string
	for _, k := range keys {
		pos := e.marks[k]
		rows = append(rows, fmt.Sprintf("'%c %d:%d", k, pos.Row+1, pos.Col+1))
	}
	return strings.Join(rows, " | ")
}

// expandMarkList turns a :delmarks argument like "a-cmZ" into the rune
// slice {a,b,c,m,Z}. Reverse ranges and bad characters are silently
// skipped — the trainer prefers permissive behaviour over Vim-faithful
// errors here.
func expandMarkList(spec string) []rune {
	var out []rune
	runes := []rune(spec)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			continue
		}
		if i+2 < len(runes) && runes[i+1] == '-' {
			start, end := r, runes[i+2]
			if end >= start {
				for c := start; c <= end; c++ {
					out = append(out, c)
				}
			}
			i += 2
			continue
		}
		out = append(out, r)
	}
	return out
}

// sortLines applies :sort to the inclusive [start, end] range.
//
// Flags supported: 'u' (unique), 'n' (numeric).  ':sort i' (case-insensitive)
// could be added later; the trainer focuses on the most-used flags.
func (e *Editor) sortLines(start, end int, flags string) {
	lines := e.currentBuffer().Lines
	if start < 0 {
		start = 0
	}
	if end >= len(lines) {
		end = len(lines) - 1
	}
	if start >= end {
		return
	}
	region := append([]string{}, lines[start:end+1]...)
	numeric := strings.Contains(flags, "n")
	unique := strings.Contains(flags, "u")
	sort.SliceStable(region, func(i, j int) bool {
		if numeric {
			ai, aerr := strconv.Atoi(strings.TrimSpace(region[i]))
			bi, berr := strconv.Atoi(strings.TrimSpace(region[j]))
			if aerr == nil && berr == nil {
				return ai < bi
			}
		}
		return region[i] < region[j]
	})
	if unique {
		seen := map[string]struct{}{}
		out := region[:0]
		for _, s := range region {
			if _, ok := seen[s]; ok {
				continue
			}
			seen[s] = struct{}{}
			out = append(out, s)
		}
		region = out
	}
	updated := append([]string{}, lines[:start]...)
	updated = append(updated, region...)
	updated = append(updated, lines[end+1:]...)
	if len(updated) == 0 {
		updated = []string{""}
	}
	e.currentBuffer().Lines = updated
	e.active().Cursor = e.clampCursor(e.active().Buffer, e.active().Cursor)
}

// retabBuffer replaces hard tabs with the configured indent unit. The
// trainer uses spaces-only indent (config.IndentUnit), so :retab here
// always normalizes towards expandtab.
func (e *Editor) retabBuffer() {
	lines := e.currentBuffer().Lines
	for i, line := range lines {
		lines[i] = strings.ReplaceAll(line, "\t", config.IndentUnit)
	}
	e.currentBuffer().Lines = lines
}

// synthesizeShellOutput returns a deterministic, lesson-friendly fake
// stdout for a `:!cmd` invocation. The trainer doesn't shell out for
// safety + reproducibility, but a few common commands have shaped
// outputs the curriculum can rely on.
func (e *Editor) synthesizeShellOutput(cmd string) []string {
	cmd = strings.TrimSpace(cmd)
	switch {
	case cmd == "" || cmd == "true":
		return []string{}
	case cmd == "date":
		return []string{"Sat Jan 1 12:00:00 UTC 2000"}
	case strings.HasPrefix(cmd, "echo "):
		return []string{strings.TrimSpace(strings.TrimPrefix(cmd, "echo "))}
	case strings.HasPrefix(cmd, "ls"):
		var names []string
		for _, b := range e.buffers {
			names = append(names, b.Name)
		}
		return names
	case strings.HasPrefix(cmd, "wc"):
		// Word/line/byte count of the active buffer.
		buf := e.currentBuffer()
		lines := len(buf.Lines)
		words := 0
		bytes := 0
		for _, l := range buf.Lines {
			words += len(strings.Fields(l))
			bytes += len(l) + 1
		}
		return []string{fmt.Sprintf(" %d %d %d %s", lines, words, bytes, buf.Name)}
	}
	return []string{fmt.Sprintf("(simulated output for %q)", cmd)}
}

// insertLinesBelow inserts the given lines immediately below the cursor.
// Used by :r ! and :r file.
func (e *Editor) insertLinesBelow(newLines []string) {
	if len(newLines) == 0 {
		return
	}
	cur := e.active()
	row := cur.Cursor.Row + 1
	lines := e.currentBuffer().Lines
	updated := append([]string{}, lines[:row]...)
	updated = append(updated, newLines...)
	updated = append(updated, lines[row:]...)
	e.currentBuffer().Lines = updated
	cur.Cursor = Position{Row: row, Col: 0}
}

// addFold registers a manual fold over the inclusive [start, end] line
// range in the active buffer. Folds are stored closed by default.
func (e *Editor) addFold(start, end int) {
	if e.folds == nil {
		e.folds = map[int][]foldRange{}
	}
	buf := e.active().Buffer
	e.folds[buf] = append(e.folds[buf], foldRange{Start: start, End: end, Closed: true})
}

func (e *Editor) toggleFoldUnderCursor(close, open bool) {
	if e.folds == nil {
		return
	}
	buf := e.active().Buffer
	row := e.currentCursor().Row
	for i := range e.folds[buf] {
		f := &e.folds[buf][i]
		if row >= f.Start && row <= f.End {
			switch {
			case close:
				f.Closed = true
			case open:
				f.Closed = false
			default:
				f.Closed = !f.Closed
			}
			return
		}
	}
}

func (e *Editor) openAllFolds() {
	if e.folds == nil {
		return
	}
	buf := e.active().Buffer
	for i := range e.folds[buf] {
		e.folds[buf][i].Closed = false
	}
}

func (e *Editor) closeAllFolds() {
	if e.folds == nil {
		return
	}
	buf := e.active().Buffer
	for i := range e.folds[buf] {
		e.folds[buf][i].Closed = true
	}
}

func (e *Editor) jumpFold(direction int) bool {
	if e.folds == nil {
		return false
	}
	buf := e.active().Buffer
	row := e.currentCursor().Row
	folds := e.folds[buf]
	if direction > 0 {
		var nearest *foldRange
		for i := range folds {
			if folds[i].Start > row {
				if nearest == nil || folds[i].Start < nearest.Start {
					nearest = &folds[i]
				}
			}
		}
		if nearest == nil {
			return false
		}
		e.active().Cursor = e.clampCursor(buf, Position{Row: nearest.Start, Col: 0})
		return true
	}
	var nearest *foldRange
	for i := range folds {
		if folds[i].End < row {
			if nearest == nil || folds[i].Start > nearest.Start {
				nearest = &folds[i]
			}
		}
	}
	if nearest == nil {
		return false
	}
	e.active().Cursor = e.clampCursor(buf, Position{Row: nearest.Start, Col: 0})
	return true
}

func (e *Editor) deleteFoldUnderCursor() {
	if e.folds == nil {
		return
	}
	buf := e.active().Buffer
	row := e.currentCursor().Row
	folds := e.folds[buf]
	for i := range folds {
		if row >= folds[i].Start && row <= folds[i].End {
			e.folds[buf] = append(folds[:i], folds[i+1:]...)
			return
		}
	}
}

// caseOpSentinel turns the user-facing case op char ('~', 'U', 'u', '?')
// into the operator sentinel routed through the pendingTextObject
// dispatcher. They share the same rune today; this indirection keeps
// the call sites self-documenting if the encoding ever needs to change.
func caseOpSentinel(op byte) rune { return rune(op) }

// applyCaseTransform applies a case operation to the rune-indexed range
// [startRow:startCol .. endRow:endCol] in the active buffer. Op chars:
//
//	'~' — toggle each char's case
//	'U' — uppercase every char
//	'u' — lowercase every char
//	'?' — rot13 each letter
func (e *Editor) applyCaseTransform(op byte, startRow, startCol, endRow, endCol int) {
	lines := e.currentBuffer().Lines
	if startRow < 0 || endRow >= len(lines) || startRow > endRow {
		return
	}
	transform := func(r rune) rune {
		switch op {
		case '~':
			if unicode.IsUpper(r) {
				return unicode.ToLower(r)
			}
			if unicode.IsLower(r) {
				return unicode.ToUpper(r)
			}
		case 'U':
			return unicode.ToUpper(r)
		case 'u':
			return unicode.ToLower(r)
		case '?':
			return rot13(r)
		}
		return r
	}
	for r := startRow; r <= endRow; r++ {
		runes := []rune(lines[r])
		fromCol := 0
		toCol := len(runes)
		if r == startRow {
			fromCol = clamp(0, startCol, len(runes))
		}
		if r == endRow {
			toCol = clamp(0, endCol, len(runes))
		}
		for i := fromCol; i < toCol; i++ {
			runes[i] = transform(runes[i])
		}
		lines[r] = string(runes)
	}
	e.currentBuffer().Lines = lines
}

func rot13(r rune) rune {
	switch {
	case r >= 'a' && r <= 'z':
		return 'a' + (r-'a'+13)%26
	case r >= 'A' && r <= 'Z':
		return 'A' + (r-'A'+13)%26
	}
	return r
}

// joinLines merges the cursor's line with the next. withSpace controls
// whether a separator space is inserted (J = true, gJ = false).
func (e *Editor) joinLines(withSpace bool) bool {
	cur := e.active()
	lines := e.currentBuffer().Lines
	row := cur.Cursor.Row
	if row+1 >= len(lines) {
		return false
	}
	first := lines[row]
	second := strings.TrimLeft(lines[row+1], " \t")
	merged := first
	if withSpace && first != "" && second != "" {
		merged += " " + second
	} else {
		merged += second
	}
	updated := append([]string{}, lines[:row]...)
	updated = append(updated, merged)
	updated = append(updated, lines[row+2:]...)
	e.currentBuffer().Lines = updated
	cur.Cursor = e.clampCursor(cur.Buffer, Position{Row: row, Col: len([]rune(first))})
	return true
}

func (e *Editor) toggleCaseUnderCursor() bool {
	cur := e.active()
	runes := e.lineRunes()
	if cur.Cursor.Col >= len(runes) {
		return false
	}
	r := runes[cur.Cursor.Col]
	switch {
	case unicode.IsUpper(r):
		runes[cur.Cursor.Col] = unicode.ToLower(r)
	case unicode.IsLower(r):
		runes[cur.Cursor.Col] = unicode.ToUpper(r)
	default:
		return false
	}
	e.setCurrentLine(string(runes))
	if cur.Cursor.Col+1 < len(runes) {
		cur.Cursor.Col++
	}
	return true
}

// adjustNumberUnderCursor increments or decrements the contiguous digit
// run at or to the right of the cursor. Returns false if no digit run
// can be found on the current line. Mirrors Vim's <C-a>/<C-x>.
func (e *Editor) adjustNumberUnderCursor(delta int) bool {
	cur := e.active()
	runes := e.lineRunes()
	if len(runes) == 0 {
		return false
	}
	// Walk from cursor right looking for the start of a digit run.
	start := cur.Cursor.Col
	for start < len(runes) && !unicode.IsDigit(runes[start]) {
		start++
	}
	if start >= len(runes) {
		return false
	}
	// If a '-' sits immediately before the digits, treat it as a sign.
	signed := start > 0 && runes[start-1] == '-'
	end := start
	for end < len(runes) && unicode.IsDigit(runes[end]) {
		end++
	}
	digits := string(runes[start:end])
	n, err := strconv.Atoi(digits)
	if err != nil {
		return false
	}
	if signed {
		n = -n
		start--
	}
	n += delta
	replacement := strconv.Itoa(n)
	newRunes := append([]rune{}, runes[:start]...)
	newRunes = append(newRunes, []rune(replacement)...)
	newRunes = append(newRunes, runes[end:]...)
	e.setCurrentLine(string(newRunes))
	cur.Cursor.Col = start + len(replacement) - 1
	return true
}

// recordVisualMarks stamps '< and '> with the current selection bounds
// so they survive into Normal mode. Called from every code path that
// leaves Visual mode.
func (e *Editor) recordVisualMarks() {
	if e.marks == nil {
		e.marks = map[rune]Position{}
	}
	a := e.visualStart
	b := e.currentCursor()
	if a.Row > b.Row || (a.Row == b.Row && a.Col > b.Col) {
		a, b = b, a
	}
	e.marks['<'] = a
	e.marks['>'] = b
}

func leadingWhitespace(s string) string {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return s[:i]
		}
	}
	return s
}

func (e *Editor) applyTextObjectOperation(op rune, objType rune, target rune) (yankData, bool) {
	sr, sc, er, ec, linewise, ok := e.textObjectRange(objType, target)
	if !ok {
		return yankData{}, false
	}
	switch op {
	case 'y':
		return e.extractRange(sr, sc, er, ec, linewise), true
	case 'd', 'c':
		return e.deleteRange(sr, sc, er, ec, linewise), true
	default:
		return yankData{}, false
	}
}

func (e *Editor) textObjectRange(objType rune, target rune) (startRow, startCol, endRow, endCol int, linewise bool, ok bool) {
	includeAround := objType == 'a'
	switch target {
	case 'w':
		runes := e.lineRunes()
		start, end, ok := e.wordBounds(includeAround)
		if !ok {
			return 0, 0, 0, 0, false, false
		}
		if includeAround {
			for start > 0 && unicode.IsSpace(runes[start-1]) {
				start--
			}
		}
		row := e.currentCursor().Row
		return row, start, row, end, false, true
	case '"', '\'', '`':
		return e.matchedQuoteObjectRange(target, includeAround)
	case '(', ')', 'b':
		return e.pairObjectRange('(', ')', includeAround)
	case '{', '}', 'B':
		return e.pairObjectRange('{', '}', includeAround)
	case '[', ']':
		return e.pairObjectRange('[', ']', includeAround)
	case '<', '>':
		return e.pairObjectRange('<', '>', includeAround)
	case 't':
		return e.tagObjectRange(includeAround)
	case 'f':
		return e.functionObjectRange(includeAround)
	case 'p':
		return e.paragraphObjectRange(includeAround)
	default:
		return 0, 0, 0, 0, false, false
	}
}

// matchedQuoteObjectRange supports i" / a", i' / a', i` / a`. Quotes are
// not nestable so the algorithm is a left-then-right scan.
func (e *Editor) matchedQuoteObjectRange(quote rune, includeQuotes bool) (startRow, startCol, endRow, endCol int, linewise bool, ok bool) {
	row := e.currentCursor().Row
	runes := e.lineRunes()
	cur := e.currentCursor().Col
	if len(runes) == 0 {
		return 0, 0, 0, 0, false, false
	}
	left := -1
	for i := cur; i >= 0 && i < len(runes); i-- {
		if runes[i] == quote {
			left = i
			break
		}
	}
	if left == -1 {
		for i := 0; i < len(runes); i++ {
			if runes[i] == quote {
				left = i
				break
			}
		}
	}
	if left == -1 {
		return 0, 0, 0, 0, false, false
	}
	right := -1
	for i := left + 1; i < len(runes); i++ {
		if runes[i] == quote {
			right = i
			break
		}
	}
	if right == -1 {
		return 0, 0, 0, 0, false, false
	}
	if includeQuotes {
		return row, left, row, right + 1, false, true
	}
	return row, left + 1, row, right, false, true
}

// pairObjectRange supports balanced pairs across multiple lines.
func (e *Editor) pairObjectRange(open, close rune, includePair bool) (startRow, startCol, endRow, endCol int, linewise bool, ok bool) {
	lines := e.currentBuffer().Lines
	cur := e.currentCursor()
	// Walk backward to find the enclosing open bracket.
	depth := 0
	leftRow, leftCol := -1, -1
	for r := cur.Row; r >= 0; r-- {
		runes := []rune(lines[r])
		startCol := len(runes) - 1
		if r == cur.Row {
			startCol = clamp(0, cur.Col, len(runes)-1)
		}
		for c := startCol; c >= 0; c-- {
			switch runes[c] {
			case close:
				if !(r == cur.Row && c == cur.Col) {
					depth++
				}
			case open:
				if depth == 0 {
					leftRow, leftCol = r, c
					goto FOUND_LEFT
				}
				depth--
			}
		}
	}
FOUND_LEFT:
	if leftRow == -1 {
		return 0, 0, 0, 0, false, false
	}
	// Walk forward to find the matching close.
	depth = 0
	rightRow, rightCol := -1, -1
	for r := leftRow; r < len(lines); r++ {
		runes := []rune(lines[r])
		startCol := 0
		if r == leftRow {
			startCol = leftCol
		}
		for c := startCol; c < len(runes); c++ {
			switch runes[c] {
			case open:
				depth++
			case close:
				depth--
				if depth == 0 {
					rightRow, rightCol = r, c
					goto FOUND_RIGHT
				}
			}
		}
	}
FOUND_RIGHT:
	if rightRow == -1 {
		return 0, 0, 0, 0, false, false
	}
	if includePair {
		return leftRow, leftCol, rightRow, rightCol + 1, false, true
	}
	if leftRow == rightRow {
		return leftRow, leftCol + 1, rightRow, rightCol, false, true
	}
	// Inner range across multiple lines.
	return leftRow, leftCol + 1, rightRow, rightCol, false, true
}

// tagObjectRange supports it / at on a line containing <tag>...</tag>.
func (e *Editor) tagObjectRange(includeTags bool) (startRow, startCol, endRow, endCol int, linewise bool, ok bool) {
	row := e.currentCursor().Row
	line := e.currentBuffer().Lines[row]
	openRe := regexp.MustCompile(`<([a-zA-Z][a-zA-Z0-9]*)[^<>]*>`)
	openLoc := openRe.FindStringSubmatchIndex(line)
	if openLoc == nil {
		return 0, 0, 0, 0, false, false
	}
	tagName := line[openLoc[2]:openLoc[3]]
	closeTag := "</" + tagName + ">"
	closeIdx := strings.Index(line[openLoc[1]:], closeTag)
	if closeIdx < 0 {
		return 0, 0, 0, 0, false, false
	}
	closeStart := openLoc[1] + closeIdx
	closeEnd := closeStart + len(closeTag)
	// Translate byte indices to rune indices for the runes-based pipeline.
	openStartRune := utf8RuneLen(line[:openLoc[0]])
	openEndRune := utf8RuneLen(line[:openLoc[1]])
	closeStartRune := utf8RuneLen(line[:closeStart])
	closeEndRune := utf8RuneLen(line[:closeEnd])
	if includeTags {
		return row, openStartRune, row, closeEndRune, false, true
	}
	return row, openEndRune, row, closeStartRune, false, true
}

// functionObjectRange supports if / af for Go-style `func name(...) { ... }`
// blocks. Native Vim does not ship if/af; this is a pragmatic
// approximation good enough for the trainer's curated buffers.
func (e *Editor) functionObjectRange(includeSignature bool) (startRow, startCol, endRow, endCol int, linewise bool, ok bool) {
	lines := e.currentBuffer().Lines
	cur := e.currentCursor()
	// Find the nearest `func ` line at or above the cursor.
	headerRow := -1
	for r := cur.Row; r >= 0; r-- {
		if strings.HasPrefix(strings.TrimLeft(lines[r], " \t"), "func ") {
			headerRow = r
			break
		}
	}
	if headerRow == -1 {
		return 0, 0, 0, 0, false, false
	}
	// Find opening brace on the header line or following lines.
	openRow := -1
	for r := headerRow; r < len(lines); r++ {
		if strings.Contains(lines[r], "{") {
			openRow = r
			break
		}
	}
	if openRow == -1 {
		return 0, 0, 0, 0, false, false
	}
	// Walk forward tracking brace depth.
	depth := 0
	closeRow := -1
	for r := openRow; r < len(lines); r++ {
		for _, ch := range lines[r] {
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
				if depth == 0 {
					closeRow = r
					break
				}
			}
		}
		if closeRow != -1 {
			break
		}
	}
	if closeRow == -1 {
		return 0, 0, 0, 0, false, false
	}
	if includeSignature {
		return headerRow, 0, closeRow, len([]rune(lines[closeRow])), true, true
	}
	// Inner: skip header line and closing brace line.
	if openRow+1 > closeRow-1 {
		return openRow, 0, openRow, 0, true, true
	}
	return openRow + 1, 0, closeRow - 1, len([]rune(lines[closeRow-1])), true, true
}

func (e *Editor) paragraphObjectRange(includeBlankAround bool) (startRow, startCol, endRow, endCol int, linewise bool, ok bool) {
	lines := e.currentBuffer().Lines
	row := e.currentCursor().Row
	if len(lines) == 0 {
		return 0, 0, 0, 0, true, false
	}
	if strings.TrimSpace(lines[row]) == "" {
		return 0, 0, 0, 0, true, false
	}
	start := row
	for start > 0 && strings.TrimSpace(lines[start-1]) != "" {
		start--
	}
	end := row
	for end+1 < len(lines) && strings.TrimSpace(lines[end+1]) != "" {
		end++
	}
	if includeBlankAround {
		if start > 0 && strings.TrimSpace(lines[start-1]) == "" {
			start--
		}
		if end+1 < len(lines) && strings.TrimSpace(lines[end+1]) == "" {
			end++
		}
	}
	return start, 0, end, len([]rune(lines[end])), true, true
}

func (e *Editor) extractRange(startRow, startCol, endRow, endCol int, linewise bool) yankData {
	lines := e.currentBuffer().Lines
	if linewise {
		if startRow < 0 || endRow >= len(lines) || startRow > endRow {
			return yankData{}
		}
		return yankData{
			Text:     strings.Join(lines[startRow:endRow+1], "\n"),
			Linewise: true,
		}
	}

	if startRow == endRow {
		runes := []rune(lines[startRow])
		startCol = clamp(0, startCol, len(runes))
		endCol = clamp(startCol, endCol, len(runes))
		return yankData{
			Text:     string(runes[startCol:endCol]),
			Linewise: false,
		}
	}

	var pieces []string
	first := []rune(lines[startRow])
	pieces = append(pieces, string(first[clamp(0, startCol, len(first)):]))
	for row := startRow + 1; row < endRow; row++ {
		pieces = append(pieces, lines[row])
	}
	last := []rune(lines[endRow])
	pieces = append(pieces, string(last[:clamp(0, endCol, len(last))]))
	return yankData{Text: strings.Join(pieces, "\n"), Linewise: false}
}

func (e *Editor) deleteRange(startRow, startCol, endRow, endCol int, linewise bool) yankData {
	data := e.extractRange(startRow, startCol, endRow, endCol, linewise)
	lines := e.currentBuffer().Lines

	if linewise {
		if startRow < 0 || endRow >= len(lines) || startRow > endRow {
			return yankData{}
		}
		updated := append([]string{}, lines[:startRow]...)
		updated = append(updated, lines[endRow+1:]...)
		if len(updated) == 0 {
			updated = []string{""}
		}
		e.currentBuffer().Lines = updated
		e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: min(startRow, len(updated)-1), Col: 0})
		return data
	}

	if startRow == endRow {
		runes := []rune(lines[startRow])
		startCol = clamp(0, startCol, len(runes))
		endCol = clamp(startCol, endCol, len(runes))
		runes = append(runes[:startCol], runes[endCol:]...)
		lines[startRow] = string(runes)
		e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: startRow, Col: startCol})
		return data
	}

	first := []rune(lines[startRow])
	last := []rune(lines[endRow])
	newLine := string(first[:clamp(0, startCol, len(first))]) + string(last[clamp(0, endCol, len(last)):])
	updated := append([]string{}, lines[:startRow]...)
	updated = append(updated, newLine)
	updated = append(updated, lines[endRow+1:]...)
	if len(updated) == 0 {
		updated = []string{""}
	}
	e.currentBuffer().Lines = updated
	e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: startRow, Col: startCol})
	return data
}

func (e *Editor) visualRange() (startRow, startCol, endRow, endCol int, linewise bool) {
	a := e.visualStart
	b := e.currentCursor()
	if a.Row > b.Row || (a.Row == b.Row && a.Col > b.Col) {
		a, b = b, a
	}
	if e.visualLine {
		return a.Row, 0, b.Row, len([]rune(e.currentBuffer().Lines[b.Row])), true
	}
	return a.Row, a.Col, b.Row, b.Col + 1, false
}

// blockRange returns the rectangular bounds of the current visual-block selection.
func (e *Editor) blockRange() (topRow, leftCol, botRow, rightCol int) {
	a := e.visualStart
	b := e.currentCursor()
	topRow, botRow = a.Row, b.Row
	if topRow > botRow {
		topRow, botRow = botRow, topRow
	}
	leftCol, rightCol = a.Col, b.Col
	if leftCol > rightCol {
		leftCol, rightCol = rightCol, leftCol
	}
	return topRow, leftCol, botRow, rightCol + 1
}

func (e *Editor) extractBlock() yankData {
	tr, lc, br, rc := e.blockRange()
	lines := e.currentBuffer().Lines
	var pieces []string
	for r := tr; r <= br && r < len(lines); r++ {
		runes := []rune(lines[r])
		left := clamp(0, lc, len(runes))
		right := clamp(left, rc, len(runes))
		pieces = append(pieces, string(runes[left:right]))
	}
	return yankData{Text: strings.Join(pieces, "\n"), Linewise: false}
}

func (e *Editor) deleteBlock() yankData {
	tr, lc, br, rc := e.blockRange()
	data := e.extractBlock()
	lines := e.currentBuffer().Lines
	for r := tr; r <= br && r < len(lines); r++ {
		runes := []rune(lines[r])
		left := clamp(0, lc, len(runes))
		right := clamp(left, rc, len(runes))
		runes = append(runes[:left], runes[right:]...)
		lines[r] = string(runes)
	}
	e.currentBuffer().Lines = lines
	e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: tr, Col: lc})
	return data
}

func (e *Editor) deleteVisualSelection() (yankData, bool) {
	if e.visualBlock {
		data := e.deleteBlock()
		return data, data.Text != ""
	}
	sr, sc, er, ec, linewise := e.visualRange()
	data := e.deleteRange(sr, sc, er, ec, linewise)
	return data, data.Text != "" || data.Linewise
}

func (e *Editor) yankVisualSelection() (yankData, bool) {
	if e.visualBlock {
		data := e.extractBlock()
		return data, data.Text != ""
	}
	sr, sc, er, ec, linewise := e.visualRange()
	data := e.extractRange(sr, sc, er, ec, linewise)
	return data, data.Text != "" || data.Linewise
}

func (e *Editor) replaceVisualSelectionWithPaste() bool {
	if e.visualBlock {
		_ = e.deleteBlock()
		return e.pasteAfter()
	}
	sr, sc, er, ec, linewise := e.visualRange()
	_ = e.deleteRange(sr, sc, er, ec, linewise)
	return e.pasteAfter()
}

// writeTextRegister handles every Vim-style register write. Behaviour
// mirrors real Vim:
//
//   - "_  (blackhole) — capturing the active register suppresses every
//     other side-effect and just discards the data.
//   - "0  — yanks land here.
//   - "-  — small (intra-line) deletes land here.
//   - "1..9 — multi-line deletes shift through this ring; 1 is newest.
//   - "a..z — explicit named registers; capital A..Z append.
//   - "+ "*  — system-clipboard aliases (engine has no real clipboard,
//     so they behave like ordinary letter registers).
//   - "" — always reflects the most recent write.
func (e *Editor) writeTextRegister(data yankData, fromDelete bool) {
	target := e.activeRegister
	e.activeRegister = 0
	// Black hole register: discard everything; do not even update "".
	if target == '_' {
		return
	}
	e.yankBuffer = data
	e.textRegisters['"'] = data
	if fromDelete {
		e.textRegisters['-'] = data
		// Multi-line deletes also push through the numbered ring; that's
		// real-Vim semantics. Detect via either Linewise or an embedded
		// newline in the deleted text.
		if data.Linewise || strings.Contains(data.Text, "\n") {
			e.pushDeleteRing(data)
		}
	} else {
		e.textRegisters['0'] = data
	}
	if target == 0 {
		return
	}
	if unicode.IsUpper(target) {
		base := unicode.ToLower(target)
		existing := e.textRegisters[base]
		if existing.Text == "" {
			e.textRegisters[base] = data
			return
		}
		if existing.Linewise || data.Linewise {
			existing.Linewise = true
			existing.Text = strings.TrimSuffix(existing.Text, "\n") + "\n" + strings.TrimPrefix(data.Text, "\n")
		} else {
			existing.Text += data.Text
		}
		e.textRegisters[base] = existing
		return
	}
	e.textRegisters[target] = data
}

// pushDeleteRing rotates "1..9 so the freshest delete is in "1.
func (e *Editor) pushDeleteRing(data yankData) {
	for i := '9'; i > '1'; i-- {
		e.textRegisters[i] = e.textRegisters[i-1]
	}
	e.textRegisters['1'] = data
}

// resolveReadRegister fetches the register contents for the next paste.
// Special registers ("/, ":, "., "%, "#, "=) are computed on demand from
// engine state rather than stored.
func (e *Editor) resolveReadRegister() yankData {
	target := e.activeRegister
	e.activeRegister = 0
	if target == 0 {
		target = '"'
	}
	switch target {
	case '_':
		return yankData{}
	case '/':
		return yankData{Text: e.lastSearch}
	case ':':
		// Last command-line command typed (with its leading colon).
		for i := len(e.commandHistory) - 1; i >= 0; i-- {
			if strings.HasPrefix(e.commandHistory[i], ":") {
				return yankData{Text: strings.TrimPrefix(e.commandHistory[i], ":")}
			}
		}
		return yankData{}
	case '.':
		return yankData{Text: strings.Join(e.lastChange, "")}
	case '%':
		return yankData{Text: e.currentBuffer().Name}
	case '#':
		// "Alternate file" register. The trainer doesn't model an alt
		// buffer, so return the previous buffer name if any.
		if e.activeWindow < len(e.windows) {
			cur := e.windows[e.activeWindow].Buffer
			for i := len(e.buffers) - 1; i >= 0; i-- {
				if i != cur {
					return yankData{Text: e.buffers[i].Name}
				}
			}
		}
		return yankData{}
	case '=':
		// Expression register; returns the most recent :echo output as
		// a stand-in for evaluated expressions.
		return yankData{Text: e.lastEcho}
	}
	if data, ok := e.textRegisters[target]; ok {
		e.yankBuffer = data
		return data
	}
	if data, ok := e.textRegisters['"']; ok {
		e.yankBuffer = data
		return data
	}
	return yankData{}
}

func (e *Editor) registerDump() string {
	if len(e.textRegisters) == 0 {
		return "no registers"
	}
	var keys []rune
	for k := range e.textRegisters {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	var out []string
	for _, key := range keys {
		val := e.textRegisters[key]
		if val.Text == "" {
			continue
		}
		display := val.Text
		display = strings.ReplaceAll(display, "\n", "\\n")
		out = append(out, fmt.Sprintf("\"%c %s", key, display))
	}
	if len(out) == 0 {
		return "no registers"
	}
	return strings.Join(out, " | ")
}

func (e *Editor) pushJump() {
	e.pushJumpAt(e.active().Buffer, e.currentCursor())
}

func (e *Editor) pushJumpAt(buffer int, pos Position) {
	entry := jumpEntry{Buffer: buffer, Pos: pos}
	if len(e.jumpList) > 0 && e.jumpList[len(e.jumpList)-1] == entry {
		e.jumpIndex = len(e.jumpList) - 1
		return
	}
	if e.jumpIndex >= 0 && e.jumpIndex < len(e.jumpList)-1 {
		e.jumpList = append([]jumpEntry{}, e.jumpList[:e.jumpIndex+1]...)
	}
	e.jumpList = append(e.jumpList, entry)
	if len(e.jumpList) > config.EditorMaxJumpListEntries {
		e.jumpList = e.jumpList[len(e.jumpList)-config.EditorMaxJumpListEntries:]
	}
	e.jumpIndex = len(e.jumpList) - 1
}

func (e *Editor) jumpOlder() bool {
	if len(e.jumpList) == 0 || e.jumpIndex <= 0 {
		return false
	}
	e.jumpIndex--
	return e.applyJump(e.jumpList[e.jumpIndex])
}

func (e *Editor) jumpNewer() bool {
	if len(e.jumpList) == 0 || e.jumpIndex >= len(e.jumpList)-1 {
		return false
	}
	e.jumpIndex++
	return e.applyJump(e.jumpList[e.jumpIndex])
}

func (e *Editor) applyJump(entry jumpEntry) bool {
	if entry.Buffer < 0 || entry.Buffer >= len(e.buffers) {
		return false
	}
	e.active().Buffer = entry.Buffer
	e.active().Cursor = e.clampCursor(entry.Buffer, entry.Pos)
	return true
}

func (e *Editor) recordChangeLocation(buffer int, pos Position) {
	entry := jumpEntry{Buffer: buffer, Pos: pos}
	if len(e.changeList) > 0 && e.changeList[len(e.changeList)-1] == entry {
		e.changeIndex = len(e.changeList) - 1
		return
	}
	e.changeList = append(e.changeList, entry)
	if len(e.changeList) > config.EditorMaxChangeListEntries {
		e.changeList = e.changeList[len(e.changeList)-config.EditorMaxChangeListEntries:]
	}
	e.changeIndex = len(e.changeList) - 1
}

func (e *Editor) changeOlder() bool {
	if len(e.changeList) == 0 {
		return false
	}
	if e.changeIndex <= 0 {
		return false
	}
	e.changeIndex--
	return e.applyJump(e.changeList[e.changeIndex])
}

func (e *Editor) changeNewer() bool {
	if len(e.changeList) == 0 {
		return false
	}
	if e.changeIndex >= len(e.changeList)-1 {
		return false
	}
	e.changeIndex++
	return e.applyJump(e.changeList[e.changeIndex])
}

// goToDefinition is the trainer's `gd` implementation. Real Vim ships gd
// as "go to local definition," which delegates to the underlying language
// server in modern setups. Here we approximate with a multi-language
// heuristic: walk the buffer top-to-bottom and stop on the first line
// that *looks like* a definition site for the word under the cursor.
//
// The patterns cover Go, Rust, Python, JavaScript / TypeScript, Lua,
// Vimscript, shell, and a handful of common idioms. False positives /
// negatives are accepted; the trainer doesn't need real symbol
// resolution, just enough fidelity for curriculum scenarios.
func (e *Editor) goToDefinition() bool {
	word := e.wordUnderCursor()
	if word == "" {
		return false
	}
	patterns := definitionPatterns(word)
	lines := e.currentBuffer().Lines
	curPos := e.currentCursor()
	for row, line := range lines {
		col := -1
		for _, pattern := range patterns {
			if c := strings.Index(line, pattern); c >= 0 {
				// Skip the cursor's own position so gd doesn't "jump"
				// to where you already are. (The shell-style `name()`
				// pattern is otherwise satisfied by the call site
				// itself.)
				if row == curPos.Row && c == curPos.Col {
					continue
				}
				if col < 0 || c < col {
					col = c
				}
			}
		}
		if col >= 0 {
			e.pushJump()
			e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: row, Col: col})
			return true
		}
	}
	return false
}

// definitionPatterns returns the language-flavored prefixes we treat as a
// definition site for the given symbol name. Order doesn't matter — the
// caller takes the leftmost match per line so multiple-keyword languages
// (e.g. `pub fn name`) still resolve correctly.
func definitionPatterns(name string) []string {
	return []string{
		// Go
		"func " + name,
		"var " + name,
		"type " + name,
		"const " + name,
		// Rust
		"fn " + name,
		"struct " + name,
		"enum " + name,
		"trait " + name,
		"impl " + name,
		"let " + name,
		"pub fn " + name,
		"pub struct " + name,
		// Python
		"def " + name,
		"class " + name,
		"async def " + name,
		// JavaScript / TypeScript
		"function " + name,
		"function* " + name,
		"async function " + name,
		"const " + name + " =",
		"let " + name + " =",
		"var " + name + " =",
		"interface " + name,
		"type " + name + " =",
		// Lua
		"local " + name + " =",
		"local function " + name,
		"function " + name + "(",
		// Vimscript
		":let " + name,
		":nnoremap " + name,
		":command! " + name,
		// Shell / Bash
		name + "()",
		name + " ()",
		"function " + name + "()",
		// Generic assignment
		name + " = ",
		name + ":=",
	}
}

func (e *Editor) populateQuickfix(pattern string) int {
	// Push the current list onto the history ring before replacing it so
	// :cnewer / :colder can step through past searches.
	if len(e.quickfix) > 0 {
		snap := append([]QuickfixItem{}, e.quickfix...)
		e.quickfixLists = append(e.quickfixLists, snap)
		if len(e.quickfixLists) > config.QuickfixHistoryCap {
			e.quickfixLists = e.quickfixLists[len(e.quickfixLists)-config.QuickfixHistoryCap:]
		}
		e.quickfixListIdx = len(e.quickfixLists)
	}
	e.quickfix = nil
	e.quickfixIndex = 0
	re := e.compileSearch(pattern)
	for b, buf := range e.buffers {
		for row, line := range buf.Lines {
			runes := []rune(line)
			var col int
			if re != nil {
				loc := matchAfter(re, runes, 0)
				if loc == nil {
					continue
				}
				col = loc[0]
			} else {
				col = strings.Index(line, pattern)
				if col < 0 {
					continue
				}
			}
			e.quickfix = append(e.quickfix, QuickfixItem{
				Buffer: b,
				Pos:    Position{Row: row, Col: col},
				Text:   line,
			})
		}
	}
	if len(e.quickfix) == 0 {
		return 0
	}
	e.quickfixSet(0)
	return len(e.quickfix)
}

func (e *Editor) quickfixSet(index int) bool {
	if len(e.quickfix) == 0 || index < 0 || index >= len(e.quickfix) {
		return false
	}
	item := e.quickfix[index]
	e.pushJump()
	e.active().Buffer = item.Buffer
	e.active().Cursor = e.clampCursor(item.Buffer, item.Pos)
	e.quickfixIndex = index
	return true
}

func (e *Editor) quickfixMove(delta int) bool {
	if len(e.quickfix) == 0 {
		return false
	}
	next := e.quickfixIndex + delta
	if next < 0 {
		next = len(e.quickfix) - 1
	}
	if next >= len(e.quickfix) {
		next = 0
	}
	return e.quickfixSet(next)
}

// quickfixHistoryStep moves through the ring of past quickfix lists.
// direction: +1 = newer, -1 = older. Returns false if there's no list to
// step to in that direction.
func (e *Editor) quickfixHistoryStep(direction int) bool {
	if len(e.quickfixLists) == 0 {
		return false
	}
	// We treat the "current" quickfix as one slot beyond the history ring
	// (index len(quickfixLists)). Older = step back into the ring; newer
	// = step forward toward the current one.
	if direction < 0 {
		if e.quickfixListIdx <= 0 {
			return false
		}
		// Push current into the ring at end if we're stepping back from "current".
		if e.quickfixListIdx == len(e.quickfixLists) && len(e.quickfix) > 0 {
			snap := append([]QuickfixItem{}, e.quickfix...)
			e.quickfixLists = append(e.quickfixLists, snap)
		}
		e.quickfixListIdx--
		e.quickfix = append([]QuickfixItem{}, e.quickfixLists[e.quickfixListIdx]...)
		e.quickfixIndex = 0
		return true
	}
	if e.quickfixListIdx >= len(e.quickfixLists)-1 {
		return false
	}
	e.quickfixListIdx++
	e.quickfix = append([]QuickfixItem{}, e.quickfixLists[e.quickfixListIdx]...)
	e.quickfixIndex = 0
	return true
}

// runCDO executes an ex command at every quickfix entry. The classic use
// is `:cdo norm @a` to replay a macro on every match. This is the move
// that turns "Vim is fast" into "Vim is unreasonably fast."
func (e *Editor) runCDO(cmd, token string) ActionResult {
	if len(e.quickfix) == 0 {
		return ActionResult{Key: "enter", Token: token, Error: "quickfix is empty", Description: "populate quickfix with :vimgrep first", Completed: true}
	}
	count := 0
	for i, item := range e.quickfix {
		if item.Buffer < 0 || item.Buffer >= len(e.buffers) {
			continue
		}
		e.active().Buffer = item.Buffer
		e.active().Cursor = e.clampCursor(item.Buffer, item.Pos)
		e.quickfixIndex = i
		// Reuse executeCommand so the body is the same surface the user
		// would type interactively. This keeps `:cdo s/foo/bar/g` and
		// `:cdo norm @a` honest.
		_ = e.executeCommand(cmd)
		count++
	}
	return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("ran %q on %d quickfix entries", cmd, count), Changed: true, Completed: true}
}

// runCFDO is the per-file variant of :cdo: the command runs once per
// distinct file in the quickfix list.
func (e *Editor) runCFDO(cmd, token string) ActionResult {
	if len(e.quickfix) == 0 {
		return ActionResult{Key: "enter", Token: token, Error: "quickfix is empty", Description: "populate quickfix with :vimgrep first", Completed: true}
	}
	seen := map[int]bool{}
	count := 0
	for _, item := range e.quickfix {
		if seen[item.Buffer] {
			continue
		}
		seen[item.Buffer] = true
		if item.Buffer < 0 || item.Buffer >= len(e.buffers) {
			continue
		}
		e.active().Buffer = item.Buffer
		e.active().Cursor = e.clampCursor(item.Buffer, item.Pos)
		_ = e.executeCommand(cmd)
		count++
	}
	return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("ran %q across %d file(s)", cmd, count), Changed: true, Completed: true}
}

func (e *Editor) insertRune(r rune) {
	cur := e.active()
	runes := e.lineRunes()
	if cur.Cursor.Col > len(runes) {
		cur.Cursor.Col = len(runes)
	}
	runes = append(runes[:cur.Cursor.Col], append([]rune{r}, runes[cur.Cursor.Col:]...)...)
	e.setCurrentLine(string(runes))
	cur.Cursor.Col++
}

func (e *Editor) backspaceInsert() bool {
	cur := e.active()
	runes := e.lineRunes()
	if cur.Cursor.Col == 0 || len(runes) == 0 {
		return false
	}
	idx := cur.Cursor.Col - 1
	runes = append(runes[:idx], runes[cur.Cursor.Col:]...)
	e.setCurrentLine(string(runes))
	cur.Cursor.Col--
	return true
}

func (e *Editor) openLineBelow() {
	cur := e.active()
	row := cur.Cursor.Row + 1
	lines := e.currentBuffer().Lines
	lines = append(lines[:row], append([]string{""}, lines[row:]...)...)
	e.currentBuffer().Lines = lines
	cur.Cursor = Position{Row: row, Col: 0}
}

func (e *Editor) openLineAbove() {
	cur := e.active()
	row := cur.Cursor.Row
	lines := e.currentBuffer().Lines
	lines = append(lines[:row], append([]string{""}, lines[row:]...)...)
	e.currentBuffer().Lines = lines
	cur.Cursor = Position{Row: row, Col: 0}
}

// compileSearch turns a Vim-style search pattern into a Go regexp, applying
// ignorecase / smartcase based on current options. Vim-specific atoms that
// have direct Go-regex analogs (\<, \>, \C, \c) are translated; everything
// else is passed through and falls back to literal substring matching when
// the regex fails to compile.
func (e *Editor) compileSearch(pattern string) *regexp.Regexp {
	if pattern == "" {
		return nil
	}
	caseFlag := ""
	hasUpper := false
	for _, r := range pattern {
		if unicode.IsUpper(r) {
			hasUpper = true
			break
		}
	}
	switch {
	case strings.Contains(pattern, `\C`):
		caseFlag = ""
	case strings.Contains(pattern, `\c`):
		caseFlag = "(?i)"
	case e.options.IgnoreCase && (!e.options.SmartCase || !hasUpper):
		caseFlag = "(?i)"
	}
	translated := pattern
	translated = strings.ReplaceAll(translated, `\<`, `\b`)
	translated = strings.ReplaceAll(translated, `\>`, `\b`)
	translated = strings.ReplaceAll(translated, `\C`, ``)
	translated = strings.ReplaceAll(translated, `\c`, ``)
	re, err := regexp.Compile(caseFlag + translated)
	if err != nil {
		// Fall back to literal match if the user typed something the regex
		// engine refuses (e.g. a stray bracket).
		re, err = regexp.Compile(caseFlag + regexp.QuoteMeta(pattern))
		if err != nil {
			return nil
		}
	}
	return re
}

func (e *Editor) findInBuffer(buf *Buffer, query string, row int, col int, direction int) bool {
	re := e.compileSearch(query)
	if re == nil {
		return false
	}
	r, c, _, _, ok := searchBufferRegex(buf, re, row, col, direction)
	if !ok {
		return false
	}
	e.active().Cursor = Position{Row: r, Col: c}
	return true
}

// findMatchRange locates the next/previous regex match of the last search
// starting from the cursor and returns its character range. Used by gn / gN.
func (e *Editor) findMatchRange(query string, direction int) (sr, sc, er, ec int, ok bool) {
	re := e.compileSearch(query)
	if re == nil {
		return 0, 0, 0, 0, false
	}
	cur := e.currentCursor()
	return searchBufferRegex(e.currentBuffer(), re, cur.Row, cur.Col, direction)
}

// searchBufferRegex searches the buffer for the first match of re in the
// requested direction starting from (row, col). It returns the (startRow,
// startCol, endRow, endCol) of the match (endCol exclusive). All matches
// are intra-line.
func searchBufferRegex(buf *Buffer, re *regexp.Regexp, row, col, direction int) (sr, sc, er, ec int, ok bool) {
	if direction >= 0 {
		for r := row; r < len(buf.Lines); r++ {
			runes := []rune(buf.Lines[r])
			start := 0
			if r == row {
				start = min(col, len(runes))
			}
			if loc := matchAfter(re, runes, start); loc != nil {
				return r, loc[0], r, loc[1], true
			}
		}
		for r := 0; r <= row; r++ {
			runes := []rune(buf.Lines[r])
			limit := len(runes)
			if r == row {
				limit = min(col, len(runes))
			}
			if loc := matchBefore(re, runes, limit); loc != nil {
				return r, loc[0], r, loc[1], true
			}
		}
		return 0, 0, 0, 0, false
	}
	for r := row; r >= 0; r-- {
		runes := []rune(buf.Lines[r])
		limit := len(runes)
		if r == row {
			limit = max(0, col)
		}
		if loc := matchBefore(re, runes, limit); loc != nil {
			return r, loc[0], r, loc[1], true
		}
	}
	for r := len(buf.Lines) - 1; r > row; r-- {
		runes := []rune(buf.Lines[r])
		if loc := matchAfter(re, runes, 0); loc != nil {
			return r, loc[0], r, loc[1], true
		}
	}
	return 0, 0, 0, 0, false
}

// matchAfter returns the rune-indexed [start,end] of the first match of re
// in runes at or after fromRune. Translates between byte indices (regexp)
// and rune indices.
func matchAfter(re *regexp.Regexp, runes []rune, fromRune int) []int {
	if fromRune > len(runes) {
		return nil
	}
	if fromRune < 0 {
		fromRune = 0
	}
	sub := string(runes[fromRune:])
	loc := re.FindStringIndex(sub)
	if loc == nil {
		return nil
	}
	startRune := fromRune + utf8RuneLen(sub[:loc[0]])
	endRune := fromRune + utf8RuneLen(sub[:loc[1]])
	return []int{startRune, endRune}
}

// matchBefore returns the last match of re strictly before limitRune.
func matchBefore(re *regexp.Regexp, runes []rune, limitRune int) []int {
	if limitRune <= 0 {
		return nil
	}
	if limitRune > len(runes) {
		limitRune = len(runes)
	}
	sub := string(runes[:limitRune])
	matches := re.FindAllStringIndex(sub, -1)
	if len(matches) == 0 {
		return nil
	}
	last := matches[len(matches)-1]
	startRune := utf8RuneLen(sub[:last[0]])
	endRune := utf8RuneLen(sub[:last[1]])
	return []int{startRune, endRune}
}

func utf8RuneLen(s string) int {
	return len([]rune(s))
}

func (e *Editor) openOrSwitchBuffer(name string) {
	prev := e.active().Buffer
	for i, buf := range e.buffers {
		if buf.Name == name {
			if i != prev {
				e.alternateBuffer = prev
			}
			e.active().Buffer = i
			e.active().Cursor = e.clampCursor(i, Position{})
			return
		}
	}
	e.buffers = append(e.buffers, Buffer{Name: name, Lines: defaultBufferLines(name)})
	e.alternateBuffer = prev
	e.active().Buffer = len(e.buffers) - 1
	e.active().Cursor = Position{}
}

func (e *Editor) openSplit() {
	win := *e.active()
	e.windows = append(e.windows, win)
	e.activeWindow = len(e.windows) - 1
}

// snapshotActiveTab freezes the live windows / activeWindow into the
// tabs slice so we can switch away cleanly.
func (e *Editor) snapshotActiveTab() {
	wins := make([]Window, len(e.windows))
	copy(wins, e.windows)
	e.tabs[e.activeTab] = tabSnapshot{windows: wins, activeWindow: e.activeWindow}
}

// restoreTab pulls a tab's stashed window state back into the live
// fields. Caller is responsible for first having snapshotActiveTab'd.
func (e *Editor) restoreTab(idx int) {
	snap := e.tabs[idx]
	if len(snap.windows) == 0 {
		// First time visiting this tab — seed with a single window on
		// the same buffer the user just left. Mirrors :tabnew with no
		// argument.
		var bufIdx int
		if e.activeWindow < len(e.windows) && e.windows[e.activeWindow].Buffer < len(e.buffers) {
			bufIdx = e.windows[e.activeWindow].Buffer
		}
		e.windows = []Window{{Buffer: bufIdx, Cursor: Position{}}}
		e.activeWindow = 0
		return
	}
	e.windows = make([]Window, len(snap.windows))
	copy(e.windows, snap.windows)
	e.activeWindow = snap.activeWindow
}

// openNewTab creates a tab and switches to it. The new tab gets a single
// window on the active buffer, mimicking `:tabnew %`.
func (e *Editor) openNewTab() {
	e.snapshotActiveTab()
	e.tabs = append(e.tabs, tabSnapshot{})
	e.activeTab = len(e.tabs) - 1
	e.restoreTab(e.activeTab)
}

// switchTab moves to the tab at idx, wrapping like Vim's gt / gT.
func (e *Editor) switchTab(idx int) {
	if len(e.tabs) <= 1 {
		return
	}
	idx = ((idx % len(e.tabs)) + len(e.tabs)) % len(e.tabs)
	if idx == e.activeTab {
		return
	}
	e.snapshotActiveTab()
	e.activeTab = idx
	e.restoreTab(e.activeTab)
}

// closeActiveTab drops the current tab page. Last tab can't be closed.
func (e *Editor) closeActiveTab() bool {
	if len(e.tabs) <= 1 {
		return false
	}
	e.tabs = append(e.tabs[:e.activeTab], e.tabs[e.activeTab+1:]...)
	if e.activeTab >= len(e.tabs) {
		e.activeTab = len(e.tabs) - 1
	}
	e.restoreTab(e.activeTab)
	return true
}

// moveActiveWindow rearranges the windows slice for Ctrl-W H/J/K/L.
// direction: 'H' = move active to start, 'L' = end, 'J' = swap with
// next, 'K' = swap with previous. Real Vim shifts the layout
// orientation; the trainer doesn't model layout so we approximate with
// a slice reorder that the lesson view can render.
func (e *Editor) moveActiveWindow(direction rune) bool {
	if len(e.windows) <= 1 {
		return false
	}
	idx := e.activeWindow
	switch direction {
	case 'H':
		if idx == 0 {
			return false
		}
		win := e.windows[idx]
		e.windows = append(e.windows[:idx], e.windows[idx+1:]...)
		e.windows = append([]Window{win}, e.windows...)
		e.activeWindow = 0
		return true
	case 'L':
		if idx == len(e.windows)-1 {
			return false
		}
		win := e.windows[idx]
		e.windows = append(e.windows[:idx], e.windows[idx+1:]...)
		e.windows = append(e.windows, win)
		e.activeWindow = len(e.windows) - 1
		return true
	case 'J':
		if idx == len(e.windows)-1 {
			return false
		}
		e.windows[idx], e.windows[idx+1] = e.windows[idx+1], e.windows[idx]
		e.activeWindow = idx + 1
		return true
	case 'K':
		if idx == 0 {
			return false
		}
		e.windows[idx], e.windows[idx-1] = e.windows[idx-1], e.windows[idx]
		e.activeWindow = idx - 1
		return true
	}
	return false
}

func (e *Editor) bufferNext() {
	e.active().Buffer = (e.active().Buffer + 1) % len(e.buffers)
	e.active().Cursor = e.clampCursor(e.active().Buffer, e.active().Cursor)
}

func (e *Editor) bufferPrev() {
	e.active().Buffer--
	if e.active().Buffer < 0 {
		e.active().Buffer = len(e.buffers) - 1
	}
	e.active().Cursor = e.clampCursor(e.active().Buffer, e.active().Cursor)
}

func (e *Editor) consumeCount() int {
	if e.pendingCount == "" {
		return 1
	}
	count, err := strconv.Atoi(e.pendingCount)
	e.pendingCount = ""
	if err != nil || count <= 0 {
		return 1
	}
	return count
}

func (e *Editor) composeToken(token string, isChange bool) string {
	count := ""
	if e.pendingCount != "" {
		count = e.pendingCount
		e.pendingCount = ""
	}
	if isChange {
		return count + token
	}
	return count + token
}

func (e *Editor) recordHistory(token string) {
	e.commandHistory = append(e.commandHistory, token)
}

func (e *Editor) applyCountToMotion(fn func(count int), token string) {
	count := e.consumeCount()
	fn(count)
	e.recordHistory(withCount(count, token))
}

func cloneBuffers(in []Buffer) []Buffer {
	out := make([]Buffer, len(in))
	for i, buf := range in {
		lines := make([]string, len(buf.Lines))
		copy(lines, buf.Lines)
		out[i] = Buffer{Name: buf.Name, Lines: lines}
	}
	return out
}

func cloneWindows(in []Window) []Window {
	out := make([]Window, len(in))
	copy(out, in)
	return out
}

// defaultBufferLines returns the placeholder content for a freshly-opened
// buffer when a lesson didn't pre-populate it. Vim opens new files as a
// single empty line; the engine matches that. Lessons that need specific
// content must declare it in Initial.Buffers.
func defaultBufferLines(name string) []string {
	_ = name // intentionally ignored — engine no longer fabricates per-name content
	return []string{""}
}

func (e *Editor) openTerminalBuffer() {
	// The trainer simulates :terminal as an opaque buffer; the actual
	// content is intentionally generic so no lesson can rely on a magic
	// string. The "terminal://" prefix is the bit lessons key off.
	name := "terminal://zsh"
	for i, buf := range e.buffers {
		if buf.Name == name {
			e.active().Buffer = i
			e.active().Cursor = Position{Row: 0, Col: 0}
			return
		}
	}
	e.buffers = append(e.buffers, Buffer{
		Name:  name,
		Lines: []string{"$"},
	})
	e.active().Buffer = len(e.buffers) - 1
	e.active().Cursor = Position{Row: 0, Col: 1}
}

// normalSpec describes a parsed `:[range]normal[!] keys` command. The
// keys field is the raw key sequence (no remap when bang). The trainer
// runs it as a literal key stream against the engine.
type normalSpec struct {
	rangeSpec string
	keys      string
}

func parseNormalCommand(cmd string) (normalSpec, bool) {
	rangeStr, rest := splitRangeAndCommand(cmd)
	if rest == "" {
		return normalSpec{}, false
	}
	switch {
	case strings.HasPrefix(rest, "normal!"):
		rest = rest[len("normal!"):]
	case strings.HasPrefix(rest, "normal"):
		// Need a non-letter after "normal" to avoid matching "normalize"
		// or future identifiers.
		if len(rest) > len("normal") && isLetterOrDigit(rest[len("normal")]) {
			return normalSpec{}, false
		}
		rest = rest[len("normal"):]
	case strings.HasPrefix(rest, "norm!"):
		rest = rest[len("norm!"):]
	case strings.HasPrefix(rest, "norm"):
		if len(rest) > len("norm") && isLetterOrDigit(rest[len("norm")]) {
			return normalSpec{}, false
		}
		rest = rest[len("norm"):]
	default:
		return normalSpec{}, false
	}
	rest = strings.TrimLeft(rest, " \t")
	return normalSpec{rangeSpec: rangeStr, keys: rest}, true
}

func (e *Editor) applyNormal(spec normalSpec, token string) ActionResult {
	if spec.keys == "" {
		return ActionResult{Key: "enter", Token: token, Error: "missing keys", Description: "use :normal <keys> or :%normal <keys>", Completed: true}
	}
	keys := tokenizeNormalKeys(spec.keys)
	run := func() {
		// Suppress macro recording while replaying so :normal keys don't
		// pollute an in-progress qa…q recording.
		prev := e.replaying
		e.replaying = true
		for _, k := range keys {
			e.processKey(k)
		}
		// Mirror Vim's :normal semantics: any leftover Insert / Visual
		// mode is force-closed at the end of the keystroke sequence.
		// Without this, `:%normal A!` enters Insert on the first line
		// and then types "A" / "!" literally on every subsequent line.
		if e.mode == ModeInsert {
			e.processKey("esc")
		} else if e.mode == ModeVisual {
			e.processKey("esc")
		}
		e.replaying = prev
	}
	if spec.rangeSpec == "" {
		run()
		return ActionResult{Key: "enter", Token: token, Description: "ran normal-mode keys at cursor", Changed: true, Completed: true}
	}
	startRow, endRow := e.resolveRange(spec.rangeSpec)
	rowsRan := 0
	for r := startRow; r <= endRow && r < len(e.currentBuffer().Lines); r++ {
		e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: r, Col: 0})
		run()
		rowsRan++
	}
	return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("ran normal-mode keys on %d line(s)", rowsRan), Changed: true, Completed: true}
}

// tokenizeNormalKeys turns "iHello\<Esc>" into ["i","H","e","l","l","o","esc"].
// Recognized angle-bracket forms: <Esc>, <CR>, <Enter>, <Tab>, <Space>,
// <BS>, <C-r>, <C-o>, <C-w>, <C-u>, <C-n>, <C-p>.
func tokenizeNormalKeys(s string) []string {
	var out []string
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		if runes[i] == '<' {
			j := i + 1
			for j < len(runes) && runes[j] != '>' {
				j++
			}
			if j < len(runes) {
				token := string(runes[i+1 : j])
				switch strings.ToLower(token) {
				case "esc":
					out = append(out, "esc")
				case "cr", "enter":
					out = append(out, "enter")
				case "tab":
					out = append(out, "tab")
				case "space":
					out = append(out, " ")
				case "bs", "backspace":
					out = append(out, "backspace")
				case "c-r":
					out = append(out, "ctrl+r")
				case "c-o":
					out = append(out, "ctrl+o")
				case "c-w":
					out = append(out, "ctrl+w")
				case "c-u":
					out = append(out, "ctrl+u")
				case "c-n":
					out = append(out, "ctrl+n")
				case "c-p":
					out = append(out, "ctrl+p")
				case "c-i":
					out = append(out, "ctrl+i")
				case "c-v":
					out = append(out, "ctrl+v")
				default:
					// Unknown angle form — pass it through verbatim so
					// the engine reports an "unsupported key" error
					// rather than silently swallowing it.
					out = append(out, "<"+token+">")
				}
				i = j
				continue
			}
		}
		out = append(out, string(runes[i]))
	}
	return out
}

// executeSpec describes a parsed `:execute "..."` command. The trainer's
// implementation handles only string-literal arguments — the most common
// shape in real Vim configs (e.g. `:exe "normal " . count . "j"`) — but
// recognizes the keyword so it doesn't 404 in lessons.
type executeSpec struct {
	body string
}

func parseExecuteCommand(cmd string) (executeSpec, bool) {
	rest := strings.TrimLeft(cmd, " \t")
	switch {
	case strings.HasPrefix(rest, "execute"):
		rest = rest[len("execute"):]
	case strings.HasPrefix(rest, "exe"):
		if len(rest) > len("exe") && isLetterOrDigit(rest[len("exe")]) {
			return executeSpec{}, false
		}
		rest = rest[len("exe"):]
	default:
		return executeSpec{}, false
	}
	rest = strings.TrimLeft(rest, " \t")
	return executeSpec{body: rest}, true
}

func (e *Editor) applyExecute(spec executeSpec, token string) ActionResult {
	if spec.body == "" {
		return ActionResult{Key: "enter", Token: token, Error: "missing argument", Description: "use :execute \"...\"", Completed: true}
	}
	// Strip outer quotes if present.
	cmd := trimVimQuotes(spec.body)
	// Recurse by handing the unquoted command back to executeCommand —
	// this is a small lie about Vim's actual concatenation semantics but
	// covers the most common :execute shape (a single quoted command).
	return e.executeCommand(cmd)
}

// substituteSpec describes a parsed :[range]s/pattern/replacement/flags command.
type substituteSpec struct {
	rangeSpec   string
	pattern     string
	replacement string
	flags       string
}

// parseSubstituteCommand returns a parsed spec if cmd is a :s-style command.
func parseSubstituteCommand(cmd string) (substituteSpec, bool) {
	rangeStr, rest := splitRangeAndCommand(cmd)
	if rest == "" {
		return substituteSpec{}, false
	}
	// rest must start with "s" then a delimiter (typically /).
	if rest[0] != 's' {
		return substituteSpec{}, false
	}
	if len(rest) < 2 {
		return substituteSpec{}, false
	}
	// Reject :set, :source, :split etc. by requiring a non-letter after `s`.
	delim := rest[1]
	if (delim >= 'a' && delim <= 'z') || (delim >= 'A' && delim <= 'Z') || (delim >= '0' && delim <= '9') {
		return substituteSpec{}, false
	}
	body := rest[2:]
	parts := splitDelimited(body, rune(delim), 3)
	if len(parts) < 2 {
		return substituteSpec{}, false
	}
	spec := substituteSpec{rangeSpec: rangeStr, pattern: parts[0]}
	if len(parts) >= 2 {
		spec.replacement = parts[1]
	}
	if len(parts) >= 3 {
		spec.flags = parts[2]
	}
	return spec, true
}

// globalSpec describes a parsed :[range]g/pattern/cmd command.
type globalSpec struct {
	rangeSpec string
	pattern   string
	command   string
	invert    bool
}

func parseGlobalCommand(cmd string) (globalSpec, bool) {
	rangeStr, rest := splitRangeAndCommand(cmd)
	if rest == "" {
		return globalSpec{}, false
	}
	invert := false
	switch {
	case strings.HasPrefix(rest, "global!"):
		invert = true
		rest = rest[len("global!"):]
	case strings.HasPrefix(rest, "global"):
		rest = rest[len("global"):]
	case strings.HasPrefix(rest, "g!"):
		invert = true
		rest = rest[2:]
	case strings.HasPrefix(rest, "g") && len(rest) >= 2 && !isLetterOrDigit(rest[1]):
		rest = rest[1:]
	case strings.HasPrefix(rest, "v") && len(rest) >= 2 && !isLetterOrDigit(rest[1]):
		invert = true
		rest = rest[1:]
	default:
		return globalSpec{}, false
	}
	if rest == "" {
		return globalSpec{}, false
	}
	delim := rest[0]
	body := rest[1:]
	parts := splitDelimited(body, rune(delim), 2)
	if len(parts) < 1 {
		return globalSpec{}, false
	}
	spec := globalSpec{rangeSpec: rangeStr, pattern: parts[0], invert: invert}
	if len(parts) >= 2 {
		spec.command = strings.TrimSpace(parts[1])
	}
	return spec, true
}

// splitRangeAndCommand peels a :range prefix off cmd. Supported forms:
//
//	%   .   $   N   N,M   .,$   .,+5
func splitRangeAndCommand(cmd string) (string, string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return "", ""
	}
	if cmd[0] == '%' {
		return "%", strings.TrimLeft(cmd[1:], " \t")
	}
	i := 0
	allowed := func(b byte) bool {
		return (b >= '0' && b <= '9') || b == ',' || b == '.' || b == '$' || b == '+' || b == '-' || b == '\'' || b == ';'
	}
	for i < len(cmd) && allowed(cmd[i]) {
		i++
	}
	if i == 0 {
		return "", cmd
	}
	return cmd[:i], strings.TrimLeft(cmd[i:], " \t")
}

// splitDelimited splits on delim up to limit parts, treating an escaped
// delimiter (e.g. "\/") as part of the segment. Trailing flags after the
// final delimiter become the last part.
func splitDelimited(s string, delim rune, limit int) []string {
	var parts []string
	var current strings.Builder
	escaped := false
	for _, r := range s {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			current.WriteRune(r)
			continue
		}
		if r == delim {
			parts = append(parts, current.String())
			current.Reset()
			if len(parts) == limit-1 {
				continue
			}
			continue
		}
		current.WriteRune(r)
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func isLetterOrDigit(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// resolveRange returns the inclusive [start, end] line range (0-indexed)
// described by the rangeSpec, clamped to the buffer. An empty rangeSpec
// targets only the current line.
func (e *Editor) resolveRange(rangeSpec string) (int, int) {
	lines := e.currentBuffer().Lines
	last := len(lines) - 1
	if last < 0 {
		last = 0
	}
	cur := e.currentCursor().Row
	if rangeSpec == "" {
		return cur, cur
	}
	if rangeSpec == "%" {
		return 0, last
	}
	resolveOne := func(token string, fallback int) int {
		token = strings.TrimSpace(token)
		if token == "" {
			return fallback
		}
		if token == "." {
			return cur
		}
		if token == "$" {
			return last
		}
		if strings.HasPrefix(token, ".") {
			delta, err := strconv.Atoi(strings.TrimPrefix(token, "."))
			if err == nil {
				return clamp(0, cur+delta, last)
			}
		}
		if strings.HasPrefix(token, "+") || strings.HasPrefix(token, "-") {
			delta, err := strconv.Atoi(token)
			if err == nil {
				return clamp(0, cur+delta, last)
			}
		}
		if n, err := strconv.Atoi(token); err == nil {
			return clamp(0, n-1, last) // ex ranges are 1-indexed
		}
		return fallback
	}
	parts := strings.SplitN(rangeSpec, ",", 2)
	start := resolveOne(parts[0], cur)
	end := start
	if len(parts) == 2 {
		end = resolveOne(parts[1], start)
	}
	if start > end {
		start, end = end, start
	}
	return clamp(0, start, last), clamp(0, end, last)
}

// expandSubstituteReplacement turns a Vim replacement string into a Go
// replacement string. Vim uses \1..\9 for capture groups and & for the
// whole match; Go regexp.ReplaceAllString uses $1..$9 and ${name}.
func expandSubstituteReplacement(repl string) string {
	var out strings.Builder
	for i := 0; i < len(repl); i++ {
		c := repl[i]
		if c == '\\' && i+1 < len(repl) {
			next := repl[i+1]
			switch {
			case next >= '0' && next <= '9':
				out.WriteByte('$')
				out.WriteByte(next)
				i++
				continue
			case next == '&':
				out.WriteByte('&')
				i++
				continue
			case next == '\\':
				out.WriteByte('\\')
				i++
				continue
			case next == 'n':
				out.WriteByte('\n')
				i++
				continue
			}
		}
		if c == '&' {
			out.WriteString("${0}")
			continue
		}
		if c == '$' {
			out.WriteString("$$")
			continue
		}
		out.WriteByte(c)
	}
	return out.String()
}

func (e *Editor) applySubstitute(spec substituteSpec, token string) ActionResult {
	if spec.pattern == "" {
		return ActionResult{Key: "enter", Token: token, Error: "empty pattern", Description: "use :s/pattern/replacement/", Completed: true}
	}
	re := e.compileSearch(spec.pattern)
	if re == nil {
		return ActionResult{Key: "enter", Token: token, Error: "invalid pattern", Description: "could not compile :s pattern", Completed: true}
	}
	replacement := expandSubstituteReplacement(spec.replacement)
	flags := spec.flags
	global := strings.Contains(flags, "g")
	confirm := strings.Contains(flags, "c")
	startRow, endRow := e.resolveRange(spec.rangeSpec)
	lines := e.currentBuffer().Lines
	if endRow >= len(lines) {
		endRow = len(lines) - 1
	}

	if confirm {
		return e.startSubstituteConfirm(re, replacement, startRow, endRow, global, token)
	}

	totalReplacements := 0
	changed := false
	for r := startRow; r <= endRow && r < len(lines); r++ {
		line := lines[r]
		var newLine string
		if global {
			newLine = re.ReplaceAllString(line, replacement)
		} else {
			replaced := false
			newLine = re.ReplaceAllStringFunc(line, func(match string) string {
				if replaced {
					return match
				}
				replaced = true
				return re.ReplaceAllString(match, replacement)
			})
		}
		if newLine != line {
			countMatches := len(re.FindAllStringIndex(line, -1))
			if !global && countMatches > 0 {
				countMatches = 1
			}
			totalReplacements += countMatches
			lines[r] = newLine
			changed = true
		}
	}
	e.currentBuffer().Lines = lines
	e.active().Cursor = e.clampCursor(e.active().Buffer, e.active().Cursor)
	if totalReplacements == 0 {
		return ActionResult{Key: "enter", Token: token, Error: "pattern not found", Description: fmt.Sprintf("no matches for %q", spec.pattern), Completed: true}
	}
	desc := fmt.Sprintf("substituted %d occurrence(s)", totalReplacements)
	return ActionResult{Key: "enter", Token: token, Description: desc, Changed: changed, Completed: true}
}

// startSubstituteConfirm precomputes every match in the requested range and
// transitions into ModeConfirm, where each match is presented to the user
// for y/n/a/q decision.
func (e *Editor) startSubstituteConfirm(re *regexp.Regexp, replacement string, startRow, endRow int, global bool, token string) ActionResult {
	lines := e.currentBuffer().Lines
	var matches []confirmMatch
	for r := startRow; r <= endRow && r < len(lines); r++ {
		runes := []rune(lines[r])
		if global {
			offset := 0
			for offset <= len(runes) {
				loc := matchAfter(re, runes, offset)
				if loc == nil {
					break
				}
				matches = append(matches, confirmMatch{Row: r, Start: loc[0], End: loc[1]})
				if loc[1] == offset {
					offset++
				} else {
					offset = loc[1]
				}
			}
		} else {
			loc := matchAfter(re, runes, 0)
			if loc != nil {
				matches = append(matches, confirmMatch{Row: r, Start: loc[0], End: loc[1]})
			}
		}
	}
	if len(matches) == 0 {
		return ActionResult{Key: "enter", Token: token, Error: "pattern not found", Description: "no matches for that pattern", Completed: true}
	}
	e.confirm = &substituteConfirmState{
		re:            re,
		replacement:   replacement,
		matches:       matches,
		originalToken: token,
	}
	e.beginChange(token)
	e.changeTokens = []string{token}
	e.mode = ModeConfirm
	first := matches[0]
	e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: first.Row, Col: first.Start})
	return ActionResult{Key: "enter", Token: token, Description: e.confirmPrompt(), Completed: true}
}

func (e *Editor) handleConfirm(key string) ActionResult {
	if e.confirm == nil {
		e.mode = ModeNormal
		return ActionResult{Key: key, Token: key, Description: "confirm prompt closed", Completed: true}
	}
	switch key {
	case "y":
		e.applyCurrentConfirmMatch()
		return e.advanceConfirm("y")
	case "n":
		return e.advanceConfirm("n")
	case "a":
		// Apply current and every remaining match.
		for e.confirm != nil && e.confirm.index < len(e.confirm.matches) {
			e.applyCurrentConfirmMatch()
			e.confirm.index++
		}
		return e.finishConfirm("a")
	case "q", "esc":
		return e.finishConfirm(key)
	default:
		return ActionResult{Key: key, Token: key, Description: "press y to replace, n to skip, a for all, q to quit", Completed: true}
	}
}

func (e *Editor) applyCurrentConfirmMatch() {
	cs := e.confirm
	if cs == nil || cs.index >= len(cs.matches) {
		return
	}
	m := cs.matches[cs.index]
	lines := e.currentBuffer().Lines
	if m.Row >= len(lines) {
		return
	}
	runes := []rune(lines[m.Row])
	if m.Start < 0 || m.End > len(runes) || m.Start > m.End {
		return
	}
	matched := string(runes[m.Start:m.End])
	replaced := cs.re.ReplaceAllString(matched, cs.replacement)
	newRunes := make([]rune, 0, len(runes)+len([]rune(replaced))-(m.End-m.Start))
	newRunes = append(newRunes, runes[:m.Start]...)
	newRunes = append(newRunes, []rune(replaced)...)
	newRunes = append(newRunes, runes[m.End:]...)
	lines[m.Row] = string(newRunes)
	e.currentBuffer().Lines = lines
	delta := len([]rune(replaced)) - (m.End - m.Start)
	for i := cs.index + 1; i < len(cs.matches); i++ {
		if cs.matches[i].Row == m.Row {
			cs.matches[i].Start += delta
			cs.matches[i].End += delta
		}
	}
	cs.replacedCount++
}

func (e *Editor) advanceConfirm(key string) ActionResult {
	cs := e.confirm
	if cs == nil {
		e.mode = ModeNormal
		return ActionResult{Key: key, Token: key, Description: "confirm closed", Completed: true}
	}
	cs.index++
	if cs.index >= len(cs.matches) {
		return e.finishConfirm(key)
	}
	next := cs.matches[cs.index]
	e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: next.Row, Col: next.Start})
	return ActionResult{Key: key, Token: key, Description: e.confirmPrompt(), Completed: true}
}

func (e *Editor) finishConfirm(key string) ActionResult {
	replaced := 0
	if e.confirm != nil {
		replaced = e.confirm.replacedCount
	}
	e.confirm = nil
	e.mode = ModeNormal
	e.finishChange(":s confirm")
	return ActionResult{Key: key, Token: "confirm", Description: fmt.Sprintf("substitute applied to %d match(es)", replaced), Completed: true, Changed: replaced > 0}
}

func (e *Editor) confirmPrompt() string {
	cs := e.confirm
	if cs == nil || cs.index >= len(cs.matches) {
		return ""
	}
	m := cs.matches[cs.index]
	if e.activeWindow >= len(e.windows) {
		return ""
	}
	bufIdx := e.windows[e.activeWindow].Buffer
	if bufIdx >= len(e.buffers) {
		return ""
	}
	runes := []rune(e.buffers[bufIdx].Lines[m.Row])
	if m.Start < 0 || m.End > len(runes) {
		return ""
	}
	matched := string(runes[m.Start:m.End])
	return fmt.Sprintf("replace %q? (y/n/a/q) [%d/%d]", matched, cs.index+1, len(cs.matches))
}

// confirmContext returns the full source line of the current match, so
// the UI can render the line with the match underlined or highlighted
// rather than just echoing the matched substring.
func (e *Editor) confirmContext() string {
	cs := e.confirm
	if cs == nil || cs.index >= len(cs.matches) {
		return ""
	}
	m := cs.matches[cs.index]
	if e.activeWindow >= len(e.windows) {
		return ""
	}
	bufIdx := e.windows[e.activeWindow].Buffer
	if bufIdx >= len(e.buffers) || m.Row >= len(e.buffers[bufIdx].Lines) {
		return ""
	}
	return e.buffers[bufIdx].Lines[m.Row]
}

func (e *Editor) confirmMatchStart() int {
	cs := e.confirm
	if cs == nil || cs.index >= len(cs.matches) {
		return 0
	}
	return cs.matches[cs.index].Start
}

func (e *Editor) confirmMatchEnd() int {
	cs := e.confirm
	if cs == nil || cs.index >= len(cs.matches) {
		return 0
	}
	return cs.matches[cs.index].End
}

func (e *Editor) applyGlobal(spec globalSpec, token string) ActionResult {
	if spec.pattern == "" {
		return ActionResult{Key: "enter", Token: token, Error: "empty pattern", Description: "use :g/pattern/cmd", Completed: true}
	}
	re := e.compileSearch(spec.pattern)
	if re == nil {
		return ActionResult{Key: "enter", Token: token, Error: "invalid pattern", Description: "could not compile :g pattern", Completed: true}
	}
	rangeSpec := spec.rangeSpec
	if rangeSpec == "" {
		rangeSpec = "%"
	}
	startRow, endRow := e.resolveRange(rangeSpec)
	lines := e.currentBuffer().Lines
	if endRow >= len(lines) {
		endRow = len(lines) - 1
	}
	command := strings.TrimSpace(spec.command)
	if command == "" {
		command = "p"
	}
	// Collect the lines that match (or don't, when inverted) before mutating.
	var targets []int
	for r := startRow; r <= endRow && r < len(lines); r++ {
		hit := re.MatchString(lines[r])
		if hit != spec.invert {
			targets = append(targets, r)
		}
	}
	if len(targets) == 0 {
		return ActionResult{Key: "enter", Token: token, Error: "no matches", Description: fmt.Sprintf("no lines matched %q", spec.pattern), Completed: true}
	}
	switch command {
	case "d":
		// Delete in reverse order to keep indices stable.
		for i := len(targets) - 1; i >= 0; i-- {
			row := targets[i]
			lines = append(lines[:row], lines[row+1:]...)
		}
		if len(lines) == 0 {
			lines = []string{""}
		}
		e.currentBuffer().Lines = lines
		e.active().Cursor = e.clampCursor(e.active().Buffer, e.active().Cursor)
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("deleted %d matching line(s)", len(targets)), Changed: true, Completed: true}
	case "p":
		var preview []string
		for _, r := range targets {
			preview = append(preview, lines[r])
		}
		e.lastEcho = strings.Join(preview, "\n")
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("matched %d line(s)", len(targets)), Completed: true}
	default:
		// :g/pat/normal <keys> — run the keys against each matching line.
		if n, ok := parseNormalCommand(command); ok {
			rowsRan := 0
			for _, r := range targets {
				e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: r, Col: 0})
				_ = e.applyNormal(normalSpec{keys: n.keys}, token)
				rowsRan++
			}
			return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("ran :normal across %d matching line(s)", rowsRan), Changed: true, Completed: true}
		}
		// Run a substitute against each target line if the command starts with s/.
		if sub, ok := parseSubstituteCommand(command); ok {
			beforeChanged := false
			total := 0
			for _, r := range targets {
				line := lines[r]
				re2 := e.compileSearch(sub.pattern)
				if re2 == nil {
					continue
				}
				replacement := expandSubstituteReplacement(sub.replacement)
				global := strings.Contains(sub.flags, "g")
				var newLine string
				if global {
					newLine = re2.ReplaceAllString(line, replacement)
				} else {
					replaced := false
					newLine = re2.ReplaceAllStringFunc(line, func(match string) string {
						if replaced {
							return match
						}
						replaced = true
						return re2.ReplaceAllString(match, replacement)
					})
				}
				if newLine != line {
					countMatches := len(re2.FindAllStringIndex(line, -1))
					if !global && countMatches > 0 {
						countMatches = 1
					}
					total += countMatches
					lines[r] = newLine
					beforeChanged = true
				}
			}
			e.currentBuffer().Lines = lines
			if total == 0 {
				return ActionResult{Key: "enter", Token: token, Error: "no replacements", Description: "global ran but no substitutions matched", Completed: true}
			}
			return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("global substituted %d occurrence(s)", total), Changed: beforeChanged, Completed: true}
		}
		return ActionResult{Key: "enter", Token: token, Error: "unsupported global command", Description: "trainer supports :g/pat/d, :g/pat/p, and :g/pat/s/from/to/g", Completed: true}
	}
}

func (e *Editor) applySedScript(rawScript string) (bool, error) {
	script := trimVimQuotes(strings.TrimSpace(rawScript))
	if script == "" {
		return false, fmt.Errorf("empty sed script")
	}

	if strings.HasPrefix(script, "s/") {
		parts := strings.SplitN(script, "/", 4)
		if len(parts) < 4 {
			return false, fmt.Errorf("use substitution form s/old/new/g")
		}
		old := parts[1]
		newValue := parts[2]
		flags := parts[3]
		if old == "" {
			return false, fmt.Errorf("sed substitution needs a non-empty pattern")
		}
		replaceAll := strings.Contains(flags, "g")
		changed := false
		for i, line := range e.currentBuffer().Lines {
			next := line
			if replaceAll {
				next = strings.ReplaceAll(line, old, newValue)
			} else {
				next = strings.Replace(line, old, newValue, 1)
			}
			if next != line {
				e.currentBuffer().Lines[i] = next
				changed = true
			}
		}
		return changed, nil
	}

	if strings.HasPrefix(script, "/") && strings.HasSuffix(script, "/d") {
		pattern := strings.TrimSuffix(strings.TrimPrefix(script, "/"), "/d")
		if pattern == "" {
			return false, fmt.Errorf("sed delete form needs a pattern like /TODO/d")
		}
		var out []string
		changed := false
		for _, line := range e.currentBuffer().Lines {
			if strings.Contains(line, pattern) {
				changed = true
				continue
			}
			out = append(out, line)
		}
		if len(out) == 0 {
			out = []string{""}
		}
		e.currentBuffer().Lines = out
		e.active().Cursor = e.clampCursor(e.active().Buffer, e.active().Cursor)
		return changed, nil
	}

	return false, fmt.Errorf("supported sed forms are s/old/new/g and /pattern/d")
}

// applyLua is the trainer's deliberately scoped Lua simulator. It is NOT
// a real Lua interpreter — it pattern-matches a small subset chosen to
// teach Neovim config muscle memory without dragging in a Lua VM.
//
// Supported forms:
//
//	vim.g.<name> = "value"             -- global variable assignment
//	print("text")                       -- echoes literal text
//	print(vim.g.<name>)                 -- echoes a previously-set global
//	vim.keymap.set("n", "<lhs>", "<rhs>") -- normal-mode mappings only
//
// Anything outside this subset returns LuaSubsetError so callers can show
// a learner-friendly message ("trainer supports only this slice — for
// real Lua, use Neovim itself").
func (e *Editor) applyLua(script string) (string, error) {
	code := strings.TrimSpace(script)
	if code == "" {
		return "", fmt.Errorf("lua code is empty")
	}

	if strings.HasPrefix(code, "vim.g.") && strings.Contains(code, "=") {
		assign := strings.SplitN(strings.TrimPrefix(code, "vim.g."), "=", 2)
		if len(assign) != 2 {
			return "", fmt.Errorf("invalid global assignment")
		}
		key := strings.TrimSpace(assign[0])
		value := trimVimQuotes(strings.TrimSpace(assign[1]))
		if key == "" {
			return "", fmt.Errorf("missing vim.g key")
		}
		e.variables["g:"+key] = value
		return "", nil
	}

	if strings.HasPrefix(code, "print(") && strings.HasSuffix(code, ")") {
		body := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(code, "print("), ")"))
		if strings.HasPrefix(body, "vim.g.") {
			key := strings.TrimSpace(strings.TrimPrefix(body, "vim.g."))
			if value, ok := e.variables["g:"+key]; ok {
				return value, nil
			}
			return "", fmt.Errorf("vim.g.%s is not set", key)
		}
		if strings.HasPrefix(body, "\"") || strings.HasPrefix(body, "'") {
			return trimVimQuotes(body), nil
		}
		return "", luaSubsetError("print supports quoted text and vim.g variables")
	}

	if strings.HasPrefix(code, "vim.keymap.set(") {
		inside := strings.TrimSuffix(strings.TrimPrefix(code, "vim.keymap.set("), ")")
		args := splitLuaArgs(inside)
		if len(args) < 3 {
			return "", fmt.Errorf("vim.keymap.set needs mode, lhs, rhs")
		}
		mode := trimVimQuotes(strings.TrimSpace(args[0]))
		lhs := trimVimQuotes(strings.TrimSpace(args[1]))
		rhs := trimVimQuotes(strings.TrimSpace(args[2]))
		if mode != "n" {
			return "", luaSubsetError("the trainer simulates only normal-mode keymaps; real Neovim supports all modes")
		}
		if lhs == "" || rhs == "" {
			return "", fmt.Errorf("keymap lhs/rhs cannot be empty")
		}
		e.normalMappings[lhs] = rhs
		return "", nil
	}

	return "", luaSubsetError("trainer's :lua supports vim.g assignment, print(...), and vim.keymap.set(\"n\", lhs, rhs)")
}

// luaSubsetError tags a parse failure with the standard "out of subset"
// preamble so learners aren't surprised when a real-Lua snippet bounces.
func luaSubsetError(detail string) error {
	return fmt.Errorf("trainer Lua is a scoped subset: %s. For full Lua, run real Neovim", detail)
}

func splitLuaArgs(raw string) []string {
	var args []string
	var current strings.Builder
	var quote rune
	for _, r := range raw {
		if quote != 0 {
			current.WriteRune(r)
			if r == quote {
				quote = 0
			}
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
			current.WriteRune(r)
		case ',':
			args = append(args, strings.TrimSpace(current.String()))
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, strings.TrimSpace(current.String()))
	}
	return args
}

func parseVimGrepPattern(cmd string) string {
	raw := strings.TrimSpace(strings.TrimPrefix(cmd, "vimgrep"))
	if raw == "" || raw[0] != '/' {
		return ""
	}
	end := strings.Index(raw[1:], "/")
	if end < 0 {
		return ""
	}
	pattern := strings.TrimSpace(raw[1 : end+1])
	return pattern
}

func parentPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "." {
		return ".."
	}
	if value == "~" {
		return "~"
	}
	clean := filepath.Clean(value)
	parent := filepath.Dir(clean)
	if parent == "." {
		return ".."
	}
	return parent
}

func parseLetCommand(cmd string) (name, value string, ok bool) {
	raw := strings.TrimSpace(strings.TrimPrefix(cmd, "let"))
	if raw == "" {
		return "", "", false
	}
	parts := strings.SplitN(raw, "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	name = strings.TrimSpace(parts[0])
	value = strings.TrimSpace(parts[1])
	if name == "" || value == "" {
		return "", "", false
	}
	value = trimVimQuotes(value)
	return name, value, true
}

func parseMapCommand(cmd string) (lhs, rhs string, ok bool) {
	fields := strings.Fields(cmd)
	if len(fields) < 3 {
		return "", "", false
	}
	lhs = fields[1]
	rhs = strings.TrimSpace(strings.Join(fields[2:], " "))
	if lhs == "" || rhs == "" {
		return "", "", false
	}
	return lhs, rhs, true
}

func trimVimQuotes(value string) string {
	if len(value) >= 2 {
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) || (strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func (e *Editor) evaluateExpression(expr string) (string, bool) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return "", false
	}
	if strings.HasPrefix(expr, "\"") || strings.HasPrefix(expr, "'") {
		return trimVimQuotes(expr), true
	}
	if value, ok := e.variables[expr]; ok {
		return value, true
	}
	if strings.HasPrefix(expr, "g:") {
		if value, ok := e.variables[expr]; ok {
			return value, true
		}
	}
	if expr == "&number" {
		if e.options.Number {
			return "1", true
		}
		return "0", true
	}
	return "", false
}

func shouldRecordKey(key string, pendingMacroQ bool, token string) bool {
	if pendingMacroQ && len([]rune(key)) == 1 {
		return false
	}
	return key != ""
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-'
}

func firstRune(s string) rune {
	runes := []rune(s)
	if len(runes) == 0 {
		return 0
	}
	return runes[0]
}

func withCount(count int, token string) string {
	if count <= 1 {
		return token
	}
	return strconv.Itoa(count) + token
}

// surroundPair maps a surround-character to its open/close pair.
// Vim-surround treats brackets idiomatically: the "inside" form (without
// padding) is keyed by the closing char; the "outside" form (with a leading
// space inside the pair) is keyed by the opening char.
func surroundPair(ch rune) (rune, rune, bool, bool) {
	switch ch {
	case '"':
		return '"', '"', false, true
	case '\'':
		return '\'', '\'', false, true
	case '`':
		return '`', '`', false, true
	case '(':
		return '(', ')', true, true
	case ')', 'b':
		return '(', ')', false, true
	case '[':
		return '[', ']', true, true
	case ']':
		return '[', ']', false, true
	case '{':
		return '{', '}', true, true
	case '}', 'B':
		return '{', '}', false, true
	case '<':
		return '<', '>', true, true
	case '>':
		return '<', '>', false, true
	}
	return 0, 0, false, false
}

func (e *Editor) replaceCharUnderCursor(r rune) bool {
	cur := e.currentCursor()
	runes := e.lineRunes()
	if len(runes) == 0 || cur.Col >= len(runes) {
		return false
	}
	runes[cur.Col] = r
	e.setCurrentLine(string(runes))
	return true
}

// handleReplace implements R (Replace mode): each printable key overwrites
// the character under the cursor and advances. Backspace moves the cursor
// back without truly restoring (a faithful implementation tracks original
// chars; the trainer keeps it simple). Esc returns to Normal.
func (e *Editor) handleReplace(key string) ActionResult {
	switch key {
	case "esc":
		e.mode = ModeNormal
		e.finishChange("R")
		return ActionResult{Key: key, Token: "esc", Description: "returned to normal mode", Completed: true}
	case "backspace":
		cur := e.active()
		if cur.Cursor.Col > 0 {
			cur.Cursor.Col--
		}
		e.changeTokens = append(e.changeTokens, "backspace")
		return ActionResult{Key: key, Token: "backspace", Description: "moved replace cursor back"}
	default:
		if len([]rune(key)) == 1 {
			r := []rune(key)[0]
			cur := e.active()
			runes := e.lineRunes()
			if cur.Cursor.Col < len(runes) {
				runes[cur.Cursor.Col] = r
				e.setCurrentLine(string(runes))
			} else {
				runes = append(runes, r)
				e.setCurrentLine(string(runes))
			}
			cur.Cursor.Col++
			e.changeTokens = append(e.changeTokens, key)
			return ActionResult{Key: key, Token: key, Description: fmt.Sprintf("replaced with %q", key), Changed: true}
		}
		return ActionResult{Key: key, Token: key, Error: "unsupported replace-mode key", Description: "type a character or press Esc", Completed: true}
	}
}

// deleteSurround removes the matching open/close pair around the cursor.
func (e *Editor) deleteSurround(ch rune) bool {
	open, close, _, ok := surroundPair(ch)
	if !ok {
		return false
	}
	var sr, sc, er, ec int
	var found bool
	if open == close {
		sr, sc, er, ec, _, found = e.matchedQuoteObjectRange(open, true)
	} else {
		sr, sc, er, ec, _, found = e.pairObjectRange(open, close, true)
	}
	if !found {
		return false
	}
	lines := e.currentBuffer().Lines
	if sr != er {
		// Remove the open char at (sr, sc) and the close char at (er, ec-1).
		first := []rune(lines[sr])
		last := []rune(lines[er])
		first = append(first[:sc], first[sc+1:]...)
		last = append(last[:ec-1], last[ec:]...)
		lines[sr] = string(first)
		lines[er] = string(last)
	} else {
		runes := []rune(lines[sr])
		// Remove close first (higher index), then open.
		runes = append(runes[:ec-1], runes[ec:]...)
		runes = append(runes[:sc], runes[sc+1:]...)
		lines[sr] = string(runes)
	}
	e.currentBuffer().Lines = lines
	e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: sr, Col: sc})
	return true
}

// changeSurround swaps the surround characters around the cursor.
func (e *Editor) changeSurround(oldCh, newCh rune) bool {
	openOld, closeOld, _, ok := surroundPair(oldCh)
	if !ok {
		return false
	}
	openNew, closeNew, addPadding, okNew := surroundPair(newCh)
	if !okNew {
		return false
	}
	var sr, sc, er, ec int
	var found bool
	if openOld == closeOld {
		sr, sc, er, ec, _, found = e.matchedQuoteObjectRange(openOld, true)
	} else {
		sr, sc, er, ec, _, found = e.pairObjectRange(openOld, closeOld, true)
	}
	if !found {
		return false
	}
	lines := e.currentBuffer().Lines
	openStr := string(openNew)
	closeStr := string(closeNew)
	if addPadding {
		// vim-surround uses padding only for the open form of brackets when
		// the user opted in (e.g. `(` vs `)`). The trainer keeps it simple
		// and never inserts padding to avoid surprising existing tests.
		_ = addPadding
	}
	if sr != er {
		first := []rune(lines[sr])
		last := []rune(lines[er])
		first = append(first[:sc], append([]rune(openStr), first[sc+1:]...)...)
		last = append(last[:ec-1], append([]rune(closeStr), last[ec:]...)...)
		lines[sr] = string(first)
		lines[er] = string(last)
	} else {
		runes := []rune(lines[sr])
		// Replace close first (higher index), then open.
		runes = append(runes[:ec-1], append([]rune(closeStr), runes[ec:]...)...)
		runes = append(runes[:sc], append([]rune(openStr), runes[sc+1:]...)...)
		lines[sr] = string(runes)
	}
	e.currentBuffer().Lines = lines
	e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: sr, Col: sc})
	return true
}

// surroundLine wraps the current line with the given surround character.
func (e *Editor) surroundLine(ch rune) bool {
	open, close, _, ok := surroundPair(ch)
	if !ok {
		return false
	}
	cur := e.currentCursor()
	lines := e.currentBuffer().Lines
	if cur.Row >= len(lines) {
		return false
	}
	lines[cur.Row] = string(open) + lines[cur.Row] + string(close)
	e.currentBuffer().Lines = lines
	e.active().Cursor = e.clampCursor(e.active().Buffer, e.active().Cursor)
	return true
}

// surroundTextObject wraps a text-object range with the given pair.
// motion is 'i' (inner) or 'a' (around); object is the text-object char.
func (e *Editor) surroundTextObject(motion, object, ch rune) bool {
	open, close, _, ok := surroundPair(ch)
	if !ok {
		return false
	}
	sr, sc, er, ec, linewise, found := e.textObjectRange(motion, object)
	if !found {
		return false
	}
	if linewise {
		// Wrap each line individually with open/close on first/last lines.
		lines := e.currentBuffer().Lines
		if sr >= len(lines) || er >= len(lines) {
			return false
		}
		lines[sr] = string(open) + lines[sr]
		lines[er] = lines[er] + string(close)
		e.currentBuffer().Lines = lines
		e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: sr, Col: 0})
		return true
	}
	if sr != er {
		// Multi-line non-linewise: wrap by inserting close at (er,ec) and
		// open at (sr,sc). Apply close first so sc indices stay valid.
		lines := e.currentBuffer().Lines
		last := []rune(lines[er])
		ec = clamp(0, ec, len(last))
		last = append(last[:ec], append([]rune{close}, last[ec:]...)...)
		lines[er] = string(last)
		first := []rune(lines[sr])
		sc = clamp(0, sc, len(first))
		first = append(first[:sc], append([]rune{open}, first[sc:]...)...)
		lines[sr] = string(first)
		e.currentBuffer().Lines = lines
		e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: sr, Col: sc})
		return true
	}
	runes := []rune(e.currentBuffer().Lines[sr])
	sc = clamp(0, sc, len(runes))
	ec = clamp(sc, ec, len(runes))
	wrapped := append([]rune{}, runes[:sc]...)
	wrapped = append(wrapped, open)
	wrapped = append(wrapped, runes[sc:ec]...)
	wrapped = append(wrapped, close)
	wrapped = append(wrapped, runes[ec:]...)
	e.currentBuffer().Lines[sr] = string(wrapped)
	e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: sr, Col: sc})
	return true
}

func clamp(minValue, value, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
