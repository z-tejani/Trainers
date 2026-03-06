#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/lib/trainer.sh"

trainer_require_cmd "zsh" "Error: zsh is not installed on this system."
trainer_init_workdir

mkdir -p "$TRAINER_WORKDIR/glob"
touch "$TRAINER_WORKDIR/glob/app.log"
touch "$TRAINER_WORKDIR/glob/worker.log"
touch "$TRAINER_WORKDIR/glob/notes.txt"

trainer_section "zsh trainer"
echo "You will run zsh-specific commands and verify output."
echo "Sample files are in: $TRAINER_WORKDIR"
echo "Tip: run exercises with zsh -c '...'"
trainer_pause

trainer_expect_contains \
  "Lesson 1: brace expansion" \
  "generate a sequence of filenames using braces" \
  "Run a command that prints: file1.txt file2.txt file3.txt (space-separated)" \
  "file1.txt file2.txt file3.txt" \
  "Use: zsh -c 'print file{1..3}.txt'"

trainer_expect_contains \
  "Lesson 2: parameter defaults" \
  "use default value expansion" \
  "Run a command that unsets NAME and prints \${NAME:-guest} in zsh" \
  "guest" \
  "Use: zsh -c 'unset NAME; print \${NAME:-guest}'"

trainer_expect_contains \
  "Lesson 3: associative arrays" \
  "read a value from a zsh associative array" \
  "Run a command that creates map[key]=42 and prints 42" \
  "42" \
  "Use typeset -A in zsh."

trainer_expect_contains \
  "Lesson 4: glob qualifiers" \
  "print only .log basenames from glob directory" \
  "Run a command that prints app.log and worker.log from $TRAINER_WORKDIR/glob using zsh globbing" \
  "worker.log" \
  "Use the (:t) tail modifier."

trainer_section "Reference answers"
cat <<EOF
1) zsh -c 'print file{1..3}.txt'
2) zsh -c 'unset NAME; print \${NAME:-guest}'
3) zsh -c 'typeset -A map; map[key]=42; print \$map[key]'
4) zsh -c 'print -l "$TRAINER_WORKDIR"/glob/*.log(:t)'
EOF

trainer_finish \
  "Nice work. You practiced core zsh features beyond basic POSIX shell." \
  "Re-run the trainer to improve your score."
