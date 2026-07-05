package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Cases transcribed from CyberChef tests/operations/tests/JA3SFingerprint.mjs.
func TestJA3SFingerprint(t *testing.T) {
	runCases(t, []opCase{
		{
			"JA3S: bed95e1b", "160301003d020000390301543dd2ddedbfe33895bd6bc676a3fa6b9fe5773a6e04d5476d1af3bcbc1dcbbb00c011000011ff01000100000b00040300010200230000", "bed95e1b525d2f41db3a6d68fac5b566",
			core.Recipe{{Op: "JA3S Fingerprint", Args: []any{"Hex", "Hash digest"}}},
		},
		{
			"JA3S: 130fac2d", "160302003d020000390302543dd2ed88131999a0120d36c14a4139671d75aae3d7d7779081d3cf7dd7725a00c013000011ff01000100000b00040300010200230000", "130fac2dc19b142500acb0abc63b6379",
			core.Recipe{{Op: "JA3S Fingerprint", Args: []any{"Hex", "Hash digest"}}},
		},
		{
			"JA3S: ccc51475", "160303003d020000390303543dd328b38b445686739d58fab733fa23838f575e0e5ad9a1b9baace6cc3b4100c02f000011ff01000100000b00040300010200230000", "ccc514751b175866924439bdbb5bba34",
			core.Recipe{{Op: "JA3S Fingerprint", Args: []any{"Hex", "Hash digest"}}},
		},
	})
}
