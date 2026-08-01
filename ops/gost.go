package ops

// Pure-Go port of the GOST block-cipher engine used by CyberChef's GOST
// operations (../CyberChef/src/core/vendor/gost/gostCipher.mjs, by Rudolf
// Nickolaev). It implements GOST 28147-89 / GOST R 34.12-2015 "Magma" (64-bit)
// and "Kuznyechik" (128-bit) in the ES (encrypt/decrypt), MAC (imitovstavka)
// and KW (key wrapping) modes. The port follows the JavaScript arithmetic
// faithfully — including its little-endian typed-array semantics and a couple
// of upstream quirks (e.g. "PKCS5" falling through to zero padding, and the
// CTR-2015 counter loop) — so output matches CyberChef byte for byte.

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/core"
)

// errGostInvalidTypedArray mirrors the JavaScript RangeError raised when the
// engine lays a typed array over a too-short buffer (e.g. Kuznyechik CryptoPro
// key wrapping, whose diversification step is inherently 64-bit). The Key Wrap
// operations translate it to the friendly "Incorrect input length" message.
var errGostInvalidTypedArray = errors.New("Invalid typed array length") //nolint:staticcheck,revive // CyberChef's verbatim RangeError text

func init() {
	core.Register(GOSTEncrypt{})
	core.Register(GOSTDecrypt{})
	core.Register(GOSTKeyWrap{})
	core.Register(GOSTKeyUnwrap{})
	core.Register(GOSTSign{})
	core.Register(GOSTVerify{})
}

// gostSBoxNames lists the paramset S-box names offered by every GOST cipher op,
// matching CyberChef's option order. Only used when the algorithm version is
// GOST 28147 (1989); the 2015 algorithms use fixed transforms.
var gostSBoxNames = []string{"E-TEST", "E-A", "E-B", "E-C", "E-D", "E-SC", "E-Z", "D-TEST", "D-A", "D-SC"}

// gostAlgorithms lists the three block-cipher variants offered by the cipher ops.
var gostAlgorithms = []string{"GOST 28147 (1989)", "GOST R 34.12 (Magma, 2015)", "GOST R 34.12 (Kuznyechik, 2015)"}

// gostToggleValues are the key/IV encoding modes shared by all GOST ops.
var gostToggleValues = []string{"Hex", "UTF8", "Latin1", "Base64"}

// gostCipherArgs builds the argument list shared by GOST Encrypt and GOST
// Decrypt (which differ only in the Input/Output option ordering).
func gostCipherArgs(inputOpts, outputOpts []string) []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: gostToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: gostToggleValues},
		{Name: "Input type", Type: core.ArgOption, Value: inputOpts},
		{Name: "Output type", Type: core.ArgOption, Value: outputOpts},
		{Name: "Algorithm", Type: core.ArgOption, Value: gostAlgorithms},
		{Name: "sBox", Type: core.ArgOption, Value: gostSBoxNames},
		{Name: "Block mode", Type: core.ArgOption, Value: []string{"ECB", "CFB", "OFB", "CTR", "CBC"}},
		{Name: "Key meshing mode", Type: core.ArgOption, Value: []string{"NO", "CP"}},
		{Name: "Padding", Type: core.ArgOption, Value: []string{"NO", "PKCS5", "ZERO", "RANDOM", "BIT"}},
	}
}

// gostBlockSize64 / gostBlockSize128 are the two supported block sizes in bytes.
const (
	gostBlockSize64  = 8
	gostBlockSize128 = 16
	gostKeySize      = 32 // GOST keys are always 256 bits.
)

// le32 / putLE32 read and write a little-endian uint32 at byte offset off,
// mirroring the Int32Array views the JavaScript engine lays over its buffers.
func le32(b []byte, off int) uint32       { return binary.LittleEndian.Uint32(b[off:]) }
func putLE32(b []byte, off int, v uint32) { binary.LittleEndian.PutUint32(b[off:], v) }

// swap32 reverses the four bytes of a 32-bit word (GOST R 34.12-2015 reads keys
// and blocks big-endian, so the 64-bit engine byte-swaps around its core).
func swap32(b uint32) uint32 {
	return (b&0xff)<<24 | (b&0xff00)<<8 | (b>>8)&0xff00 | (b>>24)&0xff
}

// ---------------------------------------------------------------------------
// S-boxes (paramset substitution tables) for GOST 28147-89.
// ---------------------------------------------------------------------------

