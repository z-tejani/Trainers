# CLI Trainers

This repo contains executable trainers for CLI tools.
Each lesson is interactive and supports `teach`, `hint`, and `skip` commands.

## Available Trainers

- `./jq` - interactive trainer for learning `jq`
- `./bat` - interactive trainer for learning `bat` (`batcat` fallback)
- `./fd` - interactive trainer for learning `fd` (`fdfind` fallback)
- `./zsh` - interactive trainer for learning advanced `zsh` usage
- `./kubectl` - interactive trainer for kubectl client-side workflows
- `./make` - interactive trainer for `make` targets and variables
- `./xargs` - interactive trainer for `xargs` batching and safety flags
- `./python-advanced` - advanced Python topics (lambda, decorators, closures, generators, caching)
- `./ripgrep` - interactive trainer for learning `ripgrep` (`rg`)
- `./tmux` - interactive trainer for learning `tmux`
- `./vim` - interactive trainer for learning `vim` (Ex-mode exercises)
- `./sed` - interactive trainer for learning `sed`
- `./awk` - interactive trainer for learning `awk`
- `./find` - interactive trainer for learning `find`
- `./grep` - interactive trainer for learning `grep`
- `./fzf` - interactive trainer for learning `fzf`
- `./bash-redirection` - interactive trainer for learning Bash redirection
- `./golang` - interactive trainer for Go keywords and programming workflow
- `./cs50` - comprehensive trainer for the full CS50 concept arc (computational thinking, C, algorithms, memory, Python, SQL, web, security, scaling)

## Common Launch Options

Every trainer now supports the same launch modes:

- `--guided` or `--training` - interactive coaching mode with `teach`, `hint`, and `skip`
- `--manual` - walkthrough mode with no prompts or scoring
- `--test` - one-attempt assessment mode with no hints
- `--mode guided|manual|test` - explicit mode selection
- `--help` - show usage for that trainer

## Run

```bash
./jq --guided
./golang --manual
./grep --test
./fzf --mode guided
```

## CS50 Companion Guide

Use [`CS50_MASTER_GUIDE.md`](CS50_MASTER_GUIDE.md) for a complete module-by-module concept map, drills, and capstone projects.

## Create a New Trainer

Use the scaffold generator:

```bash
./new-trainer sed
```

This creates `sed` with:
- shared helper import (`lib/trainer.sh`)
- sample temp file setup
- one starter interactive lesson block
- scoring and finish summary hooks
- common launch modes inherited automatically from `lib/trainer.sh`

Then edit the generated file to add your tool-specific lessons and answers.
