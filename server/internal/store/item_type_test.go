package store

import "testing"

// The built-in catalog must cover the legacy kinds 1:1 (so migration 00019 maps
// every existing item to a real type) and bind each to a coded family.
func TestDefaultItemTypes(t *testing.T) {
	byKey := map[string]ItemType{}
	for _, it := range DefaultItemTypes() {
		if !it.Builtin {
			t.Errorf("%s should be builtin", it.Key)
		}
		byKey[it.Key] = it
	}
	want := map[string]string{
		"milestone": FamilyTimelinePoint,
		"event":     FamilyTimelineRange,
		"point":     FamilyTimelinePoint,
	}
	for key, fam := range want {
		it, ok := byKey[key]
		if !ok {
			t.Fatalf("missing built-in type %q", key)
		}
		if it.Family != fam {
			t.Errorf("%s: family=%q want %q", key, it.Family, fam)
		}
		if it.Label == "" || it.Icon == "" {
			t.Errorf("%s: label/icon must be set", key)
		}
	}
}
