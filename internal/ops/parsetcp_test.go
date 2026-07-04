package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// TestParseTCPFixtures transcribes CyberChef's ParseTCP.mjs cases (expected
// values captured minified from the CyberChef-server oracle). Exercises the
// no-options, options (MSS/NOP/Window Scale/SACK) and Timestamps paths.
func TestParseTCPFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"no options", "c2eb0050a138132e70dc9fb9501804025ea70000",
			`{"Source port":49899,"Destination port":80,"Sequence number":"2704806702","Acknowledgement number":1893507001,"Data offset":"5 (20 bytes)","Flags":{"Reserved":"000","NS":0,"CWR":0,"ECE":0,"URG":0,"ACK":1,"PSH":1,"RST":0,"SYN":0,"FIN":0},"Window size":"1026 (Scaled: 1026)","Checksum":"0x5ea7","Urgent pointer":"0x0000"}`,
			core.Recipe{{Op: "Parse TCP", Args: []any{"Hex"}}},
		},
		{
			"options", "c2eb0050a1380c1f000000008002faf080950000020405b40103030801010402",
			`{"Source port":49899,"Destination port":80,"Sequence number":"2704804895","Acknowledgement number":0,"Data offset":"8 (32 bytes)","Flags":{"Reserved":"000","NS":0,"CWR":0,"ECE":0,"URG":0,"ACK":0,"PSH":0,"RST":0,"SYN":1,"FIN":0},"Window size":"64240 (Scaled: 16445440)","Checksum":"0x8095","Urgent pointer":"0x0000","Options":{"Maximum Segment Size":{"Kind":2,"Length":4,"Value":1460},"No-Operation":{"Kind":1},"Window Scale":{"Kind":3,"Length":3,"Value":{"Shift count":8,"Multiplier":256}},"SACK Permitted":{"Kind":4,"Length":2}}}`,
			core.Recipe{{Op: "Parse TCP", Args: []any{"Hex"}}},
		},
		{
			"alternate checksum option", "c2eb0050a138132e000000006002faf0000000000e030100",
			`{"Source port":49899,"Destination port":80,"Sequence number":"2704806702","Acknowledgement number":0,"Data offset":"6 (24 bytes)","Flags":{"Reserved":"000","NS":0,"CWR":0,"ECE":0,"URG":0,"ACK":0,"PSH":0,"RST":0,"SYN":1,"FIN":0},"Window size":"64240 (Scaled: 64240)","Checksum":"0x0000","Urgent pointer":"0x0000","Options":{"TCP Alternate Checksum Request (obsolete)":{"Kind":14,"Length":3,"Value":"8-bit Fletchers's algorithm (0x01)"},"End of Option List":{"Kind":0}}}`,
			core.Recipe{{Op: "Parse TCP", Args: []any{"Hex"}}},
		},
		{
			"timestamps", "9e90e11574d57b2c00000000a002ffffe5740000020405b40402080aa4e8c8f50000000001030308",
			`{"Source port":40592,"Destination port":57621,"Sequence number":"1960147756","Acknowledgement number":0,"Data offset":"10 (40 bytes)","Flags":{"Reserved":"000","NS":0,"CWR":0,"ECE":0,"URG":0,"ACK":0,"PSH":0,"RST":0,"SYN":1,"FIN":0},"Window size":"65535 (Scaled: 16776960)","Checksum":"0xe574","Urgent pointer":"0x0000","Options":{"Maximum Segment Size":{"Kind":2,"Length":4,"Value":1460},"SACK Permitted":{"Kind":4,"Length":2},"Timestamps":{"Kind":8,"Length":10,"Value":{"Current Timestamp":"2766719221","Echo Reply":"0"}},"No-Operation":{"Kind":1},"Window Scale":{"Kind":3,"Length":3,"Value":{"Shift count":8,"Multiplier":256}}}}`,
			core.Recipe{{Op: "Parse TCP", Args: []any{"Hex"}}},
		},
	})
}
