package payload

import (
	"math/rand"
	"sync"
	"time"
)

// Sampler implements probabilistic sampling.
// SampleRate 1.0 = send everything, 0.0 = drop everything, 0.5 = 50%.
type Sampler struct {
	rate float64
	rng  *rand.Rand
	mu   sync.Mutex
}

// NewSampler creates a sampler. Rate is clamped to [0.0, 1.0].
func NewSampler(rate float64) *Sampler {
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	return &Sampler{
		rate: rate,
		rng:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ShouldSample returns true if this entry should be sent.
func (s *Sampler) ShouldSample() bool {
	if s.rate >= 1.0 {
		return true
	}
	if s.rate <= 0.0 {
		return false
	}
	s.mu.Lock()
	v := s.rng.Float64()
	s.mu.Unlock()
	return v < s.rate
}

// Rate returns the configured sample rate.
func (s *Sampler) Rate() float64 {
	return s.rate
}
