# Stretch Goals

Backlog for the trainer, ordered by impact × reach. Tier 1 (replace, visual block, real `:s`, `:g`, regex search, more text objects, `gn`/`gN`, surround) already shipped in the engine. P0 and most of P1 are now landed; remaining P1 work is called out below.

Bands:

- **P0** — ship-changing. Without these the product feels like a demo. ✅ Shipped.
- **P1** — high-value, clear wins, modest scope. ✅ Mostly shipped; deferred items noted.
- **P2** — real depth. Power-user curriculum and second-order features.
- **P3** — long tail. Worth covering eventually; won't make or break the product.
- **Foundation** — code/structure work that gates other bands.

---

## P0 — ship-changing

These are the difference between "Vim trainer demo" and "tool I keep installed."

- **Sandbox / free-play mode** — every interaction today is goal-checked. Add a route that boots the engine with no eval and just logs keystrokes. Unblocks the next item and gives users a place to experiment without failure.
- **"Open my own file"** — `--file path.txt` straight into a sandbox lesson. The engine already takes a `Scenario` (`internal/engine/engine.go:48`); the wiring is small and the motivation jump is enormous (people will use this on real code).
- **Solve-in-N-keys / time targets** — `Lesson` (`internal/content/content.go:18`) has no `OptimalKeys` or `TimeTarget`. Turning the loop from "did I solve it" into "am I getting *more efficient*" is the single best mechanic for moving learners past plateau.
- **Token-by-token replay on failure (auto-shown)** — F6 already exists (`internal/ui/model.go` ~723) but is hidden. Surface it automatically on every failed check, showing the last 5 keys + how Vim parsed them. This is the "explain why my keys did the wrong thing" loop and it's irreplaceable.
- **A real refactoring lesson + macro-over-quickfix** — `*` → `cgn` → `.` → `n.`, then `qa…q` → `:vimgrep` → `:cdo norm @a`. These are the moments learners *feel* powerful Vim. Nothing in today's curriculum builds toward either.
- **Substitute confirmation flow (`:%s/foo/bar/gc`)** — the single command real Vim users run dozens of times a day. Tier 1 wired `:s` parsing; the `c` flag with an interactive prompt is the missing piece.
- **Lift lessons into data files (YAML / JSON)** — `modules.go` + `extended_modules.go` are 1,100 lines of literal Go structs. Only `Check` closures must stay code; everything else (Goal, Hints, Initial scenario, FocusTokens) becomes data. Without this, every curriculum addition below requires a Go recompile and a new release.

## P1 — high-value, modest scope ✅ mostly shipped

Shipped in this pass:

- ✅ **Real SRS scheduler** — `SkillProgress` now carries `IntervalDays`, `EaseFactor`, `DueAt`, `Lapses`; `RecordLesson` advances/resets via SM-2 logic; `RecommendedLessons` ranks by overdueness + lapse count + mistakes.
- ✅ **Tiered hints** — `lessonSession.WrongAttempts` drives `HintsUnlocked`; the lesson view reveals one more hint per 3 misfires and shows the unlock countdown. Cap respected.
- ✅ **F3 keymap cheatsheet overlay** — scoped one-page reference visible at the bottom of any lesson; F3 toggles. Bound to F3 so the F1/`?` hint toggle stays separate.
- ✅ **Browse / search the catalog** — new `routeBrowse`, "/" enters filter prompt, "x" clears, mastery glyphs (`✓ ~ !`) on each row.
- ✅ **Diagnostic placement quiz** — new `routeDiagnostic` with 7 yes/no questions; "yes" answers mark linked lessons completed and update achievements + module unlocks.
- ✅ **Insert-mode tricks** — engine: `<C-o>`, `<C-w>`, `<C-u>` in Insert mode. Two new lessons (`insert-ctrlw-ctrlu`, `insert-ctrlo`).
- ✅ **Indent / format curriculum** — engine: `>>`, `<<`, visual `>`/`<` (2-space shift unit). Two new lessons (`indent-line`, `indent-visual`). Still missing: `=ip` (auto-reindent paragraph) and `gq` (text reflow) — non-trivial; deferred.
- ✅ **Quickfix deep dive** — engine: `:cnewer`, `:colder`, `:chistory`, `:cdo`, `:cfdo`, `:normal`. The macro-over-quickfix lesson from P0 now uses real `:cdo norm @a` semantics.
- ✅ **Ex ranges in lessons** — new `exranges-delete` lesson driven by the Tier 1 range parser.
- ✅ **Achievements / milestones** — 10 default achievements (First Steps → Apprentice → Journeyman → Adept; Module Master → All Modules; Speed Demon / Perfectionist; Comeback Kid; Well Rested). Stats screen lists locked + unlocked; summary screen toasts new unlocks.
- ✅ **Detailed practice log export** — new `SessionLog` in profile; new CLI `vim-trainer export-log --file path.csv` writes time, lesson_id, mode, success, duration_ms, keystrokes, mistakes.

