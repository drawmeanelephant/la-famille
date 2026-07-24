package ask

import (
	"encoding/json"
	"io"
)

// jsonEncoder returns a json.Encoder configured for compact, deterministic
// output. Kept separate so tests can replace it with a verbose variant if
// needed in the future without touching call sites.
func jsonEncoder(w io.Writer) *json.Encoder {
	enc := json.NewEncoder(w)
	return enc
}

// decodeJSON is a tiny wrapper around json.NewDecoder configured to reject
// unknown fields strictly. The strictness stops a typo in the UI (e.g. a
// rename of "question") from silently producing "empty question" 500s.
func decodeJSON(r io.Reader, v any) error {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
