package queue

import (
	"context"
	"sync"
	"time"
)

// LogEntry represents a queued log entry.
type LogEntry struct {
	ID           string
	Message      string
	Timestamp    time.Time
	Level        int
	EntryType    int
	PayloadType  int
	Node         string
	Labels       map[string]string
	SearchTokens []string
	Retries      int
	CreatedAt    time.Time
}

// Queue is a thread-safe in-memory FIFO queue.
type Queue struct {
	items    []LogEntry
	mu       sync.RWMutex
	maxSize  int
	notEmpty chan struct{}
	closeCh  chan struct{} // closed exactly once to signal shutdown
	closed   bool
}

func NewQueue(maxSize int) *Queue {
	if maxSize <= 0 {
		maxSize = 1000
	}
	return &Queue{
		items:    make([]LogEntry, 0, maxSize),
		maxSize:  maxSize,
		notEmpty: make(chan struct{}, 1),
		closeCh:  make(chan struct{}),
	}
}

// Enqueue adds an entry. Returns false if full or closed (caller should count as overflow).
func (q *Queue) Enqueue(entry LogEntry) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed || len(q.items) >= q.maxSize {
		return false
	}
	q.items = append(q.items, entry)
	select {
	case q.notEmpty <- struct{}{}:
	default:
	}
	return true
}

// Dequeue removes and returns the oldest entry, or nil if empty.
func (q *Queue) Dequeue() *LogEntry {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil
	}
	entry := q.items[0]
	q.items[0] = LogEntry{} // clear reference to allow GC
	q.items = q.items[1:]
	q.compactLocked()
	return &entry
}

// DequeueBatch removes up to n entries and returns them.
func (q *Queue) DequeueBatch(n int) []LogEntry {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil
	}
	if n > len(q.items) {
		n = len(q.items)
	}
	batch := make([]LogEntry, n)
	copy(batch, q.items[:n])
	// Clear references for GC
	for i := 0; i < n; i++ {
		q.items[i] = LogEntry{}
	}
	q.items = q.items[n:]
	q.compactLocked()
	return batch
}

// compactLocked copies items to a fresh slice when capacity waste exceeds 2x.
// Must be called with q.mu held.
func (q *Queue) compactLocked() {
	if cap(q.items) > q.maxSize*2 && len(q.items) < cap(q.items)/4 {
		compacted := make([]LogEntry, len(q.items), q.maxSize)
		copy(compacted, q.items)
		q.items = compacted
	}
}

// DequeueWithContext blocks until an entry is available or ctx is cancelled.
// Returns nil when the context is cancelled or the queue is closed and empty.
func (q *Queue) DequeueWithContext(ctx context.Context) *LogEntry {
	for {
		if entry := q.Dequeue(); entry != nil {
			return entry
		}
		// Check if closed and empty
		q.mu.RLock()
		closed := q.closed
		empty := len(q.items) == 0
		q.mu.RUnlock()
		if closed && empty {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-q.closeCh:
			// Queue was closed; drain remaining items
			if entry := q.Dequeue(); entry != nil {
				return entry
			}
			return nil
		case <-q.notEmpty:
			continue
		}
	}
}

func (q *Queue) Size() int {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.items)
}

func (q *Queue) IsFull() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.items) >= q.maxSize
}

func (q *Queue) IsEmpty() bool {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return len(q.items) == 0
}

func (q *Queue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = q.items[:0]
}

func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.closed {
		return
	}
	q.closed = true
	close(q.closeCh)
}

func (q *Queue) GetItems() []LogEntry {
	q.mu.RLock()
	defer q.mu.RUnlock()
	items := make([]LogEntry, len(q.items))
	copy(items, q.items)
	return items
}
