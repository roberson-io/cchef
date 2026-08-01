package ops

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"crypto/rand"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"time"

	"github.com/roberson-io/cchef/core"
)

// The ZIP container, and the encryption an archive can carry.

// zipCompressionMethods are the ways an entry's data can be stored.
var zipCompressionMethods = []string{"Deflate", "None (Store)"}

// zipOperatingSystems name the system an archive says it was written on, which
// goes into the central directory and nothing depends on.
var zipOperatingSystems = []string{"MSDOS", "Unix", "Macintosh"}

// zipOSCodes are the numbers those names stand for.
var zipOSCodes = map[string]byte{"MSDOS": 0, "Unix": 3, "Macintosh": 7}

// The two ways an entry's data can be stored, as the format numbers them.
const (
	zipMethodStore   = 0
	zipMethodDeflate = 8
)

// zipEncryptedFlag marks an entry whose data is encrypted.
const zipEncryptedFlag = 1

// zipVersionNeeded is the version of the format a reader must understand, which
// for a deflated entry is 2.0.
const zipVersionNeeded = 20

// zipCryptHeaderSize is how many bytes of random data open an encrypted entry.
// They prime the cipher, and their last byte lets a reader tell a wrong
// password from a right one before decrypting anything else.
const zipCryptHeaderSize = 12

// zipNow is the clock the archive timestamps come from, replaced in tests.
var zipNow = time.Now

// Zip packs the input into a ZIP archive holding one file.
type Zip struct{}

// Meta returns the operation metadata.
func (Zip) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Zip",
		Module: "Compression",
		Description: "Compresses data using the PKZIP algorithm with the given " +
			"filename.<br><br>No support for multiple files at this time.",
		InfoURL:    "https://wikipedia.org/wiki/Zip_(file_format)",
		InputType:  core.TypeByteArray,
		OutputType: core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (Zip) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Filename", Type: core.ArgString, Value: "file.txt"},
		{Name: "Comment", Type: core.ArgString, Value: ""},
		{Name: "Password", Type: core.ArgString, Value: ""},
		{Name: "Compression method", Type: core.ArgOption, Value: zipCompressionMethods},
		{Name: "Operating system", Type: core.ArgOption, Value: zipOperatingSystems},
		{Name: "Compression type", Type: core.ArgOption, Value: deflateCompressionTypes},
	}
}

