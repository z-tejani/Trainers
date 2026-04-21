package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"vimtrainer/internal/content"
	"vimtrainer/internal/engine"
	"vimtrainer/internal/progress"
)

type Options struct {
	LessonID string
	Seed     int64
	Debug    bool
	File     string
}

type Config struct {
	Command string
	Options Options
	Catalog *content.Catalog
	Store   *progress.Store
	Profile *progress.Profile
}

type route string

const (
	routeHome         route = "home"
	routeCampaign     route = "campaign"
	routeLesson       route = "lesson"
	routeSummary      route = "summary"
	routeStats        route = "stats"
	routeSettings     route = "settings"
	routeResetConfirm route = "reset"
)

type menuItem struct {
	Label       string
	Description string
	Action      string
}

type lessonSession struct {
	Mode        string
	Lesson      content.Lesson
	Queue       []string
	QueueIndex  int
	Editor      *engine.Editor
	StartedAt   time.Time
	Mistakes    []string
	Feedback    content.Feedback
	ModuleTitle string
}

type summaryState struct {
	Title       string
	Body        string
	Commands    []string
	Mistakes    []string
	Practice    []string
	NextLabel   string
	ReturnRoute route
}

type App struct {
	cfg            Config
	catalog        *content.Catalog
	store          *progress.Store
	profile        *progress.Profile
	width          int
	height         int
	route          route
	status         string
	homeCursor     int
	moduleCursor   int
	settingsCursor int
	homeItems      []menuItem
	session        *lessonSession
	summary        *summaryState
	showReplay     bool
}

func NewApp(cfg Config) *App {
	profile := cfg.Profile
	profile.Settings.Debug = profile.Settings.Debug || cfg.Options.Debug
	profile.RefreshModules(cfg.Catalog)

	app := &App{
		cfg:     cfg,
		catalog: cfg.Catalog,
		store:   cfg.Store,
		profile: profile,
		route:   routeHome,
		homeItems: []menuItem{
			{Label: "Continue", Description: "Resume from the last lesson or the next unlocked lesson.", Action: "continue"},
			{Label: "Campaign", Description: "Follow the guided beginner-first lesson path.", Action: "campaign"},
			{Label: "Neovim Mode", Description: "Run Neovim-specific lessons and command patterns.", Action: "neovim"},
			{Label: "Practice", Description: "Run a mixed practice queue from your current skill gaps.", Action: "practice"},
			{Label: "Review Weak Spots", Description: "Target recent failures and low-mastery skills.", Action: "review"},
			{Label: "Challenge", Description: "Timed no-hints queue focused on retention of learned skills.", Action: "challenge"},
			{Label: "Stats", Description: "See module progress, mastery counts, and recent weak skills.", Action: "stats"},
			{Label: "Settings", Description: "Toggle hints and debug details.", Action: "settings"},
			{Label: "Reset Progress", Description: "Clear local progress and start over.", Action: "reset"},
		},
	}

	switch cfg.Command {
	case "campaign":
		if cfg.Options.LessonID != "" {
			app.startSingleLesson("campaign", cfg.Options.LessonID, nil, 0)
		} else {
			app.route = routeCampaign
		}
	case "practice":
		app.startGeneratedQueue("practice", progress.QueuePractice)
	case "review":
		app.startGeneratedQueue("review", progress.QueueReview)
	case "challenge":
		app.startGeneratedQueue("challenge", progress.QueueChallenge)
	case "stats":
		app.route = routeStats
	case "neovim":
		app.startModuleQueue("neovim", "neovim")
	default:
		if cfg.Options.LessonID != "" {
			app.startSingleLesson("campaign", cfg.Options.LessonID, nil, 0)
		}
	}

	return app
}

