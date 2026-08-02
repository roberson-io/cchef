package qr

import "testing"

// qrTestMatrix builds a matrix from rows of characters, where a hash is a dark
// module and anything else is light.
func qrTestMatrix(rows ...string) [][]byte {
	matrix := make([][]byte, len(rows))
	for i, row := range rows {
		matrix[i] = make([]byte, len(row))
		for j := range row {
			if row[j] == '#' {
				matrix[i][j] = 1
			}
		}
	}
	return matrix
}

// TestQRPenaltyRules covers each of the four rules on its own, against counts
// worked out by hand rather than taken from the implementation.
func TestQRPenaltyRules(t *testing.T) {
	t.Run("runs of five or more score their length less two", func(t *testing.T) {
		// Five rows and five columns of five, each scoring three.
		dark := qrTestMatrix("#####", "#####", "#####", "#####", "#####")
		if got := qrRunPenalty(dark); got != 30 {
			t.Errorf("a wholly dark code scored %d, want 30", got)
		}
		// Light runs score the same as dark ones.
		light := qrTestMatrix(".....", ".....", ".....", ".....", ".....")
		if got := qrRunPenalty(light); got != 30 {
			t.Errorf("a wholly light code scored %d, want 30", got)
		}
		// Runs of four score nothing at all.
		short := qrTestMatrix("....", "....", "....", "....")
		if got := qrRunPenalty(short); got != 0 {
			t.Errorf("runs of four scored %d, want 0", got)
		}
	})

	t.Run("blocks of four alike score three", func(t *testing.T) {
		// A two by two square holds one block.
		one := qrTestMatrix("##", "##")
		if got := qrBlockPenaltyOf(one); got != 3 {
			t.Errorf("one dark block scored %d, want 3", got)
		}
		// A three by three square holds four overlapping blocks.
		four := qrTestMatrix("###", "###", "###")
		if got := qrBlockPenaltyOf(four); got != 12 {
			t.Errorf("four dark blocks scored %d, want 12", got)
		}
		// A light square counts the same as a dark one.
		light := qrTestMatrix("..", "..")
		if got := qrBlockPenaltyOf(light); got != 3 {
			t.Errorf("one light block scored %d, want 3", got)
		}
		// A block of mixed modules scores nothing.
		mixed := qrTestMatrix("#.", ".#")
		if got := qrBlockPenaltyOf(mixed); got != 0 {
			t.Errorf("a mixed block scored %d, want 0", got)
		}
	})

	t.Run("patterns resembling a finder score forty", func(t *testing.T) {
		// The sequence with four light modules to its left, in one row of a
		// code light everywhere else.
		blank := []string{
			"............", "............", "............", "............",
			"............", "............", "............", "............",
			"............", "............", "............",
		}
		row := qrTestMatrix(append([]string{"....#.###.#."}, blank...)...)
		if got := qrFinderLikePenalty(row); got != 40 {
			t.Errorf("one finder-like row scored %d, want 40", got)
		}
		// The same sequence read downwards.
		down := make([]string, 12)
		for i, module := range "....#.###.#." {
			down[i] = string(module) + "..........."
		}
		if got := qrFinderLikePenalty(qrTestMatrix(down...)); got != 40 {
			t.Errorf("one finder-like column scored %d, want 40", got)
		}
		// The sequence itself, read out of a line of modules.
		for _, tc := range []struct {
			line string
			want bool
		}{
			{"#.###.#", true},
			{"#.##..#", false},
			{"..###.#", false},
			{"#.###..", false},
			{"##.##.#", false},
		} {
			line := tc.line
			if got := qrIsFinderSequence(func(k int) bool { return line[k] == '#' }); got != tc.want {
				t.Errorf("%q read as a finder sequence: %v, want %v", line, got, tc.want)
			}
		}

		// Without the light modules beside it, it scores nothing.
		bare := qrTestMatrix("#.###.#", ".......", ".......", ".......",
			".......", ".......", ".......")
		if got := qrFinderLikePenalty(bare); got != 0 {
			t.Errorf("a bare sequence scored %d, want 0", got)
		}
	})

	t.Run("an unbalanced code scores ten for each twentieth", func(t *testing.T) {
		// Half dark is perfectly balanced.
		balanced := qrTestMatrix("#.", ".#")
		if got := qrBalancePenaltyOf(balanced); got != 0 {
			t.Errorf("a balanced code scored %d, want 0", got)
		}
		// Wholly dark is as far from balanced as it goes.
		dark := qrTestMatrix("##", "##")
		if got := qrBalancePenaltyOf(dark); got != 100 {
			t.Errorf("a wholly dark code scored %d, want 100", got)
		}
		// Wholly light scores the same.
		light := qrTestMatrix("..", "..")
		if got := qrBalancePenaltyOf(light); got != 100 {
			t.Errorf("a wholly light code scored %d, want 100", got)
		}
		// A quarter dark is half way.
		quarter := qrTestMatrix("#...", "....", "....", "....")
		if got := qrBalancePenaltyOf(quarter); got != 80 {
			t.Errorf("a code a sixteenth dark scored %d, want 80", got)
		}
	})
}
