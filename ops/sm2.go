package ops

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(SM2Encrypt{})
	core.Register(SM2Decrypt{})
}

// SM2 public-key encryption (GM/T 0003) over the sm2p256v1 curve, an in-repo
// port of CyberChef's lib/SM2.mjs. jsrsasign supplies the elliptic-curve
// arithmetic and crypto-api the SM3 hash in CyberChef; both are reimplemented
// here (see sm3.go for SM3). Encryption output is C1 ‖ C3 ‖ C2 or C1 ‖ C2 ‖ C3
// depending on the chosen format.

// sm2 curve parameters for sm2p256v1 (the only curve CyberChef defines):
// p = 2^256 − 2^224 − 2^96 + 2^64 − 1.
var (
	sm2P  = sm2HexInt("FFFFFFFEFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF00000000FFFFFFFFFFFFFFFF")
	sm2A  = sm2HexInt("FFFFFFFEFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF00000000FFFFFFFFFFFFFFFC")
	sm2N  = sm2HexInt("FFFFFFFEFFFFFFFFFFFFFFFFFFFFFFFF7203DF6B21C6052B53BBF40939D54123")
	sm2Gx = sm2HexInt("32C4AE2C1F1981195F9904466A39C9948FE30BBFF2660BE1715A4589334C74C7")
	sm2Gy = sm2HexInt("BC3736A2F4F6779C59BDCEE36B692153D0A9877CC62A474002DF32E52139F0A0")
)

// sm2FieldBytes is the size in bytes of a curve coordinate (256 bits), and
// sm2CoordHex the corresponding hex-character width used when formatting points.
const (
	sm2FieldBytes = 32
	sm2CoordHex   = sm2FieldBytes * 2
)

// sm2HexInt parses a hex constant, panicking on malformed input (used only for
// the compile-time curve constants).
func sm2HexInt(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 16)
	if !ok {
		panic("sm2: bad curve constant " + s)
	}
	return n
}

// sm2Point is an affine curve point; a nil X denotes the point at infinity.
type sm2Point struct{ x, y *big.Int }

func sm2IsInf(p sm2Point) bool { return p.x == nil }

// sm2Mod reduces a modulo the field prime p.
func sm2Mod(a *big.Int) *big.Int { return a.Mod(a, sm2P) }

// sm2Add returns p + q on the curve.
func sm2Add(p, q sm2Point) sm2Point {
	if sm2IsInf(p) {
		return q
	}
	if sm2IsInf(q) {
		return p
	}
	if p.x.Cmp(q.x) == 0 {
		if p.y.Cmp(q.y) != 0 || p.y.Sign() == 0 {
			return sm2Point{} // p + (−p) = ∞
		}
		return sm2Double(p)
	}
	// λ = (qy − py) / (qx − px)
	num := new(big.Int).Sub(q.y, p.y)
	den := new(big.Int).Sub(q.x, p.x)
	den.ModInverse(den, sm2P)
	lam := sm2Mod(num.Mul(num, den))
	return sm2Chord(lam, p, q.x)
}

// sm2Double returns 2p on the curve.
func sm2Double(p sm2Point) sm2Point {
	if sm2IsInf(p) || p.y.Sign() == 0 {
		return sm2Point{}
	}
	// λ = (3x² + a) / (2y)
	num := new(big.Int).Mul(big.NewInt(3), new(big.Int).Mul(p.x, p.x))
	num.Add(num, sm2A)
	den := new(big.Int).Lsh(p.y, 1)
	den.ModInverse(den, sm2P)
	lam := sm2Mod(num.Mul(num, den))
	return sm2Chord(lam, p, p.x)
}

// sm2Chord finishes a point addition/doubling from the slope λ, the first point
// p, and the second x-coordinate: x₃ = λ² − px − qx, y₃ = λ(px − x₃) − py.
func sm2Chord(lam *big.Int, p sm2Point, qx *big.Int) sm2Point {
	x3 := sm2Mod(new(big.Int).Sub(new(big.Int).Sub(new(big.Int).Mul(lam, lam), p.x), qx))
	y3 := sm2Mod(new(big.Int).Sub(new(big.Int).Mul(lam, new(big.Int).Sub(p.x, x3)), p.y))
	return sm2Point{x3, y3}
}

// sm2Mul returns k·p by double-and-add.
func sm2Mul(k *big.Int, p sm2Point) sm2Point {
	r := sm2Point{}
	addend := p
	for i := 0; i < k.BitLen(); i++ {
		if k.Bit(i) == 1 {
			r = sm2Add(r, addend)
		}
		addend = sm2Double(addend)
	}
	return r
}

