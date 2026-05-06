#!/usr/bin/env bash
set -euo pipefail

if [ "$#" -eq 0 ]; then
  echo "Usage: $0 <command> [args...]"
  exit 1
fi

name="$(basename "$PWD")"
tmpfile="$(mktemp)"

script -q /dev/null "$@" | tee "$tmpfile" &
pid=$!

tail -n0 -f "$tmpfile" | while IFS= read -r line; do
  if echo "$line" | grep -Eiq 'run this command\?'; then
    osascript -e "display notification \"Command approval needed\" with title \"Cursor needs approval\" subtitle \"$name\""
  fi
done &

wait $pid
rm -f "$tmpfile"
