package progress

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"vimtrainer/internal/config"
	"vimtrainer/internal/content"
)

type Mastery string

const (
	MasteryUnseen     Mastery = "unseen"
	MasteryNovice     Mastery = "novice"
	MasteryPracticing Mastery = "practicing"
	MasteryMastered   Mastery = "mastered"
)

type SkillProgress struct {
	Attempts               int       `json:"attempts"`
	Successes              int       `json:"successes"`
	RecentMistakes         []string  `json:"recent_mistakes"`
	AverageCompletionMilli float64   `json:"average_completion_milli"`
	LastSeenAt             time.Time `json:"last_seen_at"`
	Mastery                Mastery   `json:"mastery"`
	// Spaced-repetition fields (SM-2-style). Interval is days until next
	// due; EaseFactor scales the interval on each successful repetition;
	// DueAt is when this skill becomes eligible for review again. Lapses
	// counts how many times the skill has dropped back to interval=1.
	IntervalDays int       `json:"interval_days,omitempty"`
	EaseFactor   float64   `json:"ease_factor,omitempty"`
	DueAt        time.Time `json:"due_at,omitempty"`
	Lapses       int       `json:"lapses,omitempty"`
	// StreakSuccesses counts consecutive successful attempts since the
	// last failure; resets on lapse. Used by Achievement checks.
	StreakSuccesses int `json:"streak_successes,omitempty"`
}

type Settings struct {
	ShowHints bool `json:"show_hints"`
	Debug     bool `json:"debug"`
	// P3 expansion — UI-side tunables. The trainer doesn't ship a
	// theme picker engine; these are advisory flags surfaced in the
	// Settings screen so a learner can opt into accessibility-friendly
	// rendering. Future renderers can honor them.
	Theme         string `json:"theme,omitempty"`           // "default" | "high-contrast" | "monochrome"
	ColorblindSafe bool   `json:"colorblind_safe,omitempty"` // swap red/green cues
	ReducedMotion  bool   `json:"reduced_motion,omitempty"`  // skip transition flourishes
	LargerCursor   bool   `json:"larger_cursor,omitempty"`   // emphasize cursor for low-vision users
	KeyRepeatDelay int    `json:"key_repeat_delay_ms,omitempty"`
}

type Profile struct {
	SchemaVersion     int                       `json:"schema_version,omitempty"`
	LastLessonID      string                    `json:"last_lesson_id"`
	LastMode          string                    `json:"last_mode"`
	CompletedLessons  map[string]bool           `json:"completed_lessons"`
	CompletedModules  map[string]bool           `json:"completed_modules"`
	SkillProgress     map[string]*SkillProgress `json:"skill_progress"`
	Settings          Settings                  `json:"settings"`
	LessonCompletions int                       `json:"lesson_completions"`
	Achievements      map[string]time.Time      `json:"achievements,omitempty"`
	SessionLog        []SessionEntry            `json:"session_log,omitempty"`
	ParStreak         int                       `json:"par_streak,omitempty"`
	BestParStreak     int                       `json:"best_par_streak,omitempty"`
	DiagnosticTaken   bool                      `json:"diagnostic_taken,omitempty"`
	// LessonHashes records the content hash of each lesson at the time
	// the learner most recently completed it. The stats screen uses this
	// to flag lessons whose content has been edited since (per Q1=a,
	// informational only — completion status is not reset).
	LessonHashes map[string]string `json:"lesson_hashes,omitempty"`
}

// SessionEntry records one lesson attempt outcome — kept for the practice
// log export and for achievement queries that need recent history.
type SessionEntry struct {
	Time       time.Time     `json:"time"`
	LessonID   string        `json:"lesson_id"`
	Mode       string        `json:"mode"`
	Success    bool          `json:"success"`
	Duration   time.Duration `json:"duration_ns"`
	Keystrokes int           `json:"keystrokes"`
	Mistakes   []string      `json:"mistakes,omitempty"`
}

type Store struct {
	root string
}

type RankedLesson struct {
	Lesson content.Lesson
	Score  float64
}

type QueueStyle string

const (
	QueuePractice  QueueStyle = "practice"
	QueueReview    QueueStyle = "review"
	QueueChallenge QueueStyle = "challenge"
)

func NewStore(root string) *Store {
	return &Store{root: root}
}

func (s *Store) Load() (*Profile, error) {
	path, err := s.profilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			profile := defaultProfile()
			if err := s.Save(profile); err != nil {
				return nil, err
			}
			return profile, nil
		}
		return nil, err
	}

	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}
	normalizeProfile(&profile)
	return &profile, nil
}

