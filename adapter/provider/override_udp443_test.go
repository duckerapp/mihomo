package provider

import (
	"testing"

	"github.com/metacubex/mihomo/common/structure"
)

// TestOverrideUdp443 checks that `udp-443` is part of
// the provider override whitelist and are put into the proxy mapping before
// parse, so adapter.ParseProxy applies them.
func TestOverrideUdp443(t *testing.T) {
	decoder := structure.NewDecoder(structure.Option{TagName: "provider", WeaklyTypedInput: true})
	var schema overrideSchema
	err := decoder.Decode(map[string]any{
		"udp-443": "reject",
	}, &schema)
	if err != nil {
		t.Fatal(err)
	}

	mapping := map[string]any{"name": "p"}
	if err := schema.Apply(mapping); err != nil {
		t.Fatal(err)
	}
	if mapping["udp-443"] != "reject" {
		t.Fatalf("udp-443 override not applied: %#v", mapping["udp-443"])
	}
}

// An override without udp-443 adds nothing, so a proxy's own setting stays.
func TestOverrideUdp443Unset(t *testing.T) {
	var schema overrideSchema
	mapping := map[string]any{"name": "p"}
	if err := schema.Apply(mapping); err != nil {
		t.Fatal(err)
	}
	if _, exist := mapping["udp-443"]; exist {
		t.Fatalf("unset udp-443 must not appear in mapping: %#v", mapping["udp-443"])
	}
}
