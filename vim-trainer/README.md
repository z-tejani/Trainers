# Vim Trainer

Standalone Bubble Tea app for learning Vim as a real editor workflow instead of a flat list of key facts.

## Run

```bash
cd /Users/zain/Documents/code/Trainers/vim-trainer
go run .
```

## CLI

```bash
go run . campaign
go run . neovim
go run . practice
go run . review
go run . challenge
go run . stats
go run . export-profile --file /tmp/vimtrainer-profile.json
go run . import-profile --file /tmp/vimtrainer-profile.json
go run . --lesson motions-word --debug
```

## Product Shape

- guided campaign with beginner-first modules
- practice mode generated from weak skills and due review
- review mode focused on recent mistakes
- challenge mode with no hints and retention-focused queues
- persistent local progress and mastery tracking
- home, stats, settings, and reset-progress flows
- profile import/export for local backups

## Curriculum

- onboarding: modes, `i`, `Esc`, command-line basics
- motions: `hjkl`, `w`, counts, `0`, `$`, `gg`, `G`
- operators: `x`, `dw`, `dd`, `cw`, `d$`, `yy`, `p`
- search and navigation: `/`, `n`, `N`, `*`, marks
- editing primitives: `o`, `O`, `u`, `Ctrl-r`
- text objects: `diw`, `caw`
- command-line and options: `:set number`, `:set relativenumber`, `:set hlsearch`, `:set ignorecase`, `:set smartcase`, `:set incsearch`, `:noh`, `:help`, `:source`, `:e`, `:Ex`, `:w`, `:q`, `:wq`
- external filter integration: `:%!sed 's/old/new/g'` and `:%!sed '/pattern/d'`
- windows and buffers: `:split`, `Ctrl-w w`, `:bn`
- repetition: `.`, `qa...q`, `@a`
- Vimscript essentials: `:let`, `:echo`, `:nnoremap`, `:unmap`
- Neovim mode: `:checkhealth`, `:terminal`, `:lua`
- visual mode: `v`, `V`, visual delete/yank/paste
- advanced text objects: `ci"`, `di(`, `dap`
- jumps and changes: `Ctrl-o`, `Ctrl-i`, `g;`, `g,`
- registers: `"a`, `:registers`
- quickfix: `:vimgrep`, `:cnext`, `:cprev`, `:copen`
- explorer depth: `:Explore`, `-`
- plugin/language workflows: `:Lazy`, `:Mason`, `:TSUpdate`, `:LspInfo`, `gd`
- Telescope literacy: `:Telescope find_files`, `:Telescope buffers`
- recovery and safety: `:wa`, `:qa!`, `:recover`
- performance/debugging: `:profile`, `:scriptnames`, `:verbose`

## Controls

- lesson screen uses Vim keys directly
- `?` toggles hints
- `F5` restarts the current lesson
- `F6` toggles explain-replay panel
- `F2` returns to the home screen
- `Ctrl+C` quits
