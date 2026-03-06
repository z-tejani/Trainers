# CLI Trainers

This repo contains executable trainers for CLI tools.

## Available Trainers

- `trainers/jq` - interactive trainer for learning `jq`
- `trainers/bat` - interactive trainer for learning `bat` (`batcat` fallback)
- `trainers/fd` - interactive trainer for learning `fd` (`fdfind` fallback)
- `trainers/zsh` - interactive trainer for learning advanced `zsh` usage
- `trainers/kubectl` - interactive trainer for kubectl client-side workflows
- `trainers/make` - interactive trainer for `make` targets and variables
- `trainers/xargs` - interactive trainer for `xargs` batching and safety flags
- `trainers/python-advanced` - advanced Python topics (lambda, decorators, closures, generators, caching)
- `trainers/ripgrep` - interactive trainer for learning `ripgrep` (`rg`)
- `trainers/tmux` - interactive trainer for learning `tmux`
- `trainers/vim` - interactive trainer for learning `vim` (Ex-mode exercises)
- `trainers/sed` - interactive trainer for learning `sed`
- `trainers/awk` - interactive trainer for learning `awk`
- `trainers/find` - interactive trainer for learning `find`
- `trainers/grep` - interactive trainer for learning `grep`
- `trainers/fzf` - interactive trainer for learning `fzf`
- `trainers/bash-redirection` - interactive trainer for learning Bash redirection
- `trainers/golang` - interactive trainer for Go keywords and programming workflow

## Run

```bash
./trainers/jq
./trainers/bat
./trainers/fd
./trainers/zsh
./trainers/kubectl
./trainers/make
./trainers/xargs
./trainers/python-advanced
./trainers/ripgrep
./trainers/tmux
./trainers/vim
./trainers/sed
./trainers/awk
./trainers/find
./trainers/grep
./trainers/fzf
./trainers/bash-redirection
./trainers/golang
```

## Create a New Trainer

Use the scaffold generator:

```bash
./trainers/new-trainer sed
```

This creates `trainers/sed` with:
- shared helper import (`trainers/lib/trainer.sh`)
- sample temp file setup
- one starter lesson block
- scoring and finish summary hooks

Then edit the generated file to add your tool-specific lessons and answers.
