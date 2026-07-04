package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func TestCaseOps(t *testing.T) {
	runCases(t, []opCase{
		{
			"To Upper All", "Hello, World!", "HELLO, WORLD!",
			core.Recipe{{Op: "To Upper case", Args: []any{"All"}}},
		},
		{
			"To Upper Word", "hello there world", "Hello There World",
			core.Recipe{{Op: "To Upper case", Args: []any{"Word"}}},
		},
		{
			"To Upper Sentence", "hello there. how are you?", "Hello there. How are you?",
			core.Recipe{{Op: "To Upper case", Args: []any{"Sentence"}}},
		},
		{
			"To Lower", "Hello, World!", "hello, world!",
			core.Recipe{{Op: "To Lower case"}},
		},
		{
			"Case round trip", "MiXeD", "MIXED",
			core.Recipe{
				{Op: "To Lower case"},
				{Op: "To Upper case", Args: []any{"All"}},
			},
		},
	})
}
