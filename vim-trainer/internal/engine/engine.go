package engine

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"
)

type Mode string

const (
	ModeNormal   Mode = "NORMAL"
	ModeInsert   Mode = "INSERT"
	ModeVisual   Mode = "VISUAL"
	ModeCommand  Mode = "COMMAND"
	ModeSearch   Mode = "SEARCH"
	ModeExplorer Mode = "EXPLORER"
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
	pendingMacroAt    bool
	pendingMacroQ     bool
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
	jumpList          []jumpEntry
	jumpIndex         int
	changeList        []jumpEntry
	changeIndex       int
	quickfix          []QuickfixItem
	quickfixIndex     int
	quickfixOpen      bool
	explorerPath      string
	profileActive     bool
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
	}
	if s.StartingMode != "" {
		e.mode = s.StartingMode
	}
	for mark, pos := range s.Marks {
		e.marks[mark] = pos
	}
	e.normalizeWindows()
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
	}
}

func (e *Editor) ProcessKey(key string) ActionResult {
	res := e.processKey(key)
	if e.recordingRegister != 0 && !e.replaying && shouldRecordKey(key, e.pendingMacroQ, res.Token) {
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
	case ModeCommand:
		return e.handlePrompt(key, true)
	case ModeSearch:
		return e.handlePrompt(key, false)
	case ModeExplorer:
		return e.handleExplorer(key)
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
		e.openOrSwitchBuffer("README.md")
		e.mode = ModeNormal
		e.explorerOpen = false
		return ActionResult{Key: key, Token: "enter", Description: "opened selected file from explorer", Completed: true}
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
	switch key {
	case "esc":
		e.mode = ModeNormal
		e.finishChange("insert")
		return ActionResult{Key: key, Token: "esc", Description: "returned to normal mode", Completed: true}
	case "backspace":
		if e.backspaceInsert() {
			e.changeTokens = append(e.changeTokens, "backspace")
			return ActionResult{Key: key, Token: "backspace", Description: "deleted inserted character", Changed: true}
		}
		return ActionResult{Key: key, Token: "backspace", Description: "nothing to delete"}
	default:
		if len([]rune(key)) == 1 {
			e.insertRune([]rune(key)[0])
			e.changeTokens = append(e.changeTokens, key)
			return ActionResult{Key: key, Token: key, Description: fmt.Sprintf("inserted %q", key), Changed: true}
		}
		return ActionResult{Key: key, Token: key, Error: "unsupported insert-mode key", Description: "type text or press Esc", Completed: true}
	}
}

func (e *Editor) handleVisual(key string) ActionResult {
	switch key {
	case "esc":
		e.mode = ModeNormal
		e.visualLine = false
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
		e.mode = ModeNormal
		e.visualLine = false
		e.writeTextRegister(deleted, true)
		e.finishChange("vd")
		return ActionResult{Key: key, Token: "vd", Description: "deleted visual selection", Changed: true, Completed: true}
	case "y":
		yanked, ok := e.yankVisualSelection()
		if !ok {
			return ActionResult{Key: key, Token: "y", Error: "empty visual selection", Description: "expand the visual selection before yanking", Completed: true}
		}
		e.mode = ModeNormal
		e.visualLine = false
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
		e.mode = ModeNormal
		e.visualLine = false
		e.finishChange(token)
		return ActionResult{Key: key, Token: token, Description: "replaced selection with register text", Changed: true, Completed: true}
	default:
		return ActionResult{Key: key, Token: key, Error: "unsupported visual-mode key", Description: "the trainer supports h/j/k/l/w/b/e/0/$ plus d, y, p, and Esc in visual mode", Completed: true}
	}
}

func (e *Editor) handleNormal(key string) ActionResult {
	if e.pendingMacroQ {
		e.pendingMacroQ = false
		if len([]rune(key)) == 1 {
			e.recordingRegister = []rune(key)[0]
			e.currentMacro = nil
			return ActionResult{Key: key, Token: "q" + key, Description: fmt.Sprintf("recording macro in register %s", key), Completed: true}
		}
		return ActionResult{Key: key, Token: "q", Error: "macro register must be a letter", Description: "press q followed by a register like a", Completed: true}
	}
	if e.pendingMacroAt {
		e.pendingMacroAt = false
		if len([]rune(key)) == 1 {
			reg := []rune(key)[0]
			tokens := e.macroRegisters[reg]
			if len(tokens) == 0 {
				return ActionResult{Key: key, Token: "@" + key, Error: "macro register is empty", Description: fmt.Sprintf("register %s has no recorded macro", key), Completed: true}
			}
			return e.replay(tokens, "@"+key)
		}
		return ActionResult{Key: key, Token: "@", Error: "macro register must be a letter", Description: "press @ followed by a register like a", Completed: true}
	}
	if e.pendingRegister {
		e.pendingRegister = false
		if len([]rune(key)) != 1 {
			return ActionResult{Key: key, Token: "\"", Error: "register name must be one character", Description: "use \" followed by a register like a or 0", Completed: true}
		}
		reg := []rune(key)[0]
		if unicode.IsLetter(reg) || unicode.IsDigit(reg) || reg == '"' || reg == '-' {
			e.activeRegister = reg
			token := "\"" + key
			e.recordHistory(token)
			return ActionResult{Key: key, Token: token, Description: fmt.Sprintf("using register %s for the next operation", key), Completed: true}
		}
		return ActionResult{Key: key, Token: "\""+key, Error: "unsupported register", Description: "supported register names are letters, digits, \" and -", Completed: true}
	}
	if e.pendingMarkSet {
		e.pendingMarkSet = false
		if len([]rune(key)) == 1 {
			e.marks[[]rune(key)[0]] = e.currentCursor()
			token := "m" + key
			e.recordHistory(token)
			return ActionResult{Key: key, Token: token, Description: fmt.Sprintf("set mark %s", key), Completed: true}
		}
		return ActionResult{Key: key, Token: "m", Error: "marks require a letter", Description: "press m followed by a mark name like a", Completed: true}
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
		default:
			return ActionResult{Key: key, Token: "<C-w>" + key, Error: "unsupported window command", Description: "the trainer supports <C-w>w for cycling windows", Completed: true}
		}
	}
	if e.pendingPrefix == "g" {
		e.pendingPrefix = ""
		switch key {
		case "g":
			e.pushJump()
			e.applyCountToMotion(func(count int) { e.moveFileTop() }, "gg")
			return ActionResult{Key: key, Token: "gg", Description: "jumped to the top of the file", Completed: true}
		case "d":
			if !e.goToDefinition() {
				return ActionResult{Key: key, Token: "gd", Error: "definition not found", Description: "no obvious definition match found for the symbol under cursor", Completed: true}
			}
			e.recordHistory("gd")
			return ActionResult{Key: key, Token: "gd", Description: "jumped to a definition match", Completed: true}
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
		default:
			e.pendingOperator = 0
			return ActionResult{Key: key, Token: token, Error: "text objects need d, c, or y", Description: "use a text object with an operator like diw, ci\", or yap", Completed: true}
		}
	}
	if e.pendingOperator != 0 {
		op := e.pendingOperator
		switch key {
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
		e.visualStart = e.currentCursor()
		e.recordHistory("v")
		return ActionResult{Key: key, Token: "v", Description: "entered visual mode", Completed: true}
	case "V":
		e.mode = ModeVisual
		e.visualLine = true
		e.visualStart = e.currentCursor()
		e.recordHistory("V")
		return ActionResult{Key: key, Token: "V", Description: "entered visual-line mode", Completed: true}
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
			e.macroRegisters[e.recordingRegister] = append([]string{}, e.currentMacro...)
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
	case "p":
		if !e.pasteAfter() {
			return ActionResult{Key: key, Token: "p", Description: "nothing to paste", Completed: true}
		}
		token := e.composeToken("p", true)
		e.recordHistory(token)
		return ActionResult{Key: key, Token: token, Description: "pasted yanked text", Changed: true, Completed: true}
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
			default:
				return ActionResult{Key: "enter", Token: token, Error: "unsupported option", Description: "supported options include number, relativenumber, hlsearch, ignorecase, smartcase, incsearch, and wrap", Completed: true}
			}
		}
		return ActionResult{Key: "enter", Token: token, Description: "updated editor options", Completed: true}
	case "noh", "nohlsearch":
		e.options.HLSearch = false
		return ActionResult{Key: "enter", Token: token, Description: "cleared search highlighting", Completed: true}
	case "w":
		return ActionResult{Key: "enter", Token: token, Description: "wrote the buffer (simulated)", Completed: true}
	case "q":
		return ActionResult{Key: "enter", Token: token, Description: "quit requested", Completed: true}
	case "wq":
		return ActionResult{Key: "enter", Token: token, Description: "write and quit requested", Completed: true}
	case "wa", "wall":
		return ActionResult{Key: "enter", Token: token, Description: "wrote all buffers (simulated)", Completed: true}
	case "qa", "qall":
		return ActionResult{Key: "enter", Token: token, Description: "quit all requested", Completed: true}
	case "qa!", "qall!":
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
	case "split", "vsplit":
		e.openSplit()
		return ActionResult{Key: "enter", Token: token, Description: "opened a split window", Completed: true}
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
	case "nunmap", "unmap":
		if len(fields) < 2 {
			return ActionResult{Key: "enter", Token: token, Error: "missing lhs", Description: "use :unmap <lhs>", Completed: true}
		}
		delete(e.normalMappings, fields[1])
		return ActionResult{Key: "enter", Token: token, Description: fmt.Sprintf("removed mapping for %s", fields[1]), Completed: true}
	case "registers", "reg":
		e.lastEcho = e.registerDump()
		return ActionResult{Key: "enter", Token: token, Description: "listed register contents", Completed: true}
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
		e.recordChangeLocation(e.active().Buffer, e.currentCursor())
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
	case '"':
		return e.quoteObjectRange(includeAround)
	case '(', ')':
		return e.parenObjectRange(includeAround)
	case 'p':
		return e.paragraphObjectRange(includeAround)
	default:
		return 0, 0, 0, 0, false, false
	}
}