func (a *App) Init() tea.Cmd { return nil }

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return a, tea.Quit
		}
		switch a.route {
		case routeHome:
			a.updateHome(msg)
		case routeCampaign:
			a.updateCampaign(msg)
		case routeLesson:
			a.updateLesson(msg)
		case routeSummary:
			a.updateSummary(msg)
		case routeStats:
			a.updateStats(msg)
		case routeSettings:
			a.updateSettings(msg)
		case routeResetConfirm:
			a.updateReset(msg)
		}
		return a, nil
	default:
		return a, nil
	}
}

func (a *App) View() string {
	switch a.route {
	case routeCampaign:
		return a.viewCampaign()
	case routeLesson:
		return a.viewLesson()
	case routeSummary:
		return a.viewSummary()
	case routeStats:
		return a.viewStats()
	case routeSettings:
		return a.viewSettings()
	case routeResetConfirm:
		return a.viewResetConfirm()
	default:
		return a.viewHome()
	}
}

func (a *App) updateHome(msg tea.KeyMsg) {
	switch msg.String() {
	case "j", "down":
		if a.homeCursor < len(a.homeItems)-1 {
			a.homeCursor++
		}
	case "k", "up":
		if a.homeCursor > 0 {
			a.homeCursor--
		}
	case "enter", " ":
		a.activateHome(a.homeItems[a.homeCursor].Action)
	}
}

func (a *App) activateHome(action string) {
	switch action {
	case "continue":
		if a.profile.LastLessonID != "" {
			idx := a.catalog.LessonIndex(a.profile.LastLessonID)
			nextID := a.profile.LastLessonID
			if idx >= 0 && a.profile.CompletedLessons[a.profile.LastLessonID] {
				if next := a.catalog.NextLessonID(a.profile.LastLessonID); next != "" {
					nextID = next
				}
			}
			a.startSingleLesson("campaign", nextID, nil, 0)
			return
		}
		a.route = routeCampaign
	case "campaign":
		a.route = routeCampaign
	case "neovim":
		a.startModuleQueue("neovim", "neovim")
	case "practice":
		a.startGeneratedQueue("practice", progress.QueuePractice)
	case "review":
		a.startGeneratedQueue("review", progress.QueueReview)
	case "challenge":
		a.startGeneratedQueue("challenge", progress.QueueChallenge)
	case "stats":
		a.route = routeStats
	case "settings":
		a.route = routeSettings
	case "reset":
		a.route = routeResetConfirm
	}
}

func (a *App) updateCampaign(msg tea.KeyMsg) {
	modules := a.catalog.Modules()
	switch msg.String() {
	case "esc":
		a.route = routeHome
	case "j", "down":
		if a.moduleCursor < len(modules)-1 {
			a.moduleCursor++
		}
	case "k", "up":
		if a.moduleCursor > 0 {
			a.moduleCursor--
		}
	case "enter", " ":
		module := modules[a.moduleCursor]
		if !a.profile.ModuleUnlocked(module) {
			a.status = "That module is still locked. Finish its prerequisites first."
			return
		}
		lessons := a.catalog.LessonsForModule(module.ID)
		startIdx := 0
		for i, lesson := range lessons {
			if !a.profile.CompletedLessons[lesson.ID] {
				startIdx = i
				break
			}
		}
		queue := make([]string, len(lessons))
		for i, lesson := range lessons {
			queue[i] = lesson.ID
		}
		a.startSingleLesson("campaign", queue[startIdx], queue, startIdx)
	}
}

