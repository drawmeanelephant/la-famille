# Security Fix: Prevent Path Traversal in Output Directory Creation

## Intended Steps
1. **Analyze Vulnerability**: Review `filepath.Join(cfg.OutputDir, filepath.FromSlash(relPath))` usage in `internal/generator/generator.go`, `internal/stub/stub.go`, and `internal/asset/copy.go`.
2. **Implement Fix**:
   - Clean `cfg.OutputDir`.
   - Validate that the resulting `outPath` starts with the cleaned output directory and separator, preventing any traversal payload from writing outside `public/`.
3. **Verify locally**: Run `go fmt`, `go test`, and `go vet` to ensure no errors were introduced.
4. **Submit PR**: Open a PR detailing the security fix.

## Potential Breaking Changes
None. The asset generation pipeline will now correctly drop any malicious `.md` or asset paths attempting to escape the output directory, whereas previously it might have created directories or written files outside the scope of the generator.
