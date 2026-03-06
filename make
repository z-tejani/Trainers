#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=/dev/null
source "$SCRIPT_DIR/lib/trainer.sh"

trainer_require_cmd "make" "Error: make is not installed on this system."
trainer_init_workdir

cat >"$TRAINER_WORKDIR/Makefile" <<'EOF'
NAME ?= world
SRC = app.c util.c
OBJ = $(SRC:.c=.o)

.PHONY: hello list-objs clean

hello:
	@echo "hello $(NAME)"

list-objs:
	@echo "$(OBJ)"

build/%:
	@echo "building $*"

clean:
	@echo "cleaned"
EOF

trainer_section "make trainer"
echo "You will run make commands against a temporary Makefile."
echo "Makefile path: $TRAINER_WORKDIR/Makefile"
trainer_pause

trainer_expect_contains \
  "Lesson 1: run target" \
  "invoke the hello target" \
  "Run a command that executes hello from $TRAINER_WORKDIR/Makefile" \
  "hello world" \
  "Use make -f ... hello"

trainer_expect_contains \
  "Lesson 2: override variable" \
  "override NAME at invocation time" \
  "Run a command that prints hello ada by setting NAME=ada" \
  "hello ada" \
  "Pass NAME=ada on the make command line."

trainer_expect_contains \
  "Lesson 3: inspect substitutions" \
  "show derived object names from pattern substitution" \
  "Run a command that executes list-objs target" \
  "app.o util.o" \
  "Target: list-objs"

trainer_expect_contains \
  "Lesson 4: pattern targets" \
  "run build/api and verify stem expansion" \
  "Run a command that executes target build/api" \
  "building api" \
  "Use make -f ... build/api"

trainer_section "Reference answers"
cat <<EOF
1) make -f "$TRAINER_WORKDIR/Makefile" hello
2) make -f "$TRAINER_WORKDIR/Makefile" NAME=ada hello
3) make -f "$TRAINER_WORKDIR/Makefile" list-objs
4) make -f "$TRAINER_WORKDIR/Makefile" build/api
EOF

trainer_finish \
  "Nice work. You covered make targets, variable overrides, and pattern rules." \
  "Re-run the trainer to improve your score."