func (a *App) updateLesson(msg tea.KeyMsg) {
	if msg.String() == "f2" {
		a.route = routeHome
		a.status = "Returned to the home screen."
		return
	}
	if msg.String() == "f5" {
		a.restartCurrentLesson()
		return
	}
	if msg.String() == "?" {
		a.profile.Settings.ShowHints = !a.profile.Settings.ShowHints
		_ = a.store.Save(a.profile)
		return
	}
	if msg.String() == "f6" {
		a.showReplay = !a.showReplay
		return
	}

	result := a.session.Editor.ProcessKey(msg.String())
	state := a.session.Editor.State()
	feedback := a.catalog.Evaluate(a.session.Lesson, state)
	a.session.Feedback = feedback
	if result.Error != "" && result.Token != "" {
		a.session.Mistakes = appendUnique(a.session.Mistakes, result.Token)
	}
	if feedback.Completed {
		duration := time.Since(a.session.StartedAt)
		a.profile.RecordLesson(a.session.Lesson, true, duration, "")
		a.profile.LastMode = a.session.Mode
		a.profile.LastLessonID = a.session.Lesson.ID
		a.profile.RefreshModules(a.catalog)
		_ = a.store.Save(a.profile)
		a.summary = a.buildSummary()
		a.route = routeSummary
		return
	}

	note := feedback.Body
	if result.Error != "" {
		a.profile.RecordLesson(a.session.Lesson, false, 0, note)
		_ = a.store.Save(a.profile)
	}
}

func (a *App) updateSummary(msg tea.KeyMsg) {
	switch msg.String() {
	case "enter", " ":
		a.advanceAfterSummary()
	case "esc":
		a.route = routeHome
	}
}

func (a *App) updateStats(msg tea.KeyMsg) {
	if msg.String() == "esc" {
		a.route = routeHome
	}
}

func (a *App) updateSettings(msg tea.KeyMsg) {
	switch msg.String() {
	case "esc":
		a.route = routeHome
	case "j", "down":
		if a.settingsCursor < 1 {
			a.settingsCursor++
		}
	case "k", "up":
		if a.settingsCursor > 0 {
			a.settingsCursor--
		}
	case "enter", " ":
		if a.settingsCursor == 0 {
			a.profile.Settings.ShowHints = !a.profile.Settings.ShowHints
		} else {
			a.profile.Settings.Debug = !a.profile.Settings.Debug
		}
		_ = a.store.Save(a.profile)
	}
}

func (a *App) updateReset(msg tea.KeyMsg) {
	switch msg.String() {
	case "esc":
		a.route = routeHome
	case "y":
		if err := a.store.Reset(); err != nil {
			a.status = "Failed to reset progress: " + err.Error()
			a.route = routeHome
			return
		}
		profile, err := a.store.Load()
		if err != nil {
			a.status = "Progress reset, but reload failed: " + err.Error()
			a.route = routeHome
			return
		}
		a.profile = profile
		a.profile.RefreshModules(a.catalog)
		a.session = nil
		a.summary = nil
		a.status = "Progress reset."
		a.route = routeHome
	case "enter":
		fallthrough
	case "n":
		a.route = routeHome
	}
}

func (a *App) startSingleLesson(mode, lessonID string, queue []string, index int) {
	lesson, ok := a.catalog.Lesson(lessonID)
	if !ok {
		a.status = fmt.Sprintf("lesson %q not found", lessonID)
		a.route = routeHome
		return
	}
	if len(queue) == 0 {
		queue = []string{lessonID}
		index = 0
	}
	module, _ := a.catalog.Module(lesson.ModuleID)
	a.session = &lessonSession{
		Mode:        mode,
		Lesson:      lesson,
		Queue:       queue,
		QueueIndex:  index,
		Editor:      engine.NewEditor(lesson.Initial),
		StartedAt:   time.Now(),
		Feedback:    a.catalog.Evaluate(lesson, engine.NewEditor(lesson.Initial).State()),
		ModuleTitle: module.Title,
	}
	a.showReplay = false
	a.route = routeLesson
}

func (a *App) startGeneratedQueue(mode string, style progress.QueueStyle) {
	limit := 6
	if style == progress.QueueChallenge {
		limit = 10
	}
	queueLessons := a.profile.QueueForStyle(a.catalog, style, limit)
	if len(queueLessons) == 0 {
		a.status = "No practice items are due yet. Try the campaign first."
		a.route = routeHome
		return
	}
	queue := make([]string, len(queueLessons))
	for i, lesson := range queueLessons {
		queue[i] = lesson.ID
	}
	a.startSingleLesson(mode, queue[0], queue, 0)
}

