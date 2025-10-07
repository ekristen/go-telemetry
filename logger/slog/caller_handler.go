package slog

import (
	"context"
	"log/slog"
	"runtime"
)

// CallerHandler is a slog handler that adjusts the source caller information
// to skip wrapper frames when AddSource is enabled.
type CallerHandler struct {
	base slog.Handler
	skip int // exported for use in UpdateLoggerProvider
}

// NewCallerHandler creates a new handler that adjusts caller skip frames.
// The skip parameter specifies how many additional frames to skip beyond
// the slog internals.
func NewCallerHandler(base slog.Handler, skip int) *CallerHandler {
	return &CallerHandler{
		base: base,
		skip: skip,
	}
}

// Enabled reports whether the handler handles records at the given level.
func (h *CallerHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.base.Enabled(ctx, level)
}

// Handle handles the Record with adjusted source information.
func (h *CallerHandler) Handle(ctx context.Context, record slog.Record) error {
	// If the record has source information, we need to adjust it
	// by recalculating with the correct skip count
	if record.PC != 0 {
		// Get the correct caller by skipping additional frames
		// skip + 4 accounts for: runtime.Callers, this Handle method,
		// the slog.Logger method, and our Event wrapper
		var pcs [1]uintptr
		runtime.Callers(h.skip+4, pcs[:])

		// Create a new record with the corrected PC
		newRecord := slog.NewRecord(record.Time, record.Level, record.Message, pcs[0])

		// Copy all attributes from the original record
		record.Attrs(func(attr slog.Attr) bool {
			newRecord.AddAttrs(attr)
			return true
		})

		return h.base.Handle(ctx, newRecord)
	}

	// If no source info, just pass through
	return h.base.Handle(ctx, record)
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
func (h *CallerHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &CallerHandler{
		base: h.base.WithAttrs(attrs),
		skip: h.skip,
	}
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
func (h *CallerHandler) WithGroup(name string) slog.Handler {
	return &CallerHandler{
		base: h.base.WithGroup(name),
		skip: h.skip,
	}
}
