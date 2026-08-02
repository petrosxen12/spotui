#!/usr/bin/env bash
# Runs *inside* the demo-record container (see Dockerfile + Taskfile `demo:record`).
# Builds spotui, drives it through a scripted tmux session under `asciinema rec`, then
# converts the resulting cast to a GIF with agg. Both outputs land directly in
# docs/screenshots/ on the host because /workspace is a bind mount of the repo root.
#
# The default script below covers spotui's most common real-world path: connect, use the
# managed local spotifyd player, search, navigate results with the keyboard, play a track
# (so the album-art accent color actually changes), and use autocomplete along the way.
# Edit the "drive the TUI" block whenever a change alters this path, so the recording
# keeps demonstrating reality rather than a stale script.
set -euo pipefail

cd /workspace

# spotui/lipgloss picks its color profile via termenv, which only emits true 24-bit RGB
# (rather than downsampling the album-art accent color and the connected-state green to
# the nearest ANSI-256 entry) when COLORTERM=truecolor is present. This has to be set
# before asciinema (and, transitively, spotui) starts — it is inherited through the tmux
# session below, not read fresh by spotui itself.
export TERM=xterm-256color
export COLORTERM=truecolor

mkdir -p "$HOME"
git config --global --add safe.directory /workspace
go build -o spotui ./cmd/spotui

# --- writable spotui config, seeded from the host's real (read-only mounted) one ---
# A straight symlink into the read-only mount (as used for read-only recordings) can't
# support this script: /local start needs to write spotifyd's pid/log/state files, and
# spotifyd itself needs a writable cache dir. So make a real, writable copy instead.
# This also carries over runtime/spotifyd/cache/zeroconf/credentials.json — the host's
# already-paired Spotify Connect session — so the container's spotifyd can authenticate
# immediately instead of needing a live zeroconf pairing flow (which won't work headless).
mkdir -p "$XDG_CONFIG_HOME"
cp -a /tmp/spotui-config-ro "$XDG_CONFIG_HOME/spotui"
chmod -R u+w "$XDG_CONFIG_HOME/spotui"

python3 - <<'PY'
import json
import os

path = os.path.join(os.environ["XDG_CONFIG_HOME"], "spotui", "config.json")
with open(path) as f:
    cfg = json.load(f)

lp = cfg.setdefault("local_player", {})
lp["spotifyd_path"] = "/usr/local/bin/spotifyd"
lp["backend"] = "pulseaudio"
lp["audio_device"] = ""
# No D-Bus session bus in this container; spotifyd's MPRIS integration would otherwise
# crash shortly after connecting trying to autolaunch one. See use_mpris in
# internal/config/config.go — off here only, on by default for real desktop use.
lp["use_mpris"] = False

with open(path, "w") as f:
    json.dump(cfg, f, indent=2)
PY

# --- self-contained PulseAudio server with a null sink ---
# spotifyd needs *a* pulseaudio sink to write to; a null sink means no host audio
# hardware or host PulseAudio/PipeWire server is required at all.
export XDG_RUNTIME_DIR=/tmp/pulse-runtime
mkdir -p "$XDG_RUNTIME_DIR"
chmod 700 "$XDG_RUNTIME_DIR"
pulseaudio -D --exit-idle-time=-1 --disallow-exit=1 --log-target=stderr
sleep 1
pactl load-module module-null-sink sink_name=spotui_demo_sink sink_properties=device.description=spotui_demo_sink
pactl set-default-sink spotui_demo_sink

SESSION=spotui-demo-rec
CAST=docs/screenshots/spotui-demo.cast
GIF=docs/screenshots/spotui-demo.gif

tmux kill-session -t "$SESSION" 2>/dev/null || true
tmux new-session -d -s "$SESSION" -x 100 -y 30

tmux send-keys -t "$SESSION" "asciinema rec --overwrite -c './spotui tui' $CAST" Enter
sleep 2

# --- drive the TUI (edit per recording) ---

# Autocomplete: show the suggestion panel and ghost completion for a partial command, and
# cycle through it, before clearing back to a blank prompt. Deliberately not accepted +
# submitted here — which suggestion is highlighted depends on internal ordering, and the
# local-player commands below need to run in a specific sequence (start before use).
tmux send-keys -t "$SESSION" "/lo"
sleep 1
tmux send-keys -t "$SESSION" Tab
sleep 1
tmux send-keys -t "$SESSION" Escape
sleep 1
tmux send-keys -t "$SESSION" Escape
sleep 1

# Local playback: start the managed spotifyd player and select it as the active device.
# `/local start` blocks internally (localPlayerDeviceTimeout in internal/app/service.go,
# currently 30s) polling Spotify's classic device-list endpoint for the spotifyd device —
# that endpoint lags noticeably behind the realtime Connect protocol spotifyd itself uses
# to authenticate, so this command can legitimately take up to that long to return.
tmux send-keys -t "$SESSION" "/local start"
sleep 1
tmux send-keys -t "$SESSION" Enter
sleep 32
tmux send-keys -t "$SESSION" "/local status"
sleep 1
tmux send-keys -t "$SESSION" Enter
sleep 2
tmux send-keys -t "$SESSION" "/local use"
sleep 1
tmux send-keys -t "$SESSION" Enter
sleep 3

# Search, then move focus off the input and onto the results list (Tab; arrow keys are
# only read as list navigation once the list itself is focused) before navigating it.
tmux send-keys -t "$SESSION" "fka twigs"
sleep 1
tmux send-keys -t "$SESSION" Enter
sleep 2
tmux send-keys -t "$SESSION" Tab
sleep 1
tmux send-keys -t "$SESSION" Down
sleep 1
tmux send-keys -t "$SESSION" Down
sleep 1

# Play the selected track on the local device — this is what triggers the album-art
# accent color change once Now Playing metadata comes back.
tmux send-keys -t "$SESSION" Enter
sleep 8

# ... add/replace steps here to cover whatever changed ...
# --- end drive the TUI ---

tmux send-keys -t "$SESSION" "q"
sleep 1
tmux kill-session -t "$SESSION" 2>/dev/null || true

# asciinema writes the cast asynchronously on exit; give it a moment to flush.
sleep 1

agg "$CAST" "$GIF"

echo "Wrote $CAST and $GIF"