var gostSBoxes = map[string][]uint32{
	"E-TEST": {
		0x4, 0x2, 0xF, 0x5, 0x9, 0x1, 0x0, 0x8, 0xE, 0x3, 0xB, 0xC, 0xD, 0x7, 0xA, 0x6,
		0xC, 0x9, 0xF, 0xE, 0x8, 0x1, 0x3, 0xA, 0x2, 0x7, 0x4, 0xD, 0x6, 0x0, 0xB, 0x5,
		0xD, 0x8, 0xE, 0xC, 0x7, 0x3, 0x9, 0xA, 0x1, 0x5, 0x2, 0x4, 0x6, 0xF, 0x0, 0xB,
		0xE, 0x9, 0xB, 0x2, 0x5, 0xF, 0x7, 0x1, 0x0, 0xD, 0xC, 0x6, 0xA, 0x4, 0x3, 0x8,
		0x3, 0xE, 0x5, 0x9, 0x6, 0x8, 0x0, 0xD, 0xA, 0xB, 0x7, 0xC, 0x2, 0x1, 0xF, 0x4,
		0x8, 0xF, 0x6, 0xB, 0x1, 0x9, 0xC, 0x5, 0xD, 0x3, 0x7, 0xA, 0x0, 0xE, 0x2, 0x4,
		0x9, 0xB, 0xC, 0x0, 0x3, 0x6, 0x7, 0x5, 0x4, 0x8, 0xE, 0xF, 0x1, 0xA, 0x2, 0xD,
		0xC, 0x6, 0x5, 0x2, 0xB, 0x0, 0x9, 0xD, 0x3, 0xE, 0x7, 0xA, 0xF, 0x4, 0x1, 0x8,
	},
	"E-A": {
		0x9, 0x6, 0x3, 0x2, 0x8, 0xB, 0x1, 0x7, 0xA, 0x4, 0xE, 0xF, 0xC, 0x0, 0xD, 0x5,
		0x3, 0x7, 0xE, 0x9, 0x8, 0xA, 0xF, 0x0, 0x5, 0x2, 0x6, 0xC, 0xB, 0x4, 0xD, 0x1,
		0xE, 0x4, 0x6, 0x2, 0xB, 0x3, 0xD, 0x8, 0xC, 0xF, 0x5, 0xA, 0x0, 0x7, 0x1, 0x9,
		0xE, 0x7, 0xA, 0xC, 0xD, 0x1, 0x3, 0x9, 0x0, 0x2, 0xB, 0x4, 0xF, 0x8, 0x5, 0x6,
		0xB, 0x5, 0x1, 0x9, 0x8, 0xD, 0xF, 0x0, 0xE, 0x4, 0x2, 0x3, 0xC, 0x7, 0xA, 0x6,
		0x3, 0xA, 0xD, 0xC, 0x1, 0x2, 0x0, 0xB, 0x7, 0x5, 0x9, 0x4, 0x8, 0xF, 0xE, 0x6,
		0x1, 0xD, 0x2, 0x9, 0x7, 0xA, 0x6, 0x0, 0x8, 0xC, 0x4, 0x5, 0xF, 0x3, 0xB, 0xE,
		0xB, 0xA, 0xF, 0x5, 0x0, 0xC, 0xE, 0x8, 0x6, 0x2, 0x3, 0x9, 0x1, 0x7, 0xD, 0x4,
	},
	"E-B": {
		0x8, 0x4, 0xB, 0x1, 0x3, 0x5, 0x0, 0x9, 0x2, 0xE, 0xA, 0xC, 0xD, 0x6, 0x7, 0xF,
		0x0, 0x1, 0x2, 0xA, 0x4, 0xD, 0x5, 0xC, 0x9, 0x7, 0x3, 0xF, 0xB, 0x8, 0x6, 0xE,
		0xE, 0xC, 0x0, 0xA, 0x9, 0x2, 0xD, 0xB, 0x7, 0x5, 0x8, 0xF, 0x3, 0x6, 0x1, 0x4,
		0x7, 0x5, 0x0, 0xD, 0xB, 0x6, 0x1, 0x2, 0x3, 0xA, 0xC, 0xF, 0x4, 0xE, 0x9, 0x8,
		0x2, 0x7, 0xC, 0xF, 0x9, 0x5, 0xA, 0xB, 0x1, 0x4, 0x0, 0xD, 0x6, 0x8, 0xE, 0x3,
		0x8, 0x3, 0x2, 0x6, 0x4, 0xD, 0xE, 0xB, 0xC, 0x1, 0x7, 0xF, 0xA, 0x0, 0x9, 0x5,
		0x5, 0x2, 0xA, 0xB, 0x9, 0x1, 0xC, 0x3, 0x7, 0x4, 0xD, 0x0, 0x6, 0xF, 0x8, 0xE,
		0x0, 0x4, 0xB, 0xE, 0x8, 0x3, 0x7, 0x1, 0xA, 0x2, 0x9, 0x6, 0xF, 0xD, 0x5, 0xC,
	},
	"E-C": {
		0x1, 0xB, 0xC, 0x2, 0x9, 0xD, 0x0, 0xF, 0x4, 0x5, 0x8, 0xE, 0xA, 0x7, 0x6, 0x3,
		0x0, 0x1, 0x7, 0xD, 0xB, 0x4, 0x5, 0x2, 0x8, 0xE, 0xF, 0xC, 0x9, 0xA, 0x6, 0x3,
		0x8, 0x2, 0x5, 0x0, 0x4, 0x9, 0xF, 0xA, 0x3, 0x7, 0xC, 0xD, 0x6, 0xE, 0x1, 0xB,
		0x3, 0x6, 0x0, 0x1, 0x5, 0xD, 0xA, 0x8, 0xB, 0x2, 0x9, 0x7, 0xE, 0xF, 0xC, 0x4,
		0x8, 0xD, 0xB, 0x0, 0x4, 0x5, 0x1, 0x2, 0x9, 0x3, 0xC, 0xE, 0x6, 0xF, 0xA, 0x7,
		0xC, 0x9, 0xB, 0x1, 0x8, 0xE, 0x2, 0x4, 0x7, 0x3, 0x6, 0x5, 0xA, 0x0, 0xF, 0xD,
		0xA, 0x9, 0x6, 0x8, 0xD, 0xE, 0x2, 0x0, 0xF, 0x3, 0x5, 0xB, 0x4, 0x1, 0xC, 0x7,
		0x7, 0x4, 0x0, 0x5, 0xA, 0x2, 0xF, 0xE, 0xC, 0x6, 0x1, 0xB, 0xD, 0x9, 0x3, 0x8,
	},
	"E-D": {
		0xF, 0xC, 0x2, 0xA, 0x6, 0x4, 0x5, 0x0, 0x7, 0x9, 0xE, 0xD, 0x1, 0xB, 0x8, 0x3,
		0xB, 0x6, 0x3, 0x4, 0xC, 0xF, 0xE, 0x2, 0x7, 0xD, 0x8, 0x0, 0x5, 0xA, 0x9, 0x1,
		0x1, 0xC, 0xB, 0x0, 0xF, 0xE, 0x6, 0x5, 0xA, 0xD, 0x4, 0x8, 0x9, 0x3, 0x7, 0x2,
		0x1, 0x5, 0xE, 0xC, 0xA, 0x7, 0x0, 0xD, 0x6, 0x2, 0xB, 0x4, 0x9, 0x3, 0xF, 0x8,
		0x0, 0xC, 0x8, 0x9, 0xD, 0x2, 0xA, 0xB, 0x7, 0x3, 0x6, 0x5, 0x4, 0xE, 0xF, 0x1,
		0x8, 0x0, 0xF, 0x3, 0x2, 0x5, 0xE, 0xB, 0x1, 0xA, 0x4, 0x7, 0xC, 0x9, 0xD, 0x6,
		0x3, 0x0, 0x6, 0xF, 0x1, 0xE, 0x9, 0x2, 0xD, 0x8, 0xC, 0x4, 0xB, 0xA, 0x5, 0x7,
		0x1, 0xA, 0x6, 0x8, 0xF, 0xB, 0x0, 0x4, 0xC, 0x3, 0x5, 0x9, 0x7, 0xD, 0x2, 0xE,
	},
	"E-SC": {
		0x3, 0x6, 0x1, 0x0, 0x5, 0x7, 0xd, 0x9, 0x4, 0xb, 0x8, 0xc, 0xe, 0xf, 0x2, 0xa,
		0x7, 0x1, 0x5, 0x2, 0x8, 0xb, 0x9, 0xc, 0xd, 0x0, 0x3, 0xa, 0xf, 0xe, 0x4, 0x6,
		0xf, 0x1, 0x4, 0x6, 0xc, 0x8, 0x9, 0x2, 0xe, 0x3, 0x7, 0xa, 0xb, 0xd, 0x5, 0x0,
		0x3, 0x4, 0xf, 0xc, 0x5, 0x9, 0xe, 0x0, 0x6, 0x8, 0x7, 0xa, 0x1, 0xb, 0xd, 0x2,
		0x6, 0x9, 0x0, 0x7, 0xb, 0x8, 0x4, 0xc, 0x2, 0xe, 0xa, 0xf, 0x1, 0xd, 0x5, 0x3,
		0x6, 0x1, 0x2, 0xf, 0x0, 0xb, 0x9, 0xc, 0x7, 0xd, 0xa, 0x5, 0x8, 0x4, 0xe, 0x3,
		0x0, 0x2, 0xe, 0xc, 0x9, 0x1, 0x4, 0x7, 0x3, 0xf, 0x6, 0x8, 0xa, 0xd, 0xb, 0x5,
		0x5, 0x2, 0xb, 0x8, 0x4, 0xc, 0x7, 0x1, 0xa, 0x6, 0xe, 0x0, 0x9, 0x3, 0xd, 0xf,
	},
	"E-Z": {
		0xc, 0x4, 0x6, 0x2, 0xa, 0x5, 0xb, 0x9, 0xe, 0x8, 0xd, 0x7, 0x0, 0x3, 0xf, 0x1,
		0x6, 0x8, 0x2, 0x3, 0x9, 0xa, 0x5, 0xc, 0x1, 0xe, 0x4, 0x7, 0xb, 0xd, 0x0, 0xf,
		0xb, 0x3, 0x5, 0x8, 0x2, 0xf, 0xa, 0xd, 0xe, 0x1, 0x7, 0x4, 0xc, 0x9, 0x6, 0x0,
		0xc, 0x8, 0x2, 0x1, 0xd, 0x4, 0xf, 0x6, 0x7, 0x0, 0xa, 0x5, 0x3, 0xe, 0x9, 0xb,
		0x7, 0xf, 0x5, 0xa, 0x8, 0x1, 0x6, 0xd, 0x0, 0x9, 0x3, 0xe, 0xb, 0x4, 0x2, 0xc,
		0x5, 0xd, 0xf, 0x6, 0x9, 0x2, 0xc, 0xa, 0xb, 0x7, 0x8, 0x1, 0x4, 0x3, 0xe, 0x0,
		0x8, 0xe, 0x2, 0x5, 0x6, 0x9, 0x1, 0xc, 0xf, 0x4, 0xb, 0x0, 0xd, 0xa, 0x3, 0x7,
		0x1, 0x7, 0xe, 0xd, 0x0, 0x5, 0x8, 0x3, 0x4, 0xf, 0xa, 0x6, 0x9, 0xc, 0xb, 0x2,
	},
	"D-TEST": {
		0x4, 0xA, 0x9, 0x2, 0xD, 0x8, 0x0, 0xE, 0x6, 0xB, 0x1, 0xC, 0x7, 0xF, 0x5, 0x3,
		0xE, 0xB, 0x4, 0xC, 0x6, 0xD, 0xF, 0xA, 0x2, 0x3, 0x8, 0x1, 0x0, 0x7, 0x5, 0x9,
		0x5, 0x8, 0x1, 0xD, 0xA, 0x3, 0x4, 0x2, 0xE, 0xF, 0xC, 0x7, 0x6, 0x0, 0x9, 0xB,
		0x7, 0xD, 0xA, 0x1, 0x0, 0x8, 0x9, 0xF, 0xE, 0x4, 0x6, 0xC, 0xB, 0x2, 0x5, 0x3,
		0x6, 0xC, 0x7, 0x1, 0x5, 0xF, 0xD, 0x8, 0x4, 0xA, 0x9, 0xE, 0x0, 0x3, 0xB, 0x2,
		0x4, 0xB, 0xA, 0x0, 0x7, 0x2, 0x1, 0xD, 0x3, 0x6, 0x8, 0x5, 0x9, 0xC, 0xF, 0xE,
		0xD, 0xB, 0x4, 0x1, 0x3, 0xF, 0x5, 0x9, 0x0, 0xA, 0xE, 0x7, 0x6, 0x8, 0x2, 0xC,
		0x1, 0xF, 0xD, 0x0, 0x5, 0x7, 0xA, 0x4, 0x9, 0x2, 0x3, 0xE, 0x6, 0xB, 0x8, 0xC,
	},
	"D-A": {
		0xA, 0x4, 0x5, 0x6, 0x8, 0x1, 0x3, 0x7, 0xD, 0xC, 0xE, 0x0, 0x9, 0x2, 0xB, 0xF,
		0x5, 0xF, 0x4, 0x0, 0x2, 0xD, 0xB, 0x9, 0x1, 0x7, 0x6, 0x3, 0xC, 0xE, 0xA, 0x8,
		0x7, 0xF, 0xC, 0xE, 0x9, 0x4, 0x1, 0x0, 0x3, 0xB, 0x5, 0x2, 0x6, 0xA, 0x8, 0xD,
		0x4, 0xA, 0x7, 0xC, 0x0, 0xF, 0x2, 0x8, 0xE, 0x1, 0x6, 0x5, 0xD, 0xB, 0x9, 0x3,
		0x7, 0x6, 0x4, 0xB, 0x9, 0xC, 0x2, 0xA, 0x1, 0x8, 0x0, 0xE, 0xF, 0xD, 0x3, 0x5,
		0x7, 0x6, 0x2, 0x4, 0xD, 0x9, 0xF, 0x0, 0xA, 0x1, 0x5, 0xB, 0x8, 0xE, 0xC, 0x3,
		0xD, 0xE, 0x4, 0x1, 0x7, 0x0, 0x5, 0xA, 0x3, 0xC, 0x8, 0xF, 0x6, 0x2, 0x9, 0xB,
		0x1, 0x3, 0xA, 0x9, 0x5, 0xB, 0x4, 0xF, 0x8, 0x6, 0x7, 0xE, 0xD, 0x0, 0x2, 0xC,
	},
	"D-SC": {
		0xb, 0xd, 0x7, 0x0, 0x5, 0x4, 0x1, 0xf, 0x9, 0xe, 0x6, 0xa, 0x3, 0xc, 0x8, 0x2,
		0x1, 0x2, 0x7, 0x9, 0xd, 0xb, 0xf, 0x8, 0xe, 0xc, 0x4, 0x0, 0x5, 0x6, 0xa, 0x3,
		0x5, 0x1, 0xd, 0x3, 0xf, 0x6, 0xc, 0x7, 0x9, 0x8, 0xb, 0x2, 0x4, 0xe, 0x0, 0xa,
		0xd, 0x1, 0xb, 0x4, 0x9, 0xc, 0xe, 0x0, 0x7, 0x5, 0x8, 0xf, 0x6, 0x2, 0xa, 0x3,
		0x2, 0xd, 0xa, 0xf, 0x9, 0xb, 0x3, 0x7, 0x8, 0xc, 0x5, 0xe, 0x6, 0x0, 0x1, 0x4,
		0x0, 0x4, 0x6, 0xc, 0x5, 0x3, 0x8, 0xd, 0xa, 0xb, 0xf, 0x2, 0x1, 0x9, 0x7, 0xe,
		0x1, 0x3, 0xc, 0x8, 0xa, 0x6, 0xb, 0x0, 0x2, 0xe, 0x7, 0x9, 0xf, 0x4, 0x5, 0xd,
		0xa, 0xb, 0x6, 0x0, 0x1, 0x3, 0x4, 0x7, 0xe, 0xd, 0x5, 0xf, 0x8, 0x2, 0x9, 0xc,
	},
}

