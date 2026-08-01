package ops

import (
	"testing"

	"github.com/roberson-io/cchef/core"
)

// Expected digests transcribed from CyberChef tests/operations/tests/Hash.mjs
// (input "Hello, World!").
func TestHashFixtures(t *testing.T) {
	const in = "Hello, World!"
	runCases(t, []opCase{
		{
			"MD5", in, "65a8e27d8879283831b664bd8b7f0ad4",
			core.Recipe{{Op: "MD5"}},
		},
		{
			"SHA1", in, "0a0a9f2a6772942557ab5355d76af442f8f65e01",
			core.Recipe{{Op: "SHA1"}},
		},
		{
			"SHA224", in, "72a23dfa411ba6fde01dbfabf3b00a709c93ebf273dc29e2d8b261ff",
			core.Recipe{{Op: "SHA224"}},
		},
		{
			"SHA256", in, "dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f",
			core.Recipe{{Op: "SHA256"}},
		},
		{
			"SHA384", in, "5485cc9b3365b4305dfb4e8337e0a598a574f8242bf17289e0dd6c20a3cd44a089de16ab4ab308f63e44b1170eb5f515",
			core.Recipe{{Op: "SHA384"}},
		},
		{
			"SHA512", in, "374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387",
			core.Recipe{{Op: "SHA512"}},
		},
	})
}
