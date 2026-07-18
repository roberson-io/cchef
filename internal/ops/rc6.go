package ops

import (
	"fmt"
	"math"
	"math/big"
	"strconv"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(RC6Encrypt{})
	core.Register(RC6Decrypt{})
}

// rc6P256 / rc6Q256 are the 256-bit magic constants from
// draft-krovetz-rc6-rc5-vectors-00, scaled to any word size.
var (
	rc6P256, _ = new(big.Int).SetString("b7e151628aed2a6abf7158809cf4f3c762e7160f38b4da56a784d9045190cfef", 16)
	rc6Q256, _ = new(big.Int).SetString("9e3779b97f4a7c15f39cc0605cedc8341082276bf3a27251f86c6a11d0c18e95", 16)
)

// rc6DefaultRounds returns the recommended round count for a word size.
func rc6DefaultRounds(w int) int {
	switch {
	case w <= 16:
		return 16
	case w <= 32:
		return 20
	case w <= 64:
		return 24
	default:
		return 28
	}
}

// rc6Cipher holds the round-key schedule and word-size parameters for RC6.
type rc6Cipher struct {
	w, rounds, blockSize, lgw int
	mask, modulus, lgMask     *big.Int
	wBig, lgwBig              *big.Int
	s                         []*big.Int
}

// rc6 modular helpers (all results reduced to w bits).
func (c *rc6Cipher) add(a, b *big.Int) *big.Int {
	return new(big.Int).And(new(big.Int).Add(a, b), c.mask)
}

func (c *rc6Cipher) sub(a, b *big.Int) *big.Int {
	r := new(big.Int).Sub(a, b)
	r.Add(r, c.modulus)
	return r.And(r, c.mask)
}

func (c *rc6Cipher) mul(a, b *big.Int) *big.Int {
	return new(big.Int).And(new(big.Int).Mul(a, b), c.mask)
}

// rol rotates x left by (n mod w) bits, where n is first reduced to its low
// lg(w) bits (the RC6 spec).
func (c *rc6Cipher) rol(x, n *big.Int) *big.Int {
	shift := new(big.Int).And(n, c.lgMask)
	shift.Mod(shift, c.wBig)
	s := uint(shift.Int64()) // #nosec G115 -- shift is reduced mod w (< 256)
	left := new(big.Int).Lsh(x, s)
	right := new(big.Int).Rsh(x, uint(c.w)-s)
	return new(big.Int).And(new(big.Int).Or(left, right), c.mask)
}

// ror rotates x right by (n mod w) bits (n reduced to its low lg(w) bits).
func (c *rc6Cipher) ror(x, n *big.Int) *big.Int {
	shift := new(big.Int).And(n, c.lgMask)
	shift.Mod(shift, c.wBig)
	s := uint(shift.Int64()) // #nosec G115 -- shift is reduced mod w (< 256)
	right := new(big.Int).Rsh(x, s)
	left := new(big.Int).Lsh(x, uint(c.w)-s)
	return new(big.Int).And(new(big.Int).Or(right, left), c.mask)
}

// bytesToWords packs bytes into w-bit little-endian words.
func (c *rc6Cipher) bytesToWords(bytes []byte) []*big.Int {
	bpw := c.w / 8
	var words []*big.Int
	for i := 0; i < len(bytes); i += bpw {
		word := new(big.Int)
		for j := 0; j < bpw && i+j < len(bytes); j++ {
			word.Or(word, new(big.Int).Lsh(big.NewInt(int64(bytes[i+j])), uint(j*8)))
		}
		words = append(words, word)
	}
	return words
}

// wordsToBytes unpacks w-bit words into little-endian bytes.
func (c *rc6Cipher) wordsToBytes(words []*big.Int) []byte {
	bpw := c.w / 8
	ff := big.NewInt(0xff)
	out := make([]byte, 0, len(words)*bpw)
	for _, word := range words {
		for j := range bpw {
			b := new(big.Int).And(new(big.Int).Rsh(word, uint(j*8)), ff)
			out = append(out, byte(b.Int64())) // #nosec G115 -- b is masked to a single byte (& 0xff)
		}
	}
	return out
}

