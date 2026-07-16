package ops

import (
	"crypto/aes"
	"encoding/hex"
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// aesEnc builds an AES Encrypt recipe with the standard argument order:
// Key, IV, Mode, Input, Output, AAD, Include IV in output.
func aesEnc(key, iv core.ToggleString, mode, in, out string, aad core.ToggleString, includeIV string) core.Recipe {
	return core.Recipe{{Op: "AES Encrypt", Args: []any{key, iv, mode, in, out, aad, includeIV}}}
}

// TestAESEncryptFixtures transcribes the AES Encrypt cases from
// ../CyberChef/tests/operations/tests/Crypt.mjs.
func TestAESEncryptFixtures(t *testing.T) {
	none := aesTS("Hex", "")
	k128 := aesTS("Hex", "00112233445566778899aabbccddeeff")
	iv0 := aesTS("Hex", "00000000000000000000000000000000")
	iv1 := aesTS("Hex", "00112233445566778899aabbccddeeff")
	fox := "The quick brown fox jumps over the lazy dog."

	// Binary fixtures share this input and IV.
	bin := "7a0e643132750e96d805d11e9e48e281fa39a41039286423cc1c045e5442b40bf1c3f2822bded3f9c8ef11cb25da64dda9c7ab87c246bd305385150c98f31465c2a6180fe81d31ea289b916504d5a12e1de26cb10adba84a0cb0c86f94bc14bc554f3018"
	kb128 := aesTS("Hex", "51e201d463698ef5f717f71f5b4712af")
	kb192 := aesTS("Hex", "6801ed503c9d96ee5f9d78b07ab1b295dba3c2adf81c7816")
	kb256 := aesTS("Hex", "2d767f6e9333d1c77581946e160b2b7368c2cdd5e2b80f04ca09d64e02afbfe1")
	ivb := aesTS("Hex", "1748e7179bd56570d51fa4ba287cc3e5")
	aad := aesTS("UTF8", "additional data")

	runCases(t, []opCase{
		{
			"AES-128-CBC IV0 ASCII", fox,
			"2ef6c3fdb1314b5c2c326a2087fe1a82d5e73bf605ec8431d73e847187fc1c8fbbe969c177df1ecdf8c13f2f505f9498",
			aesEnc(k128, iv0, "CBC", "Raw", "Hex", none, "Off"),
		},
		{
			"AES-128-CTR IV0 ASCII", fox,
			"a98c9e8e3b7c894384d740e4f0f4ed0be2bbb1e0e13a255812c3c6b0a629e4ad759c075b2469c6f4fb2c0cf9",
			aesEnc(k128, iv0, "CTR", "Raw", "Hex", none, "Off"),
		},
		{
			"AES-128-CBC IV1 ASCII", fox,
			"4fa077d50cc71a57393e7b542c4e3aea0fb75383b97083f2f568ffc13c0e7a47502ec6d9f25744a061a3a5e55fe95e8d",
			aesEnc(k128, iv1, "CBC", "Raw", "Hex", none, "Off"),
		},
		{
			"AES-128-CFB ASCII", fox,
			"369e1c9e5a85b0520f3e61eecc37759246ad0a02cae7a99a3d250ae39cad4743385375cf63720d52ae8cdfb9",
			aesEnc(k128, iv1, "CFB", "Raw", "Hex", none, "Off"),
		},
		{
			"AES-128-OFB ASCII", fox,
			"369e1c9e5a85b0520f3e61eecc37759288cb378c5fa9c675bd6c4ede0ae6a925eaebc8e0a6162d2a000ddc0f",
			aesEnc(k128, iv1, "OFB", "Raw", "Hex", none, "Off"),
		},
		{
			"AES-128-CTR ASCII", fox,
			"369e1c9e5a85b0520f3e61eecc37759206f6f1ba63527af96fae3b15a921844df2e542902a4f0525dbb4146b",
			aesEnc(k128, iv1, "CTR", "Raw", "Hex", none, "Off"),
		},
		{
			"AES-128-ECB ASCII", fox,
			"2ef6c3fdb1314b5c2c326a2087fe1a8238c5a5db7dff38f6f4eb75b2e55cab3d8d6113eb8d3517223b4545fcdb4c5a48",
			aesEnc(k128, none, "ECB", "Raw", "Hex", none, "Off"),
		},
		{
			"AES-128-GCM ASCII", fox,
			"d0bcace0fa3a214b0ac3cbb4ac2caaf97b965f172f66d2a4ec6304a15a4072f1b28a6f9b80473f86bfa47b2c\n\nTag: 16a3e732a605cc9ca29108f742ca0743",
			aesEnc(k128, none, "GCM", "Raw", "Hex", none, "Off"),
		},
		{
			"AES-128-GCM ASCII AAD", fox,
			"daa58faa056c52756aa488aeafbd265b6effcf4eca58220a97b0005b1a9b1e1c9e7a6725d35f5f79b9493de7\n\nTag: 3b5378917f67b0aade9891fc6c291646",
			aesEnc(k128, aesTS("Hex", "ffeeddccbbaa99887766554433221100"), "GCM", "Raw", "Hex", aad, "Off"),
		},

		{
			"AES-128-CBC Binary", bin,
			"bf2ccb148e5df181a46f39764047e24fc94cc46bbe6c8d160fc25a977e4b630883e9e04d3eeae3ccbb2d57a4c22e61909f2b6d7b24940abe95d356ce986294270d0513e0ffe7a9928fa6669e1aaae4379310281dc27c0bb9e254684b2ecd7f5f944c8218f3bc680570399a508dfe4b65",
			aesEnc(kb128, ivb, "CBC", "Hex", "Hex", none, "Off"),
		},
		{
			"AES-128-CFB Binary", bin,
			"17211941bb2fa43d54d9fa59072436422a55be7a2be164cf5ec4e50e7a0035094ab684dab8d45a4515ae95c4136ded98898f74d4ecc4ac57ae682a985031ecb7518ddea6c8d816349801aa22ff0b6ac1784d169060efcd9fb77d564477038eb09bb4e1ce",
			aesEnc(kb128, ivb, "CFB", "Hex", "Hex", none, "Off"),
		},
		{
			"AES-128-OFB Binary", bin,
			"17211941bb2fa43d54d9fa5907243642bfd805201c130c8600566720cf87562011f0872598f1e69cfe541bb864de7ed68201e0a34284157b581984dab3fe2cb0f20cb80d0046740df3e149ec4c92c0e81f2dc439a6f3a05c5ef505eae6308b301c673cfa",
			aesEnc(kb128, ivb, "OFB", "Hex", "Hex", none, "Off"),
		},
		{
			"AES-128-CTR Binary", bin,
			"17211941bb2fa43d54d9fa5907243642baf08c837003bf24d7b81a911ce41bd31de8a92f6dc6d11135b70c73ea167c3fc4ea78234f58652d25e23245dbcb895bf4165092d0515ae8f14230f8a34b06957f24ba4b24db741490e7edcd6e5310945cc159fc",
			aesEnc(kb128, ivb, "CTR", "Hex", "Hex", none, "Off"),
		},
		{
			"AES-128-GCM Binary", bin,
			"5a29debb5c5f38cdf8aee421bd94dbbf3399947faddf205f88b3ad8ecb0c51214ec0e28bf78942dfa212d7eb15259bbdcac677b4c05f473eeb9331d74f31d441d97d56eb5c73b586342d72128ca528813543dc0fc7eddb7477172cc9194c18b2e1383e4e\n\nTag: 70fad2ca19412c20f40fd06918736e56",
			aesEnc(kb128, ivb, "GCM", "Hex", "Hex", none, "Off"),
		},
		{
			"AES-128-GCM Binary AAD", bin,
			"5a29debb5c5f38cdf8aee421bd94dbbf3399947faddf205f88b3ad8ecb0c51214ec0e28bf78942dfa212d7eb15259bbdcac677b4c05f473eeb9331d74f31d441d97d56eb5c73b586342d72128ca528813543dc0fc7eddb7477172cc9194c18b2e1383e4e\n\nTag: 61cc4b70809452b0b3e38f913fa0a109",
			aesEnc(kb128, ivb, "GCM", "Hex", "Hex", aad, "Off"),
		},
		{
			"AES-128-ECB Binary", bin,
			"869c057637a58cc3363bcc4bcfa62702abf85dff44300eb9fdcfb9d845772c8acb557c8d540baae2489c6758abef83d81b74239bef87c6c944c1b00ca160882bc15be9a6a3de4e6a50a2eab8b635c634027ed7eae4c1d2f08477c38b7dc24f6915da235bc3051f3a50736b14db8863e4",
			aesEnc(kb128, ivb, "ECB", "Hex", "Hex", none, "Off"),
		},

		{
			"AES-192-CBC Binary", bin,
			"1aec90cd7f629ef68243881f3e2b793a548cbcdad69631995a6bd0c8aea1e948d8a5f3f2b7e7f9b77da77434c92a6257a9f57e937b883f4400511b990888a0b1d27c0a4b7f298e6f50b563135edc9fa7d8eceb6bc8163e6153a20cf07aa1e705bc5cb3a37b0452b4019cef8000d7c1b7",
			aesEnc(kb192, ivb, "CBC", "Hex", "Hex", none, "Off"),
		},
		{
			"AES-192-CFB Binary", bin,
			"fc370a6c013b3c05430fbce810cb97d39cb0a587320a4c1b57d0c0d08e93cb0d1221abba9df09b4b1332ce923b289f92000e6b4f7fbc55dfdab9179081d8c36ef4a0e3d3a49f1564715c5d3e88f8bf6d3dd77944f22f99a03b5535a3cd47bc44d4a9665c",
			aesEnc(kb192, ivb, "CFB", "Hex", "Hex", none, "Off"),
		},
		{
			"AES-192-OFB Binary", bin,
			"fc370a6c013b3c05430fbce810cb97d33605d11b2531c8833bc3e818003bbd7dd58b2a38d10d44d25d11bd96228b264a4d2aad1d0a7af2cfad0e70c1ade305433e95cb0ee693447f6877a59a4be5c070d19afba23ff10caf5ecfa7a9c2877b8df23d61f2",
			aesEnc(kb192, ivb, "OFB", "Hex", "Hex", none, "Off"),
		},
		{
			"AES-192-CTR Binary", bin,
			"fc370a6c013b3c05430fbce810cb97d340525303ae59c5e9b73ad5ff3e65ce3abf00431e0a292d990f732a397de589420827beb1c28623c56972eb2ddf0cf3f82e3c30e155df7f64a530419c28fc51a9091c73df78e73958bee1d1acd8676c9c0f1915ca",
			aesEnc(kb192, ivb, "CTR", "Hex", "Hex", none, "Off"),
		},
		{
			"AES-192-GCM Binary", bin,
			"318b479d919d506f0cd904f2676fab263a7921b6d7e0514f36e03ae2333b77fa66ef5600babcb2ee9718aeb71fc357412343c1f2cb351d8715bb0aedae4a6468124f9c4aaf6a721b306beddbe63a978bec8baeeba4b663be33ee5bc982746bd4aed1c38b\n\nTag: 86db597d5302595223cadbd990f1309b",
			aesEnc(kb192, ivb, "GCM", "Hex", "Hex", none, "Off"),
		},
		{
			"AES-192-GCM Binary AAD", bin,
			"318b479d919d506f0cd904f2676fab263a7921b6d7e0514f36e03ae2333b77fa66ef5600babcb2ee9718aeb71fc357412343c1f2cb351d8715bb0aedae4a6468124f9c4aaf6a721b306beddbe63a978bec8baeeba4b663be33ee5bc982746bd4aed1c38b\n\nTag: aeedf3e6ca4201577c0cf3e9ce58159d",
			aesEnc(kb192, ivb, "GCM", "Hex", "Hex", aad, "Off"),
		},
		{
			"AES-192-ECB Binary", bin,
			"56ef533db50a3b33951a76acede52b7d54fbae7fb07da20daa3e2731e5721ee4c13ab15ac80748c14dece982310530ad65480512a4cf70201473fb7bc3480446bc86b1ff9b4517c4c1f656bc236fab1aca276ae5af25f5871b671823f3cb3e426da059dd83a13f125bd6cfe600c331b0",
			aesEnc(kb192, ivb, "ECB", "Hex", "Hex", none, "Off"),
		},

		{
			"AES-256-CBC Binary", bin,
			"bc60a7613559e23e8a7be8e98a1459003fdb036f33368d8a30156c51464b49472705a4ddae05da96956ce058bb180dd301c5fd58bf6a2ded0d7dd4da85fd5ba43a4297691532bf7f4cd92bfcfd3704faf2f9bd5425049b34433ba90fb85c80646e6cb09ee4e4059e7cd753a2fef8bbad",
			aesEnc(kb256, ivb, "CBC", "Hex", "Hex", none, "Off"),
		},
		{
			"AES-256-CFB Binary", bin,
			"5dc73709da5cb0ac914ae4bcb621fd75169eac5ff13a2dde573f6380ff812e8ddb58f0e9afaec1ff0d6d2af0659e10c05b714ec97481a15f4a7aeb4c6ea84112ce897459b54ed9e77a794f023f2bef1901f013cf435432fca5fb59e2be781916247d2334",
			aesEnc(kb256, ivb, "CFB", "Hex", "Hex", none, "Off"),
		},
		{
			"AES-256-OFB Binary", bin,
			"5dc73709da5cb0ac914ae4bcb621fd75b6e1f909b88733f784b1df8a52dc200440a1076415d009a7c12cac1e8ab76bdc290e6634cd5bf8a416fda8dcfd7910e55fe9d1148cd85d7a59adad39ab089e111d8f8da246e2e874cf5d9ab7552af6308320a5ab",
			aesEnc(kb256, ivb, "OFB", "Hex", "Hex", none, "Off"),
		},
		{
			"AES-256-CTR Binary", bin,
			"5dc73709da5cb0ac914ae4bcb621fd7591356d4169898c986a90b193f4d1f0d5cba1d10b2bfc5aee8a48dce9dba174cecf56f92dddf7eb306d78360000eea7bcb50f696d84a3757a822800ed68f9edf118dc61406bacf64f022717d8cb6010049bf75d7e",
			aesEnc(kb256, ivb, "CTR", "Hex", "Hex", none, "Off"),
		},
		{
			"AES-256-GCM Binary", bin,
			"1287f188ad4d7ab0d9ff69b3c29cb11f861389532d8cb9337181da2e8cfc74a84927e8c0dd7a28a32fd485afe694259a63c199b199b95edd87c7aa95329feac340f2b78b72956a85f367044d821766b1b7135815571df44900695f1518cf3ae38ecb650f\n\nTag: 821b1e5f32dad052e502775a523d957a",
			aesEnc(kb256, ivb, "GCM", "Hex", "Hex", none, "Off"),
		},
		{
			"AES-256-GCM Binary AAD", bin,
			"1287f188ad4d7ab0d9ff69b3c29cb11f861389532d8cb9337181da2e8cfc74a84927e8c0dd7a28a32fd485afe694259a63c199b199b95edd87c7aa95329feac340f2b78b72956a85f367044d821766b1b7135815571df44900695f1518cf3ae38ecb650f\n\nTag: a8f04c4d93bbef82bef61a103371aef9",
			aesEnc(kb256, ivb, "GCM", "Hex", "Hex", aad, "Off"),
		},
		{
			"AES-256-ECB Binary", bin,
			"7e8521ba3f356ef692a51841807e141464aadc07bbc0ef2b628b8745bae356d245682a220688afca7be987b60cb120681ed42680ee93a67065619a3beaac11111a6cd88a6afa9e367722cb57df343f8548f2d691b295184da4ed5f3b763aaa8558502cb348ab58e81986337096e90caa",
			aesEnc(kb256, ivb, "ECB", "Hex", "Hex", none, "Off"),
		},
		{
			"AES-256-GCM Binary AAD prepend IV", bin,
			"1748e7179bd56570d51fa4ba287cc3e51287f188ad4d7ab0d9ff69b3c29cb11f861389532d8cb9337181da2e8cfc74a84927e8c0dd7a28a32fd485afe694259a63c199b199b95edd87c7aa95329feac340f2b78b72956a85f367044d821766b1b7135815571df44900695f1518cf3ae38ecb650f\n\nTag: a8f04c4d93bbef82bef61a103371aef9",
			aesEnc(kb256, ivb, "GCM", "Hex", "Hex", aad, "Prepend"),
		},
		{
			"AES-256-GCM Binary AAD append IV", bin,
			"1287f188ad4d7ab0d9ff69b3c29cb11f861389532d8cb9337181da2e8cfc74a84927e8c0dd7a28a32fd485afe694259a63c199b199b95edd87c7aa95329feac340f2b78b72956a85f367044d821766b1b7135815571df44900695f1518cf3ae38ecb650f1748e7179bd56570d51fa4ba287cc3e5\n\nTag: a8f04c4d93bbef82bef61a103371aef9",
			aesEnc(kb256, ivb, "GCM", "Hex", "Hex", aad, "Append"),
		},
	})
}

// TestAESEncryptNoKey verifies the invalid-key-length error message.
func TestAESEncryptNoKey(t *testing.T) {
	_, err := runOp(t, "AES Encrypt", "", aesTS("Hex", ""), aesTS("Hex", ""), "CBC", "Raw", "Hex", aesTS("Hex", ""), "Off")
	want := `Invalid key length: 0 bytes

The following algorithms will be used based on the size of the key:
  16 bytes = AES-128
  24 bytes = AES-192
  32 bytes = AES-256`
	if err == nil || err.Error() != want {
		t.Fatalf("got err %v, want %q", err, want)
	}
}

// aesDec builds an AES Decrypt recipe with the standard argument order:
// Key, IV, IV Length, Mode, Input, Output, GCM Tag, AAD, IV from input.
func aesDec(key, iv core.ToggleString, ivLen int, mode, in, out string, tag, aad core.ToggleString, ivFrom string) core.Recipe {
	return core.Recipe{{Op: "AES Decrypt", Args: []any{key, iv, ivLen, mode, in, out, tag, aad, ivFrom}}}
}

// TestAESDecryptFixtures transcribes the AES Decrypt cases from
// ../CyberChef/tests/operations/tests/Crypt.mjs.
func TestAESDecryptFixtures(t *testing.T) {
	none := aesTS("Hex", "")
	k128 := aesTS("Hex", "00112233445566778899aabbccddeeff")
	iv0 := aesTS("Hex", "00000000000000000000000000000000")
	iv1 := aesTS("Hex", "00112233445566778899aabbccddeeff")
	fox := "The quick brown fox jumps over the lazy dog."

	bin := "7a0e643132750e96d805d11e9e48e281fa39a41039286423cc1c045e5442b40bf1c3f2822bded3f9c8ef11cb25da64dda9c7ab87c246bd305385150c98f31465c2a6180fe81d31ea289b916504d5a12e1de26cb10adba84a0cb0c86f94bc14bc554f3018"
	kb128 := aesTS("Hex", "51e201d463698ef5f717f71f5b4712af")
	kb192 := aesTS("Hex", "6801ed503c9d96ee5f9d78b07ab1b295dba3c2adf81c7816")
	kb256 := aesTS("Hex", "2d767f6e9333d1c77581946e160b2b7368c2cdd5e2b80f04ca09d64e02afbfe1")
	ivb := aesTS("Hex", "1748e7179bd56570d51fa4ba287cc3e5")
	aad := aesTS("UTF8", "additional data")

	runCases(t, []opCase{
		{
			"AES-128-CBC IV0 ASCII",
			"2ef6c3fdb1314b5c2c326a2087fe1a82d5e73bf605ec8431d73e847187fc1c8fbbe969c177df1ecdf8c13f2f505f9498", fox,
			aesDec(k128, iv0, 16, "CBC", "Hex", "Raw", none, none, "Off"),
		},
		{
			"AES-128-CTR IV0 ASCII",
			"a98c9e8e3b7c894384d740e4f0f4ed0be2bbb1e0e13a255812c3c6b0a629e4ad759c075b2469c6f4fb2c0cf9", fox,
			aesDec(k128, iv0, 16, "CTR", "Hex", "Raw", none, none, "Off"),
		},
		{
			"AES-128-CBC IV1 ASCII",
			"4fa077d50cc71a57393e7b542c4e3aea0fb75383b97083f2f568ffc13c0e7a47502ec6d9f25744a061a3a5e55fe95e8d", fox,
			aesDec(k128, iv1, 16, "CBC", "Hex", "Raw", none, none, "Off"),
		},
		{
			"AES-128-CFB ASCII",
			"369e1c9e5a85b0520f3e61eecc37759246ad0a02cae7a99a3d250ae39cad4743385375cf63720d52ae8cdfb9", fox,
			aesDec(k128, iv1, 16, "CFB", "Hex", "Raw", none, none, "Off"),
		},
		{
			"AES-128-OFB ASCII",
			"369e1c9e5a85b0520f3e61eecc37759288cb378c5fa9c675bd6c4ede0ae6a925eaebc8e0a6162d2a000ddc0f", fox,
			aesDec(k128, iv1, 16, "OFB", "Hex", "Raw", none, none, "Off"),
		},
		{
			"AES-128-CTR ASCII",
			"369e1c9e5a85b0520f3e61eecc37759206f6f1ba63527af96fae3b15a921844df2e542902a4f0525dbb4146b", fox,
			aesDec(k128, iv1, 16, "CTR", "Hex", "Raw", none, none, "Off"),
		},
		{
			"AES-128-ECB ASCII",
			"2ef6c3fdb1314b5c2c326a2087fe1a8238c5a5db7dff38f6f4eb75b2e55cab3d8d6113eb8d3517223b4545fcdb4c5a48", fox,
			aesDec(k128, none, 16, "ECB", "Hex", "Raw", none, none, "Off"),
		},
		{
			"AES-128-GCM ASCII",
			"d0bcace0fa3a214b0ac3cbb4ac2caaf97b965f172f66d2a4ec6304a15a4072f1b28a6f9b80473f86bfa47b2c", fox,
			aesDec(k128, none, 16, "GCM", "Hex", "Raw", aesTS("Hex", "16a3e732a605cc9ca29108f742ca0743"), none, "Off"),
		},
		{
			"AES-128-GCM ASCII AAD",
			"daa58faa056c52756aa488aeafbd265b6effcf4eca58220a97b0005b1a9b1e1c9e7a6725d35f5f79b9493de7", fox,
			aesDec(k128, aesTS("Hex", "ffeeddccbbaa99887766554433221100"), 16, "GCM", "Hex", "Raw",
				aesTS("Hex", "3b5378917f67b0aade9891fc6c291646"), aad, "Off"),
		},

		{
			"AES-128-CBC Binary",
			"bf2ccb148e5df181a46f39764047e24fc94cc46bbe6c8d160fc25a977e4b630883e9e04d3eeae3ccbb2d57a4c22e61909f2b6d7b24940abe95d356ce986294270d0513e0ffe7a9928fa6669e1aaae4379310281dc27c0bb9e254684b2ecd7f5f944c8218f3bc680570399a508dfe4b65", bin,
			aesDec(kb128, ivb, 16, "CBC", "Hex", "Hex", none, none, "Off"),
		},
		{
			"AES-128-CFB Binary",
			"17211941bb2fa43d54d9fa59072436422a55be7a2be164cf5ec4e50e7a0035094ab684dab8d45a4515ae95c4136ded98898f74d4ecc4ac57ae682a985031ecb7518ddea6c8d816349801aa22ff0b6ac1784d169060efcd9fb77d564477038eb09bb4e1ce", bin,
			aesDec(kb128, ivb, 16, "CFB", "Hex", "Hex", none, none, "Off"),
		},
		{
			"AES-128-OFB Binary",
			"17211941bb2fa43d54d9fa5907243642bfd805201c130c8600566720cf87562011f0872598f1e69cfe541bb864de7ed68201e0a34284157b581984dab3fe2cb0f20cb80d0046740df3e149ec4c92c0e81f2dc439a6f3a05c5ef505eae6308b301c673cfa", bin,
			aesDec(kb128, ivb, 16, "OFB", "Hex", "Hex", none, none, "Off"),
		},
		{
			"AES-128-CTR Binary",
			"17211941bb2fa43d54d9fa5907243642baf08c837003bf24d7b81a911ce41bd31de8a92f6dc6d11135b70c73ea167c3fc4ea78234f58652d25e23245dbcb895bf4165092d0515ae8f14230f8a34b06957f24ba4b24db741490e7edcd6e5310945cc159fc", bin,
			aesDec(kb128, ivb, 16, "CTR", "Hex", "Hex", none, none, "Off"),
		},
		{
			"AES-128-GCM Binary",
			"5a29debb5c5f38cdf8aee421bd94dbbf3399947faddf205f88b3ad8ecb0c51214ec0e28bf78942dfa212d7eb15259bbdcac677b4c05f473eeb9331d74f31d441d97d56eb5c73b586342d72128ca528813543dc0fc7eddb7477172cc9194c18b2e1383e4e", bin,
			aesDec(kb128, ivb, 16, "GCM", "Hex", "Hex", aesTS("Hex", "70fad2ca19412c20f40fd06918736e56"), none, "Off"),
		},
		{
			"AES-128-GCM Binary AAD",
			"5a29debb5c5f38cdf8aee421bd94dbbf3399947faddf205f88b3ad8ecb0c51214ec0e28bf78942dfa212d7eb15259bbdcac677b4c05f473eeb9331d74f31d441d97d56eb5c73b586342d72128ca528813543dc0fc7eddb7477172cc9194c18b2e1383e4e", bin,
			aesDec(kb128, ivb, 16, "GCM", "Hex", "Hex", aesTS("Hex", "61cc4b70809452b0b3e38f913fa0a109"), aad, "Off"),
		},
		{
			"AES-128-ECB Binary",
			"869c057637a58cc3363bcc4bcfa62702abf85dff44300eb9fdcfb9d845772c8acb557c8d540baae2489c6758abef83d81b74239bef87c6c944c1b00ca160882bc15be9a6a3de4e6a50a2eab8b635c634027ed7eae4c1d2f08477c38b7dc24f6915da235bc3051f3a50736b14db8863e4", bin,
			aesDec(kb128, ivb, 16, "ECB", "Hex", "Hex", none, none, "Off"),
		},

		{
			"AES-192-CBC Binary",
			"1aec90cd7f629ef68243881f3e2b793a548cbcdad69631995a6bd0c8aea1e948d8a5f3f2b7e7f9b77da77434c92a6257a9f57e937b883f4400511b990888a0b1d27c0a4b7f298e6f50b563135edc9fa7d8eceb6bc8163e6153a20cf07aa1e705bc5cb3a37b0452b4019cef8000d7c1b7", bin,
			aesDec(kb192, ivb, 16, "CBC", "Hex", "Hex", none, none, "Off"),
		},
		{
			"AES-192-CFB Binary",
			"fc370a6c013b3c05430fbce810cb97d39cb0a587320a4c1b57d0c0d08e93cb0d1221abba9df09b4b1332ce923b289f92000e6b4f7fbc55dfdab9179081d8c36ef4a0e3d3a49f1564715c5d3e88f8bf6d3dd77944f22f99a03b5535a3cd47bc44d4a9665c", bin,
			aesDec(kb192, ivb, 16, "CFB", "Hex", "Hex", none, none, "Off"),
		},
		{
			"AES-192-OFB Binary",
			"fc370a6c013b3c05430fbce810cb97d33605d11b2531c8833bc3e818003bbd7dd58b2a38d10d44d25d11bd96228b264a4d2aad1d0a7af2cfad0e70c1ade305433e95cb0ee693447f6877a59a4be5c070d19afba23ff10caf5ecfa7a9c2877b8df23d61f2", bin,
			aesDec(kb192, ivb, 16, "OFB", "Hex", "Hex", none, none, "Off"),
		},
		{
			"AES-192-CTR Binary",
			"fc370a6c013b3c05430fbce810cb97d340525303ae59c5e9b73ad5ff3e65ce3abf00431e0a292d990f732a397de589420827beb1c28623c56972eb2ddf0cf3f82e3c30e155df7f64a530419c28fc51a9091c73df78e73958bee1d1acd8676c9c0f1915ca", bin,
			aesDec(kb192, ivb, 16, "CTR", "Hex", "Hex", none, none, "Off"),
		},
		{
			"AES-192-GCM Binary",
			"318b479d919d506f0cd904f2676fab263a7921b6d7e0514f36e03ae2333b77fa66ef5600babcb2ee9718aeb71fc357412343c1f2cb351d8715bb0aedae4a6468124f9c4aaf6a721b306beddbe63a978bec8baeeba4b663be33ee5bc982746bd4aed1c38b", bin,
			aesDec(kb192, ivb, 16, "GCM", "Hex", "Hex", aesTS("Hex", "86db597d5302595223cadbd990f1309b"), none, "Off"),
		},
		{
			"AES-192-GCM Binary AAD",
			"318b479d919d506f0cd904f2676fab263a7921b6d7e0514f36e03ae2333b77fa66ef5600babcb2ee9718aeb71fc357412343c1f2cb351d8715bb0aedae4a6468124f9c4aaf6a721b306beddbe63a978bec8baeeba4b663be33ee5bc982746bd4aed1c38b", bin,
			aesDec(kb192, ivb, 16, "GCM", "Hex", "Hex", aesTS("Hex", "aeedf3e6ca4201577c0cf3e9ce58159d"), aad, "Off"),
		},
		{
			"AES-192-ECB Binary",
			"56ef533db50a3b33951a76acede52b7d54fbae7fb07da20daa3e2731e5721ee4c13ab15ac80748c14dece982310530ad65480512a4cf70201473fb7bc3480446bc86b1ff9b4517c4c1f656bc236fab1aca276ae5af25f5871b671823f3cb3e426da059dd83a13f125bd6cfe600c331b0", bin,
			aesDec(kb192, ivb, 16, "ECB", "Hex", "Hex", none, none, "Off"),
		},

		{
			"AES-256-CBC Binary",
			"bc60a7613559e23e8a7be8e98a1459003fdb036f33368d8a30156c51464b49472705a4ddae05da96956ce058bb180dd301c5fd58bf6a2ded0d7dd4da85fd5ba43a4297691532bf7f4cd92bfcfd3704faf2f9bd5425049b34433ba90fb85c80646e6cb09ee4e4059e7cd753a2fef8bbad", bin,
			aesDec(kb256, ivb, 16, "CBC", "Hex", "Hex", none, none, "Off"),
		},
		{
			"AES-256-CFB Binary",
			"5dc73709da5cb0ac914ae4bcb621fd75169eac5ff13a2dde573f6380ff812e8ddb58f0e9afaec1ff0d6d2af0659e10c05b714ec97481a15f4a7aeb4c6ea84112ce897459b54ed9e77a794f023f2bef1901f013cf435432fca5fb59e2be781916247d2334", bin,
			aesDec(kb256, ivb, 16, "CFB", "Hex", "Hex", none, none, "Off"),
		},
		{
			"AES-256-OFB Binary",
			"5dc73709da5cb0ac914ae4bcb621fd75b6e1f909b88733f784b1df8a52dc200440a1076415d009a7c12cac1e8ab76bdc290e6634cd5bf8a416fda8dcfd7910e55fe9d1148cd85d7a59adad39ab089e111d8f8da246e2e874cf5d9ab7552af6308320a5ab", bin,
			aesDec(kb256, ivb, 16, "OFB", "Hex", "Hex", none, none, "Off"),
		},
		{
			"AES-256-CTR Binary",
			"5dc73709da5cb0ac914ae4bcb621fd7591356d4169898c986a90b193f4d1f0d5cba1d10b2bfc5aee8a48dce9dba174cecf56f92dddf7eb306d78360000eea7bcb50f696d84a3757a822800ed68f9edf118dc61406bacf64f022717d8cb6010049bf75d7e", bin,
			aesDec(kb256, ivb, 16, "CTR", "Hex", "Hex", none, none, "Off"),
		},
		{
			"AES-256-GCM Binary",
			"1287f188ad4d7ab0d9ff69b3c29cb11f861389532d8cb9337181da2e8cfc74a84927e8c0dd7a28a32fd485afe694259a63c199b199b95edd87c7aa95329feac340f2b78b72956a85f367044d821766b1b7135815571df44900695f1518cf3ae38ecb650f", bin,
			aesDec(kb256, ivb, 16, "GCM", "Hex", "Hex", aesTS("Hex", "821b1e5f32dad052e502775a523d957a"), none, "Off"),
		},
		{
			"AES-256-GCM Binary AAD",
			"1287f188ad4d7ab0d9ff69b3c29cb11f861389532d8cb9337181da2e8cfc74a84927e8c0dd7a28a32fd485afe694259a63c199b199b95edd87c7aa95329feac340f2b78b72956a85f367044d821766b1b7135815571df44900695f1518cf3ae38ecb650f", bin,
			aesDec(kb256, ivb, 16, "GCM", "Hex", "Hex", aesTS("Hex", "a8f04c4d93bbef82bef61a103371aef9"), aad, "Off"),
		},
		{
			"AES-256-ECB Binary",
			"7e8521ba3f356ef692a51841807e141464aadc07bbc0ef2b628b8745bae356d245682a220688afca7be987b60cb120681ed42680ee93a67065619a3beaac11111a6cd88a6afa9e367722cb57df343f8548f2d691b295184da4ed5f3b763aaa8558502cb348ab58e81986337096e90caa", bin,
			aesDec(kb256, ivb, 16, "ECB", "Hex", "Hex", none, none, "Off"),
		},

		{
			"AES-256-ECB IV from input start",
			"1748e7179bd56570d51fa4ba287cc3e57e8521ba3f356ef692a51841807e141464aadc07bbc0ef2b628b8745bae356d245682a220688afca7be987b60cb120681ed42680ee93a67065619a3beaac11111a6cd88a6afa9e367722cb57df343f8548f2d691b295184da4ed5f3b763aaa8558502cb348ab58e81986337096e90caa", bin,
			aesDec(kb256, none, 16, "ECB", "Hex", "Hex", none, none, "From start"),
		},
		{
			"AES-256-ECB IV from input end",
			"7e8521ba3f356ef692a51841807e141464aadc07bbc0ef2b628b8745bae356d245682a220688afca7be987b60cb120681ed42680ee93a67065619a3beaac11111a6cd88a6afa9e367722cb57df343f8548f2d691b295184da4ed5f3b763aaa8558502cb348ab58e81986337096e90caa1748e7179bd56570d51fa4ba287cc3e5", bin,
			aesDec(kb256, none, 16, "ECB", "Hex", "Hex", none, none, "From end"),
		},
		{
			"AES-256-GCM IV from input start AAD",
			"1748e7179bd56570d51fa4ba287cc3e51287f188ad4d7ab0d9ff69b3c29cb11f861389532d8cb9337181da2e8cfc74a84927e8c0dd7a28a32fd485afe694259a63c199b199b95edd87c7aa95329feac340f2b78b72956a85f367044d821766b1b7135815571df44900695f1518cf3ae38ecb650f", bin,
			aesDec(kb256, none, 16, "GCM", "Hex", "Hex", aesTS("Hex", "a8f04c4d93bbef82bef61a103371aef9"), aad, "From start"),
		},
		{
			"AES-256-GCM 12-byte IV from input start AAD",
			"1748e7179bd56570d51fa4ba623c81f4605da9ac3df29c67c43abe4aad5230dca82a98ab31f042fe871b81a0a1e8b8af41044d46f627828e7d11eca2d04ac27f4e7c7c9a20da87854df9868a2ddbd67d85f7db92f9ff1272cfb7955a2d279dbe715965011fddf6e730e79e7b22f89817", bin,
			aesDec(kb256, none, 12, "GCM", "Hex", "Hex", aesTS("Hex", "c311c9144f8ae145ec46e2c69179a4b7"), aad, "From start"),
		},
	})
}

// TestAESDecryptErrors covers the invalid-key and too-short-input error paths.
func TestAESDecryptErrors(t *testing.T) {
	t.Run("no key", func(t *testing.T) {
		_, err := runOp(t, "AES Decrypt", "", aesTS("Hex", ""), aesTS("Hex", ""), 16, "CBC", "Hex", "Raw",
			aesTS("Hex", ""), aesTS("Hex", ""), "Off")
		want := `Invalid key length: 0 bytes

The following algorithms will be used based on the size of the key:
  16 bytes = AES-128
  24 bytes = AES-192
  32 bytes = AES-256`
		if err == nil || err.Error() != want {
			t.Fatalf("got err %v, want %q", err, want)
		}
	})
	t.Run("iv from input too short", func(t *testing.T) {
		_, err := runOp(t, "AES Decrypt", "1748e7179bd56570d51fa4ba287cc3e5",
			aesTS("Hex", "2d767f6e9333d1c77581946e160b2b7368c2cdd5e2b80f04ca09d64e02afbfe1"), aesTS("Hex", ""), 16,
			"ECB", "Hex", "Hex", aesTS("Hex", ""), aesTS("Hex", ""), "From start")
		want := "Input is too short to contain an IV of 16 bytes."
		if err == nil || err.Error() != want {
			t.Fatalf("got err %v, want %q", err, want)
		}
	})
	t.Run("bad GCM tag", func(t *testing.T) {
		_, err := runOp(t, "AES Decrypt",
			"d0bcace0fa3a214b0ac3cbb4ac2caaf97b965f172f66d2a4ec6304a15a4072f1b28a6f9b80473f86bfa47b2c",
			aesTS("Hex", "00112233445566778899aabbccddeeff"), aesTS("Hex", ""), 16,
			"GCM", "Hex", "Raw", aesTS("Hex", "00000000000000000000000000000000"), aesTS("Hex", ""), "Off")
		want := "Unable to decrypt input with these parameters."
		if err == nil || err.Error() != want {
			t.Fatalf("got err %v, want %q", err, want)
		}
	})
}

// TestAESNoPadding covers the CBC/NoPadding and ECB/NoPadding variants (values
// from the CyberChef-server oracle).
func TestAESNoPadding(t *testing.T) {
	key := aesTS("Hex", "00112233445566778899aabbccddeeff")
	iv := aesTS("Hex", "1748e7179bd56570d51fa4ba287cc3e5")
	pt := "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"

	runCases(t, []opCase{
		{
			"CBC/NoPadding encrypt", pt,
			"386fc094040d38cd8449cbe8a5beb325c902ddee5a3ef177626829ee257cc687",
			aesEnc(key, iv, "CBC/NoPadding", "Hex", "Hex", aesTS("Hex", ""), "Off"),
		},
		{
			"ECB/NoPadding encrypt", pt,
			"62f679be2bf0d931641e039ca3401bb262f679be2bf0d931641e039ca3401bb2",
			aesEnc(key, iv, "ECB/NoPadding", "Hex", "Hex", aesTS("Hex", ""), "Off"),
		},
		{
			"CBC/NoPadding decrypt", "386fc094040d38cd8449cbe8a5beb325c902ddee5a3ef177626829ee257cc687", pt,
			aesDec(key, iv, 16, "CBC/NoPadding", "Hex", "Hex", aesTS("Hex", ""), aesTS("Hex", ""), "Off"),
		},
		{
			"ECB/NoPadding decrypt", "62f679be2bf0d931641e039ca3401bb262f679be2bf0d931641e039ca3401bb2", pt,
			aesDec(key, iv, 16, "ECB/NoPadding", "Hex", "Hex", aesTS("Hex", ""), aesTS("Hex", ""), "Off"),
		},
	})
}

// TestAESNoPaddingLengthError covers the NoPadding block-alignment check.
func TestAESNoPaddingLengthError(t *testing.T) {
	_, err := runOp(t, "AES Encrypt", "0011",
		aesTS("Hex", "00112233445566778899aabbccddeeff"), aesTS("Hex", ""), "CBC/NoPadding", "Hex", "Hex",
		aesTS("Hex", ""), "Off")
	want := "Input length must be a multiple of 16 bytes for NoPadding modes."
	if err == nil || err.Error() != want {
		t.Fatalf("got err %v, want %q", err, want)
	}
}

// TestAESRawOutputRoundTrip exercises the Raw output/input formatting for a
// non-GCM mode by round-tripping through encrypt and decrypt.
func TestAESRawOutputRoundTrip(t *testing.T) {
	key := aesTS("Hex", "00112233445566778899aabbccddeeff")
	iv := aesTS("Hex", "1748e7179bd56570d51fa4ba287cc3e5")
	plaintext := "The quick brown fox jumps over the lazy dog."

	ct, err := runOp(t, "AES Encrypt", plaintext, key, iv, "CBC", "Raw", "Raw", aesTS("Hex", ""), "Off")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	got, err := runOp(t, "AES Decrypt", ct, key, iv, 16, "CBC", "Raw", "Raw",
		aesTS("Hex", ""), aesTS("Hex", ""), "Off")
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if got != plaintext {
		t.Fatalf("round-trip got %q", got)
	}
}

// TestAESGCMRawOutput covers the GCM raw-output branch: the raw ciphertext bytes
// followed by the raw tag bytes.
func TestAESGCMRawOutput(t *testing.T) {
	ct, _ := hex.DecodeString("d0bcace0fa3a214b0ac3cbb4ac2caaf97b965f172f66d2a4ec6304a15a4072f1b28a6f9b80473f86bfa47b2c")
	tag, _ := hex.DecodeString("16a3e732a605cc9ca29108f742ca0743")
	want := string(ct) + "\n\nTag: " + string(tag)

	got, err := runOp(t, "AES Encrypt", "The quick brown fox jumps over the lazy dog.",
		aesTS("Hex", "00112233445566778899aabbccddeeff"), aesTS("Hex", ""), "GCM", "Raw", "Raw",
		aesTS("Hex", ""), "Off")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

// TestAESDecryptUnpadFailures covers the block-mode decrypt failure paths:
// invalid PKCS#7 padding and a ciphertext that is not block-aligned.
func TestAESDecryptUnpadFailures(t *testing.T) {
	key := aesTS("Hex", "00112233445566778899aabbccddeeff")
	iv := aesTS("Hex", "1748e7179bd56570d51fa4ba287cc3e5")
	want := "Unable to decrypt input with these parameters."

	t.Run("invalid padding", func(t *testing.T) {
		// This ciphertext decrypts (CBC) to plaintext ending in 0xff, which is
		// not valid PKCS#7 padding.
		_, err := runOp(t, "AES Decrypt", "386fc094040d38cd8449cbe8a5beb325c902ddee5a3ef177626829ee257cc687",
			key, iv, 16, "CBC", "Hex", "Hex", aesTS("Hex", ""), aesTS("Hex", ""), "Off")
		if err == nil || err.Error() != want {
			t.Fatalf("got err %v, want %q", err, want)
		}
	})
	t.Run("not block aligned", func(t *testing.T) {
		_, err := runOp(t, "AES Decrypt", "00112233", key, iv, 16, "CBC", "Hex", "Hex",
			aesTS("Hex", ""), aesTS("Hex", ""), "Off")
		if err == nil || err.Error() != want {
			t.Fatalf("got err %v, want %q", err, want)
		}
	})
	t.Run("ecb not block aligned", func(t *testing.T) {
		_, err := runOp(t, "AES Decrypt", "00112233", key, iv, 16, "ECB", "Hex", "Hex",
			aesTS("Hex", ""), aesTS("Hex", ""), "Off")
		if err == nil || err.Error() != want {
			t.Fatalf("got err %v, want %q", err, want)
		}
	})
}

// TestAESBadBase64 covers the Base64 decoding error paths for the toggleString
// arguments of both operations.
func TestAESBadBase64(t *testing.T) {
	bad := aesTS("Base64", "!!!not base64!!!")
	goodKey := aesTS("Hex", "00112233445566778899aabbccddeeff")
	none := aesTS("Hex", "")

	t.Run("encrypt bad key", func(t *testing.T) {
		if _, err := runOp(t, "AES Encrypt", "x", bad, none, "CBC", "Raw", "Hex", none, "Off"); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("encrypt bad iv", func(t *testing.T) {
		if _, err := runOp(t, "AES Encrypt", "x", goodKey, bad, "CBC", "Raw", "Hex", none, "Off"); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("encrypt bad aad", func(t *testing.T) {
		if _, err := runOp(t, "AES Encrypt", "x", goodKey, none, "GCM", "Raw", "Hex", bad, "Off"); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("decrypt bad key", func(t *testing.T) {
		if _, err := runOp(t, "AES Decrypt", "x", bad, none, 16, "CBC", "Hex", "Raw", none, none, "Off"); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("decrypt bad tag", func(t *testing.T) {
		if _, err := runOp(t, "AES Decrypt", "00", goodKey, none, 16, "GCM", "Hex", "Raw", bad, none, "Off"); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("decrypt bad aad", func(t *testing.T) {
		if _, err := runOp(t, "AES Decrypt", "00", goodKey, none, 16, "GCM", "Hex", "Raw", none, bad, "Off"); err == nil {
			t.Fatal("want error")
		}
	})
	t.Run("decrypt bad iv", func(t *testing.T) {
		if _, err := runOp(t, "AES Decrypt", "00", goodKey, bad, 16, "CBC", "Hex", "Raw", none, none, "Off"); err == nil {
			t.Fatal("want error")
		}
	})
}

// TestAESDecryptBytesUnknownMode covers the defensive default branch, which the
// Run path cannot reach because Mode is a validated option.
func TestAESDecryptBytesUnknownMode(t *testing.T) {
	block, err := aes.NewCipher(make([]byte, 16))
	if err != nil {
		t.Fatalf("NewCipher: %v", err)
	}
	if _, ok := aesDecryptBytes(block, "BOGUS", false, make([]byte, 16), make([]byte, 16), nil, nil); ok {
		t.Fatal("unknown mode should return ok=false")
	}
}

// TestPKCS7Unpad directly exercises the padding-validation branches that the
// Run path guards against reaching with malformed input.
func TestPKCS7Unpad(t *testing.T) {
	cases := []struct {
		name string
		in   []byte
		ok   bool
	}{
		{"empty", []byte{}, false},
		{"not block aligned", []byte{1, 2, 3}, false},
		{"zero pad byte", append(make([]byte, 15), 0x00), false},
		{"pad byte too large", append(make([]byte, 15), 0x11), false},
		{"inconsistent pad", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 0x01, 0x03, 0x03}, false},
		{"valid", []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 0x03, 0x03, 0x03}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, ok := pkcs7Unpad(c.in, 16)
			if ok != c.ok {
				t.Fatalf("pkcs7Unpad ok=%v, want %v", ok, c.ok)
			}
		})
	}
}

// --- direct tests for the helpers extracted from AESEncrypt.Run ---

// TestValidateAESKeyLen documents accepting 16/24/32-byte keys and rejecting others.
func TestValidateAESKeyLen(t *testing.T) {
	for _, n := range []int{16, 24, 32} {
		if err := validateAESKeyLen(make([]byte, n)); err != nil {
			t.Fatalf("len %d should be valid: %v", n, err)
		}
	}
	if err := validateAESKeyLen(make([]byte, 20)); err == nil {
		t.Fatal("len 20 should be invalid")
	}
}

// TestAESMaybePad documents PKCS#7 padding unless NoPadding is set.
func TestAESMaybePad(t *testing.T) {
	if got := aesMaybePad([]byte("abc"), true); string(got) != "abc" {
		t.Fatalf("noPadding changed input: %q", got)
	}
	if got := aesMaybePad([]byte("abc"), false); len(got) != aes.BlockSize {
		t.Fatalf("padded length = %d, want %d", len(got), aes.BlockSize)
	}
}

// TestApplyIncludeIV documents prepend/append/none of the IV.
func TestApplyIncludeIV(t *testing.T) {
	iv := []byte{1, 2}
	out := []byte{9}
	if got := applyIncludeIV(out, iv, "Prepend"); string(got) != "\x01\x02\x09" {
		t.Fatalf("prepend: %v", got)
	}
	if got := applyIncludeIV([]byte{9}, iv, "Append"); string(got) != "\x09\x01\x02" {
		t.Fatalf("append: %v", got)
	}
	if got := applyIncludeIV([]byte{9}, iv, "None"); string(got) != "\x09" {
		t.Fatalf("none: %v", got)
	}
}

// TestFormatAESOutput documents hex vs raw output and the GCM tag suffix.
func TestFormatAESOutput(t *testing.T) {
	if got := formatAESOutput([]byte{0xAB}, nil, "CBC", "Hex").String(); got != "ab" {
		t.Fatalf("hex: %q", got)
	}
	if got := formatAESOutput([]byte("x"), nil, "CBC", "Raw").String(); got != "x" {
		t.Fatalf("raw: %q", got)
	}
	if got := formatAESOutput([]byte{0xAB}, []byte{0xCD}, "GCM", "Hex").String(); got != "ab\n\nTag: cd" {
		t.Fatalf("gcm hex: %q", got)
	}
}

// TestAESEncryptMode documents the mode dispatch via the AES-128 ECB test vector:
// encrypting 16 zero bytes with a zero key.
func TestAESEncryptMode(t *testing.T) {
	block, _ := aes.NewCipher(make([]byte, 16))
	out, tag := aesEncryptMode("ECB", block, nil, make([]byte, 16), nil, true)
	if tag != nil || hex.EncodeToString(out) != "66e94bd4ef8a2c3b884cfa59ca342b2e" {
		t.Fatalf("ECB vector: %x (tag %v)", out, tag)
	}
}
