package main

import (
	"context"
	"io"
	"log/slog"
)

// testHandler wraps slog.Handler to add prefix and control verbosity for test output.
type testHandler struct {
	prefix  string
	verbose bool
	w       io.Writer
	h       slog.Handler
}

// newTestHandler creates a new test handler with the given configuration.
func newTestHandler(prefix string, verbose bool, w io.Writer) *testHandler {
	opts := &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}

	return &testHandler{
		prefix:  prefix,
		verbose: verbose,
		w:       w,
		h:       slog.NewTextHandler(w, opts),
	}
}

// Enabled reports whether the handler handles records at the given level.
func (t *testHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if !t.verbose {
		return false // Suppress all logs when not verbose
	}
	return t.h.Enabled(ctx, level)
}

// Handle handles the Record.
func (t *testHandler) Handle(ctx context.Context, r slog.Record) error {
	if !t.verbose {
		return nil
	}

	// Add prefix to output
	if t.prefix != "" {
		t.w.Write([]byte(t.prefix))
	}

	return t.h.Handle(ctx, r)
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
func (t *testHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &testHandler{
		prefix:  t.prefix,
		verbose: t.verbose,
		w:       t.w,
		h:       t.h.WithAttrs(attrs),
	}
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
func (t *testHandler) WithGroup(name string) slog.Handler {
	return &testHandler{
		prefix:  t.prefix,
		verbose: t.verbose,
		w:       t.w,
		h:       t.h.WithGroup(name),
	}
}