// newRC6Cipher runs the RC6 key schedule for the given key, rounds and word size.
func newRC6Cipher(key []byte, rounds, w int) *rc6Cipher {
	c := &rc6Cipher{w: w, rounds: rounds, blockSize: 4 * (w / 8)}
	c.mask = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(w)), big.NewInt(1))
	c.modulus = new(big.Int).Lsh(big.NewInt(1), uint(w))
	c.wBig = big.NewInt(int64(w))
	c.lgw = int(math.Floor(math.Log2(float64(w))))
	c.lgwBig = big.NewInt(int64(c.lgw))
	c.lgMask = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(c.lgw)), big.NewInt(1))

	bpw := w / 8
	cc := max((len(key)+bpw-1)/bpw, 1)
	padded := make([]byte, cc*bpw)
	copy(padded, key)
	l := c.bytesToWords(padded)

	t := 2*rounds + 4
	p := c.scaleConst(rc6P256)
	q := c.scaleConst(rc6Q256)
	s := make([]*big.Int, t)
	s[0] = p
	for i := 1; i < t; i++ {
		s[i] = c.add(s[i-1], q)
	}
	a, b := new(big.Int), new(big.Int)
	i, j := 0, 0
	for range 3 * max(cc, t) {
		s[i] = c.rol(c.add(c.add(s[i], a), b), big.NewInt(3))
		a = s[i]
		l[j] = c.rol(c.add(c.add(l[j], a), b), c.add(a, b))
		b = l[j]
		i = (i + 1) % t
		j = (j + 1) % cc
	}
	c.s = s
	return c
}

// scaleConst scales a 256-bit master constant to w bits and forces it odd.
func (c *rc6Cipher) scaleConst(master *big.Int) *big.Int {
	v := new(big.Int).Rsh(master, uint(256-c.w))
	return v.Or(v, big.NewInt(1))
}

// twiceP1 returns (2*x + 1) mod 2^w, the RC6 quadratic function's inner term.
func (c *rc6Cipher) twiceP1(x *big.Int) *big.Int {
	return c.add(c.mul(big.NewInt(2), x), big.NewInt(1))
}

// encryptBlock encrypts one blockSize-byte block.
func (c *rc6Cipher) encryptBlock(block []byte) []byte {
	words := c.bytesToWords(block)
	a, b, cc, d := words[0], words[1], words[2], words[3]
	b = c.add(b, c.s[0])
	d = c.add(d, c.s[1])
	for i := 1; i <= c.rounds; i++ {
		t := c.rol(c.mul(b, c.twiceP1(b)), c.lgwBig)
		u := c.rol(c.mul(d, c.twiceP1(d)), c.lgwBig)
		a = c.add(c.rol(new(big.Int).Xor(a, t), u), c.s[2*i])
		cc = c.add(c.rol(new(big.Int).Xor(cc, u), t), c.s[2*i+1])
		a, b, cc, d = b, cc, d, a
	}
	a = c.add(a, c.s[2*c.rounds+2])
	cc = c.add(cc, c.s[2*c.rounds+3])
	return c.wordsToBytes([]*big.Int{a, b, cc, d})
}

// decryptBlock decrypts one blockSize-byte block.
func (c *rc6Cipher) decryptBlock(block []byte) []byte {
	words := c.bytesToWords(block)
	a, b, cc, d := words[0], words[1], words[2], words[3]
	cc = c.sub(cc, c.s[2*c.rounds+3])
	a = c.sub(a, c.s[2*c.rounds+2])
	for i := c.rounds; i >= 1; i-- {
		a, b, cc, d = d, a, b, cc
		u := c.rol(c.mul(d, c.twiceP1(d)), c.lgwBig)
		t := c.rol(c.mul(b, c.twiceP1(b)), c.lgwBig)
		cc = new(big.Int).Xor(c.ror(c.sub(cc, c.s[2*i+1]), t), u)
		a = new(big.Int).Xor(c.ror(c.sub(a, c.s[2*i]), u), t)
	}
	d = c.sub(d, c.s[1])
	b = c.sub(b, c.s[0])
	return c.wordsToBytes([]*big.Int{a, b, cc, d})
}

// rc6Xor XORs two equal-length byte slices.
func rc6Xor(a, b []byte) []byte {
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}

// rc6Block returns data[i:i+bs] zero-padded to bs bytes (for stream modes).
func rc6Block(data []byte, i, bs int) []byte {
	block := make([]byte, bs)
	copy(block, data[i:min(i+bs, len(data))])
	return block
}

// rc6IncrementCounter increments a little-endian counter by one.
func rc6IncrementCounter(counter []byte) []byte {
	r := append([]byte{}, counter...)
	for i := range r {
		r[i]++
		if r[i] != 0 {
			break
		}
	}
	return r
}

