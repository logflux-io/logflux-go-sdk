package payload

import (
	"encoding/json"
	"testing"
)

func TestConfigure_AutoFillsSource(t *testing.T) {
	Configure("my-service", "production", "v1.2.3")
	defer Configure("", "", "")

	p := NewLog("", "test message", 7)
	ApplyContext(p)

	if p.Source != "my-service" {
		t.Errorf("expected source=my-service, got %q", p.Source)
	}
}

func TestConfigure_AutoFillsMeta(t *testing.T) {
	Configure("svc", "staging", "v2.0.0")
	defer Configure("", "", "")

	p := NewEvent("", "user.signup")
	ApplyContext(p)

	if p.Meta["environment"] != "staging" {
		t.Errorf("expected environment=staging, got %q", p.Meta["environment"])
	}
	if p.Meta["release"] != "v2.0.0" {
		t.Errorf("expected release=v2.0.0, got %q", p.Meta["release"])
	}
}

func TestConfigure_DoesNotOverwriteExplicit(t *testing.T) {
	Configure("default-svc", "prod", "v1.0")
	defer Configure("", "", "")

	p := NewLog("explicit-svc", "test", 7)
	ApplyContext(p)

	if p.Source != "explicit-svc" {
		t.Errorf("expected explicit source preserved, got %q", p.Source)
	}
}

func TestConfigure_DoesNotOverwriteExplicitMeta(t *testing.T) {
	Configure("svc", "prod", "v1.0")
	defer Configure("", "", "")

	p := NewLog("", "test", 7)
	p.SetMeta(map[string]string{"environment": "custom"})
	ApplyContext(p)

	if p.Meta["environment"] != "custom" {
		t.Errorf("expected custom meta preserved, got %q", p.Meta["environment"])
	}
	// release should still be filled
	if p.Meta["release"] != "v1.0" {
		t.Errorf("expected release filled, got %q", p.Meta["release"])
	}
}

func TestApplyContext_FullSerialization(t *testing.T) {
	Configure("billing-api", "production", "v3.1.0")
	defer Configure("", "", "")

	p := NewAudit("", "record.deleted", "usr_1", "invoice", "inv_2")
	ApplyContext(p)
	p.SetAttributes(map[string]string{"ip": "10.0.1.1"})

	data, err := Marshal(p)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]interface{}
	json.Unmarshal(data, &m)

	if m["source"] != "billing-api" {
		t.Errorf("expected source=billing-api, got %v", m["source"])
	}
	meta := m["meta"].(map[string]interface{})
	if meta["environment"] != "production" {
		t.Errorf("expected environment=production, got %v", meta["environment"])
	}
	if meta["release"] != "v3.1.0" {
		t.Errorf("expected release=v3.1.0, got %v", meta["release"])
	}
}