// gostMeshC is the constant used by CryptoPro key meshing (RFC 4357 2.3.2).
var gostMeshC = []byte{
	0x69, 0x00, 0x72, 0x22, 0x64, 0xC9, 0x04, 0x23,
	0x8D, 0x3A, 0xDB, 0x96, 0x46, 0xE9, 0x2A, 0xC4,
	0x18, 0xFE, 0xAC, 0x94, 0x00, 0xED, 0x07, 0x12,
	0xC0, 0x86, 0xDC, 0xC2, 0xEF, 0x4C, 0xA9, 0x2B,
}

// ---------------------------------------------------------------------------
// GOST 28147-89 / Magma 64-bit core.
// ---------------------------------------------------------------------------

// gostRound is one round of the GOST 64-bit Feistel network. It returns the new
// (m0, m1) pair, matching round() in gostCipher.mjs.
func gostRound(s []uint32, m0, m1, k uint32) (uint32, uint32) {
	cm := m0 + k
	om := s[0+((cm>>0)&0xF)]
	om |= s[16+((cm>>4)&0xF)] << 4
	om |= s[32+((cm>>8)&0xF)] << 8
	om |= s[48+((cm>>12)&0xF)] << 12
	om |= s[64+((cm>>16)&0xF)] << 16
	om |= s[80+((cm>>20)&0xF)] << 20
	om |= s[96+((cm>>24)&0xF)] << 24
	om |= s[112+((cm>>28)&0xF)] << 28
	cm = om<<11 | om>>21
	cm ^= m1
	return cm, m0
}

// readKey8 reads the user key as up to eight little-endian uint32 words,
// zero-extending short keys (the JS engine indexes an Int32Array past its end,
// yielding 0). Callers validate that the key length is a multiple of 4 before
// scheduling (see requireValidKey), so any trailing bytes here are impossible.
func readKey8(k []byte) [8]uint32 {
	var w [8]uint32
	for i := 0; i < 8 && (i+1)*4 <= len(k); i++ {
		w[i] = le32(k, i*4)
	}
	return w
}

// keySchedule89 expands the key into the 32-subkey schedule for GOST 28147-89.
func keySchedule89(k []byte, decrypt bool) []uint32 {
	return buildSchedule(readKey8(k), decrypt)
}

// keySchedule15 is keySchedule89 with each word byte-swapped (GOST R 34.12-2015).
func keySchedule15(k []byte, decrypt bool) []uint32 {
	key := readKey8(k)
	for i := range key {
		key[i] = swap32(key[i])
	}
	return buildSchedule(key, decrypt)
}

// buildSchedule lays the eight key words into the 32-round schedule: forward
// order three times then reversed for the final eight rounds (reversed twice
// more for decryption).
func buildSchedule(key [8]uint32, decrypt bool) []uint32 {
	sch := make([]uint32, 32)
	for i := range 8 {
		sch[i] = key[i]
	}
	if decrypt {
		for i := range 8 {
			sch[i+8] = sch[7-i]
			sch[i+16] = sch[7-i]
		}
	} else {
		for i := range 8 {
			sch[i+8] = sch[i]
			sch[i+16] = sch[i]
		}
	}
	for i := range 8 {
		sch[i+24] = sch[7-i]
	}
	return sch
}

// process89 encrypts/decrypts one 64-bit block in place (GOST 28147-89).
func (c *gostCipher) process89(k []uint32, d []byte, ofs int) {
	m0, m1 := le32(d, ofs), le32(d, ofs+4)
	for i := range 32 {
		m0, m1 = gostRound(c.sBox, m0, m1, k[i])
	}
	putLE32(d, ofs, m1)
	putLE32(d, ofs+4, m0)
}

// process15 encrypts/decrypts one 64-bit block in place (GOST R 34.12-2015 Magma).
func (c *gostCipher) process15(k []uint32, d []byte, ofs int) {
	m0, m1 := le32(d, ofs), le32(d, ofs+4)
	r := swap32(m0)
	m0 = swap32(m1)
	m1 = r
	for i := range 32 {
		m0, m1 = gostRound(c.sBox, m0, m1, k[i])
	}
	putLE32(d, ofs, swap32(m0))
	putLE32(d, ofs+4, swap32(m1))
}

// ---------------------------------------------------------------------------
// GOST R 34.12-2015 Kuznyechik 128-bit core.
// ---------------------------------------------------------------------------

// kPi is the Kuznyechik non-linear substitution; kReversePi is its inverse.
var kPi = [256]byte{
	252, 238, 221, 17, 207, 110, 49, 22, 251, 196, 250, 218, 35, 197, 4, 77,
	233, 119, 240, 219, 147, 46, 153, 186, 23, 54, 241, 187, 20, 205, 95, 193,
	249, 24, 101, 90, 226, 92, 239, 33, 129, 28, 60, 66, 139, 1, 142, 79,
	5, 132, 2, 174, 227, 106, 143, 160, 6, 11, 237, 152, 127, 212, 211, 31,
	235, 52, 44, 81, 234, 200, 72, 171, 242, 42, 104, 162, 253, 58, 206, 204,
	181, 112, 14, 86, 8, 12, 118, 18, 191, 114, 19, 71, 156, 183, 93, 135,
	21, 161, 150, 41, 16, 123, 154, 199, 243, 145, 120, 111, 157, 158, 178, 177,
	50, 117, 25, 61, 255, 53, 138, 126, 109, 84, 198, 128, 195, 189, 13, 87,
	223, 245, 36, 169, 62, 168, 67, 201, 215, 121, 214, 246, 124, 34, 185, 3,
	224, 15, 236, 222, 122, 148, 176, 188, 220, 232, 40, 80, 78, 51, 10, 74,
	167, 151, 96, 115, 30, 0, 98, 68, 26, 184, 56, 130, 100, 159, 38, 65,
	173, 69, 70, 146, 39, 94, 85, 47, 140, 163, 165, 125, 105, 213, 149, 59,
	7, 88, 179, 64, 134, 172, 29, 247, 48, 55, 107, 228, 136, 217, 231, 137,
	225, 27, 131, 73, 76, 63, 248, 254, 141, 83, 170, 144, 202, 216, 133, 97,
	32, 113, 103, 164, 45, 43, 9, 91, 203, 155, 37, 208, 190, 229, 108, 82,
	89, 166, 116, 210, 230, 244, 180, 192, 209, 102, 175, 194, 57, 75, 99, 182,
}

var kReversePi = func() [256]byte {
	var m [256]byte
	for i, v := range kPi {
		m[v] = byte(i)
	}
	return m
}()

// kB indexes gostMultTable for the linear R function.
var kB = [16]int{4, 2, 3, 1, 6, 5, 0, 7, 0, 5, 6, 1, 3, 2, 4, 0}

// gostMultTable[i][j] is x[i] multiplied by j in GF(2^8) (poly x^8+x^7+x^6+x+1),
// for the eight coefficients the R function needs.
var gostMultTable = func() [8][256]byte {
	gmul := func(a, b byte) byte {
		var p byte
		for range 8 {
			if b&1 != 0 {
				p ^= a
			}
			carry := a & 0x80
			a = (a << 1) & 0xff
			if carry != 0 {
				a ^= 0xc3
			}
			b >>= 1
		}
		return p & 0xff
	}
	x := [8]byte{1, 16, 32, 133, 148, 192, 194, 251}
	var m [8][256]byte
	for i := range 8 {
		for j := range 256 {
			m[i][j] = gmul(x[i], byte(j))
		}
	}
	return m
}()

func funcR(d []byte) {
	var sum byte
	for i := range 16 {
		sum ^= gostMultTable[kB[i]][d[i]]
	}
	for i := 15; i >= 1; i-- {
		d[i] = d[i-1]
	}
	d[0] = sum
}

func funcReverseR(d []byte) {
	tmp := d[0]
	for i := range 15 {
		d[i] = d[i+1]
	}
	d[15] = tmp
	var sum byte
	for i := range 16 {
		sum ^= gostMultTable[kB[i]][d[i]]
	}
	d[15] = sum
}

func funcS(d []byte) {
	for i := range 16 {
		d[i] = kPi[d[i]]
	}
}

func funcReverseS(d []byte) {
	for i := range 16 {
		d[i] = kReversePi[d[i]]
	}
}

func funcX(a, b []byte) {
	for i := range 16 {
		a[i] ^= b[i]
	}
}

func funcL(d []byte) {
	for range 16 {
		funcR(d)
	}
}

func funcReverseL(d []byte) {
	for range 16 {
		funcReverseR(d)
	}
}