// rc6Encrypt applies RC6 in the chosen mode with the chosen padding.
func rc6Encrypt(message, key, iv []byte, mode, padding string, rounds, w int) ([]byte, error) {
	if len(message) == 0 {
		return []byte{}, nil
	}
	c := newRC6Cipher(key, rounds, w)
	bs := c.blockSize
	padded := append([]byte{}, message...)
	if mode == "ECB" || mode == "CBC" {
		var err error
		if padded, err = blockApplyPadding(message, padding, bs); err != nil {
			return nil, err
		}
	}
	var out []byte
	switch mode {
	case "ECB":
		for i := 0; i < len(padded); i += bs {
			out = append(out, c.encryptBlock(padded[i:i+bs])...)
		}
	case "CBC":
		ivBlock := append([]byte{}, iv...)
		for i := 0; i < len(padded); i += bs {
			ivBlock = c.encryptBlock(rc6Xor(padded[i:i+bs], ivBlock))
			out = append(out, ivBlock...)
		}
	case "CFB":
		ivBlock := append([]byte{}, iv...)
		for i := 0; i < len(padded); i += bs {
			ivBlock = rc6Xor(c.encryptBlock(ivBlock), rc6Block(padded, i, bs))
			out = append(out, ivBlock...)
		}
		return out[:len(message)], nil
	case "OFB":
		ivBlock := append([]byte{}, iv...)
		for i := 0; i < len(padded); i += bs {
			ivBlock = c.encryptBlock(ivBlock)
			out = append(out, rc6Xor(ivBlock, rc6Block(padded, i, bs))...)
		}
		return out[:len(message)], nil
	default: // CTR
		counter := append([]byte{}, iv...)
		for i := 0; i < len(padded); i += bs {
			out = append(out, rc6Xor(c.encryptBlock(counter), rc6Block(padded, i, bs))...)
			counter = rc6IncrementCounter(counter)
		}
		return out[:len(message)], nil
	}
	return out, nil
}

// rc6NormalizeCipher validates the ciphertext length for ECB/CBC (must be a whole
// number of blocks) and zero-pads it to a block multiple for the stream modes.
func rc6NormalizeCipher(ciphertext []byte, mode string, bs int) ([]byte, error) {
	if mode == "ECB" || mode == "CBC" {
		if len(ciphertext)%bs != 0 {
			//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
			return nil, fmt.Errorf("Invalid ciphertext length: %d bytes. Must be a multiple of %d.", len(ciphertext), bs)
		}
		return ciphertext, nil
	}
	for len(ciphertext)%bs != 0 {
		ciphertext = append(ciphertext, 0)
	}
	return ciphertext, nil
}

// rc6Decrypt reverses rc6Encrypt.
func rc6Decrypt(ciphertext, key, iv []byte, mode, padding string, rounds, w int) ([]byte, error) {
	origLen := len(ciphertext)
	if origLen == 0 {
		return []byte{}, nil
	}
	c := newRC6Cipher(key, rounds, w)
	bs := c.blockSize
	ciphertext, err := rc6NormalizeCipher(ciphertext, mode, bs)
	if err != nil {
		return nil, err
	}
	var plain []byte
	switch mode {
	case "ECB":
		for i := 0; i < len(ciphertext); i += bs {
			plain = append(plain, c.decryptBlock(ciphertext[i:i+bs])...)
		}
	case "CBC":
		ivBlock := append([]byte{}, iv...)
		for i := 0; i < len(ciphertext); i += bs {
			block := ciphertext[i : i+bs]
			plain = append(plain, rc6Xor(c.decryptBlock(block), ivBlock)...)
			ivBlock = append([]byte{}, block...)
		}
	case "CFB":
		ivBlock := append([]byte{}, iv...)
		for i := 0; i < len(ciphertext); i += bs {
			block := ciphertext[i : i+bs]
			plain = append(plain, rc6Xor(c.encryptBlock(ivBlock), block)...)
			ivBlock = append([]byte{}, block...)
		}
		return plain[:origLen], nil
	case "OFB":
		ivBlock := append([]byte{}, iv...)
		for i := 0; i < len(ciphertext); i += bs {
			ivBlock = c.encryptBlock(ivBlock)
			plain = append(plain, rc6Xor(ivBlock, ciphertext[i:i+bs])...)
		}
		return plain[:origLen], nil
	case "CTR":
		counter := append([]byte{}, iv...)
		for i := 0; i < len(ciphertext); i += bs {
			plain = append(plain, rc6Xor(c.encryptBlock(counter), ciphertext[i:i+bs])...)
			counter = rc6IncrementCounter(counter)
		}
		return plain[:origLen], nil
	}
	return blockRemovePadding(plain, padding, bs)
}

// rc6NumStr formats a numeric argument the way JavaScript prints it (no trailing
// ".0"), for the verbatim validation error messages.
func rc6NumStr(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// rc6Validate checks the word size, IV length and round count with CyberChef's
// exact error messages, returning the integer word size and round count.
func rc6Validate(w, rounds float64, ivLen int, mode string) (int, int, error) {
	if w != math.Trunc(w) || w < 8 || w > 256 || int(w)%8 != 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return 0, 0, fmt.Errorf("Invalid word size: %s. Must be a multiple of 8 between 8 and 256.", rc6NumStr(w))
	}
	wi := int(w)
	bs := 4 * (wi / 8)
	if ivLen != bs && ivLen != 0 && mode != "ECB" {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return 0, 0, fmt.Errorf("Invalid IV length: %d bytes\n\nRC6-%d uses an IV length of %d bytes (%d bits).\nMake sure you have specified the type correctly (e.g. Hex vs UTF8).", ivLen, wi, bs, bs*8)
	}
	if rounds != math.Trunc(rounds) || rounds < 1 || rounds > 255 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return 0, 0, fmt.Errorf("Invalid number of rounds: %s\n\nRounds must be an integer between 1 and 255. Standard for w=%d is %d.", rc6NumStr(rounds), wi, rc6DefaultRounds(wi))
	}
	return wi, int(rounds), nil
}

