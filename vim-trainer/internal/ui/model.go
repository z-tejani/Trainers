package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"vimtrainer/internal/config"
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
	routeChallenge    route = "challenge"
	routeLesson       route = "lesson"
	routeSummary      route = "summary"
	routeStats        route = "stats"
	routeSettings     route = "settings"
	routeResetConfirm route = "reset"
	// routeSandbox is the no-eval free-play mode. The engine runs but no
	// lesson check is applied; the user just experiments with Vim keys
	// against a buffer (default scratch, or one loaded via --file).
	routeSandbox    route = "sandbox"
	routeBrowse     route = "browse"
	routeDiagnostic route = "diagnostic"
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
	// Keystrokes counts every key the learner sends to the engine for the
	// current lesson attempt. Used by the summary to grade efficiency
	// against the lesson's OptimalKeys.
	Keystrokes int
	// LastErrorToken is the token from the most recent failing keystroke.
	// Drives the auto-shown replay panel; cleared when the next keystroke
	// succeeds.
	LastErrorToken string
	// WrongAttempts counts every keystroke that produced an Error result.
	// Used to unlock progressively more hints as the learner struggles.
	WrongAttempts int
	// HintsUnlocked is the maximum hint index revealed so far. Sticky for
	// the duration of the lesson — once a hint is shown, restarting the
	// lesson via F5 resets it but a fresh wrong-attempt count keeps it
	// from popping back instantly.
	HintsUnlocked int
}

type summaryState struct {
	Title       string
	Body        string
	Commands    []string
	Mistakes    []string
	Practice    []string
	NextLabel   string
	ReturnRoute route
	Keystrokes  int
	OptimalKeys int
	Duration    time.Duration
	TimeTarget  time.Duration
}

type App struct {
	cfg             Config
	catalog         *content.Catalog
	store           *progress.Store
	profile         *progress.Profile
	width           int
	height          int
	route           route
	status          string
	homeCursor      int
	moduleCursor    int
	challengeCursor int
	settingsCursor  int
	homeItems       []menuItem
	session         *lessonSession
	summary         *summaryState
	showReplay      bool
	showCheatsheet  bool
	sandbox         *sandboxSession
	browseCursor    int
	browseFilter    string
	browseEditing   bool
	diagnostic      *diagnosticSession
	achievementsToast []string
	// quit is set by route handlers when the engine reports a quit
	// request (e.g. :q, :qa). Update inspects it after dispatching the
	// keypress and returns tea.Quit when set.
	quit bool
}

// diagnosticSession drives the optional placement quiz on first launch.
// Each question maps a yes/skill answer to a list of lesson IDs that
// should be marked completed (so experienced users skip past basics).
type diagnosticSession struct {
	Questions []diagnosticQuestion
	Index     int
	Answers   map[int]bool
}

type diagnosticQuestion struct {
	Prompt          string
	YesGrantsLesson []string // lesson IDs to auto-complete on "yes"
}

