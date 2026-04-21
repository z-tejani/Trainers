package progress

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

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
}

type Settings struct {
	ShowHints bool `json:"show_hints"`
	Debug     bool `json:"debug"`
}

type Profile struct {
	LastLessonID      string                    `json:"last_lesson_id"`
	LastMode          string                    `json:"last_mode"`
	CompletedLessons  map[string]bool           `json:"completed_lessons"`
	CompletedModules  map[string]bool           `json:"completed_modules"`
	SkillProgress     map[string]*SkillProgress `json:"skill_progress"`
	Settings          Settings                  `json:"settings"`
	LessonCompletions int                       `json:"lesson_completions"`
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
	for _, skill := range lesson.Skills {
		entry := p.ensureSkill(skill)
		entry.Attempts++
		if success {
			entry.Successes++
		} else if note != "" {
			entry.RecentMistakes = append([]string{note}, entry.RecentMistakes...)
			if len(entry.RecentMistakes) > 5 {
				entry.RecentMistakes = entry.RecentMistakes[:5]
			}
		}
		entry.LastSeenAt = time.Now()
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
	}
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
	var ranked []RankedLesson
	for _, lesson := range catalog.Lessons() {
		score := 0.0
		for _, skill := range lesson.Skills {
			entry := p.ensureSkill(skill)
			switch entry.Mastery {
			case MasteryUnseen:
				score += 4.0
			case MasteryNovice:
				score += 3.0
			case MasteryPracticing:
				score += 2.0
			case MasteryMastered:
				score += 0.5
			}
			if len(entry.RecentMistakes) > 0 {
				score += 2.5
			}
			if entry.Attempts > 0 && entry.Successes == 0 {
				score += 2
			}
			if !entry.LastSeenAt.IsZero() {
				score += minFloat(3, time.Since(entry.LastSeenAt).Hours()/48)
			}
			if entry.AverageCompletionMilli > 0 {
				score += minFloat(1.5, entry.AverageCompletionMilli/45000.0)
			}
		}
		if reviewOnly && score < 2 {
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
	case entry.Successes >= 3 && rate >= 0.8:
		return MasteryMastered
	case entry.Successes >= 1 && rate >= 0.5:
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