func funcLSX(a, b []byte) {
	funcX(a, b)
	funcS(a)
	funcL(a)
}

func funcReverseLSX(a, b []byte) {
	funcX(a, b)
	funcReverseL(a)
	funcReverseS(a)
}

func funcF(inputKey, inputKeySecond, iterationConst []byte) {
	tmp := make([]byte, 16)
	copy(tmp, inputKey)
	funcLSX(inputKey, iterationConst)
	funcX(inputKey, inputKeySecond)
	copy(inputKeySecond, tmp)
}

func funcC(number int, d []byte) {
	for i := range 15 {
		d[i] = 0
	}
	d[15] = byte(number) // #nosec G115 -- Kuznyechik round index (1..32) fits a byte
	funcL(d)
}

// keySchedule128 expands the key into ten 128-bit round keys (160 bytes).
func keySchedule128(k []byte) []byte {
	keys := make([]byte, 160)
	copy(keys, k)
	c := make([]byte, 16)
	for j := range 4 {
		j0, j1 := 32*j, 32*(j+1)
		copy(keys[j1:j1+32], keys[j0:j0+32])
		for i := 1; i < 9; i++ {
			funcC(j*8+i, c)
			funcF(keys[j1:j1+16], keys[j1+16:j1+32], c)
		}
	}
	return keys
}

// process128 encrypts (decrypt=false) or decrypts one 128-bit block in place.
func process128(k, d []byte, ofs int, decrypt bool) {
	r := d[ofs : ofs+16]
	if decrypt {
		for i := range 9 {
			funcReverseLSX(r, k[(9-i)*16:(9-i)*16+16])
		}
		funcX(r, k[0:16])
	} else {
		for i := range 9 {
			funcLSX(r, k[16*i:16*i+16])
		}
		funcX(r, k[16*9:16*9+16])
	}
}

// ---------------------------------------------------------------------------
// Cipher instance and dispatch.
// ---------------------------------------------------------------------------

// gostSchedKey holds an expanded key: k64 for the 64-bit ciphers, k128 for
// Kuznyechik. Key meshing rewrites the active schedule in place.
type gostSchedKey struct {
	k64  []uint32
	k128 []byte
}

// gostCipher is a configured GOST cipher instance (see newGostCipher).
type gostCipher struct {
	version        int // 1989 or 2015
	blockLength    int // 64 or 128 bits
	blockSize      int // block length in bytes
	keySize        int // always 32
	sBox           []uint32
	iv             []byte
	ukm            []byte
	macLength      int    // bits
	shiftBits      int    // bits (feedback register width for CFB/OFB/CTR)
	mode           string // ES / MAC / KW
	block          string // ECB / CFB / OFB / CTR / CBC
	keyMeshingMode string // NO / CP
	padding        string // NO / PKCS5 / ZERO / RANDOM / BIT
	keyWrapping    string // NO / CP / SC
}

// keySchedule expands a key for the current algorithm/direction. The key length
// is validated at the public entry (requireValidKey), so scheduling is total.
func (c *gostCipher) keySchedule(k []byte, decrypt bool) gostSchedKey {
	if c.blockSize == gostBlockSize128 {
		return gostSchedKey{k128: keySchedule128(k)}
	}
	if c.version == 2015 {
		return gostSchedKey{k64: keySchedule15(k, decrypt)}
	}
	return gostSchedKey{k64: keySchedule89(k, decrypt)}
}

// requireValidKey rejects a 64-bit-cipher key whose length is not a multiple of
// 4 (the JS engine throws a RangeError when it lays an Int32Array over such a
// key). The 128-bit cipher and the SignalCom unpacked key are always valid.
func (c *gostCipher) requireValidKey(k []byte) error {
	if c.blockSize == gostBlockSize64 && len(k)%4 != 0 {
		return errors.New("byte length of Int32Array should be a multiple of 4") //nolint:staticcheck,revive // CyberChef's verbatim RangeError text
	}
	return nil
}

// process encrypts/decrypts one block in place at offset ofs.
func (c *gostCipher) process(key gostSchedKey, d []byte, ofs int, decrypt bool) {
	switch {
	case c.blockSize == gostBlockSize128:
		process128(key.k128, d, ofs, decrypt)
	case c.version == 2015:
		c.process15(key.k64, d, ofs)
	default:
		c.process89(key.k64, d, ofs)
	}
}

// cloneBytes returns a copy of b.
func cloneBytes(b []byte) []byte {
	out := make([]byte, len(b))
	copy(out, b)
	return out
}

// ivOr returns a working copy of iv, or the instance IV when iv is nil.
func (c *gostCipher) ivOr(iv []byte) []byte {
	if iv == nil {
		return cloneBytes(c.iv)
	}
	return cloneBytes(iv)
}

// ---------------------------------------------------------------------------
// Padding.
// ---------------------------------------------------------------------------

// pad applies the configured padding (only ECB/CBC in ES mode pad).
func (c *gostCipher) pad(d []byte) []byte {
	if c.mode != "ES" || (c.block != "ECB" && c.block != "CBC") {
		return cloneBytes(d)
	}
	switch c.padding {
	case "RANDOM":
		return randomPad(d, c.blockSize)
	case "BIT":
		return bitPad(d, c.blockSize)
	default: // NO / PKCS5 / ZERO all zero-pad (matching the upstream switch)
		return zeroPad(d, c.blockSize)
	}
}

// unpad reverses pad. Only BIT padding is removed on decrypt; ZERO/RANDOM keep
// the padding bytes (matching the upstream unpad selection).
func (c *gostCipher) unpad(d []byte) ([]byte, error) {
	if c.mode == "ES" && (c.block == "ECB" || c.block == "CBC") && c.padding == "BIT" {
		return bitUnpad(d)
	}
	return d, nil
}

func zeroPad(d []byte, blockSize int) []byte {
	m := ((len(d) + blockSize - 1) / blockSize) * blockSize
	r := make([]byte, m)
	copy(r, d)
	return r
}

func bitPad(d []byte, blockSize int) []byte {
	m := ((len(d) + 1 + blockSize - 1) / blockSize) * blockSize
	r := make([]byte, m)
	copy(r, d)
	r[len(d)] = 1
	return r
}

func bitUnpad(d []byte) ([]byte, error) {
	n := len(d)
	for n > 1 && d[n-1] == 0 {
		n--
	}
	if d[n-1] != 1 {
		return nil, errors.New("Invalid padding") //nolint:staticcheck,revive // CyberChef's verbatim DataError text
	}
	n--
	return cloneBytes(d[:n]), nil
}

func randomPad(d []byte, blockSize int) []byte {
	m := ((len(d) + blockSize - 1) / blockSize) * blockSize
	r := make([]byte, m)
	copy(r, d)
	// The trailing (m-len(d)) bytes are random.
	_, _ = rand.Read(r[len(d):])
	return r
}

// ---------------------------------------------------------------------------
// Key meshing (RFC 4357).
// ---------------------------------------------------------------------------

// keyMeshing applies CryptoPro key meshing every 1024 bytes; NO meshing is a
// no-op. It rewrites the active schedule in place and returns the meshed key.
// All of its sub-operations take an already-valid 32-byte key, so it is total.
func (c *gostCipher) keyMeshing(k, s []byte, i int, key gostSchedKey, decrypt bool) []byte {
	if c.keyMeshingMode != "CP" || (i+1)*c.blockSize%1024 != 0 {
		return k
	}
	meshed := c.decryptBlocks(k, gostMeshC)
	copy(s, c.encryptECB(meshed, s))
	fresh := c.keySchedule(meshed, decrypt)
	if key.k64 != nil {
		copy(key.k64, fresh.k64)
	} else {
		copy(key.k128, fresh.k128)
	}
	return meshed
}

// ---------------------------------------------------------------------------
// Block modes. Key scheduling is total (see requireValidKey), so encryption and
// the raw decryption primitives never fail; only ES-mode unpadding can.
// ---------------------------------------------------------------------------

func (c *gostCipher) encryptECB(k, d []byte) []byte {
	p := c.pad(d)
	n := c.blockSize
	key := c.keySchedule(k, false)
	for i := 0; i < len(p)/n; i++ {
		c.process(key, p, n*i, false)
	}
	return p
}

// decryptBlocks decrypts each block in place without unpadding.
func (c *gostCipher) decryptBlocks(k, d []byte) []byte {
	p := cloneBytes(d)
	n := c.blockSize
	key := c.keySchedule(k, true)
	for i := 0; i < len(p)/n; i++ {
		c.process(key, p, n*i, true)
	}
	return p
}

func (c *gostCipher) decryptECB(k, d []byte) ([]byte, error) {
	return c.unpad(c.decryptBlocks(k, d))
}

func (c *gostCipher) encryptCFB(k, d, iv []byte) []byte {
	s := c.ivOr(iv)
	cc := cloneBytes(d)
	m := len(s)
	t := make([]byte, m)
	b := c.shiftBits >> 3
	r := len(cc) % b
	q := (len(cc) - r) / b
	key := c.keySchedule(k, false)
	for i := range q {
		copy(t, s)
		c.process(key, s, 0, false)
		for j := range b {
			cc[i*b+j] ^= s[j]
		}
		for j := 0; j < m-b; j++ {
			s[j] = t[b+j]
		}
		for j := range b {
			s[m-b+j] = cc[i*b+j]
		}
		k = c.keyMeshing(k, s, i, key, false)
	}
	if r > 0 {
		c.process(key, s, 0, false)
		for i := range r {
			cc[q*b+i] ^= s[i]
		}
	}
	return cc
}