// sm2PointHex returns a point's X and Y coordinates as fixed-width hex strings.
func sm2PointHex(p sm2Point) (string, string) {
	return hex.EncodeToString(sm2Pad(p.x)), hex.EncodeToString(sm2Pad(p.y))
}

// sm2Pad left-pads a coordinate to the field width in bytes.
func sm2Pad(n *big.Int) []byte {
	b := n.Bytes()
	if len(b) >= sm2FieldBytes {
		return b[len(b)-sm2FieldBytes:]
	}
	out := make([]byte, sm2FieldBytes)
	copy(out[sm2FieldBytes-len(b):], b)
	return out
}

// sm2KDF derives length bytes of key material from the shared point p2 by
// hashing X ‖ Y ‖ counter with SM3 for an increasing 32-bit counter.
func sm2KDF(p2 sm2Point, length int) []byte {
	x2, y2 := sm2Pad(p2.x), sm2Pad(p2.y)
	blocks := (length+sm3DigestLen-1)/sm3DigestLen + 1
	var km []byte
	for cnt := 1; cnt < blocks; cnt++ {
		var ctr [4]byte
		//nolint:gosec // G115: cnt is a small positive block counter, well within uint32
		binary.BigEndian.PutUint32(ctr[:], uint32(cnt))
		block := append(append(append([]byte{}, x2...), y2...), ctr[:]...)
		km = append(km, sm3Sum(block)...)
	}
	return km
}

// sm2C3 computes the C3 authentication tag: SM3(X ‖ message ‖ Y).
func sm2C3(p2 sm2Point, message []byte) []byte {
	x2, y2 := sm2Pad(p2.x), sm2Pad(p2.y)
	block := append(append(append([]byte{}, x2...), message...), y2...)
	return sm3Sum(block)
}

// sm2XOR returns message XOR the first len(message) bytes of key material.
func sm2XOR(message, key []byte) []byte {
	out := make([]byte, len(message))
	for i := range message {
		out[i] = message[i] ^ key[i]
	}
	return out
}

// SM2Encrypt encrypts a message with the SM2 public-key standard.
type SM2Encrypt struct{}

// Meta returns the operation metadata.
func (SM2Encrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "SM2 Encrypt",
		Module:      "Crypto",
		Description: "Encrypts a message utilizing the SM2 standard",
		InfoURL:     "",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (SM2Encrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Public Key X", Type: core.ArgString, Value: "DEADBEEF"},
		{Name: "Public Key Y", Type: core.ArgString, Value: "DEADBEEF"},
		{Name: "Output Format", Type: core.ArgOption, Value: []string{"C1C3C2", "C1C2C3"}},
		{Name: "Curve", Type: core.ArgOption, Value: []string{"sm2p256v1"}},
	}
}

// Run encrypts the input bytes for the given public key, emitting hex.
func (SM2Encrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	publicKeyX := args[0].(string)
	publicKeyY := args[1].(string)
	format := args[2].(string)

	if len(publicKeyX) != sm2CoordHex || len(publicKeyY) != sm2CoordHex {
		//nolint:staticcheck,revive // verbatim CyberChef OperationError text
		return nil, errors.New("Invalid Public Key - Ensure each component is 32 bytes in size and in hex")
	}
	pubX, okX := new(big.Int).SetString(publicKeyX, 16)
	pubY, okY := new(big.Int).SetString(publicKeyY, 16)
	if !okX || !okY {
		//nolint:staticcheck,revive // verbatim CyberChef OperationError text
		return nil, errors.New("Invalid Public Key - Ensure each component is 32 bytes in size and in hex")
	}

	out, err := sm2Encrypt(in.Bytes(), sm2Point{pubX, pubY}, format)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// sm2Encrypt performs the SM2 encryption and returns the hex-encoded package.
func sm2Encrypt(message []byte, publicKey sm2Point, format string) (string, error) {
	k, err := sm2RandomK()
	if err != nil {
		return "", err
	}
	c1 := sm2Mul(k, sm2Point{sm2Gx, sm2Gy})
	c1x, c1y := sm2PointHex(c1)

	p2 := sm2Mul(k, publicKey)
	c3 := hex.EncodeToString(sm2C3(p2, message))
	c2 := hex.EncodeToString(sm2XOR(message, sm2KDF(p2, len(message))))

	if format == "C1C3C2" {
		return c1x + c1y + c3 + c2, nil
	}
	return c1x + c1y + c2 + c3, nil
}

// sm2RandReader is the entropy source for encryption; a package variable so
// tests can substitute a failing reader.
var sm2RandReader = rand.Reader

// sm2RandomK returns a uniform random scalar in [1, n−1].
func sm2RandomK() (*big.Int, error) {
	limit := new(big.Int).Sub(sm2N, big.NewInt(1))
	k, err := rand.Int(sm2RandReader, limit)
	if err != nil {
		return nil, err
	}
	return k.Add(k, big.NewInt(1)), nil
}

// SM2Decrypt decrypts an SM2-encrypted message.
type SM2Decrypt struct{}

// Meta returns the operation metadata.
func (SM2Decrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "SM2 Decrypt",
		Module:      "Crypto",
		Description: "Decrypts a message utilizing the SM2 standard",
		InfoURL:     "",
		InputType:   core.TypeString,
		OutputType:  core.TypeArrayBuffer,
	}
}

