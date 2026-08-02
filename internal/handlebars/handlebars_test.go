package handlebars

import "testing"

// TestHandlebarsFormatFallback covers a value of a kind JSON cannot produce,
// which is written as Go would print it.
func TestHandlebarsFormatFallback(t *testing.T) {
	if got := hbFormat(42, false); got != "42" {
		t.Errorf("got %q, want %q", got, "42")
	}
	if got := hbTruthy(42); !got {
		t.Error("a value of an unexpected kind was treated as absent")
	}
}