func (c *gostCipher) decryptCFB(k, d, iv []byte) []byte {
	s := c.ivOr(iv)
	cc := cloneBytes(d)
	m := len(s)
	t := make([]byte, m)
	b := c.shiftBits >> 3
	r := len(cc) % b
	q := (len(cc) - r) / b
	key := c.keySchedule(k, false)
	for i := range q {
		copy(t, s)
		c.process(key, s, 0, false)
		for j := range b {
			t[j] = cc[i*b+j]
			cc[i*b+j] ^= s[j]
		}
		for j := 0; j < m-b; j++ {
			s[j] = t[b+j]
		}
		for j := range b {
			s[m-b+j] = t[j]
		}
		k = c.keyMeshing(k, s, i, key, false)
	}
	if r > 0 {
		c.process(key, s, 0, false)
		for i := range r {
			cc[q*b+i] ^= s[i]
		}
	}
	return cc
}

func (c *gostCipher) processOFB(k, d, iv []byte) []byte {
	s := c.ivOr(iv)
	cc := cloneBytes(d)
	m := len(s)
	t := make([]byte, m)
	b := c.shiftBits >> 3
	p := make([]byte, b)
	r := len(cc) % b
	q := (len(cc) - r) / b
	key := c.keySchedule(k, false)
	for i := range q {
		copy(t, s)
		c.process(key, s, 0, false)
		copy(p, s[:b])
		for j := range b {
			cc[i*b+j] ^= s[j]
		}
		for j := 0; j < m-b; j++ {
			s[j] = t[b+j]
		}
		for j := range b {
			s[m-b+j] = p[j]
		}
		k = c.keyMeshing(k, s, i, key, false)
	}
	if r > 0 {
		c.process(key, s, 0, false)
		for i := range r {
			cc[q*b+i] ^= s[i]
		}
	}
	return cc
}

// ctr89Inc advances the GOST 28147-89 counter in s (RFC 5830 gammirovanie).
func ctr89Inc(s []byte) {
	putLE32(s, 0, le32(s, 0)+0x1010101)
	tmp := uint64(le32(s, 4)) + 0x1010104
	if tmp >= 0x100000000 {
		tmp -= 0xffffffff
	}
	putLE32(s, 4, uint32(tmp)) // #nosec G115 -- 32-bit counter wrap, matching the JS Int32Array
}

func (c *gostCipher) processCTR89(k, d, iv []byte) []byte {
	s := c.ivOr(iv)
	cc := cloneBytes(d)
	b := c.blockSize
	t := make([]byte, b)
	r := len(cc) % b
	q := (len(cc) - r) / b
	key := c.keySchedule(k, false)
	c.process(key, s, 0, false)
	for i := range q {
		ctr89Inc(s)
		copy(t, s[:b])
		c.process(key, s, 0, false)
		for j := range b {
			cc[i*b+j] ^= s[j]
		}
		copy(s[:b], t)
		k = c.keyMeshing(k, s, i, key, false)
	}
	if r > 0 {
		ctr89Inc(s)
		c.process(key, s, 0, false)
		for i := range r {
			cc[q*b+i] ^= s[i]
		}
	}
	return cc
}

func (c *gostCipher) processCTR15(k, d, iv []byte) []byte {
	cc := cloneBytes(d)
	n := c.blockSize
	b := c.shiftBits >> 3
	r := len(cc) % b
	q := (len(cc) - r) / b
	s := make([]byte, n)
	t := make([]int32, n)
	key := c.keySchedule(k, false)
	copy(s, c.ivOr(iv))
	for i := 0; i < q; i++ {
		for j := range n {
			t[j] = int32(s[j])
		}
		c.process(key, s, 0, false)
		for j := range b {
			cc[b*i+j] ^= s[j]
		}
		for j := range n {
			s[j] = byte(t[j]) // #nosec G115 -- t[j] holds a saved byte value (0..255)
		}
		// Upstream counter loop: the loop variable is the OUTER index i, so a
		// carry (s[j] > 0xfe) rewinds the block index. Replicated verbatim.
		for j := n - 1; i >= 0; i-- {
			if s[j] > 0xfe {
				s[j] -= 0xfe
			} else {
				s[j]++
				break
			}
		}
	}
	if r > 0 {
		c.process(key, s, 0, false)
		for j := range r {
			cc[b*q+j] ^= s[j]
		}
	}
	return cc
}

func (c *gostCipher) encryptCBC(k, d, iv []byte) []byte {
	s := c.ivOr(iv)
	n := c.blockSize
	m := len(s)
	cc := c.pad(d)
	key := c.keySchedule(k, false)
	for i := 0; i < len(cc)/n; i++ {
		for j := range n {
			s[j] ^= cc[i*n+j]
		}
		c.process(key, s, 0, false)
		for j := range n {
			cc[i*n+j] = s[j]
		}
		if m != n {
			for j := 0; j < m-n; j++ {
				s[j] = s[n+j]
			}
			for j := range n {
				s[j+m-n] = cc[i*n+j]
			}
		}
		k = c.keyMeshing(k, s, i, key, false)
	}
	return cc
}

func (c *gostCipher) decryptCBC(k, d, iv []byte) ([]byte, error) {
	s := c.ivOr(iv)
	n := c.blockSize
	m := len(s)
	cc := cloneBytes(d)
	next := make([]byte, n)
	key := c.keySchedule(k, true)
	for i := 0; i < len(cc)/n; i++ {
		for j := range n {
			next[j] = cc[i*n+j]
		}
		c.process(key, cc, i*n, true)
		for j := range n {
			cc[i*n+j] ^= s[j]
		}
		if m != n {
			for j := 0; j < m-n; j++ {
				s[j] = s[n+j]
			}
		}
		for j := range n {
			s[j+m-n] = next[j]
		}
		k = c.keyMeshing(k, s, i, key, true)
	}
	return c.unpad(cc)
}

// ---------------------------------------------------------------------------
// Public entry points.
// ---------------------------------------------------------------------------

// Encrypt encrypts d with key k in the configured block mode.
func (c *gostCipher) Encrypt(k, d []byte) ([]byte, error) {
	if err := c.requireValidKey(k); err != nil {
		return nil, err
	}
	switch c.block {
	case "CFB":
		return c.encryptCFB(k, d, nil), nil
	case "OFB":
		return c.processOFB(k, d, nil), nil
	case "CTR":
		if c.version == 2015 {
			return c.processCTR15(k, d, nil), nil
		}
		return c.processCTR89(k, d, nil), nil
	case "CBC":
		return c.encryptCBC(k, d, nil), nil
	default: // ECB
		return c.encryptECB(k, d), nil
	}
}

// Decrypt decrypts d with key k in the configured block mode.
func (c *gostCipher) Decrypt(k, d []byte) ([]byte, error) {
	if err := c.requireValidKey(k); err != nil {
		return nil, err
	}
	switch c.block {
	case "CFB":
		return c.decryptCFB(k, d, nil), nil
	case "OFB":
		return c.processOFB(k, d, nil), nil
	case "CTR":
		if c.version == 2015 {
			return c.processCTR15(k, d, nil), nil
		}
		return c.processCTR89(k, d, nil), nil
	case "CBC":
		return c.decryptCBC(k, d, nil)
	default: // ECB
		return c.decryptECB(k, d)
	}
}

// ---------------------------------------------------------------------------
// MAC (imitovstavka) mode.
// ---------------------------------------------------------------------------

// processMAC accumulates the MAC state s over data d (dispatched by version).
func (c *gostCipher) processMAC(key gostSchedKey, s, d []byte) {
	if c.version == 2015 {
		c.processMAC15(key, s, d)
	} else {
		c.processMAC89(key, s, d)
	}
}

// processMAC89 is the GOST 28147-89 imitovstavka: each block runs 16 rounds.
func (c *gostCipher) processMAC89(key gostSchedKey, s, d []byte) {
	cc := zeroPad(d, c.blockSize)
	n := c.blockSize
	for i := 0; i < len(cc)/n; i++ {
		for j := range n {
			s[j] ^= cc[i*n+j]
		}
		m0, m1 := le32(s, 0), le32(s, 4)
		for j := range 16 {
			m0, m1 = gostRound(c.sBox, m0, m1, key.k64[j])
		}
		putLE32(s, 0, m0)
		putLE32(s, 4, m1)
	}
}

// processKeyMAC15 derives the next CMAC subkey by a one-bit left shift with the
// GF(2) reduction polynomial (0x87 for 128-bit blocks, 0x1b for 64-bit).
func processKeyMAC15(s []byte) {
	var t byte
	for i := len(s) - 1; i >= 0; i-- {
		t1 := s[i] >> 7
		s[i] = (s[i]<<1)&0xff | t
		t = t1
	}
	if t != 0 {
		if len(s) == 16 {
			s[15] ^= 0x87
		} else {
			s[7] ^= 0x1b
		}
	}
}

// processMAC15 is the GOST R 34.12-2015 CMAC-style MAC.
func (c *gostCipher) processMAC15(key gostSchedKey, s, d []byte) {
	n := c.blockSize
	r := make([]byte, n)
	c.process(key, r, 0, false)
	processKeyMAC15(r) // K1
	cc := d
	if len(d)%n != 0 {
		cc = bitPad(d, n)
		processKeyMAC15(r) // K2
	}
	q := len(cc) / n
	for i := range q {
		for j := range n {
			s[j] ^= cc[i*n+j]
		}
		if i == q-1 {
			for j := range n {
				s[j] ^= r[j]
			}
		}
		c.process(key, s, 0, false)
	}
}

// signMAC computes the MAC of d under key k (iv defaults to the instance IV).
// The key is validated by the caller, so this is total.
func (c *gostCipher) signMAC(k, d, iv []byte) []byte {
	key := c.keySchedule(k, false)
	s := c.ivOr(iv)
	m := c.macLength >> 3
	if m == 0 {
		m = c.blockSize >> 1
	}
	c.processMAC(key, s, d)
	mac := make([]byte, m)
	copy(mac, s[:m])
	return mac
}

