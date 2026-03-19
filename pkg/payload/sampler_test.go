package payload

import "testing"

func TestSampler_AlwaysSend(t *testing.T) {
	s := NewSampler(1.0)
	for i := 0; i < 100; i++ {
		if !s.ShouldSample() {
			t.Fatal("rate 1.0 should always sample")
		}
	}
}

func TestSampler_NeverSend(t *testing.T) {
	s := NewSampler(0.0)
	for i := 0; i < 100; i++ {
		if s.ShouldSample() {
			t.Fatal("rate 0.0 should never sample")
		}
	}
}

func TestSampler_HalfRate(t *testing.T) {
	s := NewSampler(0.5)
	sampled := 0
	total := 10000
	for i := 0; i < total; i++ {
		if s.ShouldSample() {
			sampled++
		}
	}
	// Expect roughly 50% ± 5%
	rate := float64(sampled) / float64(total)
	if rate < 0.45 || rate > 0.55 {
		t.Errorf("expected ~50%% sample rate, got %.1f%% (%d/%d)", rate*100, sampled, total)
	}
}

func TestSampler_ClampNegative(t *testing.T) {
	s := NewSampler(-0.5)
	if s.Rate() != 0 {
		t.Errorf("expected clamped to 0, got %f", s.Rate())
	}
}

func TestSampler_ClampOver1(t *testing.T) {
	s := NewSampler(2.0)
	if s.Rate() != 1.0 {
		t.Errorf("expected clamped to 1.0, got %f", s.Rate())
	}
}