func (a *App) startModuleQueue(mode, moduleID string) {
	lessons := a.catalog.LessonsForModule(moduleID)
	if len(lessons) == 0 {
		a.status = "No lessons found for module: " + moduleID
		a.route = routeHome
		return
	}
	queue := make([]string, 0, len(lessons))
	for _, lesson := range lessons {
		queue = append(queue, lesson.ID)
	}
	start := 0
	for i, lesson := range lessons {
		if !a.profile.CompletedLessons[lesson.ID] {
			start = i
			break
		}
	}
	a.startSingleLesson(mode, queue[start], queue, start)
}

func (a *App) restartCurrentLesson() {
	if a.session == nil {
		return
	}
	a.startSingleLesson(a.session.Mode, a.session.Lesson.ID, a.session.Queue, a.session.QueueIndex)
}

func (a *App) buildSummary() *summaryState {
	nextLabel := "Return Home"
	if a.session.QueueIndex < len(a.session.Queue)-1 {
		nextLabel = "Next Lesson"
	}
	practice := append([]string{}, a.session.Lesson.Hints...)
	if len(practice) > 3 {
		practice = practice[:3]
	}
	title := "Lesson Complete"
	if a.profile.IsModuleCompleted(a.catalog, a.session.Lesson.ModuleID) {
		title = "Module Complete"
	}
	return &summaryState{
		Title:       title,
		Body:        a.session.Feedback.Body,
		Commands:    a.session.Lesson.CommandsLearned,
		Mistakes:    a.session.Mistakes,
		Practice:    practice,
		NextLabel:   nextLabel,
		ReturnRoute: routeHome,
	}
}

func (a *App) advanceAfterSummary() {
	if a.session == nil || a.summary == nil {
		a.route = routeHome
		return
	}
	if a.session.QueueIndex < len(a.session.Queue)-1 {
		nextIdx := a.session.QueueIndex + 1
		a.startSingleLesson(a.session.Mode, a.session.Queue[nextIdx], a.session.Queue, nextIdx)
		return
	}
	switch a.session.Mode {
	case "campaign":
		a.route = routeCampaign
	default:
		a.route = routeHome
	}
	a.summary = nil
}

func (a *App) viewHome() string {
	var lines []string
	lines = append(lines, bold("Vim Trainer"))
	lines = append(lines, dim("Beginner-first Vim practice with a real command surface, persistent progress, and weak-spot drills."))
	lines = append(lines, "")
	for i, item := range a.homeItems {
		prefix := "  "
		if i == a.homeCursor {
			prefix = cyan("> ")
		}
		lines = append(lines, prefix+item.Label+reset()+" "+dim(item.Description))
	}
	lines = append(lines, "")
	lines = append(lines, a.footer("Use j/k or arrow keys, Enter to open, Ctrl+C to quit."))
	if a.status != "" {
		lines = append(lines, yellow(a.status))
	}
	return strings.Join(lines, "\n")
}

func (a *App) viewCampaign() string {
	var lines []string
	lines = append(lines, bold("Campaign"))
	lines = append(lines, dim("Work through modules in order. Locked modules open as you complete prerequisites."))
	lines = append(lines, "")
	modules := a.catalog.Modules()
	available := a.height - 10
	if available < 4 {
		available = 4
	}
	start, end := visibleRange(len(modules), a.moduleCursor, available)
	if start > 0 {
		lines = append(lines, dim("  ..."))
	}
	for i := start; i < end; i++ {
		module := modules[i]
		cursor := "  "
		if i == a.moduleCursor {
			cursor = cyan("> ")
		}
		status := "[open]"
		if !a.profile.ModuleUnlocked(module) {
			status = "[locked]"
		} else if a.profile.CompletedModules[module.ID] {
			status = "[done]"
		}
		line := cursor + module.Title + " " + status + reset()
		lines = append(lines, line)
	}
	if end < len(modules) {
		lines = append(lines, dim("  ..."))
	}
	lines = append(lines, "")
	if len(modules) > 0 && a.moduleCursor < len(modules) {
		selected := modules[a.moduleCursor]
		lines = append(lines, dim(wrap("Selected: "+selected.Summary, a.width)))
	}
	lines = append(lines, "")
	lines = append(lines, a.footer("Enter starts the selected module. Esc returns home."))
	if a.status != "" {
		lines = append(lines, yellow(a.status))
	}
	return strings.Join(lines, "\n")
}

