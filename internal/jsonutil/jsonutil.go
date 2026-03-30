package jsonutil

import (
	"bytes"
	"encoding/json"
)

// MarshalIndent is like json.MarshalIndent but does not escape HTML characters
// (<, >, &). Go's default encoder escapes these, which turns version constraints
// like ">=1.0" into "\u003e=1.0" in the output.
func MarshalIndent(v any, prefix, indent string) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent(prefix, indent)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	// Encode appends a trailing newline; trim it to match MarshalIndent behavior
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return b, nil
}
