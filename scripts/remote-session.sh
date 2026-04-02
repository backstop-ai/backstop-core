#!/usr/bin/env bash
# Starts a Claude Code session that stays alive for remote control.
# Run this before you leave your desk. Connect from your phone via claude.ai.
#
# What it does:
#   1. Prevents your Mac from sleeping (even with lid closed)
#   2. Runs Claude Code inside tmux so it survives terminal crashes
#   3. Enables remote control so you can connect from your phone
#
# Usage:
#   ./scripts/remote-session.sh
#
# To reconnect if your terminal closes:
#   tmux attach -t claude
#
# To stop everything:
#   tmux kill-session -t claude

set -euo pipefail

# Kill any existing caffeinate processes we started
cleanup() {
    if [ -n "${CAFFEINATE_PID:-}" ]; then
        kill "$CAFFEINATE_PID" 2>/dev/null || true
    fi
}
trap cleanup EXIT

# Check dependencies
if ! command -v tmux &>/dev/null; then
    echo "tmux is required. Install with: brew install tmux"
    exit 1
fi

if ! command -v claude &>/dev/null; then
    echo "claude is not on PATH"
    exit 1
fi

# Prevent Mac from sleeping (runs in background)
caffeinate -i &
CAFFEINATE_PID=$!
echo "Mac sleep prevention active (pid $CAFFEINATE_PID)"

# Start or attach to tmux session
if tmux has-session -t claude 2>/dev/null; then
    echo "Existing tmux session found. Attaching..."
    tmux attach -t claude
else
    echo "Starting new Claude Code session with remote control..."
    tmux new-session -s claude "claude --remote-control 'backstop'"
fi
