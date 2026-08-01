package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// Transcribed from ../CyberChef/tests/operations/tests/Modhex.mjs.
func TestModhexFixtures(t *testing.T) {
	multiline := "fk,dc,ie,hb,ii,dc,ht,ik,ie,hg,hr,hh,dc,ie,hk,\n" +
		"if,if,hk,hu,hi,dc,hk,hu,dc,if,hj,hg,dc,he,id,\n" +
		"hv,if,he,hj,dc,hv,hh,dc,if,hj,hg,dc,if,hj,hk,\n" +
		"ie,dc,hh,hk,hi,dc,if,id,hg,hg,dr,dc,ie,if,hb,\n" +
		"id,ih,hk,hu,hi,dc,if,hv,dc,hf,hg,hb,if,hj,dr,\n" +
		"dc,hl,ig,ie,if,dc,hd,hg,he,hb,ig,ie,hg,dc,fk,\n" +
		"dc,he,hv,ig,hr,hf,hu,di,if,dc,ht,hb,hn,hg,dc,\n" +
		"ig,ic,dc,ht,ik,dc,ht,hk,hu,hf,dc,ii,hj,hk,he,\n" +
		"hj,dc,hv,hh,dc,if,hj,hg,dc,hh,hk,hi,ie,dc,fk,\n" +
		"dc,ii,hv,ig,hr,hf,dc,he,hj,hv,hv,ie,hg,du"
	multilinePercent := "fk%dc%ie%hb%ii%dc%ht%ik%ie%hg%hr%hh%dc%ie%hk%\n" +
		"if%if%hk%hu%hi%dc%hk%hu%dc%if%hj%hg%dc%he%id%\n" +
		"hv%if%he%hj%dc%hv%hh%dc%if%hj%hg%dc%if%hj%hk%\n" +
		"ie%dc%hh%hk%hi%dc%if%id%hg%hg%dr%dc%ie%if%hb%\n" +
		"id%ih%hk%hu%hi%dc%if%hv%dc%hf%hg%hb%if%hj%dr%\n" +
		"dc%hl%ig%ie%if%dc%hd%hg%he%hb%ig%ie%hg%dc%fk%\n" +
		"dc%he%hv%ig%hr%hf%hu%di%if%dc%ht%hb%hn%hg%dc%\n" +
		"ig%ic%dc%ht%ik%dc%ht%hk%hu%hf%dc%ii%hj%hk%he%\n" +
		"hj%dc%hv%hh%dc%if%hj%hg%dc%hh%hk%hi%ie%dc%fk%\n" +
		"dc%ii%hv%ig%hr%hf%dc%he%hj%hv%hv%ie%hg%du"
	multilineSemi := "fk;dc;ie;hb;ii;dc;ht;ik;ie;hg;hr;hh;dc;ie;hk;\n" +
		"if;if;hk;hu;hi;dc;hk;hu;dc;if;hj;hg;dc;he;id;\n" +
		"hv;if;he;hj;dc;hv;hh;dc;if;hj;hg;dc;if;hj;hk;\n" +
		"ie;dc;hh;hk;hi;dc;if;id;hg;hg;dr;dc;ie;if;hb;\n" +
		"id;ih;hk;hu;hi;dc;if;hv;dc;hf;hg;hb;if;hj;dr;\n" +
		"dc;hl;ig;ie;if;dc;hd;hg;he;hb;ig;ie;hg;dc;fk;\n" +
		"dc;he;hv;ig;hr;hf;hu;di;if;dc;ht;hb;hn;hg;dc;\n" +
		"ig;ic;dc;ht;ik;dc;ht;hk;hu;hf;dc;ii;hj;hk;he;\n" +
		"hj;dc;hv;hh;dc;if;hj;hg;dc;hh;hk;hi;ie;dc;fk;\n" +
		"dc;ii;hv;ig;hr;hf;dc;he;hj;hv;hv;ie;hg;du"
	fig := "I saw myself sitting in the crotch of the this fig tree, starving to death, just because I couldn't make up my mind which of the figs I would choose."

	runCases(t, []opCase{
		{
			"ASCII to Modhex stream", "aberystwyth", "hbhdhgidikieifiiikifhj",
			core.Recipe{{Op: "To Modhex", Args: []any{"None", float64(0)}}},
		},
		{
			"ASCII to Modhex with colon deliminator", "aberystwyth", "hb:hd:hg:id:ik:ie:if:ii:ik:if:hj",
			core.Recipe{{Op: "To Modhex", Args: []any{"Colon", float64(0)}}},
		},
		{
			"Modhex stream to UTF-8", "uhkgkbuhkgkbugltlkugltkc", "救救孩子",
			core.Recipe{{Op: "From Modhex", Args: []any{"Auto"}}},
		},
		{
			"Mixed case Modhex stream to UTF-8", "uhKGkbUHkgkBUGltlkugltkc", "救救孩子",
			core.Recipe{{Op: "From Modhex", Args: []any{"Auto"}}},
		},
		{
			"Multiline Modhex with comma to ASCII (Auto Mode)", multiline, fig,
			core.Recipe{{Op: "From Modhex", Args: []any{"Auto"}}},
		},
		{
			"Multiline Modhex with percent to ASCII (Percent Mode)", multilinePercent, fig,
			core.Recipe{{Op: "From Modhex", Args: []any{"Percent"}}},
		},
		{
			"Multiline Modhex with semicolon to ASCII (Semi-colon Mode)", multilineSemi, fig,
			core.Recipe{{Op: "From Modhex", Args: []any{"Semi-colon"}}},
		},
		{
			"ASCII to Modhex with comma and line breaks", "aberystwyth", "hb,hd,hg,id,\nik,ie,if,ii,\nik,if,hj",
			core.Recipe{{Op: "To Modhex", Args: []any{"Comma", float64(4)}}},
		},
		{
			"Empty input through From Hex and To Modhex returns empty output", "", "",
			core.Recipe{
				{Op: "From Hex", Args: []any{"Auto"}},
				{Op: "To Modhex", Args: []any{"Space", float64(0)}},
			},
		},
		{
			"From Modhex with explicit None delimiter", "hbhdhgidikieifiiikifhj", "aberystwyth",
			core.Recipe{{Op: "From Modhex", Args: []any{"None"}}},
		},
	})
}
