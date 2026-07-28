package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// domainRecipe builds a recipe for the operation with the arguments given.
func domainRecipe(total, sorted, unique, underscore bool) core.Recipe {
	return core.Recipe{{
		Op:   "Extract domains",
		Args: []any{total, sorted, unique, underscore},
	}}
}

// TestExtractDomains covers the switches and the shape of a domain, each
// expectation taken from the CyberChef-server oracle.
func TestExtractDomains(t *testing.T) {
	runCases(t, []opCase{
		{
			"plain",
			"Visit www.example.com or example.co.uk today.\nmail.subdomain.EXAMPLE.COM and xn--bcher-kva.de\nnot a domain: foo.c, 1.2.3.4, _dmarc.example.com\nlocalhost, http://test.example.org/path?a=b\na-b-c.example.museum\n",
			"www.example.com\nexample.co.uk\nmail.subdomain.EXAMPLE.COM\nxn--bcher-kva.de\nexample.com\ntest.example.org\na-b-c.example.museum",
			domainRecipe(false, false, false, false),
		},
		{
			"display total",
			"Visit www.example.com or example.co.uk today.\nmail.subdomain.EXAMPLE.COM and xn--bcher-kva.de\nnot a domain: foo.c, 1.2.3.4, _dmarc.example.com\nlocalhost, http://test.example.org/path?a=b\na-b-c.example.museum\n",
			"Total found: 7\n\nwww.example.com\nexample.co.uk\nmail.subdomain.EXAMPLE.COM\nxn--bcher-kva.de\nexample.com\ntest.example.org\na-b-c.example.museum",
			domainRecipe(true, false, false, false),
		},
		{
			"sorted",
			"Visit www.example.com or example.co.uk today.\nmail.subdomain.EXAMPLE.COM and xn--bcher-kva.de\nnot a domain: foo.c, 1.2.3.4, _dmarc.example.com\nlocalhost, http://test.example.org/path?a=b\na-b-c.example.museum\n",
			"a-b-c.example.museum\nexample.co.uk\nexample.com\nmail.subdomain.EXAMPLE.COM\ntest.example.org\nwww.example.com\nxn--bcher-kva.de",
			domainRecipe(false, true, false, false),
		},
		{
			"underscore",
			"Visit www.example.com or example.co.uk today.\nmail.subdomain.EXAMPLE.COM and xn--bcher-kva.de\nnot a domain: foo.c, 1.2.3.4, _dmarc.example.com\nlocalhost, http://test.example.org/path?a=b\na-b-c.example.museum\n",
			"www.example.com\nexample.co.uk\nmail.subdomain.EXAMPLE.COM\nxn--bcher-kva.de\n_dmarc.example.com\ntest.example.org\na-b-c.example.museum",
			domainRecipe(false, false, false, true),
		},
		{
			"everything on",
			"Visit www.example.com or example.co.uk today.\nmail.subdomain.EXAMPLE.COM and xn--bcher-kva.de\nnot a domain: foo.c, 1.2.3.4, _dmarc.example.com\nlocalhost, http://test.example.org/path?a=b\na-b-c.example.museum\n",
			"Total found: 7\n\n_dmarc.example.com\na-b-c.example.museum\nexample.co.uk\nmail.subdomain.EXAMPLE.COM\ntest.example.org\nwww.example.com\nxn--bcher-kva.de",
			domainRecipe(true, true, true, true),
		},
		{
			"unique",
			"example.com and Example.com and example.com again, other.net",
			"example.com\nExample.com\nother.net",
			domainRecipe(false, false, true, false),
		},
		{
			"unique keeps the case of each",
			"example.com and Example.com and example.com again, other.net",
			"Total found: 3\n\nexample.com\nExample.com\nother.net",
			domainRecipe(true, false, true, false),
		},
		{
			"label and TLD limits",
			"a.b ab.cd foo.museum -bad.com bad-.com x--y.com 1234.com\nunder_score.com dash-.com .leading.com trailing.com.\nllllllllllllllllllllllllllllllllllllllllllllllllllllllllllllllll.com sssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssss.com\n",
			"ab.cd\nfoo.museum\nbad.com\ny.com\n1234.com\nleading.com\ntrailing.com\nsssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssss.com",
			domainRecipe(false, false, false, false),
		},
		{
			"underscore label limits",
			"a.b ab.cd foo.museum -bad.com bad-.com x--y.com 1234.com\nunder_score.com dash-.com .leading.com trailing.com.\nllllllllllllllllllllllllllllllllllllllllllllllllllllllllllllllll.com sssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssss.com\n",
			"ab.cd\nfoo.museum\nbad.com\ny.com\n1234.com\nunder_score.com\nleading.com\ntrailing.com\nsssssssssssssssssssssssssssssssssssssssssssssssssssssssssssssss.com",
			domainRecipe(false, false, false, true),
		},
		{
			"nothing found",
			"nothing here at all",
			"",
			domainRecipe(false, false, false, false),
		},
		{
			"nothing found with total",
			"nothing here at all",
			"Total found: 0\n\n",
			domainRecipe(true, false, false, false),
		},
	})
}
