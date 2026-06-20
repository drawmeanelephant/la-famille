# Task Brief: Multimedia Devlog Generation & Site Support

The goal of this task is to enable the automatic generation of multimedia development updates (10-second video scripts, animation directions, and soundtrack lyrics) and properly render them on the `la-famille` generated static site.

Jules, you are responsible for defining the routine and adding the necessary template and Go parsing support to render this multimedia content perfectly.

## Requirements

### 1. Create the `multimedia-devlog` Routine
Create a new routine file at `content/jules/multimedia-devlog.md`. This routine must instruct the AI (when executing it) to:
- Summarize the latest development cycle into a punchy, 10-second explainer video script.
- Write corresponding "animation directions" or visual cues intended for a flow video creator tool.
- Draft a short soundtrack/song lyric snippet that matches the vibe.
- Output this as a new markdown file in a new `content/devlog/` directory with specific frontmatter.

### 2. Extend Go Frontmatter Parsing
Update `cmd/la-famille/main.go` (and any related `Page` or struct representations) to parse new frontmatter fields from the markdown files. 
- You must extract the following fields from the YAML frontmatter: `video_script`, `animation_cues`, and `soundtrack_theme`.
- Ensure these new metadata fields are safely passed down into the Go template execution context.

### 3. Create Site Template Support
Design and implement a new DaisyUI template layout (e.g., `templates/devlog.html` or extend `templates/layout.html`) capable of rendering these specific fields beautifully. 
- Use DaisyUI components (like cards, mockups, or split views) to display the video script text next to the animation directions and soundtrack lyrics.
- Include a visual placeholder component for the 10-second video block where the generated video will eventually live.
- **Asset Requirement**: Use one of the new octopus mascot images (e.g., `assets/img/Octopus_mascot_cleaning_litterbox_202606200817.jpeg`) as the "poster/thumbnail" for the video placeholders.

### 4. Build and Test
- Write any necessary unit tests for the new frontmatter parsing logic.
- Run `go test ./...` and `go vet ./...` locally before pushing to verify the implementation.

### 5. Robust Bot Author Suffix Matching
Update the PR sync author matching logic (specifically in `internal/github/github.go`'s `ListOpenPRs` method) to perform case-insensitive suffix-agnostic matching. For example, the configured bot author `"google-labs-jules"` should successfully match both `"google-labs-jules"` and `"google-labs-jules[bot]"`. Add unit tests to verify this matching logic.

