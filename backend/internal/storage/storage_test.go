package storage

import "testing"

func TestNormalizedKeyUsesUnicodeCaseFoldingAfterNFC(t *testing.T) {
	for _, pair := range [][2]string{
		{"Straße", "STRASSE"},
		{"Й", "Й"},
	} {
		if left, right := normalizedKey(pair[0]), normalizedKey(pair[1]); left != right {
			t.Errorf("normalizedKey(%q) = %q, normalizedKey(%q) = %q", pair[0], left, pair[1], right)
		}
	}
}