Carried-over P1 items, all shipped in this pass:

- ✅ **Tab pages** — `:tabnew`, `:tabnext`/`gt`, `:tabprev`/`gT`, `:tabfirst`, `:tablast`, `:tabclose`, `:tabonly`, `:tabs`. Engine stores inactive tabs as `tabSnapshot{windows, activeWindow}`; the active tab's state stays in the live fields and is swapped on switch. New `tabs` module + 2 lessons.
- ✅ **Window movement** — `<C-w>H/J/K/L`, `<C-w>=`, `<C-w>c`, `<C-w>o`, `<C-w>T`, `:close`, `:only`, `:resize`/`:res`/`:vertical` (resizes accepted as no-op since sizes aren't modeled). New `windowsmove` module + 2 lessons.
- ✅ **`=ip` reindent and `==`** — `=` is now an operator that takes either a duplicate (`==`) or a text-object (`=ip`/`=ap`). Reindent aligns target lines to the first non-empty line's indent.
- ✅ **`gq` reflow with `:set textwidth=N`** — `gq` is a `g` prefix; `gqq` reflows current line, `gqip`/`gqap` reflow paragraph text-objects. `:set textwidth=20` (or `tw=20`) now sets the wrap target. New `format` module + 3 lessons.
- ✅ **`:s/.../gc` match highlighting** — engine now exposes `ConfirmContext`, `ConfirmMatchStart`, `ConfirmMatchEnd`. The lesson view renders the source line with the candidate match in inverse video above the y/n/a/q prompt, so the learner can see exactly which occurrence is being asked about.

## P2 — real depth ✅ shipped

All nine categories landed in this pass. 91 lessons across 38 modules.

- ✅ **Macros (advanced)** — `@@` (replay last), `qA` (append to existing), counted macros (`100@a`), `:[range]normal keys`, `:%normal @a`, `:g/pat/normal …`. Bug fix bundled in: register-name keystrokes after `q` no longer leak into the recorded macro.
- ✅ **Marks (advanced)** — global `A`–`Z` cross-buffer, auto marks `'.`, `'^`, `'[`, `']`, `'<`, `'>`, `:marks`, `:delmarks` (range + bang).
- ✅ **Registers (advanced)** — black-hole `"_`, system clipboard `"+`/`"*` (treated as letter regs), numbered delete ring `"1`–`"9`, special read-only registers `"-`, `"/`, `":`, `".`, `"%`, `"#`, `"=`. Plus `<C-r><reg>` paste in Insert mode.
- ✅ **Insert-mode completion** — `<C-n>` / `<C-p>` complete from buffer words.
- ✅ **Shell integration** — `:!cmd`, `:!!`, `:r !cmd`, `:r file`, `:make` (synthesizes a quickfix entry), `:compiler`, `:set makeprg=…`, `:set errorformat=…`. Output is faked deterministically — no real shell exec, for safety + reproducibility.
- ✅ **Common-config walkthrough** — `tabstop`, `shiftwidth`, `expandtab`, `scrolloff`, `undofile`, `clipboard`, `splitbelow`, `splitright`, `termguicolors`, `mouse`, `spell`, `spelllang`, `foldmethod`, `cursorline`, `list`, `listchars`, `wildmenu`, `wildmode`, `completeopt`, `lazyredraw`, `updatetime`, `timeoutlen`, `backup`, `swapfile`, `autoread`, `colorscheme` — all parsed by `:set`.
- ✅ **Deep ex commands** — `:sort` (with `u`/`n`), `:retab`, `:args`/`:next`/`:prev`/`:argdo`, `:bufdo`, `:execute`, `<C-^>` (alternate buffer via `ctrl+^` or `ctrl+6`). Plus `A` (append at end of line) and `I` (insert at first non-blank) since `:%normal A!` style commands need them.
- ✅ **Deep mappings** — `:inoremap`/`:vnoremap`/`:cnoremap`/`:tnoremap`/`:onoremap` with per-mode tables (storage; replay would require a much bigger dispatcher refactor). Plus `:command!` with `-nargs`/`-range`/`-bang` flag tolerance.
- ✅ **Folds (manual)** — `zf{motion}`, `zo`, `zc`, `za`, `zR`, `zM`, `zj`, `zk`, `zd`, `zE`. State exposes `FoldCount` / `FoldClosedCount` for lessons to verify.
- ✅ **Spell, completion, diff** — `:set spell`/`spelllang`, `]s`/`[s`/`z=`/`zg`/`zw` echo placeholders (no dictionary shipped), `:diffthis`/`:diffoff`/`:diffupdate`, `]c`/`[c` diff hunk navigation stubs.

Curriculum additions: 30 new lessons across 10 new modules — `macrosadv`, `marksadv`, `registersadv`, `shellintegration`, `configwalkthrough`, `exdeep`, `mappingsdeep`, `foldsdeep`, `spelldiff`, plus expansions to existing ones.

CheckSpec grammar grew to cover the new state — `mark_set`, `textwidth_is`, `option_int_is`, `option_string_is`, `bool_option_is`, `fold_count_is`, `fold_closed_count_is`. Still no `RegisterCheck` closures needed.

## P2 reference (kept for historical context)

Power-user curriculum and second-order features. Most users won't *need* these; learners who reach them will love the trainer for going this far.

### Macros (advanced)

Trainer covers `qa…q`, `@a`, `.` only.

- **Replay last macro**: `@@`.
- **Replay across a range**: `:'<,'>normal @a`, `:%normal @a`, `:g/pattern/normal @a`.
- **Append to a macro**: capital register `qA`.
- **Edit a macro inline**: `"ap`, edit, `0"ay$`.
- **Recursive macros**: `qaj@aq` then `@a` (pairs with `:set nowrapscan`).
- **Counted macros**: `100@a`, `5@@`.
- **Persisted macros**: `let @a = '...'` in `.vimrc`.

### Marks (advanced)

Trainer covers `ma` / `'a` / `g;` / `g,`.

- **Global / file marks**: `mA`–`mZ` jump across buffers.
- **Auto marks**: `'.` (last change), `'^` (last insert), `''` (jump-back), `'[` / `']` (last yanked / changed range), `'<` / `'>` (last visual selection), `'"` (last cursor when leaving file).
- **Numbered marks**: `'0`–`'9` from `viminfo` / `shada`.
- **Listing**: `:marks`, `:delmarks a-c`, `:delmarks!`.
- **Marks as range targets**: `:'a,'bd`, `d'a`.

### Registers (advanced)

Trainer covers `"` (unnamed), `a`–`z`, `0` (yank).

- **Black hole `"_`** — delete without clobbering yank. Probably the single most-loved register among power users.
- **System clipboard**: `"+` (clipboard), `"*` (selection). Platform-dependent.
- **Numbered delete ring**: `"1`–`"9`.
- **Special registers**: `"-` (small delete), `"/` (last search), `":` (last cmd), `".` (last insert), `"%` (file), `"#` (alt file), `"=` (expression, e.g. `"=strftime("%F")<CR>p`).
- **Insert-mode paste**: `Ctrl-R a`, `Ctrl-R Ctrl-W`.

### Shell integration

- `:!cmd`, `:!!`, `:r !date`.
- Range filters beyond canned `:%!sed`: `:%!jq .`, `:'<,'>!awk '...'`.
- `:terminal`, `Ctrl-\ Ctrl-N`, `Ctrl-Z` / `:shell` / `fg`.
- `:make` workflow: `:set makeprg`, `errorformat`, `:compiler go|eslint|...`.

### Common configuration walkthrough

A guided "starter `.vimrc` / `init.lua`" lesson chain. Most users never write a real config; this is the bridge.

- **Leader**: `let mapleader = " "`, `vim.g.mapleader = " "`.
- **Indentation**: `tabstop`, `softtabstop`, `shiftwidth`, `expandtab`, `smartindent`.
- **Display**: `number`, `relativenumber`, `cursorline`, `signcolumn`, `scrolloff`, `wrap` / `linebreak`, `colorcolumn`.
- **Whitespace**: `list`, `listchars`, `fillchars`.
- **Search**: `incsearch`, `inccommand=split` (Neovim).
- **Performance**: `lazyredraw`, `updatetime`, `timeoutlen`.
- **Files / undo**: `undofile`, `undodir`, `autoread`.
- **Clipboard**: `clipboard=unnamedplus`.
- **Completion**: `completeopt`, `pumheight`, `wildmenu`, `wildmode`.
- **Splits**: `splitbelow`, `splitright`.
- **Mouse, termguicolors, colorscheme, spell, foldmethod**.

### Ex commands (deep)

- **`:normal`** — run normal-mode keys from the cmdline, e.g. `:%normal Iprefix `. Combines with `:g` and ranges.
- **`:execute`** — build commands dynamically.
- **`:read` / `:write` to range / pipe** — `:r !date`, `:w !sudo tee %`, `:'<,'>w >> file`.
- **`:try` / `:catch` / `:finally`**.
- **`:if` / `:while` / `:for`**.
- **`:sort`** (and `u`, `n`, `/pattern/`), **`:retab`**.
- **`:cdo`, `:cfdo`, `:ldo`, `:lfdo`** — already in P1 quickfix.
- **Argument list**: `:args`, `:argdo`, `:next`, `:prev`, `:argadd`.
- **Buffer management**: `:ls`, `:b <name|N>`, `:bd` / `:bw`, `:bufdo`.
- **Alternate file**: `Ctrl-^`, `''`.

### Mappings (deep)

Trainer covers `:nnoremap` and `:unmap`.

- **Mode-specific maps**: `inoremap`, `vnoremap`, `xnoremap`, `cnoremap`, `tnoremap`, `omap`.
- **Buffer-local**: `nnoremap <buffer> ...`.
- **Flags**: `<silent>`, `<expr>`, `<nowait>`, `<unique>`.
- **Special keys**: `<CR>`, `<Esc>`, `<leader>`, `<localleader>`, `<Plug>`, `<SID>`.
- **`:command!`** with `-nargs`, `-range`, `-bang`.

### Folds (deep)

- **Methods**: `manual`, `indent`, `marker` (`{{{` / `}}}`), `syntax`, `expr`, `diff`.
- **Operations**: `zf<motion>`, `zd`, `zE`, `zo`/`zO`, `zc`/`zC`, `za`/`zA`, `zR` / `zM`, `zr` / `zm`, `zj` / `zk`, `zx`.
- **Persistence**: `:mkview` / `:loadview`.

### Spell, completion, diff

- **Spell**: `]s` / `[s`, `z=`, `zg`, `zw`, `zuw`.
- **Completion menu nav**: `Ctrl-N`/`Ctrl-P`, `Ctrl-Y` accept, `Ctrl-E` cancel.
- **Diff**: `vimdiff`, `:diffthis`, `:diffupdate`, `]c` / `[c`, `do` / `dp`.

## P3 — long tail ✅ shipped

All eight categories landed. **111 lessons across 44 modules.**

- ✅ **Misc power moves** — `<C-a>` / `<C-x>` (with `g <C-a>` deferred), `J` / `gJ`, `~` / `g~iw` / `gUiw` / `guiw`, `g??` rot13, `gi` resume insert (via `'^`). New `powermoves` module + 7 lessons.
- ✅ **Autocommands** — `:autocmd Event Pattern Cmd`, `:autocmd!`, `:augroup`/`augroup END`, `:doautocmd` (storage + count; no real event firing). New `autocmds` module + 2 lessons.
- ✅ **Sessions / views / shada** — `:mksession`, `:mkview`, `:loadview`, `:wshada`, `:rshada` (storage). New `sessions` module + 2 lessons.
- ✅ **UI surfaces** — `:set statusline=`, `tabline=`, `winbar=`, `conceallevel=`, `:sign place/list/define`, plus `popupEntry` storage. New `uisurfaces` module + 4 lessons.
- ✅ **Help + cmdline history** — `q:` / `q/` / `q?` open synthetic `history://cmdline` and `history://search` buffers populated from `commandHistory`; `:helpgrep` populates quickfix with synthetic help-tag entries; `<C-]>` and `<C-T>` reuse `goToDefinition` and `jumpOlder` for tag navigation. New `helpcmdline` module + 3 lessons.
- ✅ **Plugin ecosystem (deep)** — `:TSPlaygroundToggle`, `:TSBufToggle`, `:TSConfigInfo`, `:TSModuleInfo`; `:DapContinue`, `:DapStepOver`, `:DapStepInto`, `:DapStepOut`, `:DapToggleBreakpoint`, `:DapToggleRepl`; `:Snippets`, `:LuaSnipUnlinkCurrent`, `:LuaSnipListAvailable`. All as `(simulated)` echoes. New `ecosystem-deep` module + 3 lessons.
- ✅ **Settings expansion + accessibility** — `Settings` struct gains `Theme` (default / high-contrast / monochrome cycle), `ColorblindSafe`, `ReducedMotion`, `LargerCursor`, `KeyRepeatDelay`. Settings screen now shows six toggles with cycle-on-Enter for theme. Flags are advisory; future renderer passes will honor them.
- ✅ **`gd` fix** — broader heuristic was matching the cursor's own position via the shell-style `name()` pattern; now skips the current cursor location so `gd` and `<C-]>` actually move.

CheckSpec grammar gained `autocmd_count_is`, `session_count_is`, `view_count_is`, `sign_count_is`, `statusline_is`, `conceal_level_is`. Still zero `RegisterCheck` closures needed for the migrated catalog.

Smoke-tested: `<C-a>` / `<C-x>`, `J`, `~`, `gUiw`, `g??`, `:autocmd`, `:mksession`, `:set statusline=`, `:set conceallevel=`, `:sign place`, `q:`, `:helpgrep`, `<C-]>`, plus the existing `qa` / `@a` flow — all 15 tests pass.

## P3 reference (kept for historical context)

Cover for completeness once P0–P2 are landed.

### Popups / floating windows / UI surfaces

- Vim popups: `popup_create`, `popup_close`, `popup_menu`, `popup_notification`.
- Neovim floating windows: `nvim_open_win`, `nvim_win_set_config`. Underpins Telescope, lspsaga, noice, fzf-lua.
- `cmdheight=0`, noice-style cmdline.
- Statusline / tabline / winbar (`:set statusline=…`).
- Sign column: `:sign define`, `:sign place`.
- `conceallevel`, `concealcursor`.

### Sessions, views, shada

- `:mksession`, `:source session.vim`, `sessionoptions`.
- `:mkview`, `:loadview`.
- `shada` / `viminfo`, `:rshada`, `:wshada`.

### Help / command-line history

- `:help <topic><C-d>`, `:helpgrep`, tag jump `Ctrl-]` / `Ctrl-T`.
- Command-line window: `q:`, `q/`, `q?`.
- Cmdline history nav: `<C-p>` / `<C-n>` at `:`.

### Plugin ecosystem (Neovim)

Trainer name-drops Lazy / Mason / Telescope. Add real lessons:

- **Lazy**: `:Lazy`, update / clean / profile.
- **LSP**: `vim.lsp.buf.hover/definition/references/rename/code_action/format`; default `K`, `gd`, `gr`, `<C-]>`.
- **Treesitter**: `:TSInstall`, treesitter-textobjects (`@function.outer`, `@class.inner`).
- **DAP**: `require'dap'.continue()`, breakpoint workflow.
- **Telescope deep**: `find_files`, `live_grep`, `grep_string`, `oldfiles`, `git_files`, `lsp_references`, `<C-q>` to send to quickfix.
- **Snippets**: LuaSnip / vsnip basics.

### Autocommands & events

Engine has no event system; building one is a meaningful lift.

- Common events: `BufRead`, `BufWritePre`/`Post`, `FileType`, `InsertEnter`/`Leave`, `TextYankPost`, `VimEnter`/`Leave`, `CursorHold`.
- `augroup`, `autocmd!`, `:doautocmd`.
- Recipes: trim trailing whitespace on save; highlight yanked region; auto-format on save; restore last cursor position.

### Miscellaneous power moves

- **Increment / decrement**: `Ctrl-A` / `Ctrl-X`, `g Ctrl-A` (sequential).
- **Join**: `J`, `gJ`, `:join`.
- **Case**: `~`, `g~iw`, `gUiw`, `guiw`.
- **Format / wrap**: `gqip`, `gw`, `gqq`.
- **ROT13**: `g??`.
- **Resume insert**: `gi`.
- **`:g/pattern/normal @a`** — macro per matching line in one expression.

### Settings expansion (cosmetics / a11y)

Currently only `ShowHints` + `Debug` (`internal/ui/model.go` ~889–905).

- Theme, key-repeat delay, cursor size, line spacing.
- High-contrast / colorblind mode.
- Reduced motion.
- Screen-reader hints (no a11y labels today; `internal/ui/model.go` ~1005–1013 is hard-coded ANSI).

---

## Foundation — code / structure work that gates other bands ✅ shipped

All seven items landed in this pass:

- ✅ **Centralized config (`internal/config/config.go`)** — queue limits, mastery thresholds, SRS bumps, list caps, indent unit, achievement thresholds, schema version. Q3=a (no env / file overrides for now).
- ✅ **`Settings.Debug` decoupled from engine internals** — engine exposes `Editor.DebugSummary()`; the UI calls that instead of poking individual pending-state fields.
- ✅ **Engine decoupled from lesson filenames** — `defaultBufferLines` no longer fabricates content for specific names (`README.md`, `notes.txt`, `config.lua`); new buffers come up empty, like real Vim. Explorer Enter and `:terminal` content also genericized.
- ✅ **Broadened `gd` heuristic** — covers Go, Rust, Python, JS / TS, Lua, Vimscript, shell, plus generic assignment patterns. Picks the leftmost match per line so combined keywords (`pub fn name`) resolve correctly.
- ✅ **Lua scope-down (Q2=b)** — `:lua` keeps its existing surface but errors carry a `luaSubsetError` preamble announcing the trainer simulates a scoped subset. The `neovim-lua-basics` lesson sets the same expectation in its explanation.
- ✅ **Lesson content hash + stale flag (Q1=a)** — `Lesson.ContentHash()` over learner-facing fields; profile records the hash at completion time; stats screen lists lessons whose content has been edited since with a clear "(stats remain intact)" disclaimer.
- ✅ **Bulk JSON migration** — all 54 lessons live in `internal/content/lessons/01-base.json` and `02-extended.json`. `lessonSet()` now just delegates to the JSON loader. `CheckSpec` grammar covers every previously-Go-coded check (added: `cursor_row_is`, `mode_is`, `last_search_is`, `last_echo_is`, `mapping_absent`/`mapping_exists`, `active_buffer_name_is`/`*_has_prefix`, `window_count_is`, `active_window_is`, `explorer_open_is`/`explorer_path_is`, `confirm_active_is`, `profile_active_is`). No `RegisterCheck` closures needed in the migrated catalog.

Schema version is now 2 and stamped automatically on first save. Profile loader will read it for any future migration.

What this unlocks for future work:

- New curriculum lessons can ship as data files. No Go recompile needed unless the lesson's check requires a CheckSpec type that doesn't exist yet — and the grammar is now broad enough that most lessons won't.
- The `internal/config` package is the one place to grep when tuning anything; future env-var or file-loaded overrides become a small additive change.
- Lesson edits surface as "stale" warnings on the stats screen instead of silently nullifying the learner's progress. Authors don't need to bump anything manually.
