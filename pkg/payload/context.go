package payload

import "sync"

// GlobalContext holds SDK-wide context that gets auto-attached to all payloads.
type GlobalContext struct {
	mu          sync.RWMutex
	source      string
	environment string
	release     string
	defaultMeta map[string]string
}

var globalCtx = &GlobalContext{}

// Configure sets the global context. Called by logflux.Init().
func Configure(source, environment, release string) {
	globalCtx.mu.Lock()
	defer globalCtx.mu.Unlock()
	globalCtx.source = source
	globalCtx.environment = environment
	globalCtx.release = release
	globalCtx.defaultMeta = nil
	if environment != "" || release != "" {
		globalCtx.defaultMeta = make(map[string]string)
		if environment != "" {
			globalCtx.defaultMeta["environment"] = environment
		}
		if release != "" {
			globalCtx.defaultMeta["release"] = release
		}
	}
}

// GetSource returns the configured source (for use in payload constructors).
func GetSource() string {
	globalCtx.mu.RLock()
	defer globalCtx.mu.RUnlock()
	return globalCtx.source
}

// ApplyContext fills in source and meta from global context if not already set.
func ApplyContext(c interface {
	SetSource(string)
	SetMeta(map[string]string)
	getSource() string
	getMeta() map[string]string
}) {
	globalCtx.mu.RLock()
	defer globalCtx.mu.RUnlock()

	if c.getSource() == "" && globalCtx.source != "" {
		c.SetSource(globalCtx.source)
	}
	if globalCtx.defaultMeta != nil {
		existing := c.getMeta()
		if existing == nil {
			merged := make(map[string]string, len(globalCtx.defaultMeta))
			for k, v := range globalCtx.defaultMeta {
				merged[k] = v
			}
			c.SetMeta(merged)
		} else {
			// Don't overwrite user-set meta keys
			for k, v := range globalCtx.defaultMeta {
				if _, exists := existing[k]; !exists {
					existing[k] = v
				}
			}
		}
	}
}

// Expose getters on common for the ApplyContext interface.
func (c *common) getSource() string            { return c.Source }
func (c *common) getMeta() map[string]string    { return c.Meta }