func (e *Editor) quoteObjectRange(includeQuotes bool) (startRow, startCol, endRow, endCol int, linewise bool, ok bool) {
	row := e.currentCursor().Row
	runes := e.lineRunes()
	cur := e.currentCursor().Col
	if len(runes) == 0 {
		return 0, 0, 0, 0, false, false
	}
	left := -1
	for i := cur; i >= 0; i-- {
		if runes[i] == '"' {
			left = i
			break
		}
	}
	if left == -1 {
		for i := 0; i < len(runes); i++ {
			if runes[i] == '"' {
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
		if runes[i] == '"' {
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

func (e *Editor) parenObjectRange(includeParens bool) (startRow, startCol, endRow, endCol int, linewise bool, ok bool) {
	row := e.currentCursor().Row
	runes := e.lineRunes()
	cur := e.currentCursor().Col
	if len(runes) == 0 {
		return 0, 0, 0, 0, false, false
	}
	left := -1
	for i := cur; i >= 0; i-- {
		if runes[i] == '(' {
			left = i
			break
		}
	}
	if left == -1 {
		return 0, 0, 0, 0, false, false
	}
	depth := 0
	right := -1
	for i := left; i < len(runes); i++ {
		if runes[i] == '(' {
			depth++
		}
		if runes[i] == ')' {
			depth--
			if depth == 0 {
				right = i
				break
			}
		}
	}
	if right == -1 {
		return 0, 0, 0, 0, false, false
	}
	if includeParens {
		return row, left, row, right + 1, false, true
	}
	return row, left + 1, row, right, false, true
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

func (e *Editor) deleteVisualSelection() (yankData, bool) {
	sr, sc, er, ec, linewise := e.visualRange()
	data := e.deleteRange(sr, sc, er, ec, linewise)
	return data, data.Text != "" || data.Linewise
}

func (e *Editor) yankVisualSelection() (yankData, bool) {
	sr, sc, er, ec, linewise := e.visualRange()
	data := e.extractRange(sr, sc, er, ec, linewise)
	return data, data.Text != "" || data.Linewise
}

func (e *Editor) replaceVisualSelectionWithPaste() bool {
	sr, sc, er, ec, linewise := e.visualRange()
	_ = e.deleteRange(sr, sc, er, ec, linewise)
	return e.pasteAfter()
}

func (e *Editor) writeTextRegister(data yankData, fromDelete bool) {
	e.yankBuffer = data
	e.textRegisters['"'] = data
	if fromDelete {
		e.textRegisters['-'] = data
	} else {
		e.textRegisters['0'] = data
	}
	if e.activeRegister == 0 {
		return
	}
	target := e.activeRegister
	e.activeRegister = 0
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

func (e *Editor) resolveReadRegister() yankData {
	target := e.activeRegister
	e.activeRegister = 0
	if target == 0 {
		target = '"'
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
	if len(e.jumpList) > 200 {
		e.jumpList = e.jumpList[len(e.jumpList)-200:]
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
	if len(e.changeList) > 200 {
		e.changeList = e.changeList[len(e.changeList)-200:]
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

func (e *Editor) goToDefinition() bool {
	word := e.wordUnderCursor()
	if word == "" {
		return false
	}
	lines := e.currentBuffer().Lines
	patterns := []string{"func " + word, "var " + word, "type " + word}
	for row, line := range lines {
		for _, pattern := range patterns {
			col := strings.Index(line, pattern)
			if col >= 0 {
				e.pushJump()
				e.active().Cursor = e.clampCursor(e.active().Buffer, Position{Row: row, Col: col})
				return true
			}
		}
	}
	return false
}

func (e *Editor) populateQuickfix(pattern string) int {
	e.quickfix = nil
	e.quickfixIndex = 0
	for b, buf := range e.buffers {
		for row, line := range buf.Lines {
			col := strings.Index(line, pattern)
			if col >= 0 {
				e.quickfix = append(e.quickfix, QuickfixItem{
					Buffer: b,
					Pos:    Position{Row: row, Col: col},
					Text:   line,
				})
			}
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

func (e *Editor) findInBuffer(buf *Buffer, query string, row int, col int, direction int) bool {
	if direction >= 0 {
		for r := row; r < len(buf.Lines); r++ {
			runes := []rune(buf.Lines[r])
			start := 0
			if r == row {
				start = min(col, len(runes))
			}
			for c := start; c+len([]rune(query)) <= len(runes); c++ {
				if string(runes[c:c+len([]rune(query))]) == query {
					e.active().Cursor = Position{Row: r, Col: c}
					return true
				}
			}
		}
		for r := 0; r <= row; r++ {
			runes := []rune(buf.Lines[r])
			limit := len(runes)
			if r == row {
				limit = min(col, len(runes))
			}
			for c := 0; c+len([]rune(query)) <= limit; c++ {
				if string(runes[c:c+len([]rune(query))]) == query {
					e.active().Cursor = Position{Row: r, Col: c}
					return true
				}
			}
		}
		return false
	}

	for r := row; r >= 0; r-- {
		runes := []rune(buf.Lines[r])
		start := len(runes) - len([]rune(query))
		if r == row {
			start = min(col-1, start)
		}
		for c := start; c >= 0; c-- {
			if string(runes[c:c+len([]rune(query))]) == query {
				e.active().Cursor = Position{Row: r, Col: c}
				return true
			}
		}
	}
	for r := len(buf.Lines) - 1; r >= row; r-- {
		runes := []rune(buf.Lines[r])
		start := len(runes) - len([]rune(query))
		for c := start; c >= 0; c-- {
			if r == row && c >= col {
				continue
			}
			if string(runes[c:c+len([]rune(query))]) == query {
				e.active().Cursor = Position{Row: r, Col: c}
				return true
			}
		}
	}
	return false
}

func (e *Editor) openOrSwitchBuffer(name string) {
	for i, buf := range e.buffers {
		if buf.Name == name {
			e.active().Buffer = i
			e.active().Cursor = e.clampCursor(i, Position{})
			return
		}
	}
	e.buffers = append(e.buffers, Buffer{Name: name, Lines: defaultBufferLines(name)})
	e.active().Buffer = len(e.buffers) - 1
	e.active().Cursor = Position{}
}

func (e *Editor) openSplit() {
	win := *e.active()
	e.windows = append(e.windows, win)
	e.activeWindow = len(e.windows) - 1
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

func defaultBufferLines(name string) []string {
	switch name {
	case "README.md":
		return []string{"# Vim Trainer", "", "Practice makes motions stick."}
	case "notes.txt":
		return []string{"alpha", "beta", "gamma"}
	case "config.lua":
		return []string{"vim.o.number = true", "vim.o.relativenumber = true"}
	default:
		return []string{fmt.Sprintf("// %s", name), ""}
	}
}

func (e *Editor) openTerminalBuffer() {
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
		Lines: []string{"$ echo 'neovim terminal'", "$"},
	})
	e.active().Buffer = len(e.buffers) - 1
	e.active().Cursor = Position{Row: 1, Col: 1}
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
		return "", fmt.Errorf("print supports quoted text and vim.g variables")
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
			return "", fmt.Errorf("trainer currently supports only normal-mode keymaps")
		}
		if lhs == "" || rhs == "" {
			return "", fmt.Errorf("keymap lhs/rhs cannot be empty")
		}
		e.normalMappings[lhs] = rhs
		return "", nil
	}

	return "", fmt.Errorf("supported lua snippets are vim.g assignment, print(...), and vim.keymap.set(...)")
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
