package retrieval

import (
	"strings"
	"unicode"
)

// tokenize splits a string into normalized, lowercased tokens. It strips
// markdown noise so the ranker doesn't drown in punctuation. Returns a fresh
// slice each call — that is fine because our corpora are tiny and Load
// happens once at startup.
func tokenize(s string) []string {
	s = strings.ToLower(s)
	var sb strings.Builder
	sb.Grow(len(s))
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			sb.WriteRune(r)
		default:
			sb.WriteRune(' ')
		}
	}
	return strings.Fields(sb.String())
}

// tokenizeQuery is the same as tokenize today. We keep both functions
// because query-time tokenization may later diverge (e.g. add synonyms,
// expand code identifiers). Splitting them now keeps refactors cheap.
func tokenizeQuery(s string) []string { return tokenize(s) }
