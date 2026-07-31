package ops

import (
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// SHA2 is CyberChef's single SHA-2 operation: a size selector plus a round
// count per family. cchef also exposes sha224/sha256/sha384/sha512 as
// no-argument subcommands, but only this operation carries the name a
// CyberChef recipe or share URL uses.
//
// The round count for the 512 family is expressed in the half-steps CyberChef
// counts, so its default of 160 is the standard 80 rounds. Reduced-round
// digests have no reference outside CyberChef; every value below was recorded
// from the oracle.

func TestSHA2Sizes(t *testing.T) {
	runCases(t, []opCase{
		{
			"SHA2: 512 (default size and rounds)", "hello",
			"9b71d224bd62f3785d96d46ad3ea3d73319bfbc2890caadae2dff72519673ca7" +
				"2323c3d99ba5c11d7c7acc6e14b8c5da0c4663475c2e5c3adef46f73bcdec043",
			core.Recipe{{Op: "SHA2"}},
		},
		{
			"SHA2: 384", "hello",
			"59e1748777448c69de6b800d7a33bbfb9ff1b463e44354c3553bcdb9c666fa90125a3c79f90397bdf5f6a13de828684f",
			core.Recipe{{Op: "SHA2", Args: []any{"384", 64, 160}}},
		},
		{
			"SHA2: 256", "hello",
			"2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			core.Recipe{{Op: "SHA2", Args: []any{"256", 64, 160}}},
		},
		{
			"SHA2: 224", "hello",
			"ea09ae9cc6768c50fcee903ed054556e5bfc8347907f12598aa24193",
			core.Recipe{{Op: "SHA2", Args: []any{"224", 64, 160}}},
		},
		{
			"SHA2: 512/256", "hello",
			"e30d87cfa2a75db545eac4d61baf970366a8357c7f72fa95b52d0accb698f13a",
			core.Recipe{{Op: "SHA2", Args: []any{"512/256", 64, 160}}},
		},
		{
			"SHA2: 512/224", "hello",
			"fe8509ed1fb7dcefc27e6ac1a80eddbec4cb3d2c6fe565244374061c",
			core.Recipe{{Op: "SHA2", Args: []any{"512/224", 64, 160}}},
		},
	})
}

func TestSHA2ReducedRounds(t *testing.T) {
	runCases(t, []opCase{
		{
			"SHA2: 256 over 16 rounds", "hello",
			"bfc777c948e01d0bd5b38a85de45b21d3ef79b242258f10f80a281791857a29e",
			core.Recipe{{Op: "SHA2", Args: []any{"256", 16, 160}}},
		},
		{
			"SHA2: 256 over 32 rounds", "hello",
			"6d45fe567fb44e1d4a03e37e6a09eaf390c30dcd499b096835bea939f5ec115f",
			core.Recipe{{Op: "SHA2", Args: []any{"256", 32, 160}}},
		},
		{
			"SHA2: 256 over 48 rounds", "hello",
			"fcb4d577b14f0e46f38722647f6b8addd9f87688fb477f1635306c608c3189cd",
			core.Recipe{{Op: "SHA2", Args: []any{"256", 48, 160}}},
		},
		{
			"SHA2: 224 over 16 rounds", "hello",
			"e67deef37e2d7a0afc9c6a54bd72be8d6c1004722b1b4c8c72ec09c8",
			core.Recipe{{Op: "SHA2", Args: []any{"224", 16, 160}}},
		},
		{
			"SHA2: 224 over 48 rounds", "hello",
			"29bbf95390bc9d09e5f57c05c79718e86fd8a98479387d1e63b92478",
			core.Recipe{{Op: "SHA2", Args: []any{"224", 48, 160}}},
		},
		{
			"SHA2: 512 over 32 half-steps", "hello",
			"547b9bab27fe5b88c767168e6bef638dc9ec04e083c5bd84aed734e8c300e9ec" +
				"fe4e7b5e9bfff53778a70c4a0a6fc7f3bc9693f97f64b96cb427e20e1e057c9b",
			core.Recipe{{Op: "SHA2", Args: []any{"512", 64, 32}}},
		},
		{
			"SHA2: 512 over 80 half-steps", "hello",
			"0a5ccc84bd64ddd9bdde2313a0c0271c1cb5b7045b8b381edc5c80d5fd71c5ac" +
				"ccebb0a22fba2e419d3c58362de2b121d7ae588b09227b3cb8397c8184f5927d",
			core.Recipe{{Op: "SHA2", Args: []any{"512", 64, 80}}},
		},
		{
			"SHA2: 512 over 120 half-steps", "hello",
			"09074c7d3f0033d88f70f351df2c099f7179202fbbd9878b92f3696b02df5ec2" +
				"4c171a7387661efd229a2298d5ae63be13d0d0d4460c3b5f2846b67998ef8627",
			core.Recipe{{Op: "SHA2", Args: []any{"512", 64, 120}}},
		},
		{
			"SHA2: 512/256 over 40 half-steps", "hello",
			"058d388e604f76a068934124c46fb2092d6a71ac8a583abb20425df27a33ff10",
			core.Recipe{{Op: "SHA2", Args: []any{"512/256", 64, 40}}},
		},
		{
			"SHA2: 512/256 over 120 half-steps", "hello",
			"46be06e6c6e94f93db7fc982ab41b9baf430a9a8d0a5aed6a5a91d3e7130002d",
			core.Recipe{{Op: "SHA2", Args: []any{"512/256", 64, 120}}},
		},
	})
}

// TestSHA2OddHalfSteps pins the half-step arithmetic: CyberChef's loop advances
// by two, so an odd count does the same work as the next even one.
func TestSHA2OddHalfSteps(t *testing.T) {
	const want = "5ba7bbc56ed81f6b5bba94a24e72bc0c3ee568011a8a780c8596b8cbbc1376e4" +
		"47e9b4d49ee7aa7716e2c6aead12178f21bac955fde6026d140b4bf8215edf4a"
	runCases(t, []opCase{
		{
			"SHA2: 512 over 81 half-steps", "hello", want,
			core.Recipe{{Op: "SHA2", Args: []any{"512", 64, 81}}},
		},
		{
			"SHA2: 512 over 82 half-steps", "hello", want,
			core.Recipe{{Op: "SHA2", Args: []any{"512", 64, 82}}},
		},
	})
}

func TestSHA2MultiBlock(t *testing.T) {
	const long = "The quick brown fox jumps over the lazy dog. " +
		"The quick brown fox jumps over the lazy dog. " +
		"The quick brown fox jumps over the lazy dog."
	runCases(t, []opCase{
		{
			"SHA2: 512 over 120 half-steps, three blocks", long,
			"6758afa52ece0ee337b95c4db106be4e8d8d4a968a76d368c7cf036a440ed3c5" +
				"5e93ac4a400d8cc893b68decb8c429fc4701b470713ad401d1c913969a6581d0",
			core.Recipe{{Op: "SHA2", Args: []any{"512", 64, 120}}},
		},
	})
}

// TestSHA2MatchesTheStandaloneSubcommands checks that the four no-argument
// operations agree with SHA2 at its default round counts, so the two ways of
// asking cannot drift apart.
func TestSHA2MatchesTheStandaloneSubcommands(t *testing.T) {
	for _, tc := range []struct{ size, op string }{
		{"224", "SHA224"}, {"256", "SHA256"}, {"384", "SHA384"}, {"512", "SHA512"},
	} {
		viaSHA2, err := core.Recipe{{Op: "SHA2", Args: []any{tc.size, 64, 160}}}.Execute(sdish("hello"))
		if err != nil {
			t.Fatalf("SHA2 %s: %v", tc.size, err)
		}
		direct, err := core.Recipe{{Op: tc.op}}.Execute(sdish("hello"))
		if err != nil {
			t.Fatalf("%s: %v", tc.op, err)
		}
		if viaSHA2.String() != direct.String() {
			t.Errorf("SHA2 %s = %s, but %s = %s", tc.size, viaSHA2, tc.op, direct)
		}
	}
}

// TestSHA2RoundsBounds pins the minimums CyberChef declares and the maximums
// cchef adds. Past the end of the constant table CyberChef reads an undefined
// entry, turning the working value into NaN and then zero, and returns a
// confident digest built from it; cchef refuses instead.
func TestSHA2RoundsBounds(t *testing.T) {
	op, _ := core.Default.Get("SHA2")
	for _, tc := range []struct {
		name string
		args []any
		want string
	}{
		{
			"256 family below the minimum",
			[]any{"256", 15.0, 160.0},
			"Rounds must be greater than or equal to 16.",
		},
		{
			"256 family past the table",
			[]any{"256", 65.0, 160.0},
			"Rounds must be less than or equal to 64.",
		},
		{
			"512 family below the minimum",
			[]any{"512", 64.0, 31.0},
			"Rounds must be greater than or equal to 32.",
		},
		{
			"512 family past the table",
			[]any{"512", 64.0, 161.0},
			"Rounds must be less than or equal to 160.",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := core.CoerceArgs(op.Args(), tc.args)
			if err == nil {
				t.Fatalf("%v was accepted", tc.args)
			}
			if err.Error() != tc.want {
				t.Errorf("got %q, want %q", err.Error(), tc.want)
			}
		})
	}
}

