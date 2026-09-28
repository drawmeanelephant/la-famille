package ask

import "regexp"

var (
	logEmailPattern          = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
	logSecretPattern         = regexp.MustCompile(`(?i)\b(bearer|token|api[_-]?key|password|secret)\s*[:=]\s*[^\s,;]+`)
	logLongCredentialPattern = regexp.MustCompile(`\b[A-Za-z0-9_./+=-]{32,}\b`)
)

// scrubLogText removes common credential and identity formats before text is
// handed to a logger. Ask deliberately does not log question bodies, but this
// boundary also protects future diagnostic fields and provider errors.
func scrubLogText(raw string) string {
	scrubbed := logSecretPattern.ReplaceAllString(raw, "$1=[REDACTED]")
	scrubbed = logEmailPattern.ReplaceAllString(scrubbed, "[REDACTED_EMAIL]")
	return logLongCredentialPattern.ReplaceAllString(scrubbed, "[REDACTED_TOKEN]")
}
