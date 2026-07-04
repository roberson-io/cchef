package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestROT8000Fixtures transcribes CyberChef's ROT8000 cases from Rotate.mjs.
// ROT8000 is an involution over the valid BMP code points, so encrypt and
// decrypt use the same operation (the "backward" case round-trips the output of
// the "normal" case back to plaintext).
func TestROT8000Fixtures(t *testing.T) {
	const plain = "The Quick Brown Fox Jumped Over The Lazy Dog."
	const cipher = "籝籱籮 籚籾籲籬籴 籋类籸粀籷 籏籸粁 籓籾籶籹籮籭 籘籿籮类 籝籱籮 籕籪粃粂 籍籸籰簷"
	runCases(t, []opCase{
		{
			"ROT8000: nothing", "", "",
			core.Recipe{{Op: "ROT8000", Args: []any{}}},
		},
		{
			"ROT8000: normal", plain, cipher,
			core.Recipe{{Op: "ROT8000", Args: []any{}}},
		},
		{
			"ROT8000: backward", cipher, plain,
			core.Recipe{{Op: "ROT8000", Args: []any{}}},
		},
	})
}
