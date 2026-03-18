package payload

import "testing"

func TestBreadcrumbRing_Basic(t *testing.T) {
	ring := NewBreadcrumbRing(5)

	ring.Add(Breadcrumb{Message: "a"})
	ring.Add(Breadcrumb{Message: "b"})
	ring.Add(Breadcrumb{Message: "c"})

	snap := ring.Snapshot()
	if len(snap) != 3 {
		t.Fatalf("expected 3, got %d", len(snap))
	}
	if snap[0].Message != "a" || snap[2].Message != "c" {
		t.Error("breadcrumbs not in chronological order")
	}
}

func TestBreadcrumbRing_Overflow(t *testing.T) {
	ring := NewBreadcrumbRing(3)

	ring.Add(Breadcrumb{Message: "a"})
	ring.Add(Breadcrumb{Message: "b"})
	ring.Add(Breadcrumb{Message: "c"})
	ring.Add(Breadcrumb{Message: "d"}) // overwrites "a"
	ring.Add(Breadcrumb{Message: "e"}) // overwrites "b"

	snap := ring.Snapshot()
	if len(snap) != 3 {
		t.Fatalf("expected 3, got %d", len(snap))
	}
	// Oldest should be "c", newest "e"
	if snap[0].Message != "c" {
		t.Errorf("expected oldest=c, got %s", snap[0].Message)
	}
	if snap[2].Message != "e" {
		t.Errorf("expected newest=e, got %s", snap[2].Message)
	}
}

func TestBreadcrumbRing_Clear(t *testing.T) {
	ring := NewBreadcrumbRing(10)
	ring.Add(Breadcrumb{Message: "a"})
	ring.Add(Breadcrumb{Message: "b"})
	ring.Clear()

	if ring.Size() != 0 {
		t.Errorf("expected empty after clear, got %d", ring.Size())
	}
	snap := ring.Snapshot()
	if snap != nil {
		t.Errorf("expected nil snapshot after clear, got %v", snap)
	}
}

func TestBreadcrumbRing_Empty(t *testing.T) {
	ring := NewBreadcrumbRing(10)
	snap := ring.Snapshot()
	if snap != nil {
		t.Errorf("expected nil for empty ring, got %v", snap)
	}
}

func TestBreadcrumbRing_AutoTimestamp(t *testing.T) {
	ring := NewBreadcrumbRing(10)
	ring.Add(Breadcrumb{Message: "test"})

	snap := ring.Snapshot()
	if snap[0].Timestamp == "" {
		t.Error("expected auto-filled timestamp")
	}
}
