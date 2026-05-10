package progress

import (
	"time"

	"vimtrainer/internal/config"
	"vimtrainer/internal/content"
)

// Achievement is a milestone the learner can unlock by reaching some state
// in their profile. Definitions live in DefaultAchievements; unlocked IDs
// are stored in profile.Achievements as an ID → unlock-time map.
type Achievement struct {
	ID    string
	Title string
	Desc  string
	// Check returns true once the profile satisfies the achievement.
	// catalog is provided for module-coverage queries.
	Check func(p *Profile, catalog *content.Catalog) bool
}

// DefaultAchievements is the catalog of milestones the trainer ships with.
// Adding a new one only requires appending here; the engine evaluates them
// after every successful lesson via Profile.EvaluateAchievements.
func DefaultAchievements() []Achievement {
	return []Achievement{
		{
			ID:    "first-steps",
			Title: "First Steps",
			Desc:  "Complete your first lesson.",
			Check: func(p *Profile, _ *content.Catalog) bool { return p.LessonCompletions >= 1 },
		},
		{
			ID:    "apprentice",
			Title: "Apprentice",
			Desc:  "Complete five lessons.",
			Check: func(p *Profile, _ *content.Catalog) bool { return p.LessonCompletions >= 5 },
		},
		{
			ID:    "journeyman",
			Title: "Journeyman",
			Desc:  "Complete twenty-five lessons.",
			Check: func(p *Profile, _ *content.Catalog) bool { return p.LessonCompletions >= 25 },
		},
		{
			ID:    "adept",
			Title: "Adept",
			Desc:  "Complete fifty lessons.",
			Check: func(p *Profile, _ *content.Catalog) bool { return p.LessonCompletions >= 50 },
		},
		{
			ID:    "module-master",
			Title: "Module Master",
			Desc:  "Fully complete any one module.",
			Check: func(p *Profile, _ *content.Catalog) bool {
				for _, done := range p.CompletedModules {
					if done {
						return true
					}
				}
				return false
			},
		},
		{
			ID:    "all-modules",
			Title: "All Modules",
			Desc:  "Fully complete every module in the catalog.",
			Check: func(p *Profile, catalog *content.Catalog) bool {
				modules := catalog.Modules()
				if len(modules) == 0 {
					return false
				}
				for _, m := range modules {
					if !p.CompletedModules[m.ID] {
						return false
					}
				}
				return true
			},
		},
		{
			ID:    "speed-demon",
			Title: "Speed Demon",
			Desc:  "Solve a calibrated lesson at par or better.",
			Check: func(p *Profile, _ *content.Catalog) bool { return p.BestParStreak >= 1 },
		},
		{
			ID:    "perfectionist",
			Title: "Perfectionist",
			Desc:  "Solve ten calibrated lessons in a row at par or better.",
			Check: func(p *Profile, _ *content.Catalog) bool {
				return p.BestParStreak >= config.AchievementsPerfectionistStreak
			},
		},
		{
			ID:    "comeback-kid",
			Title: "Comeback Kid",
			Desc:  "Recover from a lapse: master a skill that previously slipped.",
			Check: func(p *Profile, _ *content.Catalog) bool {
				for _, entry := range p.SkillProgress {
					if entry == nil {
						continue
					}
					if entry.Lapses > 0 && entry.Mastery == MasteryMastered {
						return true
					}
				}
				return false
			},
		},
		{
			ID:    "well-rested",
			Title: "Well Rested",
			Desc:  "Pass an SRS-due review (a skill that came back from the schedule).",
			Check: func(p *Profile, _ *content.Catalog) bool {
				for _, entry := range p.SkillProgress {
					if entry == nil {
						continue
					}
					if entry.IntervalDays >= 7 && entry.Mastery == MasteryMastered {
						return true
					}
				}
				return false
			},
		},
	}
}

// EvaluateAchievements walks the catalog of achievements and unlocks any
// the profile now satisfies. Returns the IDs that were newly unlocked
// during this call so the UI can announce them.
func (p *Profile) EvaluateAchievements(catalog *content.Catalog) []string {
	normalizeProfile(p)
	var newly []string
	for _, ach := range DefaultAchievements() {
		if _, already := p.Achievements[ach.ID]; already {
			continue
		}
		if ach.Check(p, catalog) {
			p.Achievements[ach.ID] = time.Now()
			newly = append(newly, ach.ID)
		}
	}
	return newly
}
