package ui

import (
	"bytes"
	"sync"
)

// safeBuffer is a goroutine-safe bytes.Buffer used to capture subprocess output
// that is written from one goroutine and read from the UI goroutine.
type safeBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *safeBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *safeBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}
