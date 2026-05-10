package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"vimtrainer/internal/content"
	"vimtrainer/internal/progress"
	"vimtrainer/internal/ui"
)

func main() {
	command, options, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "vim-trainer: %v\n", err)
		os.Exit(2)
	}

	store := progress.NewStore("")

	if command == "reset-progress" {
		if err := store.Reset(); err != nil {
			fmt.Fprintf(os.Stderr, "vim-trainer: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("vim-trainer progress reset")
		return
	}
	if command == "export-profile" {
		if err := store.Export(options.File); err != nil {
			fmt.Fprintf(os.Stderr, "vim-trainer: %v\n", err)
			os.Exit(1)
		}
		target := options.File
		if target == "" {
			target = "vimtrainer-profile.json"
		}
		fmt.Printf("vim-trainer profile exported to %s\n", target)
		return
	}
	if command == "import-profile" {
		if err := store.Import(options.File); err != nil {
			fmt.Fprintf(os.Stderr, "vim-trainer: %v\n", err)
			os.Exit(1)
		}
		source := options.File
		if source == "" {
			source = "vimtrainer-profile.json"
		}
		fmt.Printf("vim-trainer profile imported from %s\n", source)
		return
	}
	if command == "export-log" {
		if err := store.ExportSessionLog(options.File); err != nil {
			fmt.Fprintf(os.Stderr, "vim-trainer: %v\n", err)
			os.Exit(1)
		}
		target := options.File
		if target == "" {
			target = "vimtrainer-sessions.csv"
		}
		fmt.Printf("vim-trainer session log exported to %s\n", target)
		return
	}

	profile, err := store.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "vim-trainer: %v\n", err)
		os.Exit(1)
	}

	catalog := content.NewCatalog()
	model := ui.NewApp(ui.Config{
		Command: command,
		Options: options,
		Catalog: catalog,
		Store:   store,
		Profile: profile,
	})

	program := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "vim-trainer: %v\n", err)
		os.Exit(1)
	}
}

func parseArgs(args []string) (string, ui.Options, error) {
	command := "home"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "campaign", "neovim", "practice", "review", "challenge", "stats", "reset-progress", "export-profile", "import-profile", "export-log", "sandbox":
			command = args[0]
			args = args[1:]
		case "help":
			printUsage()
			os.Exit(0)
		default:
			return "", ui.Options{}, fmt.Errorf("unknown subcommand %q", args[0])
		}
	}

	fs := flag.NewFlagSet("vim-trainer", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var opts ui.Options
	fs.StringVar(&opts.LessonID, "lesson", "", "deep-link a lesson id")
	fs.Int64Var(&opts.Seed, "seed", 0, "seed for practice queue generation")
	fs.BoolVar(&opts.Debug, "debug", false, "show debug overlay")
	fs.StringVar(&opts.File, "file", "", "profile import/export path, or file to open in sandbox")
	if err := fs.Parse(args); err != nil {
		return "", ui.Options{}, err
	}

	return command, opts, nil
}

func printUsage() {
	fmt.Println(`Usage:
  vim-trainer [campaign|neovim|practice|review|challenge|sandbox|stats|reset-progress|export-profile|import-profile|export-log] [--lesson ID] [--seed N] [--debug] [--file PATH]

Commands:
  campaign        start the guided campaign
  neovim          start the Neovim-specific training mode
  practice        start a generated practice queue
  review          start a weak-spot review queue
  challenge       start a no-hints retention challenge queue
  sandbox         no-eval free-play; pair with --file to open your own file
  stats           open the stats screen
  reset-progress  clear saved local progress
  export-profile  write profile JSON to --file (default: vimtrainer-profile.json)
  import-profile  load profile JSON from --file (default: vimtrainer-profile.json)
  export-log      write session log CSV to --file (default: vimtrainer-sessions.csv)`)
}