// rc6Run parses arguments, validates them (matching CyberChef), and runs RC6.
func rc6Run(in *core.Dish, args []any, decrypt bool) (*core.Dish, error) {
	keyArg, ivArg := args[0].(core.ToggleString), args[1].(core.ToggleString)
	key, err := convertToByteArray(keyArg.Value, keyArg.Option)
	if err != nil {
		return nil, err
	}
	iv, err := convertToByteArray(ivArg.Value, ivArg.Option)
	if err != nil {
		return nil, err
	}
	mode, padding := args[2].(string), args[5].(string)
	wi, ri, err := rc6Validate(args[6].(float64), args[7].(float64), len(iv), mode)
	if err != nil {
		return nil, err
	}
	if len(iv) == 0 {
		iv = make([]byte, 4*(wi/8))
	}
	input := decodeAESInput(in, args[3].(string))
	var out []byte
	if decrypt {
		out, err = rc6Decrypt(input, key, iv, mode, padding, ri, wi)
	} else {
		out, err = rc6Encrypt(input, key, iv, mode, padding, ri, wi)
	}
	if err != nil {
		return nil, err
	}
	return blowfishOutput(out, args[4].(string)), nil
}

// rc6Description is shared by both RC6 operations.
const rc6Description = "RC6 is a symmetric key block cipher derived from RC5. It was designed by Ron Rivest, Matt Robshaw, Ray Sidney, and Yiqun Lisa Yin to meet the requirements of the AES competition, and was one of the five finalists.<br><br>RC6 is parameterised as RC6-w/r/b where w is word size in bits (any multiple of 8 from 8-256), r is the number of rounds (1-255), and b is the key length in bytes. The standard AES submission uses w=32, r=20. Common word sizes: 8, 16, 32 (standard), 64, 128.<br><br><b>IV:</b> The Initialisation Vector should be 4*w/8 bytes (e.g. 16 bytes for w=32). If not entered, it will default to null bytes.<br><br><b>Padding:</b> In CBC and ECB mode, the PKCS#7 padding scheme is used."

// rc6ModeValues are the block modes both operations offer.
var rc6ModeValues = []string{"CBC", "CFB", "OFB", "CTR", "ECB"}

// rc6Args builds the shared argument list; encrypt and decrypt only differ in the
// default Input/Output option order. Word Size / Rounds are validated in Run (with
// CyberChef's exact messages) rather than via Min/Max coercion.
func rc6Args(inFmt, outFmt []string) []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: aesToggleValues},
		{Name: "Mode", Type: core.ArgOption, Value: rc6ModeValues},
		{Name: "Input", Type: core.ArgOption, Value: inFmt},
		{Name: "Output", Type: core.ArgOption, Value: outFmt},
		{Name: "Padding", Type: core.ArgOption, Value: presentPaddings},
		{Name: "Word Size", Type: core.ArgNumber, Value: float64(32)},
		{Name: "Rounds", Type: core.ArgNumber, Value: float64(20)},
	}
}

// RC6Encrypt encrypts input with the RC6 block cipher.
type RC6Encrypt struct{}

// Meta returns the operation metadata.
func (RC6Encrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "RC6 Encrypt",
		Module:      "Ciphers",
		Description: rc6Description,
		InfoURL:     "https://wikipedia.org/wiki/RC6",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RC6Encrypt) Args() []core.ArgDef {
	return rc6Args([]string{"Raw", "Hex"}, []string{"Hex", "Raw"})
}

// Run performs the encryption. Ported from CyberChef RC6Encrypt.mjs.
func (RC6Encrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return rc6Run(in, args, false)
}

// RC6Decrypt decrypts RC6 ciphertext.
type RC6Decrypt struct{}

// Meta returns the operation metadata.
func (RC6Decrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "RC6 Decrypt",
		Module:      "Ciphers",
		Description: rc6Description,
		InfoURL:     "https://wikipedia.org/wiki/RC6",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (RC6Decrypt) Args() []core.ArgDef {
	return rc6Args([]string{"Hex", "Raw"}, []string{"Raw", "Hex"})
}

// Run performs the decryption. Ported from CyberChef RC6Decrypt.mjs.
func (RC6Decrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	return rc6Run(in, args, true)
}
