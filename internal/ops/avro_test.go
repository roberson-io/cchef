package ops

// Tests for the Avro to JSON operation.
//
// CyberChef's Avro to JSON is a thin wrapper around the avsc library, and it
// ships only two tiny fixtures. The table-driven vectors here were therefore
// derived from avsc 5.7.9 (the exact version CyberChef bundles) used as an
// oracle: each Avro Object Container File is fed in as hex through From Hex and
// compared against avsc's output. They were additionally differential-tested
// against avsc across many random schemas. These are ordinary tests — edit them
// as needed.

import (
	"encoding/hex"
	"math"
	"strings"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

func TestAvroToJSONErrors(t *testing.T) {
	cases := []struct {
		name   string
		input  string
		args   []any
		errSub string
	}{
		{"empty input, force true", "", []any{true}, "Please provide an input."},
		{"empty input, force false", "", []any{false}, "Please provide an input."},
		{"bad magic", "not an avro file at all!!", []any{true}, "Error parsing Avro file."},
		// Valid magic but the metadata map has no avro.schema.
		{"magic then zeros", "\x4f\x62\x6a\x01" + strings.Repeat("\x00", 40), []any{true}, "Error parsing Avro file."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := core.Recipe{{Op: "Avro to JSON", Args: c.args}}
			_, err := r.Execute(core.NewDish([]byte(c.input), core.TypeString))
			if err == nil || !strings.Contains(err.Error(), c.errSub) {
				t.Fatalf("got err %v, want substring %q", err, c.errSub)
			}
		})
	}
}

