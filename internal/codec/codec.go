// Package codec handles JSON serialization between hook processes and AI agents.
// Hooks receive context via stdin (JSON) and respond via stdout (JSON).
package codec

import (
	"bufio"
	"encoding/json"
	"os"
)

// DecodeStdin decodes one JSON object from stdin.
// Uses json.Decoder instead of io.ReadAll so it returns as soon as a
// complete JSON value is read, without waiting for EOF. This is critical
// for Cursor, which writes JSON to the pipe but does NOT close it.
//
// Strips a leading UTF-8 BOM (\xEF\xBB\xBF) if present, because
// Cursor on Windows prepends one before the JSON payload.
func DecodeStdin(v any) error {
	r := bufio.NewReader(os.Stdin)
	if bom, err := r.Peek(3); err == nil && bom[0] == 0xEF && bom[1] == 0xBB && bom[2] == 0xBF {
		r.Discard(3)
	}
	return json.NewDecoder(r).Decode(v)
}

// EncodeStdout marshals v to JSON and writes it to stdout followed by a newline.
func EncodeStdout(v any) error {
	return json.NewEncoder(os.Stdout).Encode(v)
}
