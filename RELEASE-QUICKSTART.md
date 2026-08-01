# La Famille binary quickstart

Verify the archive with its matching `SHA256SUMS` entry, then run the binary
without a Go toolchain or source checkout:

```bash
./la-famille --version --json
./la-famille --project-root /path/to/site init
./la-famille --project-root /path/to/site new index --title Home
./la-famille --project-root /path/to/site build
./la-famille --project-root /path/to/site publish-check --output public
```

Relative paths come from `--project-root`. The generated `public/` directory
is the complete static artifact; incremental cache state stays beside the
project root.
