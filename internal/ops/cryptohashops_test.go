package ops

// Standalone hash operation tests (MD2, MD4, SHA0, HAS-160, RIPEMD, Snefru,
// Whirlpool). CyberChef ships no fixture files for these, so every expected
// value was produced by the CyberChef-server oracle (crypto-api). Empty-input
// and hex-driven cases additionally match published standard vectors.

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// fromHexThen wraps an op recipe with a leading From Hex so the input bytes are
// exact (used for byte-precise and empty-input cases).
func fromHexThen(op string, args ...any) core.Recipe {
	return core.Recipe{
		{Op: "From Hex", Args: []any{"Auto"}},
		{Op: op, Args: args},
	}
}

func TestCryptoHashOps(t *testing.T) {
	const fox = "The quick brown fox"
	runCases(t, []opCase{
		// Defaults over an ASCII input.
		{
			"MD2", fox, "ea0db54a7590419c56c4ee9f6a41ebd5",
			core.Recipe{{Op: "MD2", Args: []any{float64(18)}}},
		},
		{
			"MD4", fox, "d4559d26c7203ec400e382fd1c8a6092",
			core.Recipe{{Op: "MD4", Args: []any{}}},
		},
		{
			"SHA0", fox, "3064121bad7ced29ecb673863d1c6d61a909f9f3",
			core.Recipe{{Op: "SHA0", Args: []any{float64(80)}}},
		},
		{
			"HAS-160", fox, "b883cd7a221dd7eae27c1d02c680d5fa1e3a2743",
			core.Recipe{{Op: "HAS-160", Args: []any{float64(80)}}},
		},
		{
			"RIPEMD-320", fox, "b218d0b0605815782cd635ffb9b5dfca655c7e9049b0390f144b1c84ff01f69e26d9feb64b500392",
			core.Recipe{{Op: "RIPEMD", Args: []any{"320"}}},
		},
		{
			"RIPEMD-160", fox, "aa1dd8137a60bbfb149657beca550f4c7321060d",
			core.Recipe{{Op: "RIPEMD", Args: []any{"160"}}},
		},
		{
			"RIPEMD-256", fox, "907ddce668e4cfa07f87f3a67e084deac2dc0d883d4518d46699ae5a3571ff9b",
			core.Recipe{{Op: "RIPEMD", Args: []any{"256"}}},
		},
		{
			"RIPEMD-128", fox, "51c426ea6aaa8c6716b275b2a76d7ca0",
			core.Recipe{{Op: "RIPEMD", Args: []any{"128"}}},
		},
		{
			"Snefru 128/8", fox, "c95dd2362045fe41c7adf09daafefaaa",
			core.Recipe{{Op: "Snefru", Args: []any{float64(128), "8"}}},
		},
		{
			"Snefru 256/8", fox, "d3f1805b471e909a9dc79ec366db03fa1c51b6cef32b78c3e334fd44e43d199f",
			core.Recipe{{Op: "Snefru", Args: []any{float64(256), "8"}}},
		},
		{
			"Snefru 128/4", fox, "e539ce660233969609cecef7d63830ba",
			core.Recipe{{Op: "Snefru", Args: []any{float64(128), "4"}}},
		},
		{
			"Whirlpool", fox, "317edc3c5172ea5987902aa9c4f1defedf4d5aa59209bdf7574cc6da0039852c24b8da70ecb07997ff83e86d32d2851215d3dcbd6bb9736bdef21c349d483e6d",
			core.Recipe{{Op: "Whirlpool", Args: []any{"Whirlpool", float64(10)}}},
		},
		{
			"Whirlpool-T", fox, "9c816385a8005310214fc591985dba58cfd1de44fd25b9ef420c9eefea45f174ec91daf390f0a22675ec8bbedb146e1b4e7b5e5cf0c5732d259f80ee733369b7",
			core.Recipe{{Op: "Whirlpool", Args: []any{"Whirlpool-T", float64(10)}}},
		},
		{
			"Whirlpool-0", fox, "508d046aec849496f1c16db5bf8344152cd08d736d6a2bdb8019a934a3b78afb048074fd2e7024d80fdb96f9ad07bec9685f028a180c3dff398af7771f96b935",
			core.Recipe{{Op: "Whirlpool", Args: []any{"Whirlpool-0", float64(10)}}},
		},

		// Varied parameters.
		{
			"MD2 rounds 0 (JS ||18 quirk)", fox, "ea0db54a7590419c56c4ee9f6a41ebd5",
			core.Recipe{{Op: "MD2", Args: []any{float64(0)}}},
		},
		{
			"SHA0 rounds 16", fox, "e2d16866bb66fda541358b11d2f5388e79794613",
			core.Recipe{{Op: "SHA0", Args: []any{float64(16)}}},
		},
		{
			"HAS-160 rounds 1", fox, "1b4277ac8ace1257bd0469cf7431eda8663605d4",
			core.Recipe{{Op: "HAS-160", Args: []any{float64(1)}}},
		},
		{
			"Whirlpool rounds 1", fox, "91afc9700160b03fa080848e8a5abcb6987d104d032187edd1471e22a8207fc777c867c5ba025c45b94c982fcb598439590dd39d67ced08c905888bec63f2e02",
			core.Recipe{{Op: "Whirlpool", Args: []any{"Whirlpool", float64(1)}}},
		},
		{
			"Snefru 256/2", fox, "25c074dc9489615c5f390fe822faf023b7a6929ce694e469e48dc4f02ac9a581",
			core.Recipe{{Op: "Snefru", Args: []any{float64(256), "2"}}},
		},

		// Empty input (matches published standard vectors).
		{
			"MD2 empty", "", "8350e5a3e24c153df2275c9f80692773",
			core.Recipe{{Op: "MD2", Args: []any{float64(18)}}},
		},
		{
			"MD4 empty", "", "31d6cfe0d16ae931b73c59d7e0c089c0",
			core.Recipe{{Op: "MD4", Args: []any{}}},
		},
		{
			"SHA0 empty", "", "f96cea198ad1dd5617ac084a3d92c6107708c0ef",
			core.Recipe{{Op: "SHA0", Args: []any{float64(80)}}},
		},
		{
			"RIPEMD-160 empty", "", "9c1185a5c5e9fc54612808977ee8f548b2258d31",
			core.Recipe{{Op: "RIPEMD", Args: []any{"160"}}},
		},
		{
			"Whirlpool empty", "", "19fa61d75522a4669b44e39c1d2e1726c530232130d407f89afee0964997f7a73e83be698b288febcf88e3e03c4f0757ea8964e59b63d93708b138cc42a66eb3",
			core.Recipe{{Op: "Whirlpool", Args: []any{"Whirlpool", float64(10)}}},
		},

		// Byte-exact (hex-driven) inputs.
		{
			"MD2 hex bytes", "deadbeef00ff", "77092a74249f7c492d376921741a0589",
			fromHexThen("MD2", float64(18)),
		},
		{
			"Whirlpool hex bytes", "deadbeef00ff", "0038046a0eeeccc3a8cf6dd94aaf524d8d285c7b660c8d667ac5089049909f2d104777139e594ac03d863038362805b8d0651160d155d8dc69224cd07866cc01",
			fromHexThen("Whirlpool", "Whirlpool", float64(10)),
		},
		{
			"HAS-160 hex bytes", "deadbeef00ff", "d4b494e4284fa0ed5795ae0589099076bd239437",
			fromHexThen("HAS-160", float64(80)),
		},
		{
			"Snefru 384 hex bytes", "deadbeef00ff", "cc936307ea88f1752ca959586f6d7c1ed328b98e0ffb188b3b62bd2b8c8170091721c21a56584d068e7fdab0702e2c2e",
			fromHexThen("Snefru", float64(384), "8"),
		},
		{
			"Snefru 256 hex bytes", "deadbeef00ff", "9103257354a6e411b55b7c753da6ed4e0862b9264e79adc68b1ef6d6e7394a58",
			fromHexThen("Snefru", float64(256), "8"),
		},
	})
}
