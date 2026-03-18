package payload

import (
	"sync"
	"time"
)

// Breadcrumb is a single entry in the breadcrumb trail.
type Breadcrumb struct {
	Timestamp string            `json:"timestamp"`
	Category  string            `json:"category,omitempty"`
	Message   string            `json:"message"`
	Level     string            `json:"level,omitempty"`
	Data      map[string]string `json:"data,omitempty"`
}

// BreadcrumbRing is a thread-safe ring buffer of breadcrumbs.
type BreadcrumbRing struct {
	mu       sync.Mutex
	items    []Breadcrumb
	maxSize  int
	position int
	full     bool
}

// NewBreadcrumbRing creates a ring buffer with the given capacity.
func NewBreadcrumbRing(maxSize int) *BreadcrumbRing {
	if maxSize <= 0 {
		maxSize = 100
	}
	return &BreadcrumbRing{
		items:   make([]Breadcrumb, maxSize),
		maxSize: maxSize,
	}
}

// Add adds a breadcrumb to the ring buffer.
func (r *BreadcrumbRing) Add(b Breadcrumb) {
	if b.Timestamp == "" {
		b.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
	}
	r.mu.Lock()
	r.items[r.position] = b
	r.position = (r.position + 1) % r.maxSize
	if r.position == 0 {
		r.full = true
	}
	r.mu.Unlock()
}

// Snapshot returns a chronological copy of all breadcrumbs.
func (r *BreadcrumbRing) Snapshot() []Breadcrumb {
	r.mu.Lock()
	defer r.mu.Unlock()

	var count int
	if r.full {
		count = r.maxSize
	} else {
		count = r.position
	}
	if count == 0 {
		return nil
	}

	result := make([]Breadcrumb, count)
	if r.full {
		// Oldest entries start at r.position (wrapped around)
		n := copy(result, r.items[r.position:])
		copy(result[n:], r.items[:r.position])
	} else {
		copy(result, r.items[:r.position])
	}
	return result
}

// Clear removes all breadcrumbs.
func (r *BreadcrumbRing) Clear() {
	r.mu.Lock()
	r.position = 0
	r.full = false
	r.mu.Unlock()
}

// Size returns the current number of breadcrumbs.
func (r *BreadcrumbRing) Size() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.full {
		return r.maxSize
	}
	return r.position
}
