package store

import "testing"

func TestDefaultsForItem(t *testing.T) {
	it := Item{}
	defaultsForItem(&it)
	if it.Kind != "milestone" {
		t.Errorf("Kind default = %q, want milestone", it.Kind)
	}
	if it.Marker != "l:Diamond" {
		t.Errorf("Marker default = %q, want l:Diamond", it.Marker)
	}

	custom := Item{Kind: "event", Marker: "bar"}
	defaultsForItem(&custom)
	if custom.Kind != "event" || custom.Marker != "bar" {
		t.Errorf("defaults overwrote provided values: %+v", custom)
	}
}

func TestToDate(t *testing.T) {
	if v, err := toDate(nil); err != nil || v != nil {
		t.Errorf("toDate(nil) = (%v, %v), want (nil, nil)", v, err)
	}
	empty := ""
	if v, err := toDate(&empty); err != nil || v != nil {
		t.Errorf(`toDate("") = (%v, %v), want (nil, nil)`, v, err)
	}
	good := "2026-04-10"
	if v, err := toDate(&good); err != nil || v == nil {
		t.Errorf("toDate(%q) failed: (%v, %v)", good, v, err)
	}
	bad := "10.04.2026"
	if _, err := toDate(&bad); err == nil {
		t.Errorf("toDate(%q) should have errored", bad)
	}
}
