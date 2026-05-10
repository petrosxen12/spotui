# spotui TUI QA Review Brief

Review this bundle as a UI quality-control pass, not as a code audit. Judge whether the TUI feels minimal, functional, and polished, with the restraint and smoothness expected from an Apple-quality production surface.

## What To Review

- Visual hierarchy: is the most important information obvious within a second?
- Density: does each layout breathe, or do panels feel cramped or noisy?
- Interaction clarity: is focus, selection, and command intent easy to parse?
- Compact behavior: do narrow and short layouts degrade cleanly without awkward wrapping?
- Finish quality: does the experience feel deliberate and calm rather than merely functional?

## Output Format

1. List blockers first. Only include issues that materially hurt quality or confidence.
2. Then list polish improvements.
3. End with a short verdict: `ship`, `ship with fixes`, or `needs another pass`.

## Scenarios

- `desktop-search`: `Desktop Search Flow` (`144x34`) - Assess information hierarchy, whitespace, and right-rail usefulness on a wide layout.
- `laptop-devices`: `Laptop Device Picker` (`108x26`) - Check scanability of the device list and whether the command/status surfaces feel calm rather than busy.
- `compact-local-player`: `Compact Local Player Bootstrap` (`72x18`) - Inspect density and whether the compact layout still feels deliberate instead of cramped.
- `narrow-command-suggestions`: `Narrow Command Suggestions` (`54x14`) - Check whether autocomplete remains legible and controlled in a narrow terminal without visual spill or confusion.
