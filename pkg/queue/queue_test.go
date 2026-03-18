package queue

import (
	"context"
	"testing"
	"time"
)

func TestQueue_EnqueueDequeue(t *testing.T) {
	q := NewQueue(2)
	ts := time.Now()
	ok := q.Enqueue(LogEntry{ID: "1", Message: "a", Timestamp: ts})
	if !ok {
		t.Fatalf("enqueue failed")
	}
	if q.Size() != 1 || q.IsEmpty() {
		t.Fatalf("size/empty wrong: size %d empty %v", q.Size(), q.IsEmpty())
	}

	e := q.Dequeue()
	if e == nil || e.ID != "1" || e.Message != "a" {
		t.Fatalf("unexpected dequeue: %#v", e)
	}
	if !q.IsEmpty() {
		t.Fatalf("expected empty after dequeue")
	}
}

func TestQueue_FullAndClose(t *testing.T) {
	q := NewQueue(1)
	if !q.Enqueue(LogEntry{ID: "1"}) {
		t.Fatalf("first enqueue should succeed")
	}
	if q.Enqueue(LogEntry{ID: "2"}) {
		t.Fatalf("second enqueue should fail when full")
	}

	q.Close()
	if q.Enqueue(LogEntry{ID: "3"}) {
		t.Fatalf("enqueue should fail after close")
	}
}

func TestQueue_DequeueWithContext(t *testing.T) {
	q := NewQueue(1)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// nothing queued, should return nil on timeout/cancel
	e := q.DequeueWithContext(ctx)
	if e != nil {
		t.Fatalf("expected nil when context cancelled")
	}

	// Now enqueue and expect it to return the item promptly
	ctx2, cancel2 := context.WithTimeout(context.Background(), time.Second)
	defer cancel2()

	go func() {
		time.Sleep(20 * time.Millisecond)
		_ = q.Enqueue(LogEntry{ID: "x"})
	}()

	e2 := q.DequeueWithContext(ctx2)
	if e2 == nil || e2.ID != "x" {
		t.Fatalf("expected queued item, got %#v", e2)
	}
}