type sandboxSession struct {
	Editor    *engine.Editor
	StartedAt time.Time
	Source    string // origin label shown in header (filename or "scratch")
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
			{Label: "Review Weak Spots", Description: "Target recent failures and SRS-due skills.", Action: "review"},
			{Label: "Challenge", Description: "Timed no-hints queue focused on retention of learned skills.", Action: "challenge"},
			{Label: "Sandbox", Description: "Free-play with the editor; no goal, no eval. Optionally opens --file.", Action: "sandbox"},
			{Label: "Browse Lessons", Description: "Search and filter the full lesson catalog by name, module, or skill.", Action: "browse"},
			{Label: "Placement Quiz", Description: "Quick diagnostic; mark known skills complete to skip basics.", Action: "diagnostic"},
			{Label: "Stats", Description: "See module progress, achievements, mastery counts, and weak skills.", Action: "stats"},
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
	case "sandbox":
		app.startSandbox(cfg.Options.File)
	default:
		if cfg.Options.LessonID != "" {
			app.startSingleLesson("campaign", cfg.Options.LessonID, nil, 0)
		} else if cfg.Options.File != "" {
			// `vim-trainer --file path` with no subcommand: drop the user
			// straight into a sandbox on that file. Keeps the entry
			// frictionless ("just open this in the trainer").
			app.startSandbox(cfg.Options.File)
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
		case routeChallenge:
			a.updateChallenge(msg)
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
		case routeSandbox:
			a.updateSandbox(msg)
		case routeBrowse:
			a.updateBrowse(msg)
		case routeDiagnostic:
			a.updateDiagnostic(msg)
		}
		if a.quit {
			return a, tea.Quit
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
	case routeChallenge:
		return a.viewChallenge()
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
	case routeSandbox:
		return a.viewSandbox()
	case routeBrowse:
		return a.viewBrowse()
	case routeDiagnostic:
		return a.viewDiagnostic()
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
		a.route = routeChallenge
	case "sandbox":
		a.startSandbox(a.cfg.Options.File)
	case "browse":
		a.route = routeBrowse
		a.browseCursor = 0
		a.browseFilter = ""
		a.browseEditing = false
	case "diagnostic":
		a.startDiagnostic()
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

func (a *App) updateChallenge(msg tea.KeyMsg) {
	items := a.challengeItems()
	switch msg.String() {
	case "esc":
		a.route = routeHome
	case "j", "down":
		if a.challengeCursor < len(items)-1 {
			a.challengeCursor++
		}
	case "k", "up":
		if a.challengeCursor > 0 {
			a.challengeCursor--
		}
	case "enter", " ":
		if len(items) == 0 {
			return
		}
		action := items[a.challengeCursor].Action
		if action == "challenge-adaptive" {
			a.startGeneratedQueue("challenge", progress.QueueChallenge)
			return
		}
		if strings.HasPrefix(action, "challenge-module:") {
			moduleID := strings.TrimPrefix(action, "challenge-module:")
			a.startChallengeModuleQueue(moduleID)
			return
		}
	}
}

func (a *App) updateLesson(msg tea.KeyMsg) {
	key := msg.String()
	if key == "f2" {
		a.route = routeHome
		a.status = "Returned to the home screen."
		return
	}
	if key == "f5" {
		a.restartCurrentLesson()
		return
	}
	if key == "f3" {
		a.showCheatsheet = !a.showCheatsheet
		return
	}
	// Hint toggle: bind to ? (Vim-flavored) AND F1 (universal "help") so
	// terminals that swallow one still surface the other. Status line
	// announces the new state so the toggle is unambiguously visible.
	if key == "?" || key == "f1" {
		a.profile.Settings.ShowHints = !a.profile.Settings.ShowHints
		_ = a.store.Save(a.profile)
		if a.profile.Settings.ShowHints {
			a.status = "Hints: ON"
		} else {
			a.status = "Hints: OFF"
		}
		return
	}
	if key == "f6" {
		// Treat F6 as a unified show/dismiss for the replay panel — works
		// against both manual toggles and the auto-shown post-error panel.
		visible := a.showReplay || a.session.LastErrorToken != ""
		if visible {
			a.showReplay = false
			a.session.LastErrorToken = ""
		} else {
			a.showReplay = true
		}
		return
	}
	// Submit binding: F4 (joins the F-key family) plus Ctrl+S as a backup
	// for terminals that intercept function keys. Lesson checks already
	// fire on every keystroke, so this is mostly a "did I miss it?"
	// affordance — but the explicit nudge with status feedback also
	// confirms to the learner that *something* happened.
	if key == "f4" || key == "ctrl+s" {
		state := a.session.Editor.State()
		feedback := a.catalog.EvaluateWith(a.session.Lesson, state, a.profile.Settings.NonStrictMode)
		a.session.Feedback = feedback
		if feedback.Completed {
			a.completeLesson()
			return
		}
		a.status = "Not yet — keep going."
		return
	}

	result := a.session.Editor.ProcessKey(key)
	a.session.Keystrokes++
	state := a.session.Editor.State()
	if state.QuitRequested {
		a.quit = true
		return
	}
	feedback := a.catalog.EvaluateWith(a.session.Lesson, state, a.profile.Settings.NonStrictMode)
	a.session.Feedback = feedback
	if result.Error != "" {
		if result.Token != "" {
			a.session.Mistakes = appendUnique(a.session.Mistakes, result.Token)
		}
		a.session.LastErrorToken = result.Token
		a.session.WrongAttempts++
		// Unlock the next tier of hints every config.HintsPerErrorTier
		// wrong attempts.
		want := a.session.WrongAttempts / config.HintsPerErrorTier
		if want > a.session.HintsUnlocked {
			a.session.HintsUnlocked = want
		}
	} else {
		a.session.LastErrorToken = ""
	}
	if feedback.Completed {
		a.completeLesson()
		return
	}

	note := feedback.Body
	if result.Error != "" {
		a.profile.RecordLesson(a.session.Lesson, false, 0, note)
		_ = a.store.Save(a.profile)
	}
}

func (a *App) completeLesson() {
	duration := time.Since(a.session.StartedAt)
	a.profile.RecordLesson(a.session.Lesson, true, duration, "")
	a.profile.RecordSession(a.session.Lesson, a.session.Mode, true, duration, a.session.Keystrokes, a.session.Mistakes)
	a.profile.LastMode = a.session.Mode
	a.profile.LastLessonID = a.session.Lesson.ID
	a.profile.RefreshModules(a.catalog)
	// Evaluate achievements after the module refresh — some milestones
	// (Module Master, All Modules) depend on CompletedModules.
	if newly := a.profile.EvaluateAchievements(a.catalog); len(newly) > 0 {
		a.achievementsToast = append(a.achievementsToast, newly...)
	}
	_ = a.store.Save(a.profile)
	a.summary = a.buildSummary()
	a.route = routeSummary
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
	const numSettings = 7 // ShowHints, NonStrict, Debug, Theme, Colorblind, ReducedMotion, LargerCursor
	switch msg.String() {
	case "esc":
		a.route = routeHome
	case "j", "down":
		if a.settingsCursor < numSettings-1 {
			a.settingsCursor++
		}
	case "k", "up":
		if a.settingsCursor > 0 {
			a.settingsCursor--
		}
	case "enter", " ":
		switch a.settingsCursor {
		case 0:
			a.profile.Settings.ShowHints = !a.profile.Settings.ShowHints
		case 1:
			a.profile.Settings.NonStrictMode = !a.profile.Settings.NonStrictMode
		case 2:
			a.profile.Settings.Debug = !a.profile.Settings.Debug
		case 3:
			switch a.profile.Settings.Theme {
			case "", "default":
				a.profile.Settings.Theme = "high-contrast"
			case "high-contrast":
				a.profile.Settings.Theme = "monochrome"
			default:
				a.profile.Settings.Theme = "default"
			}
		case 4:
			a.profile.Settings.ColorblindSafe = !a.profile.Settings.ColorblindSafe
		case 5:
			a.profile.Settings.ReducedMotion = !a.profile.Settings.ReducedMotion
		case 6:
			a.profile.Settings.LargerCursor = !a.profile.Settings.LargerCursor
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
		Feedback:    a.catalog.EvaluateWith(lesson, engine.NewEditor(lesson.Initial).State(), a.profile.Settings.NonStrictMode),
		ModuleTitle: module.Title,
	}
	a.showReplay = false
	a.route = routeLesson
}

func (a *App) startGeneratedQueue(mode string, style progress.QueueStyle) {
	limit := config.PracticeQueueLimit
	if style == progress.QueueChallenge {
		limit = config.ChallengeQueueLimit
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

func (a *App) startChallengeModuleQueue(moduleID string) {
	module, ok := a.catalog.Module(moduleID)
	if !ok {
		a.status = "Unknown challenge module: " + moduleID
		a.route = routeChallenge
		return
	}
	if !a.profile.ModuleUnlocked(module) {
		a.status = "That module is still locked. Finish prerequisites first."
		a.route = routeChallenge
		return
	}

	adaptive := a.profile.QueueForStyle(a.catalog, progress.QueueChallenge, 60)
	var queue []string
	for _, lesson := range adaptive {
		if lesson.ModuleID == moduleID {
			queue = append(queue, lesson.ID)
		}
	}
	if len(queue) == 0 {
		lessons := a.catalog.LessonsForModule(moduleID)
		for _, lesson := range lessons {
			queue = append(queue, lesson.ID)
		}
	}
	if len(queue) == 0 {
		a.status = "No challenge lessons available for this module yet."
		a.route = routeChallenge
		return
	}
	if len(queue) > config.ChallengeModuleQueueLimit {
		queue = queue[:config.ChallengeModuleQueueLimit]
	}
	a.startSingleLesson("challenge", queue[0], queue, 0)
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

// startSandbox opens the no-eval free-play editor. If filePath is provided,
// the file is read into the initial buffer; otherwise a generic scratch
// buffer is used. Sandbox covers both Tier-0 P0 items: the route itself and
// the "open my own file" entry path.
func (a *App) startSandbox(filePath string) {
	scenario := engine.Scenario{
		Buffers: []engine.Buffer{{Name: "scratch.txt", Lines: []string{
			"Welcome to the sandbox.",
			"",
			"There is no goal. There is no eval.",
			"Press F2 to return home, Ctrl+C to quit.",
			"Use any Vim keys you've learned — every one is live.",
		}}},
	}
	source := "scratch"
	if filePath != "" {
		buf, err := loadFileAsBuffer(filePath)
		if err != nil {
			a.status = fmt.Sprintf("could not load %s: %v", filePath, err)
		} else {
			scenario = engine.Scenario{Buffers: []engine.Buffer{buf}}
			source = filePath
		}
	}
	a.sandbox = &sandboxSession{
		Editor:    engine.NewEditor(scenario),
		StartedAt: time.Now(),
		Source:    source,
	}
	a.route = routeSandbox
}

func loadFileAsBuffer(path string) (engine.Buffer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return engine.Buffer{}, err
	}
	text := string(data)
	// Strip a single trailing newline so we don't end up with a stray
	// empty final line for files that end in "\n" (the common case).
	text = strings.TrimSuffix(text, "\n")
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}
	return engine.Buffer{Name: filepath.Base(path), Lines: lines}, nil
}

func (a *App) updateSandbox(msg tea.KeyMsg) {
	if a.sandbox == nil {
		a.route = routeHome
		return
	}
	key := msg.String()
	if key == "f2" {
		a.route = routeHome
		return
	}
	if key == "f5" {
		// Restart with the same source.
		a.startSandbox(a.sandbox.SourcePath())
		return
	}
	a.sandbox.Editor.ProcessKey(key)
	if a.sandbox.Editor.State().QuitRequested {
		a.quit = true
	}
}

// SourcePath returns the original path used to seed the sandbox, or "" for
// the scratch buffer. Used to restart in place via F5.
func (s *sandboxSession) SourcePath() string {
	if s.Source == "scratch" {
		return ""
	}
	return s.Source
}

func (a *App) viewSandbox() string {
	if a.sandbox == nil {
		return "No sandbox session."
	}
	state := a.sandbox.Editor.State()
	var lines []string
	header := bold("Vim Trainer") + " " + cyan("[SANDBOX]") + reset()
	lines = append(lines, header)
	lines = append(lines, dim("Source: "+a.sandbox.Source+"   No goal, no eval — every Vim key is live."))
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
	if state.ConfirmActive {
		// Render the full source line with the candidate match inverted
		// so the learner can SEE which occurrence is being asked about.
		if state.ConfirmContext != "" {
			lines = append(lines, dim("→ ")+highlightMatch(state.ConfirmContext, state.ConfirmMatchStart, state.ConfirmMatchEnd))
		}
		lines = append(lines, yellow(state.ConfirmPrompt))
	}
	if state.LastEcho != "" {
		lines = append(lines, dim("Echo: "+state.LastEcho))
	}
	lines = append(lines, "")
	lines = append(lines, a.footer("F2 home, F5 restart sandbox, Ctrl+C quit."))
	if a.status != "" {
		lines = append(lines, yellow(a.status))
	}
	return strings.Join(lines, "\n")
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
		Keystrokes:  a.session.Keystrokes,
		OptimalKeys: a.session.Lesson.OptimalKeys,
		Duration:    time.Since(a.session.StartedAt),
		TimeTarget:  a.session.Lesson.TimeTarget,
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
	case "challenge":
		a.route = routeChallenge
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

func (a *App) viewChallenge() string {
	items := a.challengeItems()
	var lines []string
	lines = append(lines, bold("Challenge"))
	lines = append(lines, dim("Choose a challenge track. Adaptive uses your weak spots and due-review skills."))
	lines = append(lines, "")

	available := a.height - 10
	if available < 4 {
		available = 4
	}
	start, end := visibleRange(len(items), a.challengeCursor, available)
	if start > 0 {
		lines = append(lines, dim("  ..."))
	}
	for i := start; i < end; i++ {
		item := items[i]
		prefix := "  "
		if i == a.challengeCursor {
			prefix = cyan("> ")
		}
		lines = append(lines, prefix+item.Label+reset())
	}
	if end < len(items) {
		lines = append(lines, dim("  ..."))
	}
	lines = append(lines, "")
	if len(items) > 0 && a.challengeCursor < len(items) {
		lines = append(lines, dim(wrap("Selected: "+items[a.challengeCursor].Description, a.width)))
	}
	lines = append(lines, "")
	lines = append(lines, a.footer("Enter starts selected challenge track. Esc returns home."))
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
		// Tiered hint reveal: hint #1 always available; subsequent hints
		// unlock as the learner accumulates wrong attempts. The lesson
		// session tracks how many tiers are currently unlocked and the
		// view shows the corresponding strip.
		visible := 1 + a.session.HintsUnlocked
		if visible > len(a.session.Lesson.Hints) {
			visible = len(a.session.Lesson.Hints)
		}
		for i := 0; i < visible; i++ {
			lines = append(lines, yellow(fmt.Sprintf("Hint %d/%d: ", i+1, len(a.session.Lesson.Hints)))+a.session.Lesson.Hints[i])
		}
		if visible < len(a.session.Lesson.Hints) {
			needed := (a.session.HintsUnlocked+1)*config.HintsPerErrorTier - a.session.WrongAttempts
			if needed > 0 {
				lines = append(lines, dim(fmt.Sprintf("(%d more wrong attempts unlocks the next hint)", needed)))
			}
		}
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
	if state.ConfirmActive {
		// Render the full source line with the candidate match inverted
		// so the learner can SEE which occurrence is being asked about.
		if state.ConfirmContext != "" {
			lines = append(lines, dim("→ ")+highlightMatch(state.ConfirmContext, state.ConfirmMatchStart, state.ConfirmMatchEnd))
		}
		lines = append(lines, yellow(state.ConfirmPrompt))
	}
	if a.session.Feedback.Body != "" {
		lines = append(lines, a.session.Feedback.Title+": "+a.session.Feedback.Body)
	}
	if a.session.Feedback.Rule != "" {
		lines = append(lines, dim("Rule: "+a.session.Feedback.Rule))
	}
	autoReplay := a.session.LastErrorToken != ""
	if a.showReplay || autoReplay {
		lines = append(lines, "")
		header := cyan("Explain Replay")
		if autoReplay && !a.showReplay {
			header += dim("  (auto-shown after a misfire — press F6 to dismiss)")
		}
		lines = append(lines, header+reset())
		lines = append(lines, dim("Interpreted: "+tokenOrDash(a.session.Feedback.Interpreted)))
		if len(a.session.Lesson.FocusTokens) > 0 {
			lines = append(lines, dim("Lesson focus: "+strings.Join(a.session.Lesson.FocusTokens, ", ")))
		}
		lines = append(lines, dim("Recent input: "+recentTokens(state.CommandHistory)))
		if autoReplay {
			lines = append(lines, dim("Last failure: "+tokenOrDash(a.session.LastErrorToken)+" — Vim parsed your keys but they did not match the lesson focus."))
		}
	}
	if a.profile.Settings.Debug {
		// Engine owns the debug rendering — see Editor.DebugSummary.
		lines = append(lines, dim("Debug: "+a.session.Editor.DebugSummary()))
		// Spoiler line: surface the lesson's expected answer (shortest
		// canonical solution) so the debug session can sanity-check the
		// scenario without context-switching back to the source.
		if expected := expectedAnswer(a.session.Lesson); expected != "" {
			lines = append(lines, dim("Expected: "+expected))
		}
	}
	if a.showCheatsheet {
		lines = append(lines, "")
		lines = append(lines, cheatsheetText())
	}
	keyInfo := fmt.Sprintf("Keys: %d", a.session.Keystrokes)
	if a.session.Lesson.OptimalKeys > 0 {
		keyInfo = fmt.Sprintf("Keys: %d / par %d", a.session.Keystrokes, a.session.Lesson.OptimalKeys)
	}
	hintState := "on"
	if !a.profile.Settings.ShowHints {
		hintState = "off"
	}
	mode := "strict"
	if a.profile.Settings.NonStrictMode {
		mode = cyan("non-strict")
	}
	lines = append(lines, a.footer(fmt.Sprintf("%s  ·  %s  ·  F4 submit, F2 home, F3 cheatsheet, F5 restart, F6 replay, ?/F1 hints (%s), :q quit.", keyInfo, mode, hintState)))
	if a.status != "" {
		lines = append(lines, yellow(a.status))
	}
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
	if line := efficiencyLine(a.summary.Keystrokes, a.summary.OptimalKeys, a.summary.Duration, a.summary.TimeTarget); line != "" {
		lines = append(lines, line)
	}
	if len(a.summary.Mistakes) > 0 {
		lines = append(lines, yellow("Mistakes to review: ")+strings.Join(a.summary.Mistakes, ", "))
	} else {
		lines = append(lines, dim("Mistakes to review: none in this run"))
	}
	if len(a.summary.Practice) > 0 {
		lines = append(lines, dim("What to practice next: "+strings.Join(a.summary.Practice, " | ")))
	}
	if len(a.achievementsToast) > 0 {
		lines = append(lines, "")
		titleByID := map[string]string{}
		for _, ach := range progress.DefaultAchievements() {
			titleByID[ach.ID] = ach.Title
		}
		for _, id := range a.achievementsToast {
			title := titleByID[id]
			if title == "" {
				title = id
			}
			lines = append(lines, green("🏆 Unlocked: "+title))
		}
		a.achievementsToast = nil
	}
	lines = append(lines, "")
	lines = append(lines, a.footer("Enter for "+a.summary.NextLabel+", Esc for home."))
	return strings.Join(lines, "\n")
}

// efficiencyLine renders the keystroke / time grading band shown on the
// summary screen. Returns "" when no target is set on the lesson, so older
// lessons that haven't been calibrated stay clean.
func efficiencyLine(keystrokes, optimal int, duration, timeTarget time.Duration) string {
	if optimal <= 0 && timeTarget <= 0 {
		// Always at least show keystrokes + duration so the learner has a
		// raw signal even without a calibrated target.
		return dim(fmt.Sprintf("Keystrokes: %d   Time: %s", keystrokes, formatDuration(duration)))
	}
	parts := []string{}
	if optimal > 0 {
		ratio := float64(keystrokes) / float64(optimal)
		grade := efficiencyGrade(ratio)
		parts = append(parts, fmt.Sprintf("Keystrokes: %d / par %d  %s", keystrokes, optimal, grade))
	} else {
		parts = append(parts, fmt.Sprintf("Keystrokes: %d", keystrokes))
	}
	if timeTarget > 0 {
		ratio := float64(duration) / float64(timeTarget)
		grade := efficiencyGrade(ratio)
		parts = append(parts, fmt.Sprintf("Time: %s / par %s  %s", formatDuration(duration), formatDuration(timeTarget), grade))
	} else {
		parts = append(parts, fmt.Sprintf("Time: %s", formatDuration(duration)))
	}
	return strings.Join(parts, "   ")
}

func efficiencyGrade(ratio float64) string {
	switch {
	case ratio <= 1.0:
		return green("(par)")
	case ratio <= 1.5:
		return yellow("(close)")
	case ratio <= 2.5:
		return yellow("(over)")
	default:
		return dim("(slow)")
	}
}

func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
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
	if a.profile.BestParStreak > 0 {
		lines = append(lines, fmt.Sprintf("Best par streak: %d   (current: %d)", a.profile.BestParStreak, a.profile.ParStreak))
	}
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Mastery counts: unseen=%d novice=%d practicing=%d mastered=%d",
		masteryCounts[progress.MasteryUnseen],
		masteryCounts[progress.MasteryNovice],
		masteryCounts[progress.MasteryPracticing],
		masteryCounts[progress.MasteryMastered]))
	// Count SRS-due skills for the dashboard.
	now := time.Now()
	due := 0
	for _, entry := range a.profile.SkillProgress {
		if entry != nil && entry.IsDue(now) {
			due++
		}
	}
	lines = append(lines, fmt.Sprintf("SRS due now: %d skill(s)", due))
	lines = append(lines, "")
	if len(weak) > 0 {
		lines = append(lines, "Weak spots:")
		for _, skill := range weak {
			lines = append(lines, "  - "+skill)
		}
	} else {
		lines = append(lines, dim("No weak spots recorded yet."))
	}
	if stale := a.profile.StaleLessons(a.catalog); len(stale) > 0 {
		lines = append(lines, "")
		lines = append(lines, yellow(fmt.Sprintf("Stale completions (%d): content edited since you completed them", len(stale))))
		shown := stale
		if len(shown) > 6 {
			shown = shown[:6]
		}
		for _, id := range shown {
			lines = append(lines, dim("  - "+id))
		}
		if len(stale) > len(shown) {
			lines = append(lines, dim(fmt.Sprintf("  ... and %d more", len(stale)-len(shown))))
		}
		lines = append(lines, dim("(Stats and completion remain intact; rerun any time you want to refresh.)"))
	}
	lines = append(lines, "")
	lines = append(lines, bold("Achievements"))
	all := progress.DefaultAchievements()
	unlocked := 0
	for _, ach := range all {
		if _, ok := a.profile.Achievements[ach.ID]; ok {
			unlocked++
			lines = append(lines, fmt.Sprintf("  %s %s — %s", green("✓"), ach.Title, dim(ach.Desc)))
		} else {
			lines = append(lines, fmt.Sprintf("  %s %s — %s", dim("·"), ach.Title, dim(ach.Desc)))
		}
	}
	lines = append(lines, dim(fmt.Sprintf("(%d / %d unlocked)", unlocked, len(all))))
	lines = append(lines, "")
	lines = append(lines, a.footer("Esc returns home."))
	return strings.Join(lines, "\n")
}

func (a *App) viewSettings() string {
	theme := a.profile.Settings.Theme
	if theme == "" {
		theme = "default"
	}
	items := []string{
		fmt.Sprintf("Show hints: %t", a.profile.Settings.ShowHints),
		fmt.Sprintf("Non-strict mode: %t  (accept any key sequence that reaches the goal)", a.profile.Settings.NonStrictMode),
		fmt.Sprintf("Debug overlay: %t", a.profile.Settings.Debug),
		fmt.Sprintf("Theme: %s", theme),
		fmt.Sprintf("Colorblind-safe palette: %t", a.profile.Settings.ColorblindSafe),
		fmt.Sprintf("Reduced motion: %t", a.profile.Settings.ReducedMotion),
		fmt.Sprintf("Larger cursor: %t", a.profile.Settings.LargerCursor),
	}
	var lines []string
	lines = append(lines, bold("Settings"))
	lines = append(lines, dim("Toggle trainer UI behavior. Theme cycles default → high-contrast → monochrome."))
	lines = append(lines, "")
	for i, item := range items {
		prefix := "  "
		if i == a.settingsCursor {
			prefix = cyan("> ")
		}
		lines = append(lines, prefix+item+reset())
	}
	lines = append(lines, "")
	lines = append(lines, dim("Accessibility flags are advisory. Future renderer passes will honor them."))
	lines = append(lines, a.footer("Enter toggles or cycles. Esc returns home."))
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

func (a *App) challengeItems() []menuItem {
	items := []menuItem{
		{
			Label:       "Adaptive Challenge",
			Description: "Mixed queue based on your mastery, mistakes, and review recency.",
			Action:      "challenge-adaptive",
		},
	}
	for _, module := range a.catalog.Modules() {
		if !a.profile.ModuleUnlocked(module) {
			continue
		}
		items = append(items, menuItem{
			Label:       module.Title + " Challenge",
			Description: module.Summary,
			Action:      "challenge-module:" + module.ID,
		})
	}
	if a.challengeCursor >= len(items) {
		a.challengeCursor = len(items) - 1
	}
	if a.challengeCursor < 0 {
		a.challengeCursor = 0
	}
	return items
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

// cheatsheetText returns the static, scoped cheatsheet shown via F3 inside
// a lesson. Kept compact so it fits on a single terminal page; deeper
// references live in :help once that's wired up further.
func cheatsheetText() string {
	rows := [][2]string{
		{"Modes", "i append-after a, open o/O, visual v/V/Ctrl-V, replace r<x>/R, esc → Normal"},
		{"Motions", "h/j/k/l, w/b/e, 0/$/^, gg/G, %, *, /pat ?pat, n/N, gn/gN"},
		{"Operators", "d/c/y + motion, dd cc yy, di( ci\" da{, ds<x> cs<o><n> ys<m><x>"},
		{"Edit", "x p P u <C-r>, >> <<, =ip (n/a), . repeat, ~ swap case"},
		{"Insert tricks", "<C-o>cmd one normal cmd, <C-w> word del, <C-u> line del, <C-r>a paste reg"},
		{"Search", "/pat ?pat, * #, n/N, :noh, :s/pat/rep/[gc], :%s/.../gc"},
		{"Quickfix", ":vimgrep /pat/ %, :copen, :cnext :cprev, :cnewer :colder, :cdo cmd"},
		{"Macros", "qa…q record, @a play, @@ replay, qA append, :%normal @a everywhere"},
		{"Marks", "ma set, 'a / `a jump, '< '> visual, '. last edit, '' jump back"},
		{"Registers", "\"ayy / \"ap named, \"+ system, \"_ blackhole, \"0 last yank, :reg"},
		{"Windows", ":split :vsplit, <C-w>w cycle, :bn :bp, :e file, :Ex"},
		{"Macros + qfix", "qa…q, :vimgrep /pat/ %, :cdo norm @a (the headline workflow)"},
		{"App", "F2 home · F3 cheatsheet · F4 submit · F5 restart · F6 replay · ?/F1 hints · :q quit"},
	}
	var lines []string
	lines = append(lines, cyan("Cheatsheet")+dim(" — F3 to dismiss"))
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("  %s%-14s%s %s", boldANSI, row[0], resetANSI, row[1]))
	}
	return strings.Join(lines, "\n")
}

// ----- Browse / search lesson catalog --------------------------------------

func (a *App) updateBrowse(msg tea.KeyMsg) {
	key := msg.String()
	if a.browseEditing {
		switch key {
		case "esc":
			a.browseEditing = false
		case "enter":
			a.browseEditing = false
		case "backspace":
			runes := []rune(a.browseFilter)
			if len(runes) > 0 {
				a.browseFilter = string(runes[:len(runes)-1])
			}
			a.browseCursor = 0
		default:
			if len([]rune(key)) == 1 {
				a.browseFilter += key
				a.browseCursor = 0
			}
		}
		return
	}
	items := a.browseFilteredLessons()
	switch key {
	case "esc", "f2":
		a.route = routeHome
	case "j", "down":
		if a.browseCursor < len(items)-1 {
			a.browseCursor++
		}
	case "k", "up":
		if a.browseCursor > 0 {
			a.browseCursor--
		}
	case "/":
		a.browseEditing = true
	case "x":
		// Clear filter quickly.
		a.browseFilter = ""
		a.browseCursor = 0
	case "enter", " ":
		if len(items) == 0 {
			return
		}
		lesson := items[a.browseCursor]
		a.startSingleLesson("browse", lesson.ID, nil, 0)
	}
}

func (a *App) browseFilteredLessons() []content.Lesson {
	all := a.catalog.Lessons()
	if a.browseFilter == "" {
		return all
	}
	needle := strings.ToLower(a.browseFilter)
	var out []content.Lesson
	for _, lesson := range all {
		hay := strings.ToLower(lesson.ID + " " + lesson.Title + " " + lesson.ModuleID + " " + strings.Join(lesson.Skills, " ") + " " + strings.Join(lesson.CommandsLearned, " "))
		if strings.Contains(hay, needle) {
			out = append(out, lesson)
		}
	}
	return out
}

func (a *App) viewBrowse() string {
	items := a.browseFilteredLessons()
	if a.browseCursor >= len(items) {
		a.browseCursor = max0(len(items) - 1)
	}
	var lines []string
	lines = append(lines, bold("Browse Lessons"))
	prompt := dim("Filter:")
	if a.browseEditing {
		prompt = cyan("Filter:")
	}
	filterDisplay := a.browseFilter
	if filterDisplay == "" {
		filterDisplay = dim("(none)")
	}
	lines = append(lines, prompt+" "+filterDisplay+"   "+dim(fmt.Sprintf("(%d match%s)", len(items), pluralS(len(items)))))
	lines = append(lines, "")
	available := a.height - 8
	if available < 4 {
		available = 4
	}
	start, end := visibleRange(len(items), a.browseCursor, available)
	if start > 0 {
		lines = append(lines, dim("  ..."))
	}
	for i := start; i < end; i++ {
		lesson := items[i]
		prefix := "  "
		if i == a.browseCursor {
			prefix = cyan("> ")
		}
		mastery := "·"
		if entry, ok := a.profile.SkillProgress[firstSkill(lesson)]; ok && entry != nil {
			switch entry.Mastery {
			case progress.MasteryMastered:
				mastery = green("✓")
			case progress.MasteryPracticing:
				mastery = yellow("~")
			case progress.MasteryNovice:
				mastery = yellow("!")
			}
		}
		lines = append(lines, fmt.Sprintf("%s%s %s  %s%s", prefix, mastery, lesson.Title, dim(lesson.ModuleID+"/"+lesson.ID), reset()))
	}
	if end < len(items) {
		lines = append(lines, dim("  ..."))
	}
	lines = append(lines, "")
	if a.browseEditing {
		lines = append(lines, a.footer("Type to filter. Enter or Esc to stop editing."))
	} else {
		lines = append(lines, a.footer("/ filter · x clear · Enter open · j/k move · Esc home · :q quit"))
	}
	return strings.Join(lines, "\n")
}

func firstSkill(lesson content.Lesson) string {
	if len(lesson.Skills) == 0 {
		return ""
	}
	return lesson.Skills[0]
}

func max0(v int) int {
	if v < 0 {
		return 0
	}
	return v
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "es"
}

// ----- Diagnostic placement quiz -------------------------------------------

func (a *App) startDiagnostic() {
	a.diagnostic = &diagnosticSession{
		Questions: []diagnosticQuestion{
			{
				Prompt:          "Comfortable using h/j/k/l, w/b, gg/G, and counts like 3j?",
				YesGrantsLesson: []string{"motions-hjkl", "motions-word", "motions-anchors", "motions-file"},
			},
			{
				Prompt:          "Comfortable with i/a/o/O, x, dd, dw, cw, yy/p, and u/<C-r>?",
				YesGrantsLesson: []string{"onboarding-insert", "operators-x", "operators-dw-dd", "operators-cw-d$", "editing-openlines", "editing-undo-redo", "editing-yank-paste"},
			},
			{
				Prompt:          "Comfortable with /pattern, *, n/N, marks (ma / 'a)?",
				YesGrantsLesson: []string{"search-repeat", "search-star-noh", "search-marks"},
			},
			{
				Prompt:          "Comfortable with text objects like diw, ci\", di(, dap?",
				YesGrantsLesson: []string{"textobjects-diw", "textobjects-caw", "textobjects-quotes", "textobjects-parens", "textobjects-paragraph"},
			},
			{
				Prompt:          "Comfortable with :set options, :w/:q, :e file, :Ex, and :%!sed?",
				YesGrantsLesson: []string{"onboarding-commandline", "commandline-options", "commandline-search-tuning", "commandline-help-source", "commandline-sed-filter", "commandline-files"},
			},
			{
				Prompt:          "Comfortable recording and replaying macros (qa…q, @a, .)?",
				YesGrantsLesson: []string{"macros-repeat", "macros-basic"},
			},
			{
				Prompt:          "Comfortable with visual mode (v, V, then d/y/p, >, <)?",
				YesGrantsLesson: []string{"visual-basic-delete", "visual-line-yank-paste"},
			},
		},
		Answers: map[int]bool{},
	}
	a.route = routeDiagnostic
}

func (a *App) updateDiagnostic(msg tea.KeyMsg) {
	if a.diagnostic == nil {
		a.route = routeHome
		return
	}
	key := msg.String()
	switch key {
	case "esc", "f2":
		a.route = routeHome
		a.diagnostic = nil
	case "y":
		a.diagnostic.Answers[a.diagnostic.Index] = true
		a.advanceDiagnostic()
	case "n":
		a.diagnostic.Answers[a.diagnostic.Index] = false
		a.advanceDiagnostic()
	}
}

func (a *App) advanceDiagnostic() {
	a.diagnostic.Index++
	if a.diagnostic.Index < len(a.diagnostic.Questions) {
		return
	}
	// Apply: mark each lesson granted by a "yes" answer as completed.
	granted := 0
	for i, yes := range a.diagnostic.Answers {
		if !yes {
			continue
		}
		for _, lessonID := range a.diagnostic.Questions[i].YesGrantsLesson {
			if _, ok := a.catalog.Lesson(lessonID); !ok {
				continue
			}
			if !a.profile.CompletedLessons[lessonID] {
				a.profile.CompletedLessons[lessonID] = true
				a.profile.LessonCompletions++
				granted++
			}
		}
	}
	a.profile.DiagnosticTaken = true
	a.profile.RefreshModules(a.catalog)
	if newly := a.profile.EvaluateAchievements(a.catalog); len(newly) > 0 {
		a.achievementsToast = append(a.achievementsToast, newly...)
	}
	_ = a.store.Save(a.profile)
	a.status = fmt.Sprintf("Placement complete — marked %d lesson(s) as already known.", granted)
	a.diagnostic = nil
	a.route = routeHome
}

func (a *App) viewDiagnostic() string {
	if a.diagnostic == nil {
		return "No diagnostic in progress."
	}
	q := a.diagnostic.Questions[a.diagnostic.Index]
	var lines []string
	lines = append(lines, bold("Placement Quiz"))
	lines = append(lines, dim(fmt.Sprintf("Question %d of %d — answer honestly; this only adjusts what's marked as already-known.", a.diagnostic.Index+1, len(a.diagnostic.Questions))))
	lines = append(lines, "")
	lines = append(lines, q.Prompt)
	lines = append(lines, "")
	lines = append(lines, a.footer("y = yes, mark known · n = no, keep practicing · Esc cancel"))
	return strings.Join(lines, "\n")
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

// expectedAnswer returns the shortest canonical solution for a lesson —
// the same one parKeysFromCanonical scores against — so the debug
// overlay can surface "here's the intended chord." Returns "" when the
// lesson has no canonical solutions on file.
func expectedAnswer(lesson content.Lesson) string {
	if len(lesson.CanonicalSolutions) == 0 {
		return ""
	}
	best := lesson.CanonicalSolutions[0]
	for _, sol := range lesson.CanonicalSolutions[1:] {
		if len(sol) < len(best) {
			best = sol
		}
	}
	return best
}

func tokenOrDash(token string) string {
	if token == "" {
		return "-"
	}
	return token
}

// highlightMatch wraps the rune-indexed [start, end) substring of `line`
// in inverse video so the substitute-confirm prompt visibly shows which
// occurrence is being asked about.
func highlightMatch(line string, start, end int) string {
	runes := []rune(line)
	if start < 0 {
		start = 0
	}
	if end > len(runes) {
		end = len(runes)
	}
	if start >= end {
		return line
	}
	return string(runes[:start]) + invert(string(runes[start:end])) + string(runes[end:])
}
