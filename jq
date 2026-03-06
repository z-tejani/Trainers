#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/lib/trainer.sh"

trainer_require_cmd "jq" "Error: jq is not installed on this system."
trainer_init_workdir

cat >"$TRAINER_WORKDIR/data.json" <<'EOF'
{
  "users": [
    {"id": 1, "name": "Ada", "role": "admin", "active": true},
    {"id": 2, "name": "Lin", "role": "user", "active": false},
    {"id": 3, "name": "Sam", "role": "admin", "active": true}
  ],
  "orders": [
    {"id": "o1", "amount": 12},
    {"id": "o2", "amount": 5},
    {"id": "o3", "amount": 35}
  ]
}
EOF

trainer_section "jq trainer"
echo "You will run jq commands and verify output."
echo "Sample files are in: $TRAINER_WORKDIR"
trainer_pause

trainer_expect_contains \
  "Lesson 1: select fields" \
  "print all user names" \
  "Run a command that prints each user name from $TRAINER_WORKDIR/data.json" \
  "\"Ada\"" \
  "Use: jq '.users[].name' file"

trainer_expect_contains \
  "Lesson 2: filter objects" \
  "print names of active users only" \
  "Run a command that prints names where active is true" \
  "\"Sam\"" \
  "Use select(.active)."

trainer_expect_contains \
  "Lesson 3: aggregate values" \
  "sum all order amounts" \
  "Run a command that prints the total order amount" \
  "52" \
  "Map amounts and add them."

trainer_expect_contains \
  "Lesson 4: grouped counts" \
  "count users per role" \
  "Run a command that outputs role counts like {\"admin\":2,\"user\":1}" \
  "\"admin\": 2" \
  "Use reduce over .users[].role."

trainer_section "Reference answers"
cat <<EOF
1) jq '.users[].name' "$TRAINER_WORKDIR/data.json"
2) jq '.users[] | select(.active) | .name' "$TRAINER_WORKDIR/data.json"
3) jq '[.orders[].amount] | add' "$TRAINER_WORKDIR/data.json"
4) jq 'reduce .users[].role as \$r ({}; .[\$r] += 1)' "$TRAINER_WORKDIR/data.json"
EOF

trainer_finish \
  "Nice work. You covered jq querying, filtering, and aggregation." \
  "Re-run the trainer to improve your score."
