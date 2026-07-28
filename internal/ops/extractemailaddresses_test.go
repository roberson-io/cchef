package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// emailRecipe builds a recipe for the operation with the arguments given.
func emailRecipe(total, sorted, unique bool) core.Recipe {
	return core.Recipe{{
		Op:   "Extract email addresses",
		Args: []any{total, sorted, unique},
	}}
}

// TestExtractEmailAddressesFixtures covers CyberChef's own cases
// (../CyberChef/tests/operations/tests/ExtractEmailAddresses.mjs).
func TestExtractEmailAddressesFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"Extract email address",
			"email@example.com\nfirstname.lastname@example.com\nemail@subdomain.example.com\nfirstname+lastname@example.com\n1234567890@example.com\nemail@example-one.com\n_______@example.com email@example.name\nemail@example.museum email@example.co.jp firstname-lastname@example.com",
			"email@example.com\nfirstname.lastname@example.com\nemail@subdomain.example.com\nfirstname+lastname@example.com\n1234567890@example.com\nemail@example-one.com\n_______@example.com\nemail@example.name\nemail@example.museum\nemail@example.co.jp\nfirstname-lastname@example.com",
			emailRecipe(false, false, false),
		},
		{
			"Extract email address - Display total",
			"email@example.com\nfirstname.lastname@example.com\nemail@subdomain.example.com\nfirstname+lastname@example.com\n1234567890@example.com\nemail@example-one.com\n_______@example.com email@example.name\nemail@example.museum email@example.co.jp firstname-lastname@example.com",
			"Total found: 11\n\nemail@example.com\nfirstname.lastname@example.com\nemail@subdomain.example.com\nfirstname+lastname@example.com\n1234567890@example.com\nemail@example-one.com\n_______@example.com\nemail@example.name\nemail@example.museum\nemail@example.co.jp\nfirstname-lastname@example.com",
			emailRecipe(true, false, false),
		},
		{
			"Extract email address (Internationalized)",
			"伊昭傑@郵件.商務 ाम@मोहन.ईन्फो\nюзер@екзампл.ком θσερ@εχαμπλε.ψομ JosễSilvễ@googlễ.com\nJosễSilvễ@google.com and JosễSilva@google.com\nFoO@BaR.CoM, john@192.168.10.100\ngómez@junk.br and Abc.123@example.com.\nuser+mailbox/department=shipping@example.com\n用户@例子.广告\nउपयोगकर्ता@उदाहरण.कॉम\nюзер@екзампл.ком\nθσερ@εχαμπλε.ψομ\nDörte@Sörensen.example.com\nаджай@экзампл.рус\ntest@xn--bcher-kva.com",
			"伊昭傑@郵件.商務\nाम@मोहन.ईन्फो\nюзер@екзампл.ком\nθσερ@εχαμπλε.ψομ\nJosễSilvễ@googlễ.com\nJosễSilvễ@google.com\nJosễSilva@google.com\nFoO@BaR.CoM\njohn@192.168.10.100\ngómez@junk.br\nAbc.123@example.com\nuser+mailbox/department=shipping@example.com\n用户@例子.广告\nउपयोगकर्ता@उदाहरण.कॉम\nюзер@екзампл.ком\nθσερ@εχαμπλε.ψομ\nDörte@Sörensen.example.com\nаджай@экзампл.рус\ntest@xn--bcher-kva.com",
			emailRecipe(false, false, false),
		},
		{
			"Extract email address - Display total (Internationalized)",
			"伊昭傑@郵件.商務 ाम@मोहन.ईन्फो\nюзер@екзампл.ком θσερ@εχαμπλε.ψομ JosễSilvễ@googlễ.com\nJosễSilvễ@google.com and JosễSilva@google.com\nFoO@BaR.CoM, john@192.168.10.100\ngómez@junk.br and Abc.123@example.com.\nuser+mailbox/department=shipping@example.com\n用户@例子.广告\nउपयोगकर्ता@उदाहरण.कॉम\nюзер@екзампл.ком\nθσερ@εχαμπλε.ψομ\nDörte@Sörensen.example.com\nаджай@экзампл.рус\ntest@xn--bcher-kva.com",
			"Total found: 19\n\n伊昭傑@郵件.商務\nाम@मोहन.ईन्फो\nюзер@екзампл.ком\nθσερ@εχαμπλε.ψομ\nJosễSilvễ@googlễ.com\nJosễSilvễ@google.com\nJosễSilva@google.com\nFoO@BaR.CoM\njohn@192.168.10.100\ngómez@junk.br\nAbc.123@example.com\nuser+mailbox/department=shipping@example.com\n用户@例子.广告\nउपयोगकर्ता@उदाहरण.कॉम\nюзер@екзампл.ком\nθσερ@εχαμπλε.ψομ\nDörte@Sörensen.example.com\nаджай@экзампл.рус\ntest@xn--bcher-kva.com",
			emailRecipe(true, false, false),
		},
		{
			"Extract email address - IP address",
			"yaunwfkb\nexample@[127.0.0.1]\n091nvka",
			"example@[127.0.0.1]",
			emailRecipe(false, false, false),
		},
		{
			"Extract email address - invalid IP address",
			"yaunwfkb\nfalse_positive@[1.2.3.]\n091nvka",
			"",
			emailRecipe(false, false, false),
		},
	})
}

// TestExtractEmailAddressesOptions covers the switches the fixtures leave out,
// each expectation taken from the CyberChef-server oracle.
func TestExtractEmailAddressesOptions(t *testing.T) {
	runCases(t, []opCase{
		{
			"sorted",
			"Zoe@Example.com, adam@example.com and zoe@example.com again; bob@sub.example.org",
			"adam@example.com\nbob@sub.example.org\nZoe@Example.com\nzoe@example.com",
			emailRecipe(false, true, false),
		},
		{
			"unique",
			"Zoe@Example.com, adam@example.com and zoe@example.com again; bob@sub.example.org",
			"Zoe@Example.com\nadam@example.com\nzoe@example.com\nbob@sub.example.org",
			emailRecipe(false, false, true),
		},
		{
			"everything on",
			"Zoe@Example.com, adam@example.com and zoe@example.com again; bob@sub.example.org",
			"Total found: 4\n\nadam@example.com\nbob@sub.example.org\nZoe@Example.com\nzoe@example.com",
			emailRecipe(true, true, true),
		},
		{
			"nothing found",
			"no addresses here",
			"",
			emailRecipe(false, false, false),
		},
		{
			"nothing found with total",
			"no addresses here",
			"Total found: 0\n\n",
			emailRecipe(true, false, false),
		},
		{
			"quoted local part",
			"send to \"very.unusual@strange\"@example.com ok",
			"\"very.unusual@strange\"@example.com",
			emailRecipe(false, false, false),
		},
	})
}
