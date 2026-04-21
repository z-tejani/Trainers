#!/usr/bin/env bash

set -euo pipefail

TRAINER_SCORE=0
TRAINER_TOTAL=0
TRAINER_WORKDIR=""
TRAINER_MODE="${TRAINER_MODE:-guided}"
TRAINER_NAME=""
TRAINER_SCRIPT_PATH=""

trainer_normalize_mode() {
  case "${1:-guided}" in
    guided|training)
      printf '%s\n' "guided"
      ;;
    manual)
      printf '%s\n' "manual"
      ;;
    test)
      printf '%s\n' "test"
      ;;
    *)
      return 1
      ;;
  esac
}

trainer_print_usage() {
  local script_name
  script_name="$(basename "${TRAINER_SCRIPT_PATH:-$0}")"
  cat <<EOF
Usage: ./$script_name [--guided|--training|--manual|--test|--mode MODE]

Common launch modes:
  --guided, --training   Interactive coaching mode with teach/hint/skip support.
  --manual               Walk through the lessons without prompts or scoring.
  --test                 One-attempt assessment mode without hints.
  --mode MODE            Explicit mode selection: guided, manual, or test.
  -h, --help             Show this help text.
EOF
}

trainer_parse_args() {
  TRAINER_SCRIPT_PATH="${1:-$0}"
  TRAINER_NAME="$(basename "$TRAINER_SCRIPT_PATH")"
  shift || true

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --guided|--training)
        TRAINER_MODE="guided"
        shift
        ;;
      --manual)
        TRAINER_MODE="manual"
        shift
        ;;
      --test)
        TRAINER_MODE="test"
        shift
        ;;
      --mode)
        if [[ $# -lt 2 ]]; then
          echo "Error: --mode requires a value."
          trainer_print_usage
          exit 1
        fi
        if ! TRAINER_MODE="$(trainer_normalize_mode "$2")"; then
          echo "Error: unsupported mode '$2'."
          trainer_print_usage
          exit 1
        fi
        shift 2
        ;;
      -h|--help)
        trainer_print_usage
        exit 0
        ;;
      *)
        echo "Error: unknown option '$1'."
        trainer_print_usage
        exit 1
        ;;
    esac
  done

  export TRAINER_MODE
  export TRAINER_NAME
}

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
  if [[ "$TRAINER_MODE" != "guided" ]]; then
    return
  fi
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

trainer_teach_concept() {
  local goal="$1"
  local hint="${2:-}"
  local prompt="${3:-}"
  echo "Mini-lesson:"
  echo "- Concept: $goal"
  if [[ -n "$hint" ]]; then
    echo "- Approach: $hint"
  fi
  if [[ -n "$prompt" ]]; then
    echo "- Task framing: $prompt"
  fi
  echo "- Workflow: pick the command, run it, inspect output, iterate."
}

trainer_expect_contains() {
  local title="$1"
  local goal="$2"
  local prompt="$3"
  local expected="$4"
  local hint="${5:-}"
  local cmd output attempt

  TRAINER_TOTAL=$((TRAINER_TOTAL + 1))
  trainer_section "$title"
  echo "Goal: $goal"
  echo "Mode: $TRAINER_MODE"

  case "$TRAINER_MODE" in
    manual)
      trainer_teach_concept "$goal" "$hint" "$prompt"
      echo
      echo "$prompt"
      echo "Manual mode: review the task, try commands separately if you want, then compare with the reference answers."
      echo "Target output should include: $expected"
      return
      ;;
    test)
      echo
      echo "$prompt"
      echo "Test mode: one attempt, no teach/hint/skip commands."
      read -r -p "> " cmd

      if [[ -z "$cmd" ]]; then
        echo "No answer provided."
        echo "Expected output should include: $expected"
        return
      fi

      if ! output="$(trainer_run_cmd "$cmd")"; then
        echo "Command failed."
        if [[ -n "${output:-}" ]]; then
          printf '%s\n' "$output"
        fi
        echo "Expected output should include: $expected"
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
      return
      ;;
    *)
      trainer_teach_concept "$goal" "$hint" "$prompt"
      echo
      echo "$prompt"
      echo "Interactive commands: 'teach', 'hint', 'skip'"

      attempt=1
      while true; do
        read -r -p "attempt $attempt > " cmd

        if [[ -z "$cmd" ]]; then
          echo "Enter a command, or use: teach, hint, skip."
          continue
        fi

        if [[ "$cmd" == "teach" ]]; then
          trainer_teach_concept "$goal" "$hint" "$prompt"
          continue
        fi

        if [[ "$cmd" == "hint" ]]; then
          if [[ -n "$hint" ]]; then
            echo "Hint: $hint"
          else
            echo "Hint: break the task into command + options + input."
          fi
          continue
        fi

        if [[ "$cmd" == "skip" ]]; then
          echo "Skipped. Expected output should include: $expected"
          return
        fi

        if ! output="$(trainer_run_cmd "$cmd")"; then
          echo "Command failed."
          if [[ -n "${output:-}" ]]; then
            printf '%s\n' "$output"
          fi
          attempt=$((attempt + 1))
          continue
        fi

        if [[ "$output" == *"$expected"* ]]; then
          echo "Correct"
          TRAINER_SCORE=$((TRAINER_SCORE + 1))
          return
        fi

        echo "Output was:"
        printf '%s\n' "$output"
        echo "Not quite. Expected output to include: $expected"
        echo "Use 'teach' for concept review or 'hint' for next-step guidance."
        attempt=$((attempt + 1))
      done
      ;;
  esac
}

trainer_finish() {
  local perfect_msg="$1"
  local retry_msg="$2"
  echo
  if [[ "$TRAINER_MODE" == "manual" ]]; then
    echo "Mode: manual"
    echo "Lessons shown: $TRAINER_TOTAL"
    echo "Manual walkthrough complete. Review the reference answers and re-run in guided or test mode when you want validation."
    return
  fi

  echo "Score: $TRAINER_SCORE / $TRAINER_TOTAL"
  if [[ "$TRAINER_SCORE" -eq "$TRAINER_TOTAL" ]]; then
    echo "$perfect_msg"
  else
    echo "$retry_msg"
  fi
}

trainer_parse_args "$0" "$@"