func (a *App) viewLesson() string {
	state := a.session.Editor.State()
	var lines []string
	lines = append(lines, bold("Vim Trainer")+" "+cyan(fmt.Sprintf("[%s]", strings.ToUpper(a.session.Mode)))+reset())
	lines = append(lines, fmt.Sprintf("%s%s%s  %s", cyan("Module: "), a.session.ModuleTitle, reset(), a.session.Lesson.Title))
	lines = append(lines, wrap(a.session.Lesson.Goal, a.width))
	lines = append(lines, dim(wrap(a.session.Lesson.Explanation, a.width)))
	if a.profile.Settings.ShowHints && a.session.Mode != "challenge" && len(a.session.Lesson.Hints) > 0 {
		lines = append(lines, yellow("Hint: ")+a.session.Feedback.NextHint)
	}
	lines = append(lines, "")
	lines = append(lines, a.renderWorkspace(state))
	lines = append(lines, "")
	lines = append(lines, green(fmt.Sprintf("Mode: %s | Recent: %s", state.Mode, recentTokens(state.CommandHistory))))
	if state.Mode == engine.ModeCommand {
		lines = append(lines, cyan(":"+state.CommandBuffer))
	}
	if state.Mode == engine.ModeSearch {
		lines = append(lines, cyan("/"+state.CommandBuffer))
	}
	if a.session.Feedback.Body != "" {
		lines = append(lines, a.session.Feedback.Title+": "+a.session.Feedback.Body)
	}
	if a.session.Feedback.Rule != "" {
		lines = append(lines, dim("Rule: "+a.session.Feedback.Rule))
	}
	if a.showReplay {
		lines = append(lines, "")
		lines = append(lines, cyan("Explain Replay")+reset())
		lines = append(lines, dim("Interpreted: "+tokenOrDash(a.session.Feedback.Interpreted)))
		if len(a.session.Lesson.FocusTokens) > 0 {
			lines = append(lines, dim("Lesson focus: "+strings.Join(a.session.Lesson.FocusTokens, ", ")))
		}
		lines = append(lines, dim("Recent input: "+recentTokens(state.CommandHistory)))
	}
	if a.profile.Settings.Debug {
		lines = append(lines, dim(fmt.Sprintf("Debug: token=%q count=%q pending=%q op=%q textobj=%q recording=%q last_change=%v",
			state.LastResult.Token, state.PendingCount, state.PendingPrefix, string(state.PendingOperator), string(state.PendingTextObject), string(state.RecordingRegister), state.LastChange)))
	}
	lines = append(lines, a.footer("F2 home, F5 restart, F6 explain replay, ? hints, Ctrl+C quit."))
	return strings.Join(lines, "\n")
}

func (a *App) renderWorkspace(state engine.State) string {
	if state.ExplorerOpen {
		var rows []string
		rows = append(rows, green("Explorer"))
		rows = append(rows, dim("Press Esc to leave the explorer view."))
		rows = append(rows, "")
		for i, buf := range state.Buffers {
			prefix := "  "
			if i == state.Windows[state.ActiveWindow].Buffer {
				prefix = "> "
			}
			rows = append(rows, prefix+buf.Name)
		}
		return strings.Join(rows, "\n")
	}

	if len(state.Windows) == 1 {
		return renderBuffer(state, state.ActiveWindow)
	}

	var rows []string
	for i := range state.Windows {
		rows = append(rows, fmt.Sprintf("%sWindow %d%s", cyan(""), i+1, reset()))
		rows = append(rows, renderBuffer(state, i))
		if i < len(state.Windows)-1 {
			rows = append(rows, dim(strings.Repeat("-", 30)))
		}
	}
	return strings.Join(rows, "\n")
}

