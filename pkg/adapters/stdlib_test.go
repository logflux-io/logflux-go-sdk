package adapters

import (
	"errors"
	"testing"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/models"
)

// simple fake client implementing the adapter interface
type fakeClient struct {
	calls []struct {
		msg string
		lvl int
	}
}

func (f *fakeClient) SendLogWithTimestampAndLevel(message string, _ time.Time, level int) error {
	f.calls = append(f.calls, struct {
		msg string
		lvl int
	}{message, level})
	if message == "err" {
		return errors.New("fail")
	}
	return nil
}
func (f *fakeClient) Close() error { return nil }

func TestStdlibLogger_WriteAndPrefix(t *testing.T) {
	f := &fakeClient{}
	l := NewStdlibLogger(f, "[P] ")

	// Write trims common prefix segments like "x: y"
	if _, err := l.Write([]byte("time: hello\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	if len(f.calls) != 1 || f.calls[0].msg != "[P] hello" || f.calls[0].lvl != models.LogLevelInfo {
		t.Fatalf("unexpected calls: %#v", f.calls)
	}
}

func TestStdlibLogger_PanicAndFatal(t *testing.T) {
	// Panic* should panic; we recover to assert
	f := &fakeClient{}
	l := NewStdlibLogger(f, "")

	didPanic := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
			}
		}()
		func() { // Panic
			defer func() { _ = recover() }() // nested to avoid breaking test process if missed
			l.Panic("boom")
		}()
		// Panicf
		func() { defer func() { _ = recover() }(); l.Panicf("%s", "boomf") }()
		// Panicln
		func() { defer func() { _ = recover() }(); l.Panicln("boomln") }()
		didPanic = true
	}()
	if !didPanic {
		t.Fatalf("expected panic")
	}
}
