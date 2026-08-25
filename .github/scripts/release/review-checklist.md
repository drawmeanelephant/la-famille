# Release review checklist

Durable black-box review pass for every release cut (#500). Feed each
platform family's archive to a reviewer — human or fresh agent session (a
"wolf") — that has **zero repo access**: only the unpacked archive and
`RELEASE-QUICKSTART.md`. Their friction is our backlog.

## Ground rules for reviewers

- No source checkout, no repo clone, no prior context. Archive + quickstart only.
- Capture **verbatim** confusion points, broken steps, doc gaps, unexpected
  output. Do not paraphrase into politeness; exact wording is the data.
- Note the exact command and platform for every finding.
- File one issue per finding, labeled `review-finding`, named platform +
  command in the body. Blockers get fixed forward into v0.1.x patches;
  everything else seeds the next milestone.

## Per-platform flow

For each advertised target (`linux`, `darwin`, `windows` × amd64/arm64):

1. Download the archive and its `SHA256SUMS` entry; verify the checksum.
2. Unpack to an empty directory.
3. Quickstart flow, end to end:
   - [ ] `--version` and `--version --json`
   - [ ] `init` (fresh directory)
   - [ ] `new <slug> --title ...`
   - [ ] `check`
   - [ ] `build`
   - [ ] `serve` (page loads, Ctrl-C stops cleanly)
   - [ ] `publish-check --output public`
4. Realistic author tasks:
   - [ ] Write a real post with frontmatter (title/date/tags)
   - [ ] Pick another bundled theme as site default and rebuild
   - [ ] Switch layout on a single page via frontmatter
   - [ ] Deploy `public/` to GitHub Pages (or document exactly where it breaks)
5. Discoverability:
   - [ ] `themes` lists every bundled theme with a description
   - [ ] Theme pick/switch achievable from docs alone

## Platform notes per pass

Record how each family was actually exercised; never mark unexecuted
platforms as passing.

| Family | How it was run | Pass date | Findings |
| --- | --- | --- | --- |
| darwin/arm64 | native on Apple Silicon | 2026-08-25 | #516, #517, #518 |
| darwin/amd64 | Rosetta 2 on Apple Silicon | 2026-08-25 | none beyond arm64 pass |
| linux/* | CI runner or container required | _not executed this pass_ | rerun before advertising linux support |
| windows/* | Windows runner/VM required | _not executed this pass_ | rerun before advertising windows support |

## v0.1.0-prealpha pass (2026-08-25)

- darwin/arm64: full pass completed natively against the published release
  (download → SHA256SUMS verify → unpack → quickstart flow → author tasks:
  tagged post, theme default switch via config edit, per-page layout switch,
  broken-link stub behavior, serve on custom port with clean stop).
- darwin/amd64: binary executes under Rosetta (`--version` verified);
  not a substitute for real Intel hardware.
- linux/*: NOT executed this pass — needs a Linux runner or container; rerun
  before advertising linux support.
- windows/*: NOT executed this pass — needs a Windows runner or VM; rerun
  before advertising windows support.
- Findings triaged: #516 (publish-check misses stub pages), #517 (fresh-site
  description warnings), #518 (binary-only Pages deploy docs). Everything
  else in the flow worked as documented; no blockers found on darwin.
