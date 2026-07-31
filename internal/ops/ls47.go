package ops

import (
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(LS47Encrypt{})
	core.Register(LS47Decrypt{})
}

// ls47Description is shared by both LS47 operations.
const ls47Description = "This is a slight improvement of the ElsieFour cipher as described by Alan Kaminsky. We use 7x7 characters instead of original (barely fitting) 6x6, to be able to encrypt some structured information. We also describe a simple key-expansion algorithm, because remembering passwords is popular. Similar security considerations as with ElsieFour hold."

// ls47Letters is the LS47 alphabet: a permutation of these 49 characters forms
// the 7x7 grid used as the cipher state.
const ls47Letters = "_abcdefghijklmnopqrstuvwxyz.0123456789,-+*/:?!'()"

// ls47Size is the grid dimension (7x7 = 49 tiles).
const ls47Size = 7

// ls47Pos is a (row, col) coordinate on the grid.
type ls47Pos [2]int

// ls47RotateRight rotates the given row of the key n places to the right.
func ls47RotateRight(key string, row, n int) string {
	mid := key[row*ls47Size : (row+1)*ls47Size]
	n = (ls47Size - n%ls47Size) % ls47Size
	return key[:ls47Size*row] + mid[n:] + mid[:n] + key[ls47Size*(row+1):]
}

// ls47RotateDown rotates the given column of the key n places downwards.
func ls47RotateDown(key string, col, n int) string {
	lefts := make([]string, ls47Size)
	mids := make([]byte, ls47Size)
	rights := make([]string, ls47Size)
	for i := range ls47Size {
		line := key[i*ls47Size : (i+1)*ls47Size]
		lefts[i] = line[:col]
		mids[i] = line[col]
		rights[i] = line[col+1:]
	}
	n = (ls47Size - n%ls47Size) % ls47Size
	mids = append(mids[n:], mids[:n]...)
	var b strings.Builder
	for i := range ls47Size {
		b.WriteString(lefts[i])
		b.WriteByte(mids[i])
		b.WriteString(rights[i])
	}
	return b.String()
}

// ls47Ix returns a letter's position in the alphabet. The caller guarantees the
// letter is in the alphabet (it comes from the key/grid).
func ls47Ix(letter rune) ls47Pos {
	i := strings.IndexRune(ls47Letters, letter)
	return ls47Pos{i / ls47Size, i % ls47Size}
}

// ls47FindIx returns a letter's position in the alphabet, erroring if the letter
// is not part of it (used for untrusted password characters).
func ls47FindIx(letter rune) (ls47Pos, error) {
	i := strings.IndexRune(ls47Letters, letter)
	if i < 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return ls47Pos{}, fmt.Errorf("Letter %c is not included in LS47", letter)
	}
	return ls47Pos{i / ls47Size, i % ls47Size}, nil
}

// ls47PosOf returns a letter's position in the key. The caller guarantees the
// letter is present (it was just read from the key).
func ls47PosOf(key string, letter rune) ls47Pos {
	i := strings.IndexRune(key, letter)
	return ls47Pos{i / ls47Size, i % ls47Size}
}

// ls47FindPos returns a letter's position in the key, erroring if it is absent
// (used for untrusted plaintext/ciphertext characters).
func ls47FindPos(key string, letter rune) (ls47Pos, error) {
	i := strings.IndexRune(key, letter)
	if i >= 0 && i < ls47Size*ls47Size {
		return ls47Pos{i / ls47Size, i % ls47Size}, nil
	}
	//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	return ls47Pos{}, fmt.Errorf("Letter %c is not in the key", letter)
}

// ls47AtPos returns the character at the given grid position.
func ls47AtPos(key string, p ls47Pos) rune {
	return rune(key[p[1]+p[0]*ls47Size])
}

// ls47Add adds two positions modulo the grid size.
func ls47Add(a, b ls47Pos) ls47Pos {
	return ls47Pos{(a[0] + b[0]) % ls47Size, (a[1] + b[1]) % ls47Size}
}

// ls47Sub subtracts two positions using Euclidean (always-positive) modulo,
// matching the JS implementation's manual floor-division fix for negatives.
func ls47Sub(a, b ls47Pos) ls47Pos {
	return ls47Pos{
		((a[0]-b[0])%ls47Size + ls47Size) % ls47Size,
		((a[1]-b[1])%ls47Size + ls47Size) % ls47Size,
	}
}

