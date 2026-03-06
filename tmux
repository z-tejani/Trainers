#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/lib/trainer.sh"

trainer_require_cmd "tmux" "Error: tmux is not installed on this system."
trainer_init_workdir

TMUX_SOCKET="$TRAINER_WORKDIR/tmux.sock"
export TMUX_SOCKET
TMUX_SESSION="trainer-$RANDOM-$$"
export TMUX_SESSION

if tmux -S "$TMUX_SOCKET" new-session -d -s "$TMUX_SESSION" -n shell >/dev/null 2>&1 \
  && tmux -S "$TMUX_SOCKET" ls >/dev/null 2>&1; then
  TMUX_PREFIX='tmux -S "$TMUX_SOCKET"'
  TMUX_MODE_DESC="isolated socket: $TMUX_SOCKET"
elif tmux new-session -d -s "$TMUX_SESSION" -n shell >/dev/null 2>&1 \
  && tmux ls >/dev/null 2>&1; then
  TMUX_PREFIX='tmux'
  TMUX_MODE_DESC="default tmux server (socket isolation unavailable here)"
else
  echo "Error: tmux is installed but cannot start a tmux server in this environment."
  echo "This usually means tmux socket creation is blocked by sandbox permissions."
  exit 1
fi

trainer_section "tmux trainer"
echo "Use this command prefix:"
echo "  $TMUX_PREFIX"
echo "Mode: $TMUX_MODE_DESC"
echo "Sample session already exists: $TMUX_SESSION"
trainer_pause

trainer_expect_contains \
  "Lesson 1: list sessions" \
  "show available sessions" \
  "Run a command that lists tmux sessions using: $TMUX_PREFIX" \
  "$TMUX_SESSION" \
  "Use: $TMUX_PREFIX ls"

trainer_expect_contains \
  "Lesson 2: create a window and list windows" \
  "create a window named logs in the trainer session, then list windows" \
  "Run a command that creates window 'logs' and then shows windows for session $TMUX_SESSION" \
  "logs" \
  "Use $TMUX_PREFIX new-window ... ; list-windows ..."

trainer_expect_contains \
  "Lesson 3: split panes" \
  "create a second pane in window logs and list pane indexes" \
  "Run a command that splits $TMUX_SESSION:logs and then lists panes in that window" \
  "1:" \
  "Use $TMUX_PREFIX split-window ... ; list-panes ..."

trainer_expect_contains \
  "Lesson 4: send keys" \
  "send a command into the first pane and capture output" \
  "Run a command that sends 'echo tmux-ok' to $TMUX_SESSION:logs.0 and captures pane text" \
  "tmux-ok" \
  "Use $TMUX_PREFIX send-keys ... ; capture-pane -p ..."

trainer_section "Reference answers"
cat <<EOF
1) $TMUX_PREFIX ls
2) $TMUX_PREFIX new-window -t "$TMUX_SESSION" -n logs \; list-windows -t "$TMUX_SESSION"
3) $TMUX_PREFIX split-window -t "$TMUX_SESSION:logs" \; list-panes -t "$TMUX_SESSION:logs" -F '#{pane_index}:#{pane_active}'
4) $TMUX_PREFIX send-keys -t "$TMUX_SESSION:logs.0" 'echo tmux-ok' Enter \; capture-pane -p -t "$TMUX_SESSION:logs.0"
EOF

trainer_finish \
  "Nice work. You covered tmux essentials: sessions, windows, panes, and send-keys." \
  "Re-run the trainer to improve your score."
