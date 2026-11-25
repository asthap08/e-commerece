package catalog

import "testing"

func TestSlugifyAndEscape(t *testing.T) {
	if got := Slugify("  Wireless Earbuds Pro (2026)! "); got != "wireless-earbuds-pro-2026" {
		t.Fatalf("slugify = %q", got)
	}
	if got := escapeLike(`50%_off\`); got != `50\%\_off\\` {
		t.Fatalf("escapeLike = %q", got)
	}
}
