// Package varcore is the pure functional core of the Go port of Vár: parse →
// match → plan → execute, plus the conformance projections and drift. It has no
// dependency on the facade, runner, or any test framework, and performs no I/O.
package varcore

import (
	"bytes"
	"encoding/json"
)

// CanonicalStringify serializes value to the canonical JSON wire format shared
// by every Vár port. Port of canonical_json.py / canonicalStringify in
// conformance.ts. The output is byte-for-byte compatible with the JavaScript
// reference:
//
//		JSON.stringify(sortKeys(value), null, 2) + "\n"
//
//	  - object keys recursively sorted (encoding/json sorts map[string]any keys,
//	    so artifacts are built from maps, never structs, to inherit the sort);
//	  - 2-space indentation;
//	  - non-ASCII emitted raw (emoji/CJK/accents appear literally), and '<', '>',
//	    '&' left unescaped (SetEscapeHTML(false));
//	  - LF line endings with a single trailing newline (Encoder.Encode appends
//	    exactly one '\n').
func CanonicalStringify(value any) (string, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return "", err
	}
	return buf.String(), nil
}
