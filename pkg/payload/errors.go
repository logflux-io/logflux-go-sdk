package payload

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// ErrorPayload extends Log with error-specific fields (stack trace, breadcrumbs).
type ErrorPayload struct {
	common
	Message       string        `json:"message"`
	Logger        string        `json:"logger,omitempty"`
	ErrorType     string        `json:"error_type,omitempty"`
	ErrorChain    []ChainedError `json:"error_chain,omitempty"`
	StackTrace    []StackFrame  `json:"stack_trace,omitempty"`
	Breadcrumbs   []Breadcrumb  `json:"breadcrumbs,omitempty"`
}

// ChainedError represents one error in an unwrapped chain.
type ChainedError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// StackFrame represents a single frame in a stack trace.
type StackFrame struct {
	Function string `json:"function"`
	File     string `json:"file"`
	Line     int    `json:"line"`
}

// NewErrorPayload creates an error payload from a Go error with auto stack trace
// and error chain unwrapping (follows errors.Unwrap up to 10 levels).
func NewErrorPayload(source string, err error) *ErrorPayload {
	p := &ErrorPayload{
		common:    newCommon("log", source, 4), // error level
		Message:   err.Error(),
		ErrorType: errorTypeName(err),
	}
	p.ErrorChain = unwrapErrorChain(err)
	p.StackTrace = captureStackTrace(3) // skip captureStackTrace, NewErrorPayload, caller
	return p
}

// NewErrorPayloadWithMessage creates an error payload with a custom message.
func NewErrorPayloadWithMessage(source string, err error, message string) *ErrorPayload {
	p := &ErrorPayload{
		common:    newCommon("log", source, 4),
		Message:   message,
		ErrorType: errorTypeName(err),
	}
	if err != nil {
		if p.Attributes == nil {
			p.Attributes = make(map[string]string)
		}
		p.Attributes["error"] = err.Error()
	}
	p.StackTrace = captureStackTrace(3)
	return p
}

// WithBreadcrumbs attaches breadcrumbs from a ring buffer.
func (p *ErrorPayload) WithBreadcrumbs(ring *BreadcrumbRing) *ErrorPayload {
	if ring != nil {
		p.Breadcrumbs = ring.Snapshot()
	}
	return p
}

// captureStackTrace captures the current goroutine's stack trace.
func captureStackTrace(skip int) []StackFrame {
	var frames []StackFrame
	pcs := make([]uintptr, 32)
	n := runtime.Callers(skip, pcs)
	if n == 0 {
		return nil
	}

	runtimeFrames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := runtimeFrames.Next()
		// Skip runtime internals
		if strings.HasPrefix(frame.Function, "runtime.") {
			if !more {
				break
			}
			continue
		}
		frames = append(frames, StackFrame{
			Function: frame.Function,
			File:     frame.File,
			Line:     frame.Line,
		})
		if !more || len(frames) >= 20 {
			break
		}
	}
	return frames
}

// unwrapErrorChain follows errors.Unwrap() up to 10 levels deep.
// Only populated if the chain has more than one error.
func unwrapErrorChain(err error) []ChainedError {
	if err == nil {
		return nil
	}
	var chain []ChainedError
	current := err
	for i := 0; i < 10 && current != nil; i++ {
		chain = append(chain, ChainedError{
			Type:    errorTypeName(current),
			Message: current.Error(),
		})
		current = errors.Unwrap(current)
	}
	// Only include chain if there's more than one error (otherwise redundant with top-level fields)
	if len(chain) <= 1 {
		return nil
	}
	return chain
}

// errorTypeName extracts a type name from an error.
func errorTypeName(err error) string {
	if err == nil {
		return ""
	}
	t := fmt.Sprintf("%T", err)
	t = strings.TrimPrefix(t, "*")
	return t
}
