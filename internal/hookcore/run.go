package hookcore

import (
	"fmt"
	"os"

	"github.com/CheckmarxDev/ast-cx-hooks/internal/codec"
)

// Run reads one JSON event from stdin, passes it to handler, and writes the
// result to stdout. Any stdin/stdout codec error is logged to stderr and the
// process exits 0 so a bad payload never blocks an agent. Shared by the root
// Process and by every platform adapter so error handling lives in one place.
func Run[I any, O any](handler func(I) O) {
	var in I
	if err := codec.DecodeStdin(&in); err != nil {
		fmt.Fprintf(os.Stderr, "agenthooks: stdin decode error: %v\n", err)
		os.Exit(0)
	}
	if err := codec.EncodeStdout(handler(in)); err != nil {
		fmt.Fprintf(os.Stderr, "agenthooks: stdout encode error: %v\n", err)
		os.Exit(0)
	}
}

// RunE is like Run but lets the handler signal a blocking error: a non-nil error
// is written to stderr and the process exits 2, which causes supporting agents to
// surface the message and block the pending action.
func RunE[I any, O any](handler func(I) (O, error)) {
	var in I
	if err := codec.DecodeStdin(&in); err != nil {
		fmt.Fprintf(os.Stderr, "agenthooks: stdin decode error: %v\n", err)
		os.Exit(0)
	}
	out, err := handler(in)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	if err := codec.EncodeStdout(out); err != nil {
		fmt.Fprintf(os.Stderr, "agenthooks: stdout encode error: %v\n", err)
		os.Exit(0)
	}
}

// Ptr returns a pointer to v. Useful for optional JSON fields modeled as *T.
func Ptr[T any](v T) *T { return &v }
