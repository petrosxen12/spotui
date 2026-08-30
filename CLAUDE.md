# spotui

Terminal Spotify controller (CLI + Bubble Tea TUI). See README.md for usage, config, and troubleshooting.

## Keeping the demo current

When a change meaningfully alters the TUI's look or behavior (new view, changed layout, new/renamed slash command, changed keybindings), invoke the `update-demo` skill before considering the work done. It re-records the asciinema demo and updates README.md's Screenshots/Features sections to match.

## Keeping the architecture doc current

When a change adds/removes/renames a package, changes which packages call which, or changes the runtime/local-player flow, invoke the `update-architecture` skill before considering the work done. It keeps ARCHITECTURE.md's prose and Mermaid diagram in sync with the code.

## PR descriptions

Keep them concise and to the point. Extract the semantic meaning of the change — why it matters and what it does — rather than listing files touched or restating the diff.

## Commit messages

Releases are automated with [release-please](https://github.com/googleapis/release-please), which parses commit history to version bumps and CHANGELOG.md. All commits on `master` must follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

Common types: `feat` (minor bump), `fix` (patch bump), `chore`, `docs`, `ci`, `refactor`, `test`, `perf`. Add `!` after the type/scope (e.g. `feat!:`) or a `BREAKING CHANGE:` footer for a major bump. Squash-merge PRs with a conventional commit title, since that's the message release-please reads.