func renderBuffer(state engine.State, windowIndex int) string {
	window := state.Windows[windowIndex]
	buffer := state.Buffers[window.Buffer]
	var rows []string
	rows = append(rows, buffer.Name)
	for rowIndex, line := range buffer.Lines {
		number := renderLineNumber(state, windowIndex, rowIndex)
		rows = append(rows, number+renderLine(window, rowIndex, line))
	}
	return strings.Join(rows, "\n")
}

func renderLineNumber(state engine.State, windowIndex, row int) string {
	if !state.Options.Number && !state.Options.RelativeNumber {
		return ""
	}
	window := state.Windows[windowIndex]
	value := ""
	switch {
	case state.Options.Number && state.Options.RelativeNumber:
		if row == window.Cursor.Row {
			value = fmt.Sprintf("%d", row+1)
		} else {
			value = fmt.Sprintf("%d", abs(row-window.Cursor.Row))
		}
	case state.Options.Number:
		value = fmt.Sprintf("%d", row+1)
	case state.Options.RelativeNumber:
		if row == window.Cursor.Row {
			value = "0"
		} else {
			value = fmt.Sprintf("%d", abs(row-window.Cursor.Row))
		}
	}
	return fmt.Sprintf("%3s ", value)
}

func renderLine(window engine.Window, row int, line string) string {
	if row != window.Cursor.Row {
		return line
	}
	runes := []rune(line)
	if len(runes) == 0 {
		return invert(" ")
	}
	var out strings.Builder
	for i, r := range runes {
		if i == window.Cursor.Col {
			out.WriteString(invert(string(r)))
			continue
		}
		out.WriteRune(r)
	}
	if window.Cursor.Col == len(runes) {
		out.WriteString(invert(" "))
	}
	return out.String()
}

func (a *App) viewSummary() string {
	var lines []string
	lines = append(lines, bold(a.summary.Title))
	lines = append(lines, a.summary.Body)
	lines = append(lines, "")
	lines = append(lines, green("Commands learned: ")+strings.Join(a.summary.Commands, ", "))
	if len(a.summary.Mistakes) > 0 {
		lines = append(lines, yellow("Mistakes to review: ")+strings.Join(a.summary.Mistakes, ", "))
	} else {
		lines = append(lines, dim("Mistakes to review: none in this run"))
	}
	if len(a.summary.Practice) > 0 {
		lines = append(lines, dim("What to practice next: "+strings.Join(a.summary.Practice, " | ")))
	}
	lines = append(lines, "")
	lines = append(lines, a.footer("Enter for "+a.summary.NextLabel+", Esc for home."))
	return strings.Join(lines, "\n")
}

func (a *App) viewStats() string {
	a.profile.RefreshModules(a.catalog)
	masteryCounts := map[progress.Mastery]int{}
	var weak []string
	for skill, entry := range a.profile.SkillProgress {
		masteryCounts[entry.Mastery]++
		if len(entry.RecentMistakes) > 0 || entry.Mastery == progress.MasteryNovice {
			weak = append(weak, skill)
		}
	}
	sort.Strings(weak)
	if len(weak) > 8 {
		weak = weak[:8]
	}
	var lines []string
	lines = append(lines, bold("Stats"))
	lines = append(lines, fmt.Sprintf("Completed lessons: %d / %d", len(a.profile.CompletedLessons), len(a.catalog.Lessons())))
	lines = append(lines, fmt.Sprintf("Completed modules: %d / %d", len(doneModules(a.profile.CompletedModules)), len(a.catalog.Modules())))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Mastery counts: unseen=%d novice=%d practicing=%d mastered=%d",
		masteryCounts[progress.MasteryUnseen],
		masteryCounts[progress.MasteryNovice],
		masteryCounts[progress.MasteryPracticing],
		masteryCounts[progress.MasteryMastered]))
	lines = append(lines, "")
	if len(weak) > 0 {
		lines = append(lines, "Weak spots:")
		for _, skill := range weak {
			lines = append(lines, "  - "+skill)
		}
	} else {
		lines = append(lines, dim("No weak spots recorded yet."))
	}
	lines = append(lines, "")
	lines = append(lines, a.footer("Esc returns home."))
	return strings.Join(lines, "\n")
}

