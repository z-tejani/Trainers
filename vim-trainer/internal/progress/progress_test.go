package progress

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"vimtrainer/internal/content"
)

func TestStoreLoadCreatesDefaultProfile(t *testing.T) {
	store := NewStore(t.TempDir())
	profile, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !profile.Settings.ShowHints {
		t.Fatal("expected default hints to be enabled")
	}
}

func TestRecordLessonUpdatesSkillProgress(t *testing.T) {
	catalog := content.NewCatalog()
	lesson, ok := catalog.Lesson("onboarding-insert")
	if !ok {
		t.Fatal("lesson not found")
	}
	profile := defaultProfile()
	profile.RecordLesson(lesson, true, 2*time.Second, "")

	for _, skill := range lesson.Skills {
		if profile.SkillProgress[skill] == nil {
			t.Fatalf("expected skill progress for %s", skill)
		}
	}
}

func TestRecommendedLessonsPrioritizesWeakSkills(t *testing.T) {
	catalog := content.NewCatalog()
	profile := defaultProfile()
	lesson, _ := catalog.Lesson("motions-word")
	profile.RecordLesson(lesson, false, 0, "used l instead of w")

	recs := profile.RecommendedLessons(catalog, true, 3)
	if len(recs) == 0 {
		t.Fatal("expected review recommendations")
	}
}

func TestStoreSaveRoundTrip(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	profile := defaultProfile()
	profile.LastLessonID = "motions-word"
	if err := store.Save(profile); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.LastLessonID != "motions-word" {
		t.Fatalf("LastLessonID = %q", loaded.LastLessonID)
	}
	if _, err := filepath.Abs(root); err != nil {
		t.Fatal(err)
	}
}

func TestQueueForChallengePrefersSeenSkills(t *testing.T) {
	catalog := content.NewCatalog()
	profile := defaultProfile()
	lesson, _ := catalog.Lesson("motions-word")
	profile.RecordLesson(lesson, true, time.Second, "")

	queue := profile.QueueForStyle(catalog, QueueChallenge, 5)
	if len(queue) == 0 {
		t.Fatal("expected challenge queue items")
	}
}

func TestStoreExportImport(t *testing.T) {
	root := t.TempDir()
	store := NewStore(root)
	profile, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	profile.LastLessonID = "macros-basic"
	if err := store.Save(profile); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	out := filepath.Join(root, "backup", "profile.json")
	if err := store.Export(out); err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	if _, err := os.Stat(out); err != nil {
		t.Fatalf("export file missing: %v", err)
	}

	if err := store.Reset(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	if err := store.Import(out); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.LastLessonID != "macros-basic" {
		t.Fatalf("LastLessonID = %q", loaded.LastLessonID)
	}
}