// verifyMAC recomputes the MAC and compares it to the supplied value.
func (c *gostCipher) verifyMAC(k, mac, d, iv []byte) bool {
	computed := c.signMAC(k, d, iv)
	return len(computed) == len(mac) && bytes.Equal(computed, mac)
}

// Sign computes the MAC of d under key k.
func (c *gostCipher) Sign(k, d []byte) ([]byte, error) {
	if err := c.requireValidKey(k); err != nil {
		return nil, err
	}
	return c.signMAC(k, d, nil), nil
}

// Verify checks mac against the MAC of d under key k.
func (c *gostCipher) Verify(k, mac, d []byte) (bool, error) {
	if err := c.requireValidKey(k); err != nil {
		return false, err
	}
	return c.verifyMAC(k, mac, d, nil), nil
}

// ---------------------------------------------------------------------------
// Key wrapping (RFC 4357).
// ---------------------------------------------------------------------------

// diversifyKEK implements the CryptoPro KEK diversification algorithm (RFC 4357
// 6.5). It is a 64-bit-only construction; a 128-bit block errors, matching the
// upstream typed-array failure.
func (c *gostCipher) diversifyKEK(kek, ukm []byte) ([]byte, error) {
	if c.blockSize != gostBlockSize64 {
		return nil, errGostInvalidTypedArray
	}
	n := c.blockSize
	kbytes := cloneBytes(kek)
	for i := range n {
		var ui byte
		if i < len(ukm) {
			ui = ukm[i]
		}
		var s0, s1 uint32
		for j := range 8 {
			// For a key shorter than 8 words, k[j] is undefined in JS and
			// `s = (s + undefined) & 0xffffffff` evaluates to 0 — the accumulator
			// is reset, not left unchanged.
			defined := (j+1)*4 <= len(kbytes)
			var w uint32
			if defined {
				w = le32(kbytes, j*4)
			}
			if (ui>>uint(j))&1 == 1 {
				if defined {
					s0 += w
				} else {
					s0 = 0
				}
			} else if defined {
				s1 += w
			} else {
				s1 = 0
			}
		}
		iv := make([]byte, 8)
		putLE32(iv, 0, s0)
		putLE32(iv, 4, s1)
		kbytes = c.encryptCFB(kbytes, kbytes, iv)
	}
	return kbytes, nil
}

// assembleWrap builds the (CEK_ENC | zero-fill | CEK_MAC) wrapped key structure
// shared by the GOST, CryptoPro and SignalCom wrapping algorithms.
func (c *gostCipher) assembleWrap(enc, mac []byte) []byte {
	r := make([]byte, c.keySize+c.blockSize>>1)
	copy(r, enc)
	copy(r[c.keySize:], mac)
	return r
}

// errGostNoUKM is CyberChef's verbatim message when a KW operation lacks a UKM.
// It is unreachable through the operations (they always supply one, validated by
// the constructor), but the wrap primitives are exported as a faithful port.
//
//nolint:staticcheck,revive // CyberChef's verbatim DataError text
var errGostNoUKM = errors.New("UKM must be defined")

// wrapKeyGOST wraps a CEK under a KEK (RFC 4357 6.1 GOST 28147-89 Key Wrap).
func (c *gostCipher) wrapKeyGOST(kek, cek []byte) ([]byte, error) {
	if c.ukm == nil {
		return nil, errGostNoUKM
	}
	mac := c.signMAC(kek, cek, c.ukm)
	enc := c.encryptECB(kek, cek)
	return c.assembleWrap(enc, mac), nil
}

// unwrapKeyGOST reverses wrapKeyGOST (RFC 4357 6.2).
func (c *gostCipher) unwrapKeyGOST(kek, data []byte) ([]byte, error) {
	length := c.keySize + c.blockSize>>1
	if len(data) != length {
		return nil, fmt.Errorf("Wrapping key size must be %d bytes", length) //nolint:staticcheck,revive // CyberChef's verbatim DataError text
	}
	if c.ukm == nil {
		return nil, errGostNoUKM
	}
	cek := c.decryptBlocks(kek, data[:c.keySize])
	mac := data[c.keySize : c.keySize+c.blockSize>>1]
	if !c.verifyMAC(kek, mac, cek, c.ukm) {
		return nil, errors.New("Error verify MAC of wrapping key") //nolint:staticcheck,revive // CyberChef's verbatim DataError text
	}
	return cek, nil
}

// wrapKeyCP wraps a CEK under a diversified KEK (RFC 4357 6.3 CryptoPro Key Wrap).
func (c *gostCipher) wrapKeyCP(kek, cek []byte) ([]byte, error) {
	if c.ukm == nil {
		return nil, errGostNoUKM
	}
	dek, err := c.diversifyKEK(kek, c.ukm)
	if err != nil {
		return nil, err
	}
	mac := c.signMAC(dek, cek, c.ukm)
	enc := c.encryptECB(dek, cek)
	return c.assembleWrap(enc, mac), nil
}

// unwrapKeyCP reverses wrapKeyCP (RFC 4357 6.4).
func (c *gostCipher) unwrapKeyCP(kek, data []byte) ([]byte, error) {
	length := c.keySize + c.blockSize>>1
	if len(data) != length {
		return nil, fmt.Errorf("Wrapping key size must be %d bytes", length) //nolint:staticcheck,revive // CyberChef's verbatim DataError text
	}
	if c.ukm == nil {
		return nil, errGostNoUKM
	}
	dek, err := c.diversifyKEK(kek, c.ukm)
	if err != nil {
		return nil, err
	}
	cek := c.decryptBlocks(dek, data[:c.keySize])
	mac := data[c.keySize : c.keySize+c.blockSize>>1]
	if !c.verifyMAC(dek, mac, cek, c.ukm) {
		return nil, errors.New("Error verify MAC of wrapping key") //nolint:staticcheck,revive // CyberChef's verbatim DataError text
	}
	return cek, nil
}

// unpackKeySC recovers a clear KEK from a SignalCom packed master key (mk.db3 +
// masks): magic byte, mask count, MAC, then the XOR of the mask blocks. The MAC
// is checked against the default S-box, then a set of alternates.
func (c *gostCipher) unpackKeySC(packed []byte) ([]byte, error) {
	m, n := c.blockSize>>1, c.keySize
	if len(packed) < 2+m {
		return nil, errGostInvalidTypedArray
	}
	if packed[0] != 0x22 {
		return nil, errors.New("Invalid magic number") //nolint:staticcheck,revive // CyberChef's verbatim DataError text
	}
	mcount := int(packed[1])
	mac := packed[2 : 2+m]
	if 2+m+n*mcount > len(packed) {
		return nil, errGostInvalidTypedArray
	}
	key := make([]byte, n)
	for i := range mcount {
		mask := packed[2+m+n*i : 2+m+n*i+n]
		for j := range n {
			key[j] ^= mask[j]
		}
	}
	zero := make([]byte, n)
	ok := c.verifyMAC(key, mac, zero, nil)
	for _, name := range []string{"E-A", "E-B", "E-C", "E-D", "E-SC"} {
		if ok {
			break
		}
		c.sBox = gostSBoxes[name]
		ok = c.verifyMAC(key, mac, zero, nil)
	}
	if !ok {
		return nil, errors.New("Invalid main key MAC") //nolint:staticcheck,revive // CyberChef's verbatim DataError text
	}
	return key, nil
}

// wrapKeySC wraps a CEK under a SignalCom KEK (clear 32-byte key or packed blob).
func (c *gostCipher) wrapKeySC(kek, cek []byte) ([]byte, error) {
	if len(kek) != c.keySize {
		unpacked, err := c.unpackKeySC(kek)
		if err != nil {
			return nil, err
		}
		kek = unpacked
	}
	enc := c.encryptECB(kek, cek)
	mac := c.signMAC(kek, cek, nil)
	return c.assembleWrap(enc, mac), nil
}

// unwrapKeySC reverses wrapKeySC.
func (c *gostCipher) unwrapKeySC(kek, cek []byte) ([]byte, error) {
	m, n := c.blockSize>>1, c.keySize
	if len(kek) != n {
		unpacked, err := c.unpackKeySC(kek)
		if err != nil {
			return nil, err
		}
		kek = unpacked
	}
	if len(cek) < n+m {
		return nil, errGostInvalidTypedArray
	}
	d := c.decryptBlocks(kek, cek[:n])
	mac := cek[n : n+m]
	if !c.verifyMAC(kek, mac, d, nil) {
		return nil, errors.New("Invalid key MAC") //nolint:staticcheck,revive // CyberChef's verbatim DataError text
	}
	return d, nil
}

// WrapKey wraps cek under kek using the configured key-wrapping algorithm. For
// GOST/CryptoPro the KEK is the raw cipher key (length-validated); SignalCom
// unpacks its KEK to a fixed 32 bytes, so no validation applies there.
func (c *gostCipher) WrapKey(kek, cek []byte) ([]byte, error) {
	switch c.keyWrapping {
	case "CP":
		if err := c.requireValidKey(kek); err != nil {
			return nil, err
		}
		return c.wrapKeyCP(kek, cek)
	case "SC":
		return c.wrapKeySC(kek, cek)
	default:
		if err := c.requireValidKey(kek); err != nil {
			return nil, err
		}
		return c.wrapKeyGOST(kek, cek)
	}
}

// UnwrapKey reverses WrapKey.
func (c *gostCipher) UnwrapKey(kek, data []byte) ([]byte, error) {
	switch c.keyWrapping {
	case "CP":
		if err := c.requireValidKey(kek); err != nil {
			return nil, err
		}
		return c.unwrapKeyCP(kek, data)
	case "SC":
		return c.unwrapKeySC(kek, data)
	default:
		if err := c.requireValidKey(kek); err != nil {
			return nil, err
		}
		return c.unwrapKeyGOST(kek, data)
	}
}

