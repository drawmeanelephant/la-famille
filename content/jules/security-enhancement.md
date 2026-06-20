---
Title: Routine - Implement Security Enhancement
Author: The Human
Date: 2026-06-19
---

# Routine: Implement Security Enhancement

**Goal:** Identify and fix one small security issue or add one security enhancement that makes the application more secure.

## Task Details
1. **Identify Opportunity:** Scan the codebase for a single security vulnerability or enhancement opportunity that can be fixed cleanly in under 50 lines of code. Focus on:
    *   **Input Validation & Sanitization:** Using `html.EscapeString` for XSS prevention when injecting data into HTML, validating paths with `filepath.IsLocal` to prevent path traversal.
    *   **Secrets & Config:** Ensuring no hardcoded secrets or sensitive data exist in the codebase.
    *   **Error Handling:** Ensuring error messages do not leak sensitive internal information (e.g., stack traces).
    *   **File Handling:** Securely managing file reads/writes and user-provided inputs that affect file paths.

2. **Prioritize:** Choose the highest impact issue that is feasible within the 50-line boundary. Critical vulnerabilities (path traversal, XSS, hardcoded secrets) take precedence over general enhancements.

3. **Implement:** Write secure, defensive Go code.
    *   Add comments explaining the specific security concern being addressed.
    *   Use established Go standard library functions (`html`, `path/filepath`, `crypto`, etc.) or the project's existing sanitization libraries (like `bluemonday`).
    *   Ensure the fix handles errors securely and fails closed.

4. **Log Critical Learnings:** ONLY if the task reveals a specific, non-routine insight about a vulnerability pattern specific to this codebase, log it in `.jules/sentinel.md`. Do not log routine generic security fixes.
    *   **Format:**
        `## YYYY-MM-DD - [Title]`
        `**Vulnerability:** [What was found]`
        `**Learning:** [Why it existed]`
        `**Prevention:** [How to avoid next time]`

## Execution Reminders
*   **Boundaries:** Do not perform large architectural security refactors or add major new security dependencies without checking first. Do not expose vulnerability details if the repo is public.
*   **Verification:** Run `go vet ./...` and `go test ./...` to ensure no functionality is broken and code formatting is correct. If applicable, write a quick unit test for the security fix.
*   **Commit:** Use the title format `🛡️ Sentinel: [Severity] Fix [type]` (e.g., `🛡️ Sentinel: [HIGH] Fix XSS in template generation`) for your PR/commit. The description should include Severity, Vulnerability, Impact, Fix, and Verification steps.
*   **Upon successful completion, you MUST write a short log (including date, routine name, success status, and any learnings or suggestions for improving this routine) to a new markdown file in `content/jules/reports/` (e.g., `content/jules/reports/[date]-[routine-name].md`).**