func (s *Store) Save(profile *Profile) error {
	normalizeProfile(profile)
	path, err := s.profilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s *Store) Reset() error {
	path, err := s.profilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (p *Profile) RecordLesson(lesson content.Lesson, success bool, duration time.Duration, note string) {
	normalizeProfile(p)
	now := time.Now()
	for _, skill := range lesson.Skills {
		entry := p.ensureSkill(skill)
		entry.Attempts++
		if success {
			entry.Successes++
			entry.StreakSuccesses++
			advanceSRSOnSuccess(entry, now)
		} else {
			if note != "" {
				entry.RecentMistakes = append([]string{note}, entry.RecentMistakes...)
				if len(entry.RecentMistakes) > config.RecentMistakesCap {
					entry.RecentMistakes = entry.RecentMistakes[:config.RecentMistakesCap]
				}
			}
			advanceSRSOnLapse(entry, now)
		}
		entry.LastSeenAt = now
		if duration > 0 {
			ms := float64(duration.Milliseconds())
			if entry.Attempts == 1 {
				entry.AverageCompletionMilli = ms
			} else {
				entry.AverageCompletionMilli = ((entry.AverageCompletionMilli * float64(entry.Attempts-1)) + ms) / float64(entry.Attempts)
			}
		}
		entry.Mastery = deriveMastery(*entry)
	}
	if success {
		p.CompletedLessons[lesson.ID] = true
		p.LastLessonID = lesson.ID
		p.LessonCompletions++
		// Record the content fingerprint of this completion so the stats
		// screen can later flag the lesson as stale if its content
		// changes (Q1=a).
		if p.LessonHashes == nil {
			p.LessonHashes = map[string]string{}
		}
		p.LessonHashes[lesson.ID] = lesson.ContentHash()
	}
}

// StaleLessons returns the IDs of lessons the learner completed in the
// past whose content has been edited since. Order is stable (sorted) so
// the stats screen renders deterministically.
func (p *Profile) StaleLessons(catalog *content.Catalog) []string {
	normalizeProfile(p)
	var stale []string
	for id, recorded := range p.LessonHashes {
		lesson, ok := catalog.Lesson(id)
		if !ok {
			continue
		}
		if lesson.ContentHash() != recorded {
			stale = append(stale, id)
		}
	}
	sort.Strings(stale)
	return stale
}

// RecordSession appends a single attempt outcome to the session log and
// updates the par-streak counter. The full lesson is passed in so we can
// grade keystrokes against OptimalKeys.
func (p *Profile) RecordSession(lesson content.Lesson, mode string, success bool, duration time.Duration, keystrokes int, mistakes []string) {
	normalizeProfile(p)
	entry := SessionEntry{
		Time:       time.Now(),
		LessonID:   lesson.ID,
		Mode:       mode,
		Success:    success,
		Duration:   duration,
		Keystrokes: keystrokes,
		Mistakes:   append([]string{}, mistakes...),
	}
	p.SessionLog = append(p.SessionLog, entry)
	if len(p.SessionLog) > config.SessionLogCap {
		p.SessionLog = p.SessionLog[len(p.SessionLog)-config.SessionLogCap:]
	}
	if success && lesson.OptimalKeys > 0 && keystrokes <= lesson.OptimalKeys {
		p.ParStreak++
		if p.ParStreak > p.BestParStreak {
			p.BestParStreak = p.ParStreak
		}
	} else if !success {
		p.ParStreak = 0
	}
}

// advanceSRSOnSuccess promotes the SRS interval the way SM-2 does: first
// success → SRSInitialIntervalDays, second → SRSSecondIntervalDays, then
// multiply by ease factor. Ease factor nudges up (capped at SRSEaseMax)
// on each clean success.
func advanceSRSOnSuccess(entry *SkillProgress, now time.Time) {
	if entry.EaseFactor < config.SRSEaseMin {
		entry.EaseFactor = config.SRSEaseDefault
	}
	switch {
	case entry.IntervalDays <= 0:
		entry.IntervalDays = config.SRSInitialIntervalDays
	case entry.IntervalDays == config.SRSInitialIntervalDays:
		entry.IntervalDays = config.SRSSecondIntervalDays
	default:
		next := float64(entry.IntervalDays) * entry.EaseFactor
		entry.IntervalDays = int(next + 0.5)
	}
	entry.EaseFactor += config.SRSEaseBumpOnSuccess
	if entry.EaseFactor > config.SRSEaseMax {
		entry.EaseFactor = config.SRSEaseMax
	}
	entry.DueAt = now.AddDate(0, 0, entry.IntervalDays)
}

// advanceSRSOnLapse resets the interval and dings the ease factor.
func advanceSRSOnLapse(entry *SkillProgress, now time.Time) {
	entry.Lapses++
	entry.StreakSuccesses = 0
	if entry.EaseFactor < config.SRSEaseMin {
		entry.EaseFactor = config.SRSEaseDefault
	}
	entry.EaseFactor -= config.SRSEaseBumpOnLapse
	if entry.EaseFactor < config.SRSEaseMin {
		entry.EaseFactor = config.SRSEaseMin
	}
	entry.IntervalDays = config.SRSInitialIntervalDays
	entry.DueAt = now.AddDate(0, 0, config.SRSInitialIntervalDays)
}

// IsDue reports whether a skill is currently due for review under the SRS
// schedule. Treats unseen / never-scheduled skills as due so a fresh
// profile gets work to do.
func (entry *SkillProgress) IsDue(now time.Time) bool {
	if entry.IntervalDays <= 0 {
		return true
	}
	if entry.DueAt.IsZero() {
		return true
	}
	return !entry.DueAt.After(now)
}

func (p *Profile) ensureSkill(skill string) *SkillProgress {
	entry, ok := p.SkillProgress[skill]
	if !ok {
		entry = &SkillProgress{Mastery: MasteryUnseen}
		p.SkillProgress[skill] = entry
	}
	return entry
}

func (p *Profile) IsModuleCompleted(catalog *content.Catalog, moduleID string) bool {
	lessons := catalog.LessonsForModule(moduleID)
	if len(lessons) == 0 {
		return false
	}
	for _, lesson := range lessons {
		if !p.CompletedLessons[lesson.ID] {
			return false
		}
	}
	return true
}

func (p *Profile) RefreshModules(catalog *content.Catalog) {
	normalizeProfile(p)
	for _, module := range catalog.Modules() {
		p.CompletedModules[module.ID] = p.IsModuleCompleted(catalog, module.ID)
	}
}

func (p *Profile) ModuleUnlocked(module content.Module) bool {
	for _, prereq := range module.PrerequisiteModules {
		if !p.CompletedModules[prereq] {
			return false
		}
	}
	return true
}

func (p *Profile) RecommendedLessons(catalog *content.Catalog, reviewOnly bool, limit int) []content.Lesson {
	normalizeProfile(p)
	now := time.Now()
	var ranked []RankedLesson
	for _, lesson := range catalog.Lessons() {
		score := 0.0
		anyDue := false
		anyTroubled := false
		for _, skill := range lesson.Skills {
			entry := p.ensureSkill(skill)
			// Mastery-tier base weight (lower mastery = more practice
			// pressure).
			switch entry.Mastery {
			case MasteryUnseen:
				score += 4.0
			case MasteryNovice:
				score += 3.0
			case MasteryPracticing:
				score += 1.5
			case MasteryMastered:
				score += 0.3
			}
			// SRS due signal: overdue lessons score higher in proportion
			// to how late they are. Capped so single ancient skills don't
			// dominate the queue.
			if entry.IsDue(now) {
				anyDue = true
				if !entry.DueAt.IsZero() {
					overdueDays := now.Sub(entry.DueAt).Hours() / 24
					if overdueDays > 0 {
						score += minFloat(3.0, overdueDays/2)
					}
				} else {
					score += 1.0
				}
			}
			if len(entry.RecentMistakes) > 0 {
				score += 2.5
				anyTroubled = true
			}
			if entry.Attempts > 0 && entry.Successes == 0 {
				score += 2
				anyTroubled = true
			}
			if entry.Lapses > 0 {
				score += minFloat(2.0, float64(entry.Lapses)*0.5)
			}
			if entry.AverageCompletionMilli > 0 {
				score += minFloat(1.5, entry.AverageCompletionMilli/45000.0)
			}
		}
		// Review queue should only surface lessons with a real reason to
		// repeat them: due, recent mistake, or never-mastered.
		if reviewOnly && !anyDue && !anyTroubled {
			continue
		}
		ranked = append(ranked, RankedLesson{Lesson: lesson, Score: score})
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].Lesson.ID < ranked[j].Lesson.ID
		}
		return ranked[i].Score > ranked[j].Score
	})

	if limit > 0 && len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]content.Lesson, 0, len(ranked))
	for _, item := range ranked {
		out = append(out, item.Lesson)
	}
	return out
}

