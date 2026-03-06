#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/lib/trainer.sh"

if command -v bat >/dev/null 2>&1; then
  BAT_CMD="bat"
elif command -v batcat >/dev/null 2>&1; then
  BAT_CMD="batcat"
else
  echo "Error: bat is not installed on this system."
  echo "On some systems it is available as batcat."
  exit 1
fi

trainer_init_workdir

cat >"$TRAINER_WORKDIR/colors.txt" <<'EOF'
red
blue
green
yellow
EOF

cat >"$TRAINER_WORKDIR/app.py" <<'EOF'
def add(a, b):
    return a + b

print(add(2, 3))
EOF

trainer_section "bat trainer"
echo "You will run bat commands and verify output."
echo "Sample files are in: $TRAINER_WORKDIR"
echo "Detected bat command: $BAT_CMD"
trainer_pause

trainer_expect_contains \
  "Lesson 1: plain output" \
  "print file content without decorations" \
  "Run a command that prints $TRAINER_WORKDIR/colors.txt with no headers or grid lines" \
  "green" \
  "Use --paging=never --style=plain."

trainer_expect_contains \
  "Lesson 2: line numbers" \
  "show file with line numbers" \
  "Run a command that prints colors.txt with line numbers enabled" \
  "2 blue" \
  "Use -n with --paging=never."

trainer_expect_contains \
  "Lesson 3: line range" \
  "print only lines 2 through 3" \
  "Run a command that prints only lines 2-3 of colors.txt" \
  "blue" \
  "Use --line-range 2:3."

trainer_expect_contains \
  "Lesson 4: language hint" \
  "render a txt file as Python syntax" \
  "Run a command that shows $TRAINER_WORKDIR/app.py with --language=python" \
  "def add(a, b):" \
  "Use --language python and --paging=never."

trainer_section "Reference answers"
cat <<EOF
1) $BAT_CMD --paging=never --style=plain "$TRAINER_WORKDIR/colors.txt"
2) $BAT_CMD --paging=never -n "$TRAINER_WORKDIR/colors.txt"
3) $BAT_CMD --paging=never --style=plain --line-range 2:3 "$TRAINER_WORKDIR/colors.txt"
4) $BAT_CMD --paging=never --language=python "$TRAINER_WORKDIR/app.py"
EOF

trainer_finish \
  "Nice work. You covered practical bat viewing options." \
  "Re-run the trainer to improve your score."
