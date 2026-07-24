package content

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v2"
)

// StringList decodes a YAML value that may be written either as a sequence or
// as a single scalar string. It exists so `tags: golang` and `tags: [golang]`
// mean the same thing, the way `category: news` and `categories: [news]`
// already do.
//
// The sequence path delegates to a plain []string decode, so every value that
// yaml coerced into a string before (numbers, booleans, dates) is still
// coerced now. Only a non-string scalar is refused, and it is reported as an
// error rather than dropped silently.
type StringList []string

// UnmarshalYAML implements yaml.Unmarshaler.
func (s *StringList) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var list []string
	listErr := unmarshal(&list)
	if listErr == nil {
		*s = StringList(list)
		return nil
	}

	var scalar interface{}
	if err := unmarshal(&scalar); err == nil {
		if str, ok := scalar.(string); ok {
			*s = StringList{str}
			return nil
		}
	}

	// Neither shape applies. Keep whatever the sequence decode salvaged --
	// that is exactly what a plain []string field would have retained -- and
	// hand the type error back so the caller can warn about it.
	*s = StringList(list)
	return listErr
}

// NormalizeFrontmatterKeys lowercases every frontmatter key.
//
// Case-variant spellings of one key (Title / title / TITLE) collapse into a
// single lowercase key. The winner is the exact lowercase spelling when it is
// present and otherwise the first variant in sorted order, so the surviving
// value never depends on Go's randomized map iteration order. Groups are
// visited in sorted order so the returned warnings are stable too.
//
// Every collapsed group holding more than one spelling produces a warning:
// silently discarding one of two values that both look authoritative is how a
// page's title, or its published URL via `slug`, changes between builds.
func NormalizeFrontmatterKeys(raw map[string]interface{}) (map[string]interface{}, []string) {
	normalized := make(map[string]interface{}, len(raw))
	variants := make(map[string][]string, len(raw))
	lowered := make([]string, 0, len(raw))

	for k := range raw {
		lower := strings.ToLower(k)
		if _, seen := variants[lower]; !seen {
			lowered = append(lowered, lower)
		}
		variants[lower] = append(variants[lower], k)
	}
	sort.Strings(lowered)

	var warnings []string
	for _, lower := range lowered {
		spellings := variants[lower]
		sort.Strings(spellings)
		winner := spellings[0]
		for _, k := range spellings {
			if k == lower {
				winner = k
				break
			}
		}
		normalized[lower] = raw[winner]
		if len(spellings) > 1 {
			warnings = append(warnings, fmt.Sprintf("duplicate frontmatter key %v all normalize to %q, using %q", spellings, lower, winner))
		}
	}

	return normalized, warnings
}

// DecodeFrontmatter normalizes the raw frontmatter map returned by
// frontmatter.Parse and decodes it into out, which must be a pointer to a
// struct carrying yaml tags. It returns human-readable warnings for duplicate
// case-variant keys and for values whose type does not fit their destination
// field -- both of which used to be discarded without a trace.
//
// It is exported so the build path and the `check` path share one definition
// of what a frontmatter block means; internal/checker currently carries a
// private copy of this logic that should be replaced by a call to this
// function.
func DecodeFrontmatter(raw map[string]interface{}, out interface{}) []string {
	if raw == nil {
		return nil
	}

	normalized, warnings := NormalizeFrontmatterKeys(raw)

	yamlBytes, err := yaml.Marshal(normalized)
	if err != nil {
		return append(warnings, fmt.Sprintf("frontmatter could not be re-encoded, all values ignored: %s", flattenError(err)))
	}
	if err := yaml.Unmarshal(yamlBytes, out); err != nil {
		warnings = append(warnings, fmt.Sprintf("frontmatter value ignored: %s", flattenError(err)))
	}

	return warnings
}

// flattenError renders an error on a single line. yaml.TypeError in particular
// is multi-line, and these strings end up in sorted warning lists and in the
// persisted build cache where an embedded newline is unreadable.
func flattenError(err error) string {
	if err == nil {
		return ""
	}
	var typeErr *yaml.TypeError
	if errors.As(err, &typeErr) {
		return strings.Join(typeErr.Errors, "; ")
	}
	lines := strings.Split(err.Error(), "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.Join(lines, "; ")
}
