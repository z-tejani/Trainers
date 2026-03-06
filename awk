#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/lib/trainer.sh"

trainer_require_cmd "awk" "Error: awk is not installed on this system."
trainer_init_workdir

cat >"$TRAINER_WORKDIR/users.csv" <<'EOF'
id,name,role
1,Ada,admin
2,Lin,user
3,Sam,admin
EOF

cat >"$TRAINER_WORKDIR/sales.csv" <<'EOF'
item,amount
book,12
pen,5
keyboard,35
EOF

trainer_section "awk trainer"
echo "You will run awk commands and verify output."
echo "Sample files are in: $TRAINER_WORKDIR"
trainer_pause

trainer_expect_contains \
  "Lesson 1: select a field" \
  "print only names from users.csv" \
  "Run a command that prints the name column (without header) from $TRAINER_WORKDIR/users.csv" \
  "Ada" \
  "Use -F, and print \$2 for NR>1."

trainer_expect_contains \
  "Lesson 2: filter rows" \
  "print only admin names" \
  "Run a command that prints names where role is admin in $TRAINER_WORKDIR/users.csv" \
  "Sam" \
  "Filter on \$3 == \"admin\" and print \$2."

trainer_expect_contains \
  "Lesson 3: aggregate values" \
  "sum the amount column" \
  "Run a command that prints total amount from $TRAINER_WORKDIR/sales.csv" \
  "52" \
  "Keep a running sum and print in END."

trainer_expect_contains \
  "Lesson 4: formatted output" \
  "print line number and row for data rows" \
  "Run a command that prints '3:2,Lin,user' style output for $TRAINER_WORKDIR/users.csv (skip header)" \
  "3:2,Lin,user" \
  "Use NR and \$0 together for NR>1."

trainer_section "Reference answers"
cat <<EOF
1) awk -F, 'NR>1 {print \$2}' "$TRAINER_WORKDIR/users.csv"
2) awk -F, 'NR>1 && \$3==\"admin\" {print \$2}' "$TRAINER_WORKDIR/users.csv"
3) awk -F, 'NR>1 {sum+=\$2} END {print sum}' "$TRAINER_WORKDIR/sales.csv"
4) awk 'NR>1 {print NR \":\" \$0}' "$TRAINER_WORKDIR/users.csv"
EOF

trainer_finish \
  "Nice work. You covered awk basics: fields, filters, sums, and formatting." \
  "Re-run the trainer to improve your score."
