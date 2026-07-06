package ops

import "testing"

// The jsSlice/jsSliceNeg helpers port JavaScript's String.prototype.slice bound
// handling (clamping instead of panicking); jsFloatString ports Number.toString.
// Their edge branches are reached only by defensive callers, so they are unit
// tested directly against the JavaScript semantics they mirror.

func TestJSSlice(t *testing.T) {
	cases := []struct {
		name       string
		s          string
		start, end int
		want       string
	}{
		{"in bounds", "hello", 1, 3, "el"},
		{"start below zero clamps to 0", "hello", -2, 3, "hel"},
		{"start past end clamps to len", "hello", 10, 12, ""},
		{"end past len clamps to len", "hello", 2, 10, "llo"},
		{"end before start collapses", "hello", 4, 2, ""},
		{"empty string", "", 0, 5, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := jsSlice(c.s, c.start, c.end); got != c.want {
				t.Fatalf("jsSlice(%q,%d,%d) = %q, want %q", c.s, c.start, c.end, got, c.want)
			}
		})
	}
}

func TestJSSliceNeg(t *testing.T) {
	cases := []struct {
		name       string
		s          string
		start, end int
		want       string
	}{
		{"positive in bounds", "hello", 1, 3, "el"},
		{"negative start and end", "hello", -3, -1, "ll"},
		{"negative start past front clamps to 0", "hello", -100, 2, "he"},
		{"positive start with negative end", "hello", 2, -1, "ll"},
		{"start past len clamps to len", "hello", 100, 100, ""},
		{"end past len clamps to len", "hello", 1, 100, "ello"},
		{"end before start collapses", "hello", 3, 1, ""},
		{"negative end past front collapses", "hello", 0, -100, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := jsSliceNeg(c.s, c.start, c.end); got != c.want {
				t.Fatalf("jsSliceNeg(%q,%d,%d) = %q, want %q", c.s, c.start, c.end, got, c.want)
			}
		})
	}
}

func TestJSFloatString(t *testing.T) {
	cases := []struct {
		name string
		f    float64
		want string
	}{
		{"small integer", 42, "42"},
		{"large integer below 1e21", 1e20, "100000000000000000000"},
		{"non-integer uses shortest form", 42.5, "42.5"},
		{"at 1e21 switches to exponential", 1e21, "1e+21"},
		{"above 1e21 exponential", 1.5e21, "1.5e+21"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := jsFloatString(c.f); got != c.want {
				t.Fatalf("jsFloatString(%v) = %q, want %q", c.f, got, c.want)
			}
		})
	}
}