// Args returns the argument definitions.
func (SM2Decrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Private Key", Type: core.ArgString, Value: "DEADBEEF"},
		{Name: "Input Format", Type: core.ArgOption, Value: []string{"C1C3C2", "C1C2C3"}},
		{Name: "Curve", Type: core.ArgOption, Value: []string{"sm2p256v1"}},
	}
}

// Run decrypts the hex-encoded input with the given private key.
func (SM2Decrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	privateKey := args[0].(string)
	format := args[1].(string)

	if len(privateKey) != sm2CoordHex {
		//nolint:staticcheck,revive // verbatim CyberChef OperationError text
		return nil, errors.New("Input private key must be in hex; and should be 32 bytes")
	}
	priv, ok := new(big.Int).SetString(privateKey, 16)
	if !ok {
		//nolint:staticcheck,revive // verbatim CyberChef OperationError text
		return nil, errors.New("Input private key must be in hex; and should be 32 bytes")
	}

	message, err := sm2Decrypt(in.String(), priv, format)
	if err != nil {
		return nil, err
	}
	return core.NewDish(message, core.TypeArrayBuffer), nil
}

// errSM2HashMismatch is CyberChef's failure when the recomputed C3 tag does not
// match the ciphertext.
var errSM2HashMismatch = errors.New("Decryption Error -- Computed Hashes Do Not Match") //nolint:staticcheck,revive // verbatim CyberChef text

// sm2Decrypt recovers the plaintext from a hex-encoded SM2 package.
func sm2Decrypt(input string, priv *big.Int, format string) ([]byte, error) {
	c1x := sm2Slice(input, 0, 64)
	c1y := sm2Slice(input, 64, 128)

	var c3Hex, c2Hex string
	if format == "C1C3C2" {
		c3Hex = sm2Slice(input, 128, 192)
		c2Hex = sm2Slice(input, 192, len(input))
	} else {
		c2Hex = sm2Slice(input, 128, len(input)-64)
		c3Hex = sm2Slice(input, len(input)-64, len(input))
	}

	c2, err := hex.DecodeString(c2Hex)
	if err != nil {
		return nil, errSM2HashMismatch
	}
	c1, ok := sm2DecodePoint(c1x, c1y)
	if !ok {
		return nil, errSM2HashMismatch
	}

	p2 := sm2Mul(priv, c1)
	if sm2IsInf(p2) {
		return nil, errSM2HashMismatch
	}

	message := sm2XOR(c2, sm2KDF(p2, len(c2)))
	if hex.EncodeToString(sm2C3(p2, message)) != c3Hex {
		return nil, errSM2HashMismatch
	}
	return message, nil
}

// sm2DecodePoint parses X and Y coordinate hex into a curve point.
func sm2DecodePoint(xHex, yHex string) (sm2Point, bool) {
	x, okX := new(big.Int).SetString(xHex, 16)
	y, okY := new(big.Int).SetString(yHex, 16)
	if !okX || !okY {
		return sm2Point{}, false
	}
	return sm2Point{x, y}, true
}

// sm2Slice mirrors JavaScript String.slice(start, end) for the already
// non-negative index arithmetic the decrypt path uses, clamping to bounds.
func sm2Slice(s string, start, end int) string {
	if start < 0 {
		start = 0
	}
	if end > len(s) {
		end = len(s)
	}
	if start >= end {
		return ""
	}
	return s[start:end]
}