// TestSHA2UnknownSize covers the size selector's rejection path.
func TestSHA2UnknownSize(t *testing.T) {
	op, _ := core.Default.Get("SHA2")
	if _, err := core.CoerceArgs(op.Args(), []any{"640", 64.0, 160.0}); err == nil {
		t.Fatal("an unknown digest size was accepted")
	}
}

// TestSHA2PaddingBoundary covers the 128-byte block padding where the length
// field does not fit beside the message and a further block is needed. A
// message of 112 to 127 bytes is the case that spills.
func TestSHA2PaddingBoundary(t *testing.T) {
	runCases(t, []opCase{
		{
			"SHA2: 112 bytes, the first length that spills", strings.Repeat("A", 112),
			"1a008b0480a4eb64d292db671d4f43f46fc57e077b72ad3ec0a3b0b63b320357" +
				"a11418ea916038e9b659ccf39ae574ef8a8f683f1eff954788591c13022fcd81",
			core.Recipe{{Op: "SHA2", Args: []any{"512", 64, 160}}},
		},
		{
			"SHA2: 120 bytes, still spilling", strings.Repeat("A", 120),
			"44f0b49466b767aad4665e4e774e5f554b0936becebdf323ca951ee7e8ab0dc5" +
				"5fd5ed553e54d9ba5723c546402afbcf84f4da8134c4ed3d276efa9f3a6cd433",
			core.Recipe{{Op: "SHA2", Args: []any{"512", 64, 160}}},
		},
	})
}

// TestSHA2HasherShape covers the hash.Hash reporting methods, which the
// operation itself never calls but the interface requires.
func TestSHA2HasherShape(t *testing.T) {
	for _, tc := range []struct {
		size               string
		wantSum, wantBlock int
	}{
		{"512", 64, 128},
		{"512/224", 28, 128},
		{"256", 32, 64},
		{"224", 28, 64},
	} {
		h := newSHA2(sha2Variants[tc.size], 64, 160)
		if h.Size() != tc.wantSum {
			t.Errorf("%s: Size() = %d, want %d", tc.size, h.Size(), tc.wantSum)
		}
		if h.BlockSize() != tc.wantBlock {
			t.Errorf("%s: BlockSize() = %d, want %d", tc.size, h.BlockSize(), tc.wantBlock)
		}
		if got := len(h.Sum(nil)); got != tc.wantSum {
			t.Errorf("%s: digest is %d bytes, want %d", tc.size, got, tc.wantSum)
		}
	}
}