func (p *Profile) QueueForStyle(catalog *content.Catalog, style QueueStyle, limit int) []content.Lesson {
	switch style {
	case QueueReview:
		return p.RecommendedLessons(catalog, true, limit)
	case QueueChallenge:
		items := p.challengeLessons(catalog, limit)
		if len(items) > 0 {
			return items
		}
		return p.RecommendedLessons(catalog, false, limit)
	default:
		return p.RecommendedLessons(catalog, false, limit)
	}
}

func (p *Profile) challengeLessons(catalog *content.Catalog, limit int) []content.Lesson {
	normalizeProfile(p)
	var ranked []RankedLesson
	for _, lesson := range catalog.Lessons() {
		score := 0.0
		eligible := true
		for _, skill := range lesson.Skills {
			entry := p.ensureSkill(skill)
			if entry.Attempts == 0 {
				eligible = false
				break
			}
			switch entry.Mastery {
			case MasteryMastered:
				score += 3
			case MasteryPracticing:
				score += 2
			case MasteryNovice:
				score += 1.5
			}
			if !entry.LastSeenAt.IsZero() {
				score += minFloat(3, time.Since(entry.LastSeenAt).Hours()/72)
			}
			if len(entry.RecentMistakes) > 0 {
				score += 1.25
			}
		}
		if !eligible {
			continue
		}
		ranked = append(ranked, RankedLesson{Lesson: lesson, Score: score})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].Lesson.ID < ranked[j].Lesson.ID
		}
		return ranked[i].Score > ranked[j].Score
	})
	if limit > 0 && len(ranked) > limit {
		ranked = ranked[:limit]
	}
	out := make([]content.Lesson, 0, len(ranked))
	for _, item := range ranked {
		out = append(out, item.Lesson)
	}
	return out
}

