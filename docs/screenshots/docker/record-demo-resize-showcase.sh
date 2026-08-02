#!/usr/bin/env bash
# Runs *inside* the demo-record container (see Dockerfile + Taskfile `demo:resize-record`).
# Demonstrates spotui's adaptive layout: a real tmux split-pane layout with spotui in one
# pane and `top` in the other, live-resized while recording — showing the TUI reflow from
# a wide layout down to a genuinely narrow split-pane width and back, with a track actively
# playing throughout so the album-art accent color change is visible across every size,
# not just at the size it started at. This is the same reflow behavior the QA review
# bundle (`task qa:tui-review`) checks statically at fixed sizes; this recording shows it
# live, in motion, in color.
#
# Technique: an INNER tmux session holds the real split-pane layout. A separate RECORDER
# session runs `asciinema rec -c "tmux attach -t $INNER"` — so the recording captures the
# attached client's rendered view (pane borders, status line, live reflow) rather than a
# single program's own pty. Resize/interaction commands are sent from this script (the
# "control channel") to the INNER session while the RECORDER's attach passively captures
# whatever it renders.
#
# Only ever resize-pane (redistribute columns between panes), never resize-window (change
# the overall terminal size): the installed asciinema version doesn't record live
# terminal-size changes into the cast, so a whole-window resize silently desyncs the
# recording from what agg renders. See the inline comment further down for details.
#
# Edit the "drive the demo" block whenever spotui's layout behavior changes meaningfully,
# so this keeps demonstrating reality rather than a stale script.
set -euo pipefail

cd /workspace

# spotui/lipgloss picks its color profile via termenv, which only emits true 24-bit RGB
# (rather than downsampling the album-art accent color to the nearest ANSI-256 entry) when
# COLORTERM=truecolor is present — see record-demo.sh for the full explanation.
export TERM=xterm-256color
export COLORTERM=truecolor

mkdir -p "$HOME"
git config --global --add safe.directory /workspace
go build -o spotui ./cmd/spotui

# --- writable spotui config, seeded from the host's real (read-only mounted) one ---
# Needed for real local playback here (see record-demo.sh for the full rationale): /local
# start needs to write spotifyd's pid/log/state files, spotifyd needs a writable cache dir,
# and reusing runtime/spotifyd/cache/zeroconf/credentials.json (the host's already-paired
# Spotify Connect session) lets the container's spotifyd authenticate immediately instead
# of needing a live zeroconf pairing flow.
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
# crash shortly after connecting. See use_mpris in internal/config/config.go — off here
# only, on by default for real desktop use.
lp["use_mpris"] = False

with open(path, "w") as f:
    json.dump(cfg, f, indent=2)
PY

# --- self-contained PulseAudio server with a null sink ---
# spotifyd needs *a* pulseaudio sink to write to; a null sink means no host audio hardware
# or host PulseAudio/PipeWire server is required at all.
export XDG_RUNTIME_DIR=/tmp/pulse-runtime
mkdir -p "$XDG_RUNTIME_DIR"
chmod 700 "$XDG_RUNTIME_DIR"
pulseaudio -D --exit-idle-time=-1 --disallow-exit=1 --log-target=stderr
sleep 1
pactl load-module module-null-sink sink_name=spotui_demo_sink sink_properties=device.description=spotui_demo_sink
pactl set-default-sink spotui_demo_sink

INNER=spotui-resize-inner
RECORDER=spotui-resize-recorder
CAST=docs/screenshots/spotui-resize-demo.cast
GIF=docs/screenshots/spotui-resize-demo.gif

tmux kill-session -t "$INNER" 2>/dev/null || true
tmux kill-session -t "$RECORDER" 2>/dev/null || true

# Wide starting canvas: spotui pane (0.0) gets ~139 cols, `top` (0.1) alongside it. This
# needs to clear internal/ui/layout.go's context-rail threshold (bodyWidth >= 118, i.e.
# terminal width >= ~120 once padding is subtracted) — a first cut of this script gave
# spotui only 119 cols here, 1 short of the threshold, so the rail never appeared at all.
tmux new-session -d -s "$INNER" -x 160 -y 40
tmux split-window -h -t "$INNER" -l 20
tmux send-keys -t "${INNER}:0.0" "./spotui tui" Enter
tmux send-keys -t "${INNER}:0.1" "top" Enter
sleep 2