// Run builds the archive.
func (Zip) Run(in *core.Dish, args []any) (*core.Dish, error) {
	filename, _ := args[0].(string)
	comment, _ := args[1].(string)
	password, _ := args[2].(string)
	method, _ := args[3].(string)
	osName, _ := args[4].(string)
	compressionType, _ := args[5].(string)

	out, err := zipArchive(in.Bytes(), filename, comment, password, method, osName, compressionType)
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// zipArchive builds a one-file archive: the entry's local header and data, then
// a central directory describing it, then the record closing the archive.
func zipArchive(data []byte, filename, comment, password, method, osName, compressionType string) ([]byte, error) {
	body, methodCode, err := zipEntryBody(data, method, compressionType)
	if err != nil {
		return nil, err
	}

	checksum := crc32.ChecksumIEEE(data)
	var flags uint16
	if password != "" {
		body = zipEncrypt(body, password, checksum)
		flags |= zipEncryptedFlag
	}

	modTime, modDate := zipDOSTime(zipNow())
	name := []byte(filename)

	var out []byte
	out = append(out, 'P', 'K', 3, 4)
	out = append(out, zipVersionNeeded, 0)
	out = zipAppend16(out, flags)
	out = zipAppend16(out, uint16(methodCode)) // #nosec G115 -- the method is 0 or 8
	out = zipAppend16(out, modTime)
	out = zipAppend16(out, modDate)
	out = zipAppend32(out, checksum)
	out = zipAppend32(out, uint32(len(body))) // #nosec G115 -- an archive this writer builds is far below 4 GB
	out = zipAppend32(out, uint32(len(data))) // #nosec G115 -- likewise
	out = zipAppend16(out, uint16(len(name))) // #nosec G115 -- a filename is far below 64 KB
	out = zipAppend16(out, 0)                 // no extra field
	out = append(out, name...)
	out = append(out, body...)

	// #nosec G115 -- the local part is far below 4 GB
	localSize := uint32(len(out))
	central := zipCentralDirectory(name, []byte(comment), flags, methodCode,
		modTime, modDate, checksum, len(body), len(data), osName)

	out = append(out, central...)
	out = append(out, 'P', 'K', 5, 6)
	out = zipAppend16(out, 0)                    // this disk
	out = zipAppend16(out, 0)                    // the disk the directory starts on
	out = zipAppend16(out, 1)                    // entries on this disk
	out = zipAppend16(out, 1)                    // entries altogether
	out = zipAppend32(out, uint32(len(central))) // #nosec G115 -- one entry's directory is tiny
	out = zipAppend32(out, localSize)
	out = zipAppend16(out, 0) // no archive comment
	return out, nil
}

// zipCentralDirectory writes the one entry's directory record, which repeats
// the local header and adds where to find it.
func zipCentralDirectory(name, comment []byte, flags uint16, methodCode int,
	modTime, modDate uint16, checksum uint32, bodyLen, plainLen int, osName string,
) []byte {
	var out []byte
	out = append(out, 'P', 'K', 1, 2)
	out = append(out, zipVersionNeeded, zipOSCodes[osName])
	out = append(out, zipVersionNeeded, 0)
	out = zipAppend16(out, flags)
	out = zipAppend16(out, uint16(methodCode)) // #nosec G115 -- the method is 0 or 8
	out = zipAppend16(out, modTime)
	out = zipAppend16(out, modDate)
	out = zipAppend32(out, checksum)
	out = zipAppend32(out, uint32(bodyLen))      // #nosec G115 -- far below 4 GB
	out = zipAppend32(out, uint32(plainLen))     // #nosec G115 -- likewise
	out = zipAppend16(out, uint16(len(name)))    // #nosec G115 -- far below 64 KB
	out = zipAppend16(out, 0)                    // no extra field
	out = zipAppend16(out, uint16(len(comment))) // #nosec G115 -- far below 64 KB
	out = zipAppend16(out, 0)                    // the disk it starts on
	out = zipAppend16(out, 0)                    // internal attributes
	out = zipAppend32(out, 0)                    // external attributes
	out = zipAppend32(out, 0)                    // it is the first entry, so at zero
	out = append(out, name...)
	return append(out, comment...)
}

// zipEntryBody stores or compresses the entry's data, and reports which of the
// two the header should say was done.
func zipEntryBody(data []byte, method, compressionType string) ([]byte, int, error) {
	switch method {
	case "None (Store)":
		return data, zipMethodStore, nil
	case "Deflate":
		body, err := deflateEncode(data, compressionType)
		if err != nil {
			return nil, 0, err
		}
		return body, zipMethodDeflate, nil
	}
	return nil, 0, fmt.Errorf("unknown compression method: %s", method)
}

// zipAppend16 and zipAppend32 write a number least significant byte first,
// which is how every field in the format is laid out.
func zipAppend16(out []byte, v uint16) []byte {
	// #nosec G115 -- the number is written a byte at a time, least significant first
	return append(out, byte(v), byte(v>>8))
}

func zipAppend32(out []byte, v uint32) []byte {
	// #nosec G115 -- the number is written a byte at a time, least significant first
	return append(out, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

// zipDOSTime renders a time the way the format records it: two bytes of time to
// a resolution of two seconds, and two bytes of date counting years from 1980.
func zipDOSTime(t time.Time) (modTime, modDate uint16) {
	// #nosec G115 -- every field is masked into its own few bits
	modTime = uint16(t.Hour()<<11 | t.Minute()<<5 | t.Second()/2)
	// #nosec G115 -- likewise
	modDate = uint16((t.Year()-1980)&0x7f<<9 | int(t.Month())<<5 | t.Day())
	return modTime, modDate
}

// Unzip reads the files out of a ZIP archive.
type Unzip struct{}

// Meta returns the operation metadata.
func (Unzip) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Unzip",
		Module: "Compression",
		Description: "Decompresses data using the PKZIP algorithm and displays it " +
			"per file, with support for passwords.",
		InfoURL:    "https://wikipedia.org/wiki/Zip_(file_format)",
		InputType:  core.TypeByteArray,
		OutputType: core.TypeFileList,
	}
}

// Args returns the argument definitions.
func (Unzip) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Password", Type: core.ArgString, Value: ""},
		{Name: "Verify result", Type: core.ArgBoolean, Value: false},
	}
}

// Run reads the archive.
func (Unzip) Run(in *core.Dish, args []any) (*core.Dish, error) {
	password, _ := args[0].(string)
	verify, _ := args[1].(bool)

	files, err := zipRead(in.Bytes(), password, verify)
	if err != nil {
		return nil, err
	}
	return core.NewFileListDish(files), nil
}

// zipRead pulls every file out of an archive, in the order the directory lists
// them.
func zipRead(data []byte, password string, verify bool) ([]core.NamedFile, error) {
	if len(data) == 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim error text
		return nil, errors.New("Please provide an input.")
	}
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}

	files := make([]core.NamedFile, 0, len(r.File))
	for _, entry := range r.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		content, err := zipEntryContents(entry, password, verify)
		if err != nil {
			return nil, err
		}
		files = append(files, core.NamedFile{Name: entry.Name, Data: content})
	}
	return files, nil
}