func defaultProfile() *Profile {
	return &Profile{
		CompletedLessons: map[string]bool{},
		CompletedModules: map[string]bool{},
		SkillProgress:    map[string]*SkillProgress{},
		Settings: Settings{
			ShowHints: true,
			Debug:     false,
		},
	}
}

func normalizeProfile(profile *Profile) {
	if profile.CompletedLessons == nil {
		profile.CompletedLessons = map[string]bool{}
	}
	if profile.CompletedModules == nil {
		profile.CompletedModules = map[string]bool{}
	}
	if profile.SkillProgress == nil {
		profile.SkillProgress = map[string]*SkillProgress{}
	}
	if profile.Achievements == nil {
		profile.Achievements = map[string]time.Time{}
	}
	if profile.LessonHashes == nil {
		profile.LessonHashes = map[string]string{}
	}
	// Stamp the schema version on first save so future migrations can
	// read it without ambiguity.
	if profile.SchemaVersion == 0 {
		profile.SchemaVersion = config.SchemaVersion
	}
	if !profile.Settings.ShowHints && !profile.Settings.Debug && profile.Settings == (Settings{}) {
		profile.Settings.ShowHints = true
	}
}

func deriveMastery(entry SkillProgress) Mastery {
	if entry.Attempts == 0 {
		return MasteryUnseen
	}
	rate := float64(entry.Successes) / float64(entry.Attempts)
	switch {
	case entry.Successes >= config.MasterySuccessesRequired && rate >= config.MasteryRateRequired:
		return MasteryMastered
	case entry.Successes >= 1 && rate >= config.PracticingRateRequired:
		return MasteryPracticing
	default:
		return MasteryNovice
	}
}

func (s *Store) profilePath() (string, error) {
	if s.root != "" {
		return filepath.Join(s.root, "profile.json"), nil
	}
	if override := os.Getenv("VIMTRAINER_PROFILE_DIR"); override != "" {
		return filepath.Join(override, "profile.json"), nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", homeErr
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "vimtrainer", "profile.json"), nil
}

func (s *Store) Export(path string) error {
	profile, err := s.Load()
	if err != nil {
		return err
	}
	if path == "" {
		path = "vimtrainer-profile.json"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// ExportSessionLog writes the profile's session log out as CSV. Useful
// for self-review without parsing the JSON profile.
func (s *Store) ExportSessionLog(path string) error {
	profile, err := s.Load()
	if err != nil {
		return err
	}
	if path == "" {
		path = "vimtrainer-sessions.csv"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	if err := w.Write([]string{"time", "lesson_id", "mode", "success", "duration_ms", "keystrokes", "mistakes"}); err != nil {
		return err
	}
	for _, e := range profile.SessionLog {
		row := []string{
			e.Time.Format(time.RFC3339),
			e.LessonID,
			e.Mode,
			fmt.Sprintf("%t", e.Success),
			fmt.Sprintf("%d", e.Duration.Milliseconds()),
			fmt.Sprintf("%d", e.Keystrokes),
			strings.Join(e.Mistakes, "|"),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Import(path string) error {
	if path == "" {
		path = "vimtrainer-profile.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return err
	}
	normalizeProfile(&profile)
	return s.Save(&profile)
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