// TestAvroToJSONBareMagic covers avsc's quirk: a stream containing only the
// 4-byte magic (no header body) yields an empty result rather than an error.
func TestAvroToJSONBareMagic(t *testing.T) {
	runCases(t, []opCase{
		{
			"bare magic -> empty array", "4f626a01", "[]",
			core.Recipe{{Op: "From Hex", Args: []any{"Auto"}}, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"bare magic -> empty (force false)", "4f626a01", "",
			core.Recipe{{Op: "From Hex", Args: []any{"Auto"}}, {Op: "Avro to JSON", Args: []any{false}}},
		},
	})
}

func TestAvroToJSONVectors(t *testing.T) {
	fromHex := core.RecipeOp{Op: "From Hex", Args: []any{"Auto"}}
	runCases(t, []opCase{
		{
			"scalars true", `4f626a0104166176726f2e736368656d61e4037b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226e75222c2274797065223a226e756c6c227d2c7b226e616d65223a22626f222c2274797065223a22626f6f6c65616e227d2c7b226e616d65223a22696e222c2274797065223a22696e74227d2c7b226e616d65223a226c67222c2274797065223a226c6f6e67227d2c7b226e616d65223a22666c222c2274797065223a22666c6f6174227d2c7b226e616d65223a226462222c2274797065223a22646f75626c65227d2c7b226e616d65223a227374222c2274797065223a22737472696e67227d5d7d146176726f2e636f646563086e756c6c00c3ffff42fe048bd58189339ce141aa550236010d9693d89fee470000003f6e861bf0f92109400c68c3a96c6c6fc3ffff42fe048bd58189339ce141aa55`, `{
    "nu": null,
    "bo": true,
    "in": -7,
    "lg": 1234567890123,
    "fl": 0.5,
    "db": 3.14159,
    "st": "héllo"
}`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"scalars false", `4f626a0104166176726f2e736368656d61e4037b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226e75222c2274797065223a226e756c6c227d2c7b226e616d65223a22626f222c2274797065223a22626f6f6c65616e227d2c7b226e616d65223a22696e222c2274797065223a22696e74227d2c7b226e616d65223a226c67222c2274797065223a226c6f6e67227d2c7b226e616d65223a22666c222c2274797065223a22666c6f6174227d2c7b226e616d65223a226462222c2274797065223a22646f75626c65227d2c7b226e616d65223a227374222c2274797065223a22737472696e67227d5d7d146176726f2e636f646563086e756c6c003a319855b39ebf89645cd24b1bbae1640236010d9693d89fee470000003f6e861bf0f92109400c68c3a96c6c6f3a319855b39ebf89645cd24b1bbae164`, `{"nu":null,"bo":true,"in":-7,"lg":1234567890123,"fl":0.5,"db":3.14159,"st":"héllo"}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
		{
			"bytes", `4f626a0104166176726f2e736368656d6186017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2262222c2274797065223a226279746573227d5d7d146176726f2e636f646563086e756c6c0014134a3ffd1ec560ad621a9c6414ed7f0208060102ff14134a3ffd1ec560ad621a9c6414ed7f`, `{
    "b": {
        "type": "Buffer",
        "data": [
            1,
            2,
            255
        ]
    }
}`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"empty bytes", `4f626a0104166176726f2e736368656d6186017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2262222c2274797065223a226279746573227d5d7d146176726f2e636f646563086e756c6c00539c76c4d84d4b00c8fe754a3efe17e4020200539c76c4d84d4b00c8fe754a3efe17e4`, `{"b":{"type":"Buffer","data":[]}}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
		{
			"fixed", `4f626a0104166176726f2e736368656d61c2017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2266222c2274797065223a7b226e616d65223a226d64222c2274797065223a226669786564222c2273697a65223a347d7d5d7d146176726f2e636f646563086e756c6c002f2125fc97f6eea63fcab5fcff41244d0208deadbeef2f2125fc97f6eea63fcab5fcff41244d`, `{
    "f": {
        "type": "Buffer",
        "data": [
            222,
            173,
            190,
            239
        ]
    }
}`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"enum", `4f626a0104166176726f2e736368656d61ea017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2265222c2274797065223a7b226e616d65223a2253756974222c2274797065223a22656e756d222c2273796d626f6c73223a5b2248222c2244222c2243222c2253225d7d7d5d7d146176726f2e636f646563086e756c6c00f6fdfe2b9e0f37abae6ee4615ba2e5ea020202f6fdfe2b9e0f37abae6ee4615ba2e5ea`, `{"e":"D"}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
		{
			"union null/string", `4f626a0104166176726f2e736368656d619a017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2275222c2274797065223a5b226e756c6c222c22737472696e67225d7d5d7d146176726f2e636f646563086e756c6c00747aa7425eb9e4d6b329a39e769cb0be040a0204686900747aa7425eb9e4d6b329a39e769cb0be`, `{"u":"hi"}
{"u":null}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
		{
			"union int/string", `4f626a0104166176726f2e736368656d6198017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2275222c2274797065223a5b22696e74222c22737472696e67225d7d5d7d146176726f2e636f646563086e756c6c00ee20661c00b402bba7cfab7c98c6fd92040a000a020278ee20661c00b402bba7cfab7c98c6fd92`, `[
    {
        "u": 5
    },
    {
        "u": "x"
    }
]`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"union string/bytes", `4f626a0104166176726f2e736368656d619c017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2275222c2274797065223a5b22737472696e67222c226279746573225d7d5d7d146176726f2e636f646563086e756c6c00dfde8ac7f2470fb14cfe6cd3dba4372e04100004686902040102dfde8ac7f2470fb14cfe6cd3dba4372e`, `{"u":"hi"}
{"u":{"type":"Buffer","data":[1,2]}}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
		{
			"wrapped union int/long", `4f626a0104166176726f2e736368656d6194017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2275222c2274797065223a5b22696e74222c226c6f6e67225d7d5d7d146176726f2e636f646563086e756c6c00a4ab11aa1aae0b6d06ce2af48e8dbaf00408000a020ca4ab11aa1aae0b6d06ce2af48e8dbaf0`, `{"u":{"int":5}}
{"u":{"long":6}}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
		{
			"wrapped union null/string/int/long", `4f626a0104166176726f2e736368656d61b4017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2275222c2274797065223a5b226e756c6c222c22737472696e67222c22696e74222c226c6f6e67225d7d5d7d146176726f2e636f646563086e756c6c00de675b1114a8b6c9f8fcbd610c68150a081000020273040a060cde675b1114a8b6c9f8fcbd610c68150a`, `{"u":null}
{"u":{"string":"s"}}
{"u":{"int":5}}
{"u":{"long":6}}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
		{
			"wrapped union records", `4f626a0104166176726f2e736368656d6182037b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2275222c2274797065223a5b7b226e616d65223a2241222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2278222c2274797065223a22696e74227d5d7d2c7b226e616d65223a2242222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2279222c2274797065223a22696e74227d5d7d5d7d5d7d146176726f2e636f646563086e756c6c00e7897998d19616fdcc4331bb8910b59f040800020204e7897998d19616fdcc4331bb8910b59f`, `[
    {
        "u": {
            "A": {
                "x": 1
            }
        }
    },
    {
        "u": {
            "B": {
                "y": 2
            }
        }
    }
]`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"array", `4f626a0104166176726f2e736368656d61b4017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2261222c2274797065223a7b2274797065223a226172726179222c226974656d73223a22696e74227d7d5d7d146176726f2e636f646563086e756c6c000498c8256d3e0615a1dcf9e0349903f8040c0602040600000498c8256d3e0615a1dcf9e0349903f8`, `{"a":[1,2,3]}
{"a":[]}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
		{
			"map ordered", `4f626a0104166176726f2e736368656d61b2017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226d222c2274797065223a7b2274797065223a226d6170222c2276616c756573223a22696e74227d7d5d7d146176726f2e636f646563086e756c6c007d932b4fd6c1ec8be12769a3d7cbae0d022e060a7a65627261020a6170706c65040a6d616e676f06007d932b4fd6c1ec8be12769a3d7cbae0d`, `{
    "m": {
        "zebra": 1,
        "apple": 2,
        "mango": 3
    }
}`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"nested record", `4f626a0104166176726f2e736368656d61bc027b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226368696c64222c2274797065223a7b226e616d65223a2263222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2278222c2274797065223a22696e74227d2c7b226e616d65223a2279222c2274797065223a22737472696e67227d5d7d7d5d7d146176726f2e636f646563086e756c6c00da23c0af2e05ecc7ba2f03340cb9d5320206020261da23c0af2e05ecc7ba2f03340cb9d532`, `{
    "child": {
        "x": 1,
        "y": "a"
    }
}`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"empty record", `4f626a0104166176726f2e736368656d61507b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b5d7d146176726f2e636f646563086e756c6c0084b6cf5034a55a1c79f55bfc600b665c020084b6cf5034a55a1c79f55bfc600b665c`, `{}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
		{
			"top-level array schema", `4f626a0104166176726f2e736368656d613c7b2274797065223a226172726179222c226974656d73223a22696e74227d146176726f2e636f646563086e756c6c00d3046d1fabc37c665b01f34c3a255916040e04020400020600d3046d1fabc37c665b01f34c3a255916`, `[
    [
        1,
        2
    ],
    [
        3
    ]
]`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"multiple records true", `4f626a0104166176726f2e736368656d6182017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226e222c2274797065223a22696e74227d5d7d146176726f2e636f646563086e756c6c004867c874e1cf2818690710a13342cb4f06060204064867c874e1cf2818690710a13342cb4f`, `[
    {
        "n": 1
    },
    {
        "n": 2
    },
    {
        "n": 3
    }
]`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"float edges", `4f626a0104166176726f2e736368656d6186017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2266222c2274797065223a22666c6f6174227d5d7d146176726f2e636f646563086e756c6c0074a5f8fa77e93bb560a0397e94518bd50618cdcccc3dec78ad600000008074a5f8fa77e93bb560a0397e94518bd5`, `{"f":0.10000000149011612}
{"f":100000002004087730000}
{"f":0}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
		{
			"double edges", `4f626a0104166176726f2e736368656d6188017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2264222c2274797065223a22646f75626c65227d5d7d146176726f2e636f646563086e756c6c002abc2cdd2c5c723b303fecfeb9350e5406309a9999999999b93f50efe2d6e41a4b4448afbc9af2d77a3e2abc2cdd2c5c723b303fecfeb9350e54`, `{"d":0.1}
{"d":1e+21}
{"d":1e-7}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
		{
			"string escapes", `4f626a0104166176726f2e736368656d6188017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2273222c2274797065223a22737472696e67227d5d7d146176726f2e636f646563086e756c6c0037be315138c6332dc0d974b0c37d6f9a02302e6122625c630a090d203c3e262fe697a5e69cacf09f988037be315138c6332dc0d974b0c37d6f9a`, `{"s":"a\"b\\c\n\t\r <>&/日本😀"}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
		{
			"deflate", `4f626a0104166176726f2e736368656d6188017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2273222c2274797065223a22737472696e67227d5d7d146176726f2e636f6465630e6465666c617465005a216e6bbc48d3d95ec05df14998e443026605c1810dc0200804c0557e009722f56b49500ccffee95d4cbe614d6ce2064d1ce8cf05170c9167b1a02e3f0b9d7872dfa2f4035a216e6bbc48d3d95ec05df14998e443`, `{
    "s": "deflate me please, this is a longer string to compress"
}`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"deflate multi", `4f626a0104166176726f2e736368656d6182017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226e222c2274797065223a22696e74227d5d7d146176726f2e636f6465630e6465666c617465008352d18fb34f54130c8f06a7257f5d51646863606261e3e0e2e1131012119390929153505251d3d0d2d133303231b3b0b2b173707271f3f0f2f10b080a098b888a894b4802008352d18fb34f54130c8f06a7257f5d51`, `{"n":0}
{"n":1}
{"n":2}
{"n":3}
{"n":4}
{"n":5}
{"n":6}
{"n":7}
{"n":8}
{"n":9}
{"n":10}
{"n":11}
{"n":12}
{"n":13}
{"n":14}
{"n":15}
{"n":16}
{"n":17}
{"n":18}
{"n":19}
{"n":20}
{"n":21}
{"n":22}
{"n":23}
{"n":24}
{"n":25}
{"n":26}
{"n":27}
{"n":28}
{"n":29}
{"n":30}
{"n":31}
{"n":32}
{"n":33}
{"n":34}
{"n":35}
{"n":36}
{"n":37}
{"n":38}
{"n":39}
{"n":40}
{"n":41}
{"n":42}
{"n":43}
{"n":44}
{"n":45}
{"n":46}
{"n":47}
{"n":48}
{"n":49}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
		{
			"upstream small record true", `4f626a0104166176726f2e736368656d618e017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226e616d65222c2274797065223a22737472696e67227d5d7d146176726f2e636f646563086e756c6c007fb1e4db830e51228b2e019f09edadd8020e0c6d796e616d657fb1e4db830e51228b2e019f09edadd8`, `{
    "name": "myname"
}`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"upstream small record false", `4f626a0104166176726f2e736368656d618e017b226e616d65223a2272222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226e616d65222c2274797065223a22737472696e67227d5d7d146176726f2e636f646563086e756c6c00e2752fb5a2a7dc359c7de4d651a58da9020e0c6d796e616d65e2752fb5a2a7dc359c7de4d651a58da9`, `{"name":"myname"}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
	})
}

func TestAvroToJSONEdges(t *testing.T) {
	fromHex := core.RecipeOp{Op: "From Hex", Args: []any{"Auto"}}
	runCases(t, []opCase{
		{
			"truncated at 15%", `4f626a0104166176726f2e736368656d61bc017b226e616d65223a2252222c2274797065223a227265636f7264222c`, `[]`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"truncated at 35%", `4f626a0104166176726f2e736368656d61bc017b226e616d65223a2252222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226e222c2274797065223a22696e74227d2c7b226e616d65223a2273222c2274797065223a22737472696e67`, `[]`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"truncated at 50%", `4f626a0104166176726f2e736368656d61bc017b226e616d65223a2252222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226e222c2274797065223a22696e74227d2c7b226e616d65223a2273222c2274797065223a22737472696e67227d5d7d146176726f2e636f646563086e756c6c00ca4e094b08869260c9fd935cb0b7b896040a0000020276ca4e094b`, `[]`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"truncated at 70%", `4f626a0104166176726f2e736368656d61bc017b226e616d65223a2252222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226e222c2274797065223a22696e74227d2c7b226e616d65223a2273222c2274797065223a22737472696e67227d5d7d146176726f2e636f646563086e756c6c00ca4e094b08869260c9fd935cb0b7b896040a0000020276ca4e094b08869260c9fd935cb0b7b896020804047676ca4e094b08869260c9fd935cb0b7b896040e06067676760800ca4e094b08869260c9fd935cb0b7b896040e0a`, `[
    {
        "n": 0,
        "s": ""
    },
    {
        "n": 1,
        "s": "v"
    },
    {
        "n": 2,
        "s": "vv"
    },
    {
        "n": 3,
        "s": "vvv"
    },
    {
        "n": 4,
        "s": ""
    }
]`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"truncated at 90%", `4f626a0104166176726f2e736368656d61bc017b226e616d65223a2252222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226e222c2274797065223a22696e74227d2c7b226e616d65223a2273222c2274797065223a22737472696e67227d5d7d146176726f2e636f646563086e756c6c00ca4e094b08869260c9fd935cb0b7b896040a0000020276ca4e094b08869260c9fd935cb0b7b896020804047676ca4e094b08869260c9fd935cb0b7b896040e06067676760800ca4e094b08869260c9fd935cb0b7b896040e0a02760c047676ca4e094b08869260c9fd935cb0b7b896040e0e067676761000ca4e094b08869260c9fd935cb0b7b896040e12027614047676ca4e094b088692`, `[
    {
        "n": 0,
        "s": ""
    },
    {
        "n": 1,
        "s": "v"
    },
    {
        "n": 2,
        "s": "vv"
    },
    {
        "n": 3,
        "s": "vvv"
    },
    {
        "n": 4,
        "s": ""
    },
    {
        "n": 5,
        "s": "v"
    },
    {
        "n": 6,
        "s": "vv"
    },
    {
        "n": 7,
        "s": "vvv"
    },
    {
        "n": 8,
        "s": ""
    }
]`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"truncated at 97%", `4f626a0104166176726f2e736368656d61bc017b226e616d65223a2252222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226e222c2274797065223a22696e74227d2c7b226e616d65223a2273222c2274797065223a22737472696e67227d5d7d146176726f2e636f646563086e756c6c00ca4e094b08869260c9fd935cb0b7b896040a0000020276ca4e094b08869260c9fd935cb0b7b896020804047676ca4e094b08869260c9fd935cb0b7b896040e06067676760800ca4e094b08869260c9fd935cb0b7b896040e0a02760c047676ca4e094b08869260c9fd935cb0b7b896040e0e067676761000ca4e094b08869260c9fd935cb0b7b896040e12027614047676ca4e094b08869260c9fd935cb0b7b896020a1606767676ca4e094b0886`, `[
    {
        "n": 0,
        "s": ""
    },
    {
        "n": 1,
        "s": "v"
    },
    {
        "n": 2,
        "s": "vv"
    },
    {
        "n": 3,
        "s": "vvv"
    },
    {
        "n": 4,
        "s": ""
    },
    {
        "n": 5,
        "s": "v"
    },
    {
        "n": 6,
        "s": "vv"
    },
    {
        "n": 7,
        "s": "vvv"
    },
    {
        "n": 8,
        "s": ""
    },
    {
        "n": 9,
        "s": "v"
    },
    {
        "n": 10,
        "s": "vv"
    }
]`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"trailing garbage", `4f626a0104166176726f2e736368656d61bc017b226e616d65223a2252222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226e222c2274797065223a22696e74227d2c7b226e616d65223a2273222c2274797065223a22737472696e67227d5d7d146176726f2e636f646563086e756c6c00aefd95efc1c9062326e3bc9a6ae5829f040e02026104046262aefd95efc1c9062326e3bc9a6ae5829f0602ff`, `{"n":1,"s":"a"}
{"n":2,"s":"bb"}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
		{
			"special doubles true", `4f626a0104166176726f2e736368656d6188017b226e616d65223a2244222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2264222c2274797065223a22646f75626c65227d5d7d146176726f2e636f646563086e756c6c00e6f2055eecb3c814701068666cd797520c60000000000000f87f000000000000f07f000000000000f0ff00000000000000800000000000000000000000000000f83fe6f2055eecb3c814701068666cd79752`, `[
    {
        "d": null
    },
    {
        "d": null
    },
    {
        "d": null
    },
    {
        "d": 0
    },
    {
        "d": 0
    },
    {
        "d": 1.5
    }
]`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}},
		},
		{
			"special doubles false", `4f626a0104166176726f2e736368656d6188017b226e616d65223a2244222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2264222c2274797065223a22646f75626c65227d5d7d146176726f2e636f646563086e756c6c00e6f2055eecb3c814701068666cd797520c60000000000000f87f000000000000f07f000000000000f0ff00000000000000800000000000000000000000000000f83fe6f2055eecb3c814701068666cd79752`, `{"d":null}
{"d":null}
{"d":null}
{"d":0}
{"d":0}
{"d":1.5}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
		{
			"special floats false", `4f626a0104166176726f2e736368656d6186017b226e616d65223a2246222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2266222c2274797065223a22666c6f6174227d5d7d146176726f2e636f646563086e756c6c00c929dc98fad3c97d2c1bef01a8e8501c06180000c07f0000807f000080ffc929dc98fad3c97d2c1bef01a8e8501c`, `{"f":null}
{"f":null}
{"f":null}
`,
			core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{false}}},
		},
	})
}

func TestAvroToJSONEdgeErrors(t *testing.T) {
	fromHex := core.RecipeOp{Op: "From Hex", Args: []any{"Auto"}}
	cases := []struct {
		name string
		hex  string
	}{
		{"sync marker corrupted", `4f626a0104166176726f2e736368656d61bc017b226e616d65223a2252222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226e222c2274797065223a22696e74227d2c7b226e616d65223a2273222c2274797065223a22737472696e67227d5d7d146176726f2e636f646563086e756c6c00aefd95efc1c9062326e3bc9a6ae5829f040e02026104046262aefd95efc1c9062326e3bc9a6ae58260`},
		{"block data corrupted", `4f626a0104169e76726f2e736368656d61bc017b226e616d65223a2252222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226e222c2274797065223a22696e74227d2c7b226e616d65223a2273222c2274797065223a22737472696e67227d5d7d146176726f2e636f646563086e756c6c00aefd95efc1c9062326e3bc9a6ae5829f040e02026104046262aefd95efc1c9062326e3bc9a6ae5829f`},
		{"snappy codec", `4f626a0104166176726f2e736368656d61bc017b226e616d65223a2252222c2274797065223a227265636f7264222c226669656c6473223a5b7b226e616d65223a226e222c2274797065223a22696e74227d2c7b226e616d65223a2273222c2274797065223a22737472696e67227d5d7d146176726f2e636f6465630c736e6170707900aefd95efc1c9062326e3bc9a6ae5829f040e02026104046262aefd95efc1c9062326e3bc9a6ae5829f`},
		{"unknown schema type", `4f626a0104166176726f2e736368656d6190017b2274797065223a227265636f7264222c226e616d65223a2252222c226669656c6473223a5b7b226e616d65223a2278222c2274797065223a224e6f5375636854797065227d5d7d146176726f2e636f646563086e756c6c0007070707070707070707070707070707`},
		{"malformed schema json", `4f626a0104166176726f2e736368656d61207b2274797065223a227265636f726422146176726f2e636f646563086e756c6c0007070707070707070707070707070707`},
		{"numeric schema node", `4f626a0104166176726f2e736368656d61043432146176726f2e636f646563086e756c6c0007070707070707070707070707070707`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := core.Recipe{fromHex, {Op: "Avro to JSON", Args: []any{true}}}
			_, err := r.Execute(core.NewDish([]byte(c.hex), core.TypeString))
			if err == nil || !strings.Contains(err.Error(), "Error parsing Avro file.") {
				t.Fatalf("got err %v, want Avro parse error", err)
			}
		})
	}
}

func TestAvroReader(t *testing.T) {
	if _, err := (&areader{data: []byte{1}}).take(5); err == nil {
		t.Fatal("take past end should error")
	}
	if _, err := (&areader{data: nil}).byte(); err == nil {
		t.Fatal("byte at EOF should error")
	}
	if _, err := (&areader{data: nil}).readLong(); err == nil {
		t.Fatal("readLong at EOF should error")
	}
	// A varint with continuation bits that never terminates overflows.
	overflow := &areader{data: []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}}
	if _, err := overflow.readLong(); err == nil || !strings.Contains(err.Error(), "overflow") {
		t.Fatalf("readLong overflow: got %v", err)
	}
	// readBytes with a negative length (zig-zag of an odd value).
	if _, err := (&areader{data: []byte{0x01}}).readBytes(); err == nil {
		t.Fatal("readBytes negative length should error")
	}
	// readBytes with a length longer than the remaining data.
	if _, err := (&areader{data: []byte{0x08}}).readBytes(); err == nil {
		t.Fatal("readBytes truncated should error")
	}
}

// TestAvroReadMetaMap covers the metadata map's negative-count (blocked) form
// and truncation.
func TestAvroReadMetaMap(t *testing.T) {
	// count -1 (zig-zag 0x01), then a block byte size (0x02), one k/v pair,
	// then terminating 0x00.
	data := []byte{0x01, 0x02, 0x02, 'k', 0x02, 'v', 0x00}
	m, err := (&areader{data: data}).readMetaMap()
	if err != nil || string(m["k"]) != "v" {
		t.Fatalf("readMetaMap blocked form: m=%v err=%v", m, err)
	}
	if _, err := (&areader{data: []byte{0x02}}).readMetaMap(); err == nil {
		t.Fatal("truncated meta map should error")
	}
	// Negative count (0x01 -> -1) followed by a truncated block-size long.
	if _, err := (&areader{data: []byte{0x01}}).readMetaMap(); err == nil {
		t.Fatal("truncated blocked meta map should error")
	}
	// A key with a negative length (0x01 -> -1) is a non-EOF corruption error.
	if _, err := (&areader{data: []byte{0x02, 0x01}}).readMetaMap(); err == nil {
		t.Fatal("negative-length metadata key should error")
	}
	// An entry count whose varint overflows.
	if _, err := (&areader{data: []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80}}).readMetaMap(); err == nil {
		t.Fatal("overflowing metadata count should error")
	}
}

// TestAvroToJSONMalformed covers the remaining error-propagation branches that
// require a specific malformed container: a corrupt deflate block, a metadata
// map with an invalid length, and schemas with invalid nested types. avsc
// rejects all of these too.
func TestAvroToJSONMalformed(t *testing.T) {
	// A valid deflate OCF whose compressed block payload has been corrupted
	// (sync intact) — decompression fails on a complete block.
	corruptDeflate, err := hex.DecodeString(
		"4f626a0104166176726f2e736368656d6188017b226e616d65223a2272222c22747970" +
			"65223a227265636f7264222c226669656c6473223a5b7b226e616d65223a2273222c22" +
			"74797065223a22737472696e67227d5d7d146176726f2e636f6465630e6465666c6174" +
			"6500874cc8839748833e605ec993afcb2ba00222d3cb48cdc9c95728cf2fca4951bf62" +
			"0300874cc8839748833e605ec993afcb2ba0")
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string][]byte{
		"corrupt deflate block": corruptDeflate,
		// metadata map, count 1, then a key with a negative length.
		"negative metadata key length": {0x4f, 0x62, 0x6a, 0x01, 0x02, 0x01},
		"invalid array items":          buildAvroOCF(`{"type":"array","items":42}`),
		"invalid map values":           buildAvroOCF(`{"type":"map","values":42}`),
		"invalid union branch":         buildAvroOCF(`["null",42]`),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := avroDecodeOCF(data); err == nil {
				t.Fatalf("%s: expected an error", name)
			}
		})
	}
}

// TestAvroDecodeValueTruncated drives each schema kind's read-error branch by
// decoding from a reader that runs out of bytes.
func TestAvroDecodeValueTruncated(t *testing.T) {
	empty := func() *areader { return &areader{data: nil} }
	schemas := []*avroSchema{
		{kind: "boolean"},
		{kind: "int"},
		{kind: "long"},
		{kind: "float"},
		{kind: "double"},
		{kind: "bytes"},
		{kind: "string"},
		{kind: "fixed", size: 4},
		{kind: "enum", symbols: []string{"A"}},
		{kind: "record", fields: []avroField{{name: "x", schema: &avroSchema{kind: "int"}}}},
		{kind: "array", items: &avroSchema{kind: "int"}},
		{kind: "map", values: &avroSchema{kind: "int"}},
		{kind: "union", branches: []*avroSchema{{kind: "null"}, {kind: "int"}}},
	}
	for _, s := range schemas {
		if _, err := decodeValue(s, empty()); err == nil {
			t.Fatalf("decodeValue(%s) on empty reader should error", s.kind)
		}
	}

	// float/double/fixed with some-but-not-enough bytes.
	if _, err := decodeValue(&avroSchema{kind: "float"}, &areader{data: []byte{1, 2}}); err == nil {
		t.Fatal("float short read should error")
	}
	if _, err := decodeValue(&avroSchema{kind: "double"}, &areader{data: []byte{1, 2, 3, 4}}); err == nil {
		t.Fatal("double short read should error")
	}
	if _, err := decodeValue(&avroSchema{kind: "fixed", size: 4}, &areader{data: []byte{1, 2}}); err == nil {
		t.Fatal("fixed short read should error")
	}

	// enum index out of range (zig-zag 0x04 -> 2, only 1 symbol).
	if _, err := decodeValue(&avroSchema{kind: "enum", symbols: []string{"A"}}, &areader{data: []byte{0x04}}); err == nil {
		t.Fatal("enum out-of-range index should error")
	}
	// union index out of range (zig-zag 0x08 -> 4, only 2 branches).
	if _, err := decodeValue(&avroSchema{kind: "union", branches: []*avroSchema{{kind: "null"}, {kind: "int"}}}, &areader{data: []byte{0x08}}); err == nil {
		t.Fatal("union out-of-range index should error")
	}
	// map key truncated (positive count, but no key bytes).
	if _, err := decodeValue(&avroSchema{kind: "map", values: &avroSchema{kind: "int"}}, &areader{data: []byte{0x02}}); err == nil {
		t.Fatal("map truncated key should error")
	}
	// map value truncated after a key.
	if _, err := decodeValue(&avroSchema{kind: "map", values: &avroSchema{kind: "int"}}, &areader{data: []byte{0x02, 0x02, 'k'}}); err == nil {
		t.Fatal("map truncated value should error")
	}
	// union branch value truncated (valid index, but no branch bytes).
	if _, err := decodeValue(&avroSchema{kind: "union", branches: []*avroSchema{{kind: "null"}, {kind: "int"}}}, &areader{data: []byte{0x02}}); err == nil {
		t.Fatal("union truncated branch value should error")
	}
	// array item truncated after a positive count.
	if _, err := decodeValue(&avroSchema{kind: "array", items: &avroSchema{kind: "int"}}, &areader{data: []byte{0x02}}); err == nil {
		t.Fatal("array truncated item should error")
	}
	// unknown kind.
	if _, err := decodeValue(&avroSchema{kind: "bogus"}, empty()); err == nil {
		t.Fatal("unknown kind should error")
	}
}

// TestAvroDecodeBlocksNegativeCount covers the negative-count (byte-sized) block
// form used by arrays and maps.
func TestAvroDecodeBlocksNegativeCount(t *testing.T) {
	// count -2 (zig-zag 0x03), block byte size 0x04, two ints (2 -> 0x04, 3 -> 0x06),
	// then terminating 0x00.
	r := &areader{data: []byte{0x03, 0x04, 0x04, 0x06, 0x00}}
	v, err := decodeValue(&avroSchema{kind: "array", items: &avroSchema{kind: "int"}}, r)
	if err != nil {
		t.Fatalf("negative-count array: %v", err)
	}
	arr, _ := v.([]any)
	if len(arr) != 2 || arr[0].(int64) != 2 || arr[1].(int64) != 3 {
		t.Fatalf("negative-count array decoded %v", arr)
	}
	// truncated negative-count block (missing the byte-size long).
	if _, err := decodeValue(&avroSchema{kind: "array", items: &avroSchema{kind: "int"}}, &areader{data: []byte{0x03}}); err == nil {
		t.Fatal("truncated negative-count block should error")
	}
	if _, err := decodeValue(&avroSchema{kind: "map", values: &avroSchema{kind: "int"}}, &areader{data: []byte{0x03}}); err == nil {
		t.Fatal("truncated negative-count map block should error")
	}
}

// TestAvroParseSchemaErrors covers schema-parsing error and reference branches.
func TestAvroParseSchemaErrors(t *testing.T) {
	reg := map[string]*avroSchema{}
	if _, err := parseAvroSchema("NoSuchType", reg, ""); err == nil {
		t.Fatal("unknown named type should error")
	}
	if _, err := parseAvroSchema(float64(42), reg, ""); err == nil {
		t.Fatal("numeric schema node should error")
	}
	// record with a non-object field.
	badField := map[string]any{"type": "record", "name": "R", "fields": []any{float64(1)}}
	if _, err := parseAvroSchema(badField, reg, ""); err == nil {
		t.Fatal("bad field should error")
	}
	// primitive wrapped in an object with a logical type (ignored).
	s, err := parseAvroSchema(map[string]any{"type": "int", "logicalType": "date"}, reg, "")
	if err != nil || s.kind != "int" {
		t.Fatalf("logical int: %v %v", s, err)
	}
	// reference resolved via the enclosing namespace.
	reg2 := map[string]*avroSchema{}
	rec := map[string]any{"type": "record", "name": "Node", "namespace": "com.ex", "fields": []any{
		map[string]any{"name": "next", "type": []any{"null", "Node"}},
	}}
	if _, err := parseAvroSchema(rec, reg2, ""); err != nil {
		t.Fatalf("namespaced self-reference should resolve: %v", err)
	}
	if _, ok := reg2["com.ex.Node"]; !ok {
		t.Fatal("record should register under its full name")
	}
	// array with a missing items schema.
	if _, err := parseAvroSchema(map[string]any{"type": "array"}, reg2, ""); err == nil {
		t.Fatal("array without items should error")
	}
	// non-namespaced self-reference resolves via the direct registry lookup.
	reg3 := map[string]*avroSchema{}
	plainRec := map[string]any{"type": "record", "name": "Node", "fields": []any{
		map[string]any{"name": "next", "type": []any{"null", "Node"}},
	}}
	if _, err := parseAvroSchema(plainRec, reg3, ""); err != nil {
		t.Fatalf("plain self-reference should resolve: %v", err)
	}
	// a dotted "name" supplies the child namespace.
	reg4 := map[string]*avroSchema{}
	dotted := map[string]any{"type": "record", "name": "a.b.C", "fields": []any{
		map[string]any{"name": "x", "type": "int"},
	}}
	if _, err := parseAvroSchema(dotted, reg4, ""); err != nil {
		t.Fatalf("dotted name: %v", err)
	}
	if _, ok := reg4["a.b.C"]; !ok {
		t.Fatal("dotted-name record should register under its full name")
	}
	// a complex type nested under "type".
	nested := map[string]any{"type": map[string]any{"type": "array", "items": "int"}}
	if s, err := parseAvroSchema(nested, reg4, ""); err != nil || s.kind != "array" {
		t.Fatalf("nested type under type: %v %v", s, err)
	}
}

// TestAvroBucketAndBranchName covers every schema-kind bucket and branch-name.
func TestAvroBucketAndBranchName(t *testing.T) {
	want := map[string]string{
		"boolean": "boolean", "int": "number", "long": "number", "float": "number",
		"double": "number", "string": "string", "enum": "string", "bytes": "buffer",
		"fixed": "buffer", "record": "object", "map": "object", "array": "array",
		"null": "null", "union": "null",
	}
	for kind, bucket := range want {
		if got := avroBucket(&avroSchema{kind: kind}); got != bucket {
			t.Errorf("avroBucket(%s) = %q want %q", kind, got, bucket)
		}
	}
	if n := avroBranchName(&avroSchema{kind: "record", name: "com.ex.A"}); n != "com.ex.A" {
		t.Errorf("branchName record = %q", n)
	}
	if n := avroBranchName(&avroSchema{kind: "int"}); n != "int" {
		t.Errorf("branchName int = %q", n)
	}
	if n := avroBranchName(&avroSchema{kind: "map"}); n != "map" {
		t.Errorf("branchName map = %q", n)
	}
}

// TestAvroDecodeOCFShort covers the fewer-than-four-bytes leniency.
func TestAvroDecodeOCFShort(t *testing.T) {
	for _, in := range [][]byte{nil, {0x4f}, {0x4f, 0x62}, {0x4f, 0x62, 0x6a}} {
		res, err := avroDecodeOCF(in)
		if err != nil || len(res) != 0 {
			t.Fatalf("avroDecodeOCF(%v) = %v, %v", in, res, err)
		}
	}
}

// TestAvroFormatNumber covers the special-number branches directly.
func TestAvroFormatNumber(t *testing.T) {
	cases := map[float64]string{
		0:                     "0",
		math.Copysign(0, -1):  "0",
		1.5:                   "1.5",
		math.NaN():            "null",
		math.Inf(1):           "null",
		math.Inf(-1):          "null",
		100000002004087730000: "100000002004087730000",
	}
	for f, want := range cases {
		if got := avroFormatNumber(f); got != want {
			t.Errorf("avroFormatNumber(%v) = %q want %q", f, got, want)
		}
	}
}

// TestAvroStringifyScalars covers the boolean-false and compact-object branches
// of the serialiser.
func TestAvroStringifyScalars(t *testing.T) {
	if got := avroStringify(avroObject{{k: "b", v: false}, {k: "t", v: true}}, 0); got != `{"b":false,"t":true}` {
		t.Fatalf("compact bools: %q", got)
	}
	if got := avroStringify(false, 0); got != "false" {
		t.Fatalf("bare false: %q", got)
	}
}

// avroVarint zig-zag encodes n as an Avro varint (test helper).
func avroVarint(n int64) []byte {
	u := uint64((n << 1) ^ (n >> 63))
	var b []byte
	for u > 0x7f {
		b = append(b, byte(u)|0x80)
		u >>= 7
	}
	return append(b, byte(u))
}

// buildAvroOCF assembles an OCF header (magic, metadata with the given schema
// and null codec, a zero sync marker) followed by each raw data block plus a
// matching sync — for crafting malformed blocks the encoder cannot produce.
func buildAvroOCF(schema string, blocks ...[]byte) []byte {
	lenPrefixed := func(b []byte) []byte { return append(avroVarint(int64(len(b))), b...) }
	var meta []byte
	meta = append(meta, avroVarint(2)...)
	meta = append(meta, lenPrefixed([]byte("avro.schema"))...)
	meta = append(meta, lenPrefixed([]byte(schema))...)
	meta = append(meta, lenPrefixed([]byte("avro.codec"))...)
	meta = append(meta, lenPrefixed([]byte("null"))...)
	meta = append(meta, avroVarint(0)...)

	sync := make([]byte, 16)
	out := append([]byte{0x4f, 0x62, 0x6a, 0x01}, meta...)
	out = append(out, sync...)
	for _, blk := range blocks {
		out = append(out, blk...)
		out = append(out, sync...)
	}
	return out
}

// TestAvroBlockFramingErrors covers the data-block framing error branches that a
// well-formed encoder never produces.
func TestAvroBlockFramingErrors(t *testing.T) {
	// A block whose data has more bytes than the declared count consumes.
	leftover := buildAvroOCF(`"int"`, []byte{0x02, 0x04, 0x02, 0xff}) // count 1, size 2, int 1 + stray byte
	if _, err := avroDecodeOCF(leftover); err == nil || !strings.Contains(err.Error(), "not fully consumed") {
		t.Fatalf("leftover block: %v", err)
	}
	// A block header with a negative count.
	negCount := buildAvroOCF(`"int"`, []byte{0x01, 0x00}) // count -1, size 0
	if _, err := avroDecodeOCF(negCount); err == nil || !strings.Contains(err.Error(), "invalid block header") {
		t.Fatalf("negative count: %v", err)
	}
	// A block whose count varint overflows (non-EOF error, not truncation).
	overflow := buildAvroOCF(`"int"`, []byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80})
	if _, err := avroDecodeOCF(overflow); err == nil || !strings.Contains(err.Error(), "overflow") {
		t.Fatalf("overflow count: %v", err)
	}

	// Truncation leniency: incomplete framing after the header yields the
	// records decoded so far (here, none) with no error.
	base := buildAvroOCF(`"int"`)       // magic + meta + sync, no blocks
	headerNoSync := base[:len(base)-16] // truncated before the header sync
	lenient := [][]byte{
		headerNoSync,
		append(append([]byte(nil), base...), 0x80), // block count varint truncated
		append(append([]byte(nil), base...), 0x02), // count ok, block size truncated
	}
	for i, in := range lenient {
		res, err := avroDecodeOCF(in)
		if err != nil || len(res) != 0 {
			t.Fatalf("lenient case %d: res=%v err=%v", i, res, err)
		}
	}
	// count ok, block size varint overflows -> error (not truncation).
	sizeOverflow := append(append([]byte(nil), base...), 0x02, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80)
	if _, err := avroDecodeOCF(sizeOverflow); err == nil || !strings.Contains(err.Error(), "overflow") {
		t.Fatalf("size overflow: %v", err)
	}
}