// zipEntryContents reads one entry, decrypting it first when it is encrypted.
// The raw bytes are taken rather than letting the reader unpack them, so that
// the checksum is only looked at when it was asked for.
func zipEntryContents(entry *zip.File, password string, verify bool) ([]byte, error) {
	body, err := zipRawBytes(entry)
	if err != nil {
		return nil, err
	}

	encrypted := entry.Flags&zipEncryptedFlag != 0
	if encrypted {
		if password == "" {
			return nil, fmt.Errorf("%s is encrypted: a password is required", entry.Name)
		}
		if body, err = zipDecrypt(body, password, entry.Name); err != nil {
			return nil, err
		}
	}

	content, err := zipInflate(body, entry.Method, entry.Name)
	if encrypted && (err != nil || crc32.ChecksumIEEE(content) != entry.CRC32) {
		// The checksum is what tells a wrong password from a right one. The byte
		// the format sets aside for that would be quicker, but not every writer
		// fills it in correctly, and the checksum is both stronger and always
		// there.
		return nil, fmt.Errorf("wrong password for %s", entry.Name)
	}
	if err != nil {
		return nil, err
	}
	if verify && crc32.ChecksumIEEE(content) != entry.CRC32 {
		return nil, fmt.Errorf("%s does not match its checksum", entry.Name)
	}
	return content, nil
}

// zipRawBytes returns an entry's data as it is stored, without unpacking it,
// so that the checksum is only looked at when it was asked for.
func zipRawBytes(entry *zip.File) ([]byte, error) {
	rc, err := entry.OpenRaw()
	if err != nil {
		return nil, err
	}
	return io.ReadAll(rc)
}

// zipInflate turns an entry's stored bytes back into its contents.
func zipInflate(body []byte, method uint16, name string) ([]byte, error) {
	switch method {
	case zipMethodStore:
		return body, nil
	case zipMethodDeflate:
		out, err := io.ReadAll(flate.NewReader(bytes.NewReader(body)))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		return out, nil
	}
	return nil, fmt.Errorf("%s uses compression method %d, which is not supported", name, method)
}

// zipEncrypt puts the twelve-byte header in front of the data and runs the lot
// through the cipher.
func zipEncrypt(data []byte, password string, checksum uint32) []byte {
	header := make([]byte, zipCryptHeaderSize)
	// crypto/rand.Read fills the whole slice and never fails.
	_, _ = rand.Read(header[:zipCryptHeaderSize-1])
	// The last byte is what a reader checks a password against: the top byte of
	// the entry's checksum.
	header[zipCryptHeaderSize-1] = byte(checksum >> 24)

	keys := newZipCrypto(password)
	out := make([]byte, 0, len(header)+len(data))
	for _, b := range header {
		out = append(out, keys.encrypt(b))
	}
	for _, b := range data {
		out = append(out, keys.encrypt(b))
	}
	return out
}

// zipDecrypt takes the twelve-byte header off the front and decrypts what
// follows. The header is run through the cipher rather than examined: its last
// byte is meant to identify a wrong password, but writers exist that fill it
// with anything, so the caller checks the contents against their checksum
// instead.
func zipDecrypt(data []byte, password, name string) ([]byte, error) {
	if len(data) < zipCryptHeaderSize {
		return nil, fmt.Errorf("%s is too short to hold an encryption header", name)
	}

	keys := newZipCrypto(password)
	for i := range zipCryptHeaderSize {
		keys.decrypt(data[i])
	}

	out := make([]byte, 0, len(data)-zipCryptHeaderSize)
	for _, b := range data[zipCryptHeaderSize:] {
		out = append(out, keys.decrypt(b))
	}
	return out, nil
}

// zipCrypto is the stream cipher PKZIP has carried since 2.0. Three keys are
// stirred by every byte of plaintext that passes through, so encrypting and
// decrypting differ only in which of the two bytes is fed back.
type zipCrypto struct{ key0, key1, key2 uint32 }

// The values the three keys start from, before the password is mixed in.
const (
	zipCryptKey0 = 305419896
	zipCryptKey1 = 591751049
	zipCryptKey2 = 878082192
)

// newZipCrypto starts the cipher from a password.
func newZipCrypto(password string) *zipCrypto {
	z := &zipCrypto{key0: zipCryptKey0, key1: zipCryptKey1, key2: zipCryptKey2}
	for _, b := range []byte(password) {
		z.update(b)
	}
	return z
}

// update stirs one byte of plaintext into the keys.
func (z *zipCrypto) update(b byte) {
	// #nosec G115 -- the low byte of the key is what the table is indexed by
	z.key0 = crc32.IEEETable[byte(z.key0)^b] ^ z.key0>>8
	z.key1 = (z.key1+z.key0&0xff)*134775813 + 1
	// #nosec G115 -- likewise, and the key's top byte is what feeds it
	z.key2 = crc32.IEEETable[byte(z.key2)^byte(z.key1>>24)] ^ z.key2>>8
}

// next returns the byte the keys currently stand for, which is what the data is
// exclusive-ored with.
func (z *zipCrypto) next() byte {
	temp := z.key2 | 2
	// #nosec G115 -- the cipher takes one byte from the middle of the product
	return byte(temp * (temp ^ 1) >> 8)
}

func (z *zipCrypto) encrypt(plain byte) byte {
	cipher := plain ^ z.next()
	z.update(plain)
	return cipher
}

func (z *zipCrypto) decrypt(cipher byte) byte {
	plain := cipher ^ z.next()
	z.update(plain)
	return plain
}

func init() {
	core.Register(Zip{})
	core.Register(Unzip{})
}