# RECORDER attaches to INNER under asciinema — this is what actually gets captured.
tmux new-session -d -s "$RECORDER" -x 160 -y 40
tmux send-keys -t "$RECORDER" "asciinema rec --overwrite -c 'tmux attach -t $INNER' $CAST" Enter
sleep 2

# --- drive the demo (edit per recording) ---

# Local playback: start the managed spotifyd player and select it as the active device.
# `/local start` blocks internally (localPlayerDeviceTimeout in internal/app/service.go,
# currently 30s) polling Spotify's classic device-list endpoint for the spotifyd device —
# that endpoint lags noticeably behind the realtime Connect protocol spotifyd itself uses
# to authenticate, so this command can legitimately take up to that long to return.
tmux send-keys -t "${INNER}:0.0" "/local start"
sleep 1
tmux send-keys -t "${INNER}:0.0" Enter
sleep 32
tmux send-keys -t "${INNER}:0.0" "/local use"
sleep 1
tmux send-keys -t "${INNER}:0.0" Enter
sleep 3

# Search, then move focus off the input and onto the results list (Tab; arrow keys are
# only read as list navigation once the list itself is focused), and change the selection —
# this also drives the context rail's "further info about the selected track" panel, which
# otherwise only ever shows the default first item.
tmux send-keys -t "${INNER}:0.0" "fka twigs"
sleep 1
tmux send-keys -t "${INNER}:0.0" Enter
sleep 2
tmux send-keys -t "${INNER}:0.0" Tab
sleep 1
tmux send-keys -t "${INNER}:0.0" Down
sleep 1
tmux send-keys -t "${INNER}:0.0" Down
sleep 1

# Play the selected track on the local device — this is what triggers the album-art accent
# color change. Everything from here on (context rail, results list, playbar) should
# render in that color instead of the default green, at every pane width below.
tmux send-keys -t "${INNER}:0.0" Enter
sleep 8

# Shrink the spotui pane's share of the split in steps — this is the literal "make the
# spotui pane really small while another pane is open" scenario. Deliberately only ever
# resize-pane (redistributing columns between panes within the same fixed window size),
# never resize-window: this asciinema version doesn't record live terminal-size changes
# into the cast (no resize events in the v2 stream, confirmed by direct testing), so a
# whole-window resize changes the actual pty size under agg's feet without agg's fixed-size
# render ever finding out, producing silently misaligned frames. A pane-proportion resize
# keeps the overall window size constant throughout, so it renders correctly.
#
# This first step (139 -> 100) crosses the context-rail threshold (bodyWidth >= 118) going
# the other direction, so the rail should visibly disappear and the results list should
# reclaim that width — the same reflow the QA bundle's desktop-search/laptop-devices
# scenarios check statically. The playback progress bar length (playbarProgressLen in
# internal/ui/layout.go, scales with mainWidth) should visibly shrink here too, all in the
# accent color from the still-playing track.
tmux resize-pane -t "${INNER}:0.0" -x 100
sleep 3
tmux resize-pane -t "${INNER}:0.0" -x 70
sleep 2

# While moderately cramped, prove autocomplete still renders sanely, not just static text.
tmux send-keys -t "${INNER}:0.0" Tab
sleep 1
tmux send-keys -t "${INNER}:0.0" "/lo"
sleep 2
# One Escape closes the suggestion panel. A second Escape here would be wrong: the earlier
# search already pushed a view state, so a second Escape falls through to popViewState()
# and silently pops back to the pre-search idle view instead of just clearing the input —
# wiping out the results for the rest of the recording. Tab (toggleFocus, touches no view
# state) moves focus back onto the results list instead.
tmux send-keys -t "${INNER}:0.0" Escape
sleep 1
tmux send-keys -t "${INNER}:0.0" Tab
sleep 1

# Shrink further to a genuinely narrow split — well past the point most apps degrade badly.
tmux resize-pane -t "${INNER}:0.0" -x 32
sleep 3

# Grow back to the original wide layout to show the reflow works cleanly in both
# directions — including the context rail reappearing once past the 118 bodyWidth threshold.
tmux resize-pane -t "${INNER}:0.0" -x 139
sleep 2

# --- end drive the demo ---

tmux send-keys -t "${INNER}:0.0" "q"
sleep 1
tmux send-keys -t "${INNER}:0.1" "q"
sleep 1

tmux kill-session -t "$RECORDER" 2>/dev/null || true
tmux kill-session -t "$INNER" 2>/dev/null || true

# asciinema writes the cast asynchronously on exit; give it a moment to flush.
sleep 1

agg "$CAST" "$GIF"

echo "Wrote $CAST and $GIF"
