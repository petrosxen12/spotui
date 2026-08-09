# spotui

Terminal Spotify controller (CLI + Bubble Tea TUI). See README.md for usage, config, and troubleshooting.

## Keeping the demo current

When a change meaningfully alters the TUI's look or behavior (new view, changed layout, new/renamed slash command, changed keybindings), invoke the `update-demo` skill before considering the work done. It re-records the asciinema demo and updates README.md's Screenshots/Features sections to match.

## Keeping the architecture doc current

When a change adds/removes/renames a package, changes which packages call which, or changes the runtime/local-player flow, invoke the `update-architecture` skill before considering the work done. It keeps ARCHITECTURE.md's prose and Mermaid diagram in sync with the code.

## PR descriptions

Keep them concise and to the point. Extract the semantic meaning of the change — why it matters and what it does — rather than listing files touched or restating the diff.
