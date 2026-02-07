package events

import (
	"context"
	"io"
	"log/slog"
	"sync"
)

// BroadcastHandler is an slog.Handler that writes to both an underlying handler
// and broadcasts log records to all registered listeners.
type BroadcastHandler struct {
	underlying slog.Handler
	levelVar   *slog.LevelVar
	mu         sync.RWMutex
	listeners  []chan string
}

// NewBroadcastHandler creates a handler that broadcasts to listeners while also
// delegating to the underlying handler. Note: levelVar will be nil, so SetLogLevel won't work.
func NewBroadcastHandler(underlying slog.Handler) *BroadcastHandler {
	return &BroadcastHandler{
		underlying: underlying,
		levelVar:   nil,
		listeners:  make([]chan string, 0),
	}
}

func (h *BroadcastHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.underlying.Enabled(ctx, level)
}

func (h *BroadcastHandler) Handle(ctx context.Context, r slog.Record) error {
	// Write to underlying handler first
	if err := h.underlying.Handle(ctx, r); err != nil {
		return err
	}

	// Format the log message for broadcast
	msg := h.formatRecord(r)

	// Broadcast to all listeners (non-blocking)
	h.mu.RLock()
	for _, ch := range h.listeners {
		select {
		case ch <- msg:
		default:
			// Drop if channel is full
		}
	}
	h.mu.RUnlock()

	return nil
}

func (h *BroadcastHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &BroadcastHandler{
		underlying: h.underlying.WithAttrs(attrs),
		levelVar:   h.levelVar,
		listeners:  h.listeners,
		mu:         h.mu,
	}
}

func (h *BroadcastHandler) WithGroup(name string) slog.Handler {
	return &BroadcastHandler{
		underlying: h.underlying.WithGroup(name),
		levelVar:   h.levelVar,
		listeners:  h.listeners,
		mu:         h.mu,
	}
}

// Subscribe adds a new listener channel for log broadcasts.
func (h *BroadcastHandler) Subscribe() chan string {
	ch := make(chan string, 100)
	h.mu.Lock()
	h.listeners = append(h.listeners, ch)
	h.mu.Unlock()
	return ch
}

// Unsubscribe removes a listener channel.
func (h *BroadcastHandler) Unsubscribe(ch chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, l := range h.listeners {
		if l == ch {
			h.listeners = append(h.listeners[:i], h.listeners[i+1:]...)
			close(ch)
			return
		}
	}
}

// formatRecord formats a log record as a simple text string.
func (h *BroadcastHandler) formatRecord(r slog.Record) string {
	// Simple format: time level message key=value ...
	var buf []byte
	buf = append(buf, r.Time.Format("2006-01-02 15:04:05")...)
	buf = append(buf, ' ')
	buf = append(buf, r.Level.String()...)
	buf = append(buf, ' ')
	buf = append(buf, r.Message...)
	r.Attrs(func(a slog.Attr) bool {
		buf = append(buf, ' ')
		buf = append(buf, a.Key...)
		buf = append(buf, '=')
		buf = append(buf, a.Value.String()...)
		return true
	})
	return string(buf)
}

// SetLogLevel updates the log level dynamically.
func (h *BroadcastHandler) SetLogLevel(logLevel string) {
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}
	if h.levelVar != nil {
		h.levelVar.Set(level)
	}
}

// NewBroadcastLogger creates a logger with broadcast capability.
func NewBroadcastLogger(logLevel string, w io.Writer) (*slog.Logger, *BroadcastHandler) {
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn", "warning":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	levelVar := &slog.LevelVar{}
	levelVar.Set(level)
	opts := &slog.HandlerOptions{Level: levelVar}
	underlying := slog.NewTextHandler(w, opts)
	broadcast := &BroadcastHandler{
		underlying: underlying,
		levelVar:   levelVar,
		listeners:  make([]chan string, 0),
	}
	return slog.New(broadcast), broadcast
}
