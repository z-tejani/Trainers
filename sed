#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/lib/trainer.sh"

trainer_require_cmd "sed" "Error: sed is not installed on this system."
trainer_init_workdir

cat >"$TRAINER_WORKDIR/tasks.txt" <<'EOF'
TODO: update docs
TODO: ship release
DONE: write tests
EOF

cat >"$TRAINER_WORKDIR/greek.txt" <<'EOF'
alpha
beta
gamma
delta
EOF

cat >"$TRAINER_WORKDIR/team.csv" <<'EOF'
name,role
Ada,admin
Lin,user
Sam,admin
EOF

cat >"$TRAINER_WORKDIR/spaced.txt" <<'EOF'
red

blue

green
EOF

trainer_section "sed trainer"
echo "You will run sed commands and verify output."
echo "Sample files are in: $TRAINER_WORKDIR"
trainer_pause

trainer_expect_contains \
  "Lesson 1: substitution" \
  "replace TODO with DONE" \
  "Run a command that replaces TODO with DONE in $TRAINER_WORKDIR/tasks.txt" \
  "DONE: update docs" \
  "Try: sed 's/TODO/DONE/' file"

trainer_expect_contains \
  "Lesson 2: print one line" \
  "print only line 3 from greek.txt" \
  "Run a command that prints only line 3 from $TRAINER_WORKDIR/greek.txt" \
  "gamma" \
  "Use -n with an address like '3p'."

trainer_expect_contains \
  "Lesson 3: delete blank lines" \
  "remove empty lines from spaced.txt" \
  "Run a command that outputs $TRAINER_WORKDIR/spaced.txt without blank lines" \
  "blue" \
  "Use a delete command on empty-line regex."

trainer_expect_contains \
  "Lesson 4: capture groups" \
  "reformat name,role to name (role)" \
  "Run a command that prints rows as 'Ada (admin)' from $TRAINER_WORKDIR/team.csv (skip header)" \
  "Ada (admin)" \
  "Use sed -n -E with captures like 's/.../.../p'."

trainer_section "Reference answers"
cat <<EOF
1) sed 's/TODO/DONE/' "$TRAINER_WORKDIR/tasks.txt"
2) sed -n '3p' "$TRAINER_WORKDIR/greek.txt"
3) sed '/^$/d' "$TRAINER_WORKDIR/spaced.txt"
4) sed -n -E '2,\$s/^([^,]+),([^,]+)$/\1 (\2)/p' "$TRAINER_WORKDIR/team.csv"
EOF

trainer_finish \
  "Nice work. You covered core sed operations: s///, -n, addresses, and deletes." \
  "Re-run the trainer to improve your score."