// ls47DeriveKey expands a password into a 49-character key. An empty password
// yields the identity key (the alphabet itself).
func ls47DeriveKey(password string) (string, error) {
	i := 0
	k := ls47Letters
	for _, c := range password {
		p, err := ls47FindIx(c)
		if err != nil {
			return "", err
		}
		k = ls47RotateDown(ls47RotateRight(k, i, p[1]), i, p[0])
		i = (i + 1) % ls47Size
	}
	return k, nil
}

// ls47EncryptRaw encrypts plaintext with the key (no padding handling).
func ls47EncryptRaw(key, plaintext string) (string, error) {
	mp := ls47Pos{0, 0}
	var ct strings.Builder
	for _, p := range plaintext {
		pp, err := ls47FindPos(key, p)
		if err != nil {
			return "", err
		}
		cp := ls47Add(pp, ls47Ix(ls47AtPos(key, mp)))
		c := ls47AtPos(key, cp)
		ct.WriteRune(c)
		key = ls47RotateRight(key, pp[0], 1)
		key = ls47RotateDown(key, ls47PosOf(key, c)[1], 1)
		mp = ls47Add(mp, ls47Ix(c))
	}
	return ct.String(), nil
}

// ls47DecryptRaw decrypts ciphertext with the key (no padding handling).
func ls47DecryptRaw(key, ciphertext string) (string, error) {
	mp := ls47Pos{0, 0}
	var pt strings.Builder
	for _, c := range ciphertext {
		cp, err := ls47FindPos(key, c)
		if err != nil {
			return "", err
		}
		pp := ls47Sub(cp, ls47Ix(ls47AtPos(key, mp)))
		pt.WriteRune(ls47AtPos(key, pp))
		key = ls47RotateRight(key, pp[0], 1)
		key = ls47RotateDown(key, ls47PosOf(key, c)[1], 1)
		mp = ls47Add(mp, ls47Ix(c))
	}
	return pt.String(), nil
}

// ls47EncryptPad prepends paddingSize random characters, then the plaintext, a
// "---" separator and the signature, and encrypts the whole thing.
func ls47EncryptPad(key, plaintext, signature string, paddingSize int) (string, error) {
	var padding strings.Builder
	for range paddingSize {
		padding.WriteByte(ls47Letters[randInt(len(ls47Letters))])
	}
	return ls47EncryptRaw(key, padding.String()+plaintext+"---"+signature)
}

// ls47DecryptPad decrypts ciphertext and drops the leading paddingSize
// characters (JS slice semantics: a count past the end yields an empty string).
func ls47DecryptPad(key, ciphertext string, paddingSize int) (string, error) {
	plaintext, err := ls47DecryptRaw(key, ciphertext)
	if err != nil {
		return "", err
	}
	start := paddingSize
	if start < 0 {
		start += len(plaintext)
		if start < 0 {
			start = 0
		}
	} else if start > len(plaintext) {
		start = len(plaintext)
	}
	return plaintext[start:], nil
}

// LS47Encrypt encrypts with the LS47 cipher.
type LS47Encrypt struct{}

// Meta returns the operation metadata.
func (LS47Encrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "LS47 Encrypt",
		Module:      "Crypto",
		Description: ls47Description,
		InfoURL:     "https://github.com/exaexa/ls47",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (LS47Encrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Password", Type: core.ArgString, Value: ""},
		{Name: "Padding", Type: core.ArgNumber, Integer: true, Value: float64(10)},
		{Name: "Signature", Type: core.ArgString, Value: ""},
	}
}

// Run encrypts the input.
func (LS47Encrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	password, signature := args[0].(string), args[2].(string)
	paddingSize := int(args[1].(float64))
	key, err := ls47DeriveKey(password)
	if err != nil {
		return nil, err
	}
	out, err := ls47EncryptPad(key, in.String(), signature, paddingSize)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// LS47Decrypt decrypts with the LS47 cipher.
type LS47Decrypt struct{}

// Meta returns the operation metadata.
func (LS47Decrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "LS47 Decrypt",
		Module:      "Crypto",
		Description: ls47Description,
		InfoURL:     "https://github.com/exaexa/ls47",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (LS47Decrypt) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Password", Type: core.ArgString, Value: ""},
		{Name: "Padding", Type: core.ArgNumber, Integer: true, Value: float64(10)},
	}
}

// Run decrypts the input.
func (LS47Decrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	password := args[0].(string)
	paddingSize := int(args[1].(float64))
	key, err := ls47DeriveKey(password)
	if err != nil {
		return nil, err
	}
	out, err := ls47DecryptPad(key, in.String(), paddingSize)
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}
