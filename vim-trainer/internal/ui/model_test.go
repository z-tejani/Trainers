package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"vimtrainer/internal/content"
	"vimtrainer/internal/progress"
)

func TestHomeEnterStartsCampaignScreen(t *testing.T) {
	store := progress.NewStore(t.TempDir())
	profile, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	app := NewApp(Config{
		Command: "home",
		Options: Options{},
		Catalog: content.NewCatalog(),
		Store:   store,
		Profile: profile,
	})
	app.homeCursor = 1

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := model.(*App)
	if got.route != routeCampaign {
		t.Fatalf("route = %s, want %s", got.route, routeCampaign)
	}
}

func TestChallengeCommandStartsLessonQueue(t *testing.T) {
	store := progress.NewStore(t.TempDir())
	profile, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	app := NewApp(Config{
		Command: "challenge",
		Options: Options{},
		Catalog: content.NewCatalog(),
		Store:   store,
		Profile: profile,
	})
	if app.route != routeLesson {
		t.Fatalf("route = %s, want %s", app.route, routeLesson)
	}
	if app.session == nil {
		t.Fatal("expected active challenge session")
	}
}

func TestHomeChallengeOpensChallengeBrowser(t *testing.T) {
	store := progress.NewStore(t.TempDir())
	profile, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	app := NewApp(Config{
		Command: "home",
		Options: Options{},
		Catalog: content.NewCatalog(),
		Store:   store,
		Profile: profile,
	})
	app.homeCursor = 5 // Challenge

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := model.(*App)
	if got.route != routeChallenge {
		t.Fatalf("route = %s, want %s", got.route, routeChallenge)
	}
}

func TestChallengeBrowserEnterStartsQueue(t *testing.T) {
	store := progress.NewStore(t.TempDir())
	profile, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	app := NewApp(Config{
		Command: "home",
		Options: Options{},
		Catalog: content.NewCatalog(),
		Store:   store,
		Profile: profile,
	})
	app.route = routeChallenge

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := model.(*App)
	if got.route != routeLesson {
		t.Fatalf("route = %s, want %s", got.route, routeLesson)
	}
	if got.session == nil {
		t.Fatal("expected active challenge lesson session")
	}
}

func TestVisibleRangeKeepsCursorInWindow(t *testing.T) {
	start, end := visibleRange(20, 0, 5)
	if start != 0 || end != 5 {
		t.Fatalf("top range = (%d,%d), want (0,5)", start, end)
	}
	start, end = visibleRange(20, 10, 5)
	if start > 10 || end <= 10 {
		t.Fatalf("middle cursor not visible: (%d,%d)", start, end)
	}
	start, end = visibleRange(20, 19, 5)
	if start != 15 || end != 20 {
		t.Fatalf("bottom range = (%d,%d), want (15,20)", start, end)
	}
}