// ---------------------------------------------------------------------------
// Constructor.
// ---------------------------------------------------------------------------

// gostAlgo describes a cipher configuration (a subset of the JS algorithm
// identifier — only the fields the CyberChef operations set).
type gostAlgo struct {
	version     int    // 1989 or 2015
	length      int    // block length in bits (64 or 128)
	mode        string // ES / MAC / KW
	sBoxName    string // paramset name for 1989; "" for 2015
	block       string // ES block mode
	keyMeshing  string // NO / CP
	padding     string // NO / PKCS5 / ZERO / RANDOM / BIT
	keyWrapping string // NO / CP / SC
	macLength   int    // bits (MAC mode)
	iv          []byte // optional
	ukm         []byte // optional (KW mode)
}

// newGostCipher builds a configured cipher, validating the S-box, IV and UKM
// exactly as the JS GostCipher constructor does.
func newGostCipher(a gostAlgo) (*gostCipher, error) {
	c := &gostCipher{
		version:        a.version,
		blockLength:    a.length,
		blockSize:      a.length >> 3,
		keySize:        gostKeySize,
		mode:           a.mode,
		block:          a.block,
		keyMeshingMode: a.keyMeshing,
		padding:        a.padding,
		keyWrapping:    a.keyWrapping,
		macLength:      a.macLength,
	}
	c.shiftBits = c.resolveShiftBits()
	sBox, err := resolveGostSBox(a.sBoxName)
	if err != nil {
		return nil, err
	}
	c.sBox = sBox
	if err := c.setIV(a.iv); err != nil {
		return nil, err
	}
	if err := c.setUKM(a.ukm); err != nil {
		return nil, err
	}
	return c, nil
}

// resolveShiftBits returns the feedback-register width (bits) for the modes that
// use one (CFB/OFB, CTR-2015, and CryptoPro key wrapping); 0 otherwise.
func (c *gostCipher) resolveShiftBits() int {
	streaming := c.block == "CFB" || c.block == "OFB"
	switch {
	case c.mode == "KW" && c.keyWrapping == "CP":
		return c.blockLength
	case c.version == 2015 && (streaming || c.block == "CTR"):
		return c.blockLength
	case c.version == 1989 && streaming:
		return c.blockLength
	default:
		return 0
	}
}

// resolveGostSBox looks up a paramset S-box by name (defaulting to E-Z, which is
// the fixed table used by the 2015 algorithms and when no name is given).
func resolveGostSBox(name string) ([]uint32, error) {
	if name == "" {
		name = "E-Z"
	}
	sBox, ok := gostSBoxes[name]
	if !ok {
		return nil, fmt.Errorf("Unknown sBox name: %s", name) //nolint:staticcheck,revive // CyberChef's verbatim SyntaxError text
	}
	return sBox, nil
}

// setIV validates and stores the IV, defaulting to a zero block when none is
// given. CTR-2015 uses a half-block IV; the 1989 ciphers require exactly one
// block; the other 2015 modes require a whole number of blocks.
func (c *gostCipher) setIV(iv []byte) error {
	if iv == nil {
		c.iv = make([]byte, c.blockSize)
		return nil
	}
	c.iv = cloneBytes(iv)
	isCTR15 := c.version == 2015 && c.block == "CTR"
	switch {
	case len(c.iv) != c.blockSize && c.version == 1989:
		return fmt.Errorf("Length of iv must be %d bits", c.blockLength) //nolint:staticcheck,revive // CyberChef's verbatim SyntaxError text
	case len(c.iv) != c.blockSize>>1 && isCTR15:
		// The JS message here is the string "0" due to an operator-precedence
		// bug; we surface a clearer message instead.
		return fmt.Errorf("Length of iv must be %d bits", c.blockLength>>1) //nolint:staticcheck,revive // CyberChef's verbatim SyntaxError text
	case len(c.iv)%c.blockSize != 0 && !isCTR15:
		return fmt.Errorf("Length of iv must be a multiple of %d bits", c.blockLength) //nolint:staticcheck,revive // CyberChef's verbatim SyntaxError text
	}
	return nil
}

// setUKM validates and stores the user key material (KW mode), which must be
// exactly one block long.
func (c *gostCipher) setUKM(ukm []byte) error {
	if ukm == nil {
		return nil
	}
	c.ukm = cloneBytes(ukm)
	if len(c.ukm)*8 != c.blockLength {
		return fmt.Errorf("Length of ukm must be %d bits", c.blockLength) //nolint:staticcheck,revive // CyberChef's verbatim SyntaxError text
	}
	return nil
}

// gostVersionBlock maps the Algorithm option to a (version, block length in
// bits) pair, matching the switch in every GOST cipher operation.
func gostVersionBlock(algorithm string) (version, blockLength int, err error) {
	switch algorithm {
	case "GOST 28147 (1989)":
		return 1989, 64, nil
	case "GOST R 34.12 (Magma, 2015)":
		return 2015, 64, nil
	case "GOST R 34.12 (Kuznyechik, 2015)":
		return 2015, 128, nil
	default:
		return 0, 0, fmt.Errorf("Unknown algorithm version: %s", algorithm) //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
}

// gostHexEncode formats bytes as lowercase hex, inserting "\r\n" every 32 bytes,
// exactly like CryptoGost's coding.Hex.encode (which the operations use for Hex
// output — so 44-byte wrapped keys keep their line break).
func gostHexEncode(d []byte) string {
	const hexDigits = "0123456789abcdef"
	var b strings.Builder
	for i, by := range d {
		if i > 0 && i%32 == 0 {
			b.WriteString("\r\n")
		}
		b.WriteByte(hexDigits[by>>4])
		b.WriteByte(hexDigits[by&0xf])
	}
	return b.String()
}

// gostOutput renders cipher output as CryptoGost Hex or as raw bytes.
func gostOutput(data []byte, outputType string) *core.Dish {
	if outputType == "Hex" {
		return core.NewDish([]byte(gostHexEncode(data)), core.TypeString)
	}
	return core.NewDish(data, core.TypeString)
}

// gostToggleBytes decodes a toggleString argument (Hex/UTF8/Latin1/Base64).
func gostToggleBytes(arg any) ([]byte, error) {
	ts := arg.(core.ToggleString)
	return convertToByteArray(ts.Value, ts.Option)
}

// gostOptionalIV returns iv only when non-empty, so an empty IV field selects
// the default zero vector (mirroring CyberChef's `if (iv)` guard).
func gostOptionalIV(iv []byte) []byte {
	if len(iv) == 0 {
		return nil
	}
	return iv
}

// GOSTEncrypt encrypts input with a GOST block cipher (Magma or Kuznyechik).
type GOSTEncrypt struct{}

// Meta returns the operation metadata.
func (GOSTEncrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "GOST Encrypt",
		Module:      "Ciphers",
		Description: "The GOST block cipher (Magma), defined in the standard GOST 28147-89 (RFC 5830), is a Soviet and Russian government standard symmetric key block cipher with a block size of 64 bits. The original standard, published in 1989, did not give the cipher any name, but the most recent revision of the standard, GOST R 34.12-2015 (RFC 7801, RFC 8891), specifies that it may be referred to as Magma. The GOST hash function is based on this cipher. The new standard also specifies a new 128-bit block cipher called Kuznyechik.",
		InfoURL:     "https://wikipedia.org/wiki/GOST_(block_cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (GOSTEncrypt) Args() []core.ArgDef {
	return gostCipherArgs([]string{"Raw", "Hex"}, []string{"Hex", "Raw"})
}

// buildGostES parses the nine shared cipher arguments into a configured cipher
// and the decoded key (used by both GOST Encrypt and GOST Decrypt).
func buildGostES(args []any) (*gostCipher, []byte, error) {
	key, err := gostToggleBytes(args[0])
	if err != nil {
		return nil, nil, err
	}
	iv, err := gostToggleBytes(args[1])
	if err != nil {
		return nil, nil, err
	}
	version, blockLength, err := gostVersionBlock(args[4].(string))
	if err != nil {
		return nil, nil, err
	}
	sBoxName := ""
	if version == 1989 {
		sBoxName = args[5].(string)
	}
	cipher, err := newGostCipher(gostAlgo{
		version:    version,
		length:     blockLength,
		mode:       "ES",
		sBoxName:   sBoxName,
		block:      args[6].(string),
		keyMeshing: args[7].(string),
		padding:    args[8].(string),
		iv:         gostOptionalIV(iv),
	})
	if err != nil {
		return nil, nil, err
	}
	return cipher, key, nil
}

// Run performs the encryption. Ported from CyberChef GOSTEncrypt.mjs.
func (GOSTEncrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	cipher, key, err := buildGostES(args)
	if err != nil {
		return nil, err
	}
	out, err := cipher.Encrypt(key, decodeAESInput(in, args[2].(string)))
	if err != nil {
		return nil, err
	}
	return gostOutput(out, args[3].(string)), nil
}

// GOSTDecrypt decrypts GOST ciphertext (Magma or Kuznyechik).
type GOSTDecrypt struct{}

// Meta returns the operation metadata.
func (GOSTDecrypt) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "GOST Decrypt",
		Module:      "Ciphers",
		Description: "The GOST block cipher (Magma), defined in the standard GOST 28147-89 (RFC 5830), is a Soviet and Russian government standard symmetric key block cipher with a block size of 64 bits. The original standard, published in 1989, did not give the cipher any name, but the most recent revision of the standard, GOST R 34.12-2015 (RFC 7801, RFC 8891), specifies that it may be referred to as Magma. The GOST hash function is based on this cipher. The new standard also specifies a new 128-bit block cipher called Kuznyechik.",
		InfoURL:     "https://wikipedia.org/wiki/GOST_(block_cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (GOSTDecrypt) Args() []core.ArgDef {
	return gostCipherArgs([]string{"Hex", "Raw"}, []string{"Raw", "Hex"})
}

