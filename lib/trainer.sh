#!/usr/bin/env bash

set -euo pipefail

TRAINER_SCORE=0
TRAINER_TOTAL=0
TRAINER_WORKDIR=""

trainer_require_cmd() {
  local cmd="$1"
  local msg="${2:-Error: required command '$cmd' is not installed.}"
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "$msg"
    exit 1
  fi
}

trainer_init_workdir() {
  TRAINER_WORKDIR="$(mktemp -d)"
  export TRAINER_WORKDIR
  trap 'rm -rf "$TRAINER_WORKDIR"' EXIT
}

trainer_line() {
  printf '%*s\n' "72" '' | tr ' ' '-'
}

trainer_section() {
  trainer_line
  echo "$1"
  trainer_line
}

trainer_pause() {
  read -r -p "Press Enter to continue... " _
}

trainer_run_cmd() {
  local cmd="$1"
  local output
  if ! output="$(bash -lc "$cmd" 2>&1)"; then
    printf '%s\n' "$output"
    return 1
  fi
  printf '%s\n' "$output"
}

trainer_expect_contains() {
  local title="$1"
  local goal="$2"
  local prompt="$3"
  local expected="$4"
  local hint="${5:-}"
  local cmd output

  TRAINER_TOTAL=$((TRAINER_TOTAL + 1))
  trainer_section "$title"
  echo "Goal: $goal"
  echo
  echo "$prompt"
  if [[ -n "$hint" ]]; then
    echo "Type 'hint' to get a clue."
  fi
  read -r -p "> " cmd

  if [[ -n "$hint" && "$cmd" == "hint" ]]; then
    echo "Hint: $hint"
    read -r -p "> " cmd
  fi

  if ! output="$(trainer_run_cmd "$cmd")"; then
    echo "Command failed."
    if [[ -n "${output:-}" ]]; then
      printf '%s\n' "$output"
    fi
    return
  fi

  if [[ "$output" == *"$expected"* ]]; then
    echo "Correct"
    TRAINER_SCORE=$((TRAINER_SCORE + 1))
    return
  fi

  echo "Output was:"
  printf '%s\n' "$output"
  echo "Not quite. Expected output to include: $expected"
}

trainer_finish() {
  local perfect_msg="$1"
  local retry_msg="$2"
  echo
  echo "Score: $TRAINER_SCORE / $TRAINER_TOTAL"
  if [[ "$TRAINER_SCORE" -eq "$TRAINER_TOTAL" ]]; then
    echo "$perfect_msg"
  else
    echo "$retry_msg"
  fi
}
