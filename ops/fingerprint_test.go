package ops

import (
	"errors"
	"testing"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/bytestream"
)

// TestCatchStreamError covers what the recovery does and does not swallow. Only
// an out-of-bounds move is turned into an error: anything else is a fault in
// the operation rather than in the data it was given, and must not be reported
// as though the input were at fault.
func TestCatchStreamError(t *testing.T) {
	t.Run("passes a result through", func(t *testing.T) {
		want := core.NewDish([]byte("ok"), core.TypeString)
		got, err := catchStreamError(func() (*core.Dish, error) { return want, nil })
		if err != nil || got != want {
			t.Errorf("got (%v, %v), want (%v, nil)", got, err, want)
		}
	})

	t.Run("passes an error through", func(t *testing.T) {
		want := errors.New("no")
		got, err := catchStreamError(func() (*core.Dish, error) { return nil, want })
		if !errors.Is(err, want) || got != nil {
			t.Errorf("got (%v, %v), want (nil, %v)", got, err, want)
		}
	})

	t.Run("returns an out-of-bounds move", func(t *testing.T) {
		got, err := catchStreamError(func() (*core.Dish, error) {
			panic(bytestream.StreamError{Pos: 7})
		})
		if got != nil {
			t.Errorf("got %v, want no output", got)
		}
		if want := "Cannot move to position 7 in stream. Out of bounds."; err == nil || err.Error() != want {
			t.Errorf("error = %v, want %q", err, want)
		}
	})

	t.Run("re-raises another error", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("an unrelated error was swallowed")
			}
		}()
		_, _ = catchStreamError(func() (*core.Dish, error) { panic(errors.New("something else")) })
	})

	t.Run("re-raises a value that is not an error", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Error("a non-error panic was swallowed")
			}
		}()
		_, _ = catchStreamError(func() (*core.Dish, error) { panic("not an error") })
	})
}
