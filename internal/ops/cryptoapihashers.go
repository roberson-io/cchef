package ops

import (
	"crypto/md5"  // #nosec G501 -- user-selectable hash for a ported KDF, not a security control
	"crypto/sha1" // #nosec G505 -- user-selectable hash for a ported KDF, not a security control
	"crypto/sha256"
	"crypto/sha512"
	"hash"
)

// cryptoapiHashers maps CyberChef's crypto-api hash-function names (as used by
// the Derive HKDF key operation) to Go hash constructors. It is intended to be
// shared with future ports of the standalone MD2/MD4/SHA0/RIPEMD/HAS-160/
// Whirlpool/Snefru hash operations. Names match CyberChef's option strings
// exactly (including "SHA512/224", "Whirlpool-0", etc.).
var cryptoapiHashers = map[string]func() hash.Hash{
	"MD2":         newMD2,
	"MD4":         newMD4,
	"MD5":         md5.New,
	"SHA0":        newSHA0,
	"SHA1":        sha1.New,
	"HAS160":      newHAS160,
	"SHA224":      sha256.New224,
	"SHA256":      sha256.New,
	"SHA384":      sha512.New384,
	"SHA512":      sha512.New,
	"SHA512/224":  sha512.New512_224,
	"SHA512/256":  sha512.New512_256,
	"RIPEMD128":   newRIPEMD128,
	"RIPEMD160":   newRIPEMD160,
	"RIPEMD256":   newRIPEMD256,
	"RIPEMD320":   newRIPEMD320,
	"Whirlpool":   newWhirlpool,
	"Whirlpool-0": newWhirlpool0,
	"Whirlpool-T": newWhirlpoolT,
	"Snefru":      newSnefru,
}