// Run performs the decryption. Ported from CyberChef GOSTDecrypt.mjs.
func (GOSTDecrypt) Run(in *core.Dish, args []any) (*core.Dish, error) {
	cipher, key, err := buildGostES(args)
	if err != nil {
		return nil, err
	}
	out, err := cipher.Decrypt(key, decodeAESInput(in, args[2].(string)))
	if err != nil {
		return nil, err
	}
	return gostOutput(out, args[3].(string)), nil
}

// gostKeyWrapArgs builds the argument list shared by GOST Key Wrap and Unwrap.
func gostKeyWrapArgs(inputOpts, outputOpts []string) []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: gostToggleValues},
		{Name: "User Key Material", Type: core.ArgToggleString, Value: "", ToggleValues: gostToggleValues},
		{Name: "Input type", Type: core.ArgOption, Value: inputOpts},
		{Name: "Output type", Type: core.ArgOption, Value: outputOpts},
		{Name: "Algorithm", Type: core.ArgOption, Value: gostAlgorithms},
		{Name: "sBox", Type: core.ArgOption, Value: gostSBoxNames},
		{Name: "Key wrapping", Type: core.ArgOption, Value: []string{"NO", "CP", "SC"}},
	}
}

// buildGostKW parses the seven shared key-wrapping arguments into a configured
// KW cipher and the decoded KEK. The UKM is always passed (even when empty) so
// its length is validated exactly as CyberChef does.
func buildGostKW(args []any) (*gostCipher, []byte, error) {
	kek, err := gostToggleBytes(args[0])
	if err != nil {
		return nil, nil, err
	}
	ukm, err := gostToggleBytes(args[1])
	if err != nil {
		return nil, nil, err
	}
	if ukm == nil {
		ukm = []byte{}
	}
	version, blockLength, err := gostVersionBlock(args[4].(string))
	if err != nil {
		return nil, nil, err
	}
	sBoxName := ""
	if version == 1989 {
		sBoxName = args[5].(string)
	}
	cipher, err := newGostCipher(gostAlgo{
		version:     version,
		length:      blockLength,
		mode:        "KW",
		sBoxName:    sBoxName,
		keyWrapping: args[6].(string),
		ukm:         ukm,
	})
	if err != nil {
		return nil, nil, err
	}
	return cipher, kek, nil
}

// mapGostKWError translates the internal typed-array error to CyberChef's
// friendly key-wrapping length message.
func mapGostKWError(err error) error {
	if strings.Contains(err.Error(), "Invalid typed array length") {
		return errors.New("Incorrect input length. Must be a multiple of the block size.") //nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	}
	return err
}

// GOSTKeyWrap wraps a content-encryption key under a KEK using a GOST cipher.
type GOSTKeyWrap struct{}

// Meta returns the operation metadata.
func (GOSTKeyWrap) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "GOST Key Wrap",
		Module:      "Ciphers",
		Description: "A key wrapping algorithm for protecting keys in untrusted storage using one of the GOST block cipers.",
		InfoURL:     "https://wikipedia.org/wiki/GOST_(block_cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (GOSTKeyWrap) Args() []core.ArgDef {
	return gostKeyWrapArgs([]string{"Raw", "Hex"}, []string{"Hex", "Raw"})
}

// Run performs the key wrapping. Ported from CyberChef GOSTKeyWrap.mjs.
func (GOSTKeyWrap) Run(in *core.Dish, args []any) (*core.Dish, error) {
	cipher, kek, err := buildGostKW(args)
	if err != nil {
		return nil, err
	}
	out, err := cipher.WrapKey(kek, decodeAESInput(in, args[2].(string)))
	if err != nil {
		return nil, mapGostKWError(err)
	}
	return gostOutput(out, args[3].(string)), nil
}

// GOSTKeyUnwrap decrypts a key wrapped with a GOST cipher.
type GOSTKeyUnwrap struct{}

// Meta returns the operation metadata.
func (GOSTKeyUnwrap) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "GOST Key Unwrap",
		Module:      "Ciphers",
		Description: "A decryptor for keys wrapped using one of the GOST block ciphers.",
		InfoURL:     "https://wikipedia.org/wiki/GOST_(block_cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (GOSTKeyUnwrap) Args() []core.ArgDef {
	return gostKeyWrapArgs([]string{"Hex", "Raw"}, []string{"Raw", "Hex"})
}

// Run performs the key unwrapping. Ported from CyberChef GOSTKeyUnwrap.mjs.
func (GOSTKeyUnwrap) Run(in *core.Dish, args []any) (*core.Dish, error) {
	cipher, kek, err := buildGostKW(args)
	if err != nil {
		return nil, err
	}
	out, err := cipher.UnwrapKey(kek, decodeAESInput(in, args[2].(string)))
	if err != nil {
		return nil, mapGostKWError(err)
	}
	return gostOutput(out, args[3].(string)), nil
}

// gostMACCipher builds a MAC-mode cipher from the shared key/iv/algorithm/sBox
// arguments plus a MAC length in bits, returning the cipher and decoded key.
func gostMACCipher(keyArg, ivArg any, algorithm, sBox string, macLength int) (*gostCipher, []byte, error) {
	key, err := convertToByteArray(keyArg.(core.ToggleString).Value, keyArg.(core.ToggleString).Option)
	if err != nil {
		return nil, nil, err
	}
	iv, err := convertToByteArray(ivArg.(core.ToggleString).Value, ivArg.(core.ToggleString).Option)
	if err != nil {
		return nil, nil, err
	}
	version, blockLength, err := gostVersionBlock(algorithm)
	if err != nil {
		return nil, nil, err
	}
	sBoxName := ""
	if version == 1989 {
		sBoxName = sBox
	}
	cipher, err := newGostCipher(gostAlgo{
		version:   version,
		length:    blockLength,
		mode:      "MAC",
		sBoxName:  sBoxName,
		macLength: macLength,
		iv:        gostOptionalIV(iv),
	})
	if err != nil {
		return nil, nil, err
	}
	return cipher, key, nil
}

// gostMACMin / gostMACMax bound the GOST Sign "MAC length" argument (bits).
var gostMACMin, gostMACMax = 8.0, 64.0

// gostSignArgs builds the GOST Sign argument list.
func gostSignArgs() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: gostToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: gostToggleValues},
		{Name: "Input type", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
		{Name: "Output type", Type: core.ArgOption, Value: []string{"Hex", "Raw"}},
		{Name: "Algorithm", Type: core.ArgOption, Value: gostAlgorithms},
		{Name: "sBox", Type: core.ArgOption, Value: gostSBoxNames},
		{Name: "MAC length", Type: core.ArgNumber, Integer: true, Value: 32, Min: &gostMACMin, Max: &gostMACMax},
	}
}

// GOSTSign signs a message with a GOST block-cipher MAC (imitovstavka).
type GOSTSign struct{}

// Meta returns the operation metadata.
func (GOSTSign) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "GOST Sign",
		Module:      "Ciphers",
		Description: "Sign a plaintext message using one of the GOST block ciphers.",
		InfoURL:     "https://wikipedia.org/wiki/GOST_(block_cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (GOSTSign) Args() []core.ArgDef { return gostSignArgs() }

// Run computes the MAC. Ported from CyberChef GOSTSign.mjs.
func (GOSTSign) Run(in *core.Dish, args []any) (*core.Dish, error) {
	cipher, key, err := gostMACCipher(args[0], args[1], args[4].(string), args[5].(string), int(args[6].(float64)))
	if err != nil {
		return nil, err
	}
	out, err := cipher.Sign(key, decodeAESInput(in, args[2].(string)))
	if err != nil {
		return nil, err
	}
	return gostOutput(out, args[3].(string)), nil
}

// GOSTVerify verifies a GOST block-cipher MAC against a message.
type GOSTVerify struct{}

// Meta returns the operation metadata.
func (GOSTVerify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "GOST Verify",
		Module:      "Ciphers",
		Description: "Verify the signature of a plaintext message using one of the GOST block ciphers. Enter the signature in the MAC field.",
		InfoURL:     "https://wikipedia.org/wiki/GOST_(block_cipher)",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (GOSTVerify) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Key", Type: core.ArgToggleString, Value: "", ToggleValues: gostToggleValues},
		{Name: "IV", Type: core.ArgToggleString, Value: "", ToggleValues: gostToggleValues},
		{Name: "MAC", Type: core.ArgToggleString, Value: "", ToggleValues: gostToggleValues},
		{Name: "Input type", Type: core.ArgOption, Value: []string{"Raw", "Hex"}},
		{Name: "Algorithm", Type: core.ArgOption, Value: gostAlgorithms},
		{Name: "sBox", Type: core.ArgOption, Value: gostSBoxNames},
	}
}

// Run verifies the MAC. Ported from CyberChef GOSTVerify.mjs.
func (GOSTVerify) Run(in *core.Dish, args []any) (*core.Dish, error) {
	mac, err := gostToggleBytes(args[2])
	if err != nil {
		return nil, err
	}
	cipher, key, err := gostMACCipher(args[0], args[1], args[4].(string), args[5].(string), len(mac)*8)
	if err != nil {
		return nil, err
	}
	ok, err := cipher.Verify(key, mac, decodeAESInput(in, args[3].(string)))
	if err != nil {
		return nil, err
	}
	msg := "The signature does not match"
	if ok {
		msg = "The signature matches"
	}
	return core.NewDish([]byte(msg), core.TypeString), nil
}
