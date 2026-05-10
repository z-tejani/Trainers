// Package config centralizes the trainer's tunable constants. Values are
// exported (not loaded from disk) so they're easy to grep and override at
// the source. Per the foundation decision in stretch.md (Q3=a), this is
// not a runtime config story — it just removes the inline literals.
//
// Treat this package as the single point of truth: queue sizes, mastery
// thresholds, SRS bumps, achievement thresholds, session-log caps. When
// adding a new tunable elsewhere in the codebase, add it here first and
// reference it from the use site.
package config

import "time"

// Queue ----------------------------------------------------------------------

// PracticeQueueLimit caps how many lessons get queued in a Practice run.
const PracticeQueueLimit = 6

// ChallengeQueueLimit caps how many lessons get queued in a Challenge run.
const ChallengeQueueLimit = 10

// ChallengeModuleQueueLimit caps a per-module Challenge queue.
const ChallengeModuleQueueLimit = 10

// Mastery --------------------------------------------------------------------

// MasterySuccessesRequired is the minimum number of clean successes
// before a skill is eligible for the Mastered tier.
const MasterySuccessesRequired = 3

// MasteryRateRequired is the minimum success-rate threshold for the
// Mastered tier (0.0–1.0).
const MasteryRateRequired = 0.8

// PracticingRateRequired is the success-rate threshold above which a
// skill graduates from Novice to Practicing.
const PracticingRateRequired = 0.5

// SRS ------------------------------------------------------------------------

// SRSInitialIntervalDays is how many days the SRS schedules a skill out
// after its first clean success.
const SRSInitialIntervalDays = 1

// SRSSecondIntervalDays is the bump after the second clean success
// (before the ease-factor multiplier kicks in for subsequent reps).
const SRSSecondIntervalDays = 3

// SRSEaseDefault is the initial / reset ease factor for new skills.
const SRSEaseDefault = 2.5

// SRSEaseMin is the floor; ease can't drop below this even after lapses.
const SRSEaseMin = 1.3

// SRSEaseMax is the ceiling; ease can't grow past this.
const SRSEaseMax = 2.6

// SRSEaseBumpOnSuccess is the additive nudge per clean success.
const SRSEaseBumpOnSuccess = 0.1

// SRSEaseBumpOnLapse is the additive penalty per lapse (subtracted).
const SRSEaseBumpOnLapse = 0.2

// Session log ---------------------------------------------------------------

// SessionLogCap is the maximum number of session entries kept on the
// profile. Older entries are dropped from the front when the cap is hit.
const SessionLogCap = 500

// QuickfixHistoryCap is how many past quickfix lists the engine retains
// for :cnewer / :colder navigation.
const QuickfixHistoryCap = 10

// Mistakes -------------------------------------------------------------------

// RecentMistakesCap is how many recent mistake notes per skill are kept.
const RecentMistakesCap = 5

// Hints / lesson view --------------------------------------------------------

// HintsPerErrorTier is how many wrong attempts unlock the next hint.
const HintsPerErrorTier = 3

// Editor -------------------------------------------------------------------

// IndentUnit is the trainer's standard shift width — used by `>>`, `<<`,
// and visual `>` / `<`.
const IndentUnit = "  "

// Achievements ---------------------------------------------------------------

// AchievementsPerfectionistStreak is the par streak required for the
// Perfectionist achievement.
const AchievementsPerfectionistStreak = 10

// SchemaVersion is bumped when the on-disk profile shape changes in a
// non-additive way. Loaders consult it for migrations.
const SchemaVersion = 2

// EditorMaxJumpListEntries caps the engine's jumplist depth.
const EditorMaxJumpListEntries = 200

// EditorMaxChangeListEntries caps the engine's changelist depth.
const EditorMaxChangeListEntries = 200

// Time helpers ---------------------------------------------------------------

// SRSInterval converts a day count to a duration. Kept here so any
// future "minutes for testing" adjustment is one place to change.
func SRSInterval(days int) time.Duration {
	return time.Duration(days) * 24 * time.Hour
}
