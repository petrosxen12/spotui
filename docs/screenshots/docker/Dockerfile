# Toolchain image for recording the spotui asciinema demo and converting it to a GIF.
# The repo itself is bind-mounted at /workspace at `docker run` time (see record-demo.sh /
# .claude/skills/update-demo/SKILL.md) rather than copied in, so this image only needs to
# provide: Go (to build spotui), tmux + asciinema (to script the recording), agg (to
# convert the resulting .cast to .gif), and a self-contained local-playback stack (a
# PulseAudio server with a null sink + the upstream spotifyd binary) so the demo can
# exercise `/local start` / `/local use` for real without depending on the host's audio
# server at all.

FROM rust:1.86-slim AS agg-builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    git pkg-config libfontconfig1-dev ca-certificates \
    && rm -rf /var/lib/apt/lists/*

RUN cargo install --git https://github.com/asciinema/agg --root /usr/local

FROM golang:1.24-bookworm

RUN apt-get update && apt-get install -y --no-install-recommends \
    tmux asciinema ca-certificates fontconfig fonts-dejavu-core fonts-noto-mono \
    pulseaudio pulseaudio-utils curl \
    && rm -rf /var/lib/apt/lists/* \
    && fc-cache -f

COPY --from=agg-builder /usr/local/bin/agg /usr/local/bin/agg

# Upstream spotifyd release (README recommends this over the Homebrew build for the same
# reason it matters here: a full backend set, no host dependency).
RUN curl -fsSL -o /tmp/spotifyd.tar.gz \
    https://github.com/Spotifyd/spotifyd/releases/latest/download/spotifyd-linux-x86_64-full.tar.gz \
    && tar -xzf /tmp/spotifyd.tar.gz -C /usr/local/bin \
    && rm /tmp/spotifyd.tar.gz \
    && chmod +x /usr/local/bin/spotifyd

WORKDIR /workspace