func (a *App) viewSettings() string {
	items := []string{
		fmt.Sprintf("Show hints: %t", a.profile.Settings.ShowHints),
		fmt.Sprintf("Debug overlay: %t", a.profile.Settings.Debug),
	}
	var lines []string
	lines = append(lines, bold("Settings"))
	lines = append(lines, dim("Toggle trainer UI behavior."))
	lines = append(lines, "")
	for i, item := range items {
		prefix := "  "
		if i == a.settingsCursor {
			prefix = cyan("> ")
		}
		lines = append(lines, prefix+item+reset())
	}
	lines = append(lines, "")
	lines = append(lines, a.footer("Enter toggles. Esc returns home."))
	return strings.Join(lines, "\n")
}

func (a *App) viewResetConfirm() string {
	return strings.Join([]string{
		bold("Reset Progress"),
		"This deletes the saved local profile for vim-trainer.",
		"",
		a.footer("Press y to reset or Esc to cancel."),
	}, "\n")
}

func doneModules(modules map[string]bool) []string {
	var out []string
	for module, done := range modules {
		if done {
			out = append(out, module)
		}
	}
	return out
}

func recentTokens(tokens []string) string {
	if len(tokens) == 0 {
		return "none yet"
	}
	start := len(tokens) - 5
	if start < 0 {
		start = 0
	}
	return strings.Join(tokens[start:], " ")
}

func appendUnique(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func wrap(s string, width int) string {
	if width <= 0 || len([]rune(s)) <= width {
		return s
	}
	limit := width - 2
	if limit < 20 {
		limit = 20
	}
	var out []string
	words := strings.Fields(s)
	var current strings.Builder
	for _, word := range words {
		if current.Len() == 0 {
			current.WriteString(word)
			continue
		}
		if len([]rune(current.String()))+1+len([]rune(word)) > limit {
			out = append(out, current.String())
			current.Reset()
			current.WriteString(word)
			continue
		}
		current.WriteString(" ")
		current.WriteString(word)
	}
	if current.Len() > 0 {
		out = append(out, current.String())
	}
	return strings.Join(out, "\n")
}

const (
	resetANSI  = "\033[0m"
	boldANSI   = "\033[1m"
	dimANSI    = "\033[2m"
	cyanANSI   = "\033[36m"
	greenANSI  = "\033[32m"
	yellowANSI = "\033[33m"
	invertANSI = "\033[7m"
)

func reset() string          { return resetANSI }
func bold(s string) string   { return boldANSI + s + resetANSI }
func dim(s string) string    { return dimANSI + s + resetANSI }
func cyan(s string) string   { return cyanANSI + s + resetANSI }
func green(s string) string  { return greenANSI + s + resetANSI }
func yellow(s string) string { return yellowANSI + s + resetANSI }
func invert(s string) string { return invertANSI + s + resetANSI }

func (a *App) footer(text string) string {
	return dim(text)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func visibleRange(total, cursor, maxVisible int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if maxVisible <= 0 || maxVisible >= total {
		return 0, total
	}
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= total {
		cursor = total - 1
	}
	start := cursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > total {
		end = total
		start = end - maxVisible
	}
	if start < 0 {
		start = 0
	}
	return start, end
}

func tokenOrDash(token string) string {
	if token == "" {
		return "-"
	}
	return token
}
