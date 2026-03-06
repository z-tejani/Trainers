#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/lib/trainer.sh"

trainer_require_cmd "vim" "Error: vim is not installed on this system."
trainer_init_workdir

cat >"$TRAINER_WORKDIR/notes.txt" <<'EOF'
TODO write docs
TODO add tests
done ship build
EOF

cat >"$TRAINER_WORKDIR/list.txt" <<'EOF'
alpha
beta
gamma
EOF

cat >"$TRAINER_WORKDIR/blanky.txt" <<'EOF'
red

blue

green
EOF

trainer_section "vim trainer"
echo "Use non-interactive Ex mode so exercises can be checked automatically."
echo "Sample files are in: $TRAINER_WORKDIR"
trainer_pause

trainer_expect_contains \
  "Lesson 1: substitute in file" \
  "replace TODO with DONE using vim commands" \
  "Run a command that updates $TRAINER_WORKDIR/notes.txt with TODO->DONE and then prints file contents" \
  "DONE write docs" \
  "Try: vim -Nu NONE -n -es -c '%s/TODO/DONE/g' -c 'wq' file && cat file"

trainer_expect_contains \
  "Lesson 2: print specific line" \
  "print only line 2 using Ex mode" \
  "Run a command that prints line 2 from $TRAINER_WORKDIR/list.txt using vim Ex commands" \
  "beta" \
  "Try vim -Nu NONE -n -es -c '2p' -c 'q!' file"

trainer_expect_contains \
  "Lesson 3: delete blank lines" \
  "remove empty lines in-place" \
  "Run a command that deletes blank lines from $TRAINER_WORKDIR/blanky.txt and then cats the file" \
  "blue" \
  "Use g/^$/d in Ex mode, then write and quit."

trainer_expect_contains \
  "Lesson 4: append text" \
  "append one line at end of file" \
  "Run a command that appends 'omega' to $TRAINER_WORKDIR/list.txt using vim Ex mode and prints file" \
  "omega" \
  "Use \$put='omega' then wq."

trainer_section "Reference answers"
cat <<EOF
1) vim -Nu NONE -n -es -c '%s/TODO/DONE/g' -c 'wq' "$TRAINER_WORKDIR/notes.txt" && cat "$TRAINER_WORKDIR/notes.txt"
2) vim -Nu NONE -n -es -c '2p' -c 'q!' "$TRAINER_WORKDIR/list.txt"
3) vim -Nu NONE -n -es -c 'g/^$/d' -c 'wq' "$TRAINER_WORKDIR/blanky.txt" && cat "$TRAINER_WORKDIR/blanky.txt"
4) vim -Nu NONE -n -es -c "\$put='omega'" -c 'wq' "$TRAINER_WORKDIR/list.txt" && cat "$TRAINER_WORKDIR/list.txt"
EOF

trainer_finish \
  "Nice work. You covered practical Vim Ex workflows for scripted edits." \
  "Re-run the trainer to improve your score."
