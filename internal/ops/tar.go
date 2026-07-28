package ops

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/roberson-io/cchef/internal/core"
)

// Tar packs one file into an archive, and Untar reads an archive back.
//
// An archive is a run of 512-byte blocks: a header of fixed-width fields, the
// file's bytes padded out to whole blocks, and two blocks of nothing to close
// it. Reading is left to the standard library, which also copes with the
// extensions the tar command reaches for when a name will not fit; writing is
// done by hand, because the fields have to be laid out the way CyberChef lays
// them out rather than the way Go would.

// Where the rest of a tar header's fields sit. The block size, the size field
// and the format identifier are named alongside the archive carver already.
const (
	tarNameWidth      = 100
	tarModeOffset     = 100
	tarUIDOffset      = 108
	tarGIDOffset      = 116
	tarMTimeOffset    = 136
	tarMTimeWidth     = 12
	tarChecksumOffset = 148
	tarChecksumWidth  = 8
	tarChecksumDigits = 7
	tarTypeOffset     = 156
	// tarEndBlocks is how many blocks of nothing close an archive.
	tarEndBlocks = 2
)

// What the fixed fields hold. Nothing here comes from a real filesystem: the
// input is a stream of bytes, not a file that was ever on disk.
const (
	tarFileMode    = "0000664"
	tarOwner       = "0"
	tarRegularFile = '0'
	// tarUSTAR is the format identifier and the version that follows it.
	tarUSTAR = "ustar\x0000"
	// tarOctalBits is how much a single octal digit carries.
	tarOctalBits = 3
)

// Tar packs the input into a tarball under one name.
type Tar struct{}

// Meta returns the operation metadata.
func (Tar) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Tar",
		Module:      "Compression",
		Description: "Packs the input into a tarball.\n\nNo support for multiple files at this time.",
		InfoURL:     "https://wikipedia.org/wiki/Tar_(computing)",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeByteArray,
	}
}

// Args returns the argument definitions.
func (Tar) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Filename", Type: core.ArgString, Value: "file.txt"},
	}
}

// Run packs the input.
func (Tar) Run(in *core.Dish, args []any) (*core.Dish, error) {
	name, _ := args[0].(string)
	out, err := tarPack(in.Bytes(), name, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	return core.NewDish(out, core.TypeByteArray), nil
}

// tarPack writes an archive holding one file.
func tarPack(data []byte, name string, modTime int64) ([]byte, error) {
	header, err := tarHeader(name, len(data), modTime)
	if err != nil {
		return nil, err
	}
	// The data occupies whole blocks, however little of the last one it fills.
	padded := tarWholeBlocks(len(data))
	tail := padded - len(data) + tarEndBlocks*tarBlockSize

	out := make([]byte, 0, tarBlockSize+padded+tarEndBlocks*tarBlockSize)
	out = append(out, header...)
	out = append(out, data...)
	return append(out, make([]byte, tail)...), nil
}

// tarWholeBlocks rounds a length up to the blocks it takes.
func tarWholeBlocks(n int) int {
	return (n + tarBlockSize - 1) / tarBlockSize * tarBlockSize
}

// tarHeader builds the 512-byte header that opens a file's blocks.
func tarHeader(name string, size int, modTime int64) ([]byte, error) {
	if len(name) > tarNameWidth {
		return nil, fmt.Errorf("the filename takes %d bytes, more than the %d a tar header keeps for it",
			len(name), tarNameWidth)
	}
	if !tarFits(int64(size), tarSizeWidth) {
		return nil, fmt.Errorf("the input is %d bytes, more than a tar header can record", size)
	}
	if !tarFits(modTime, tarSizeWidth) {
		return nil, errors.New("the time of writing does not fit a tar header")
	}

	h := make([]byte, tarBlockSize)
	copy(h, name)
	copy(h[tarModeOffset:], tarFileMode)
	copy(h[tarUIDOffset:], tarOwner)
	copy(h[tarGIDOffset:], tarOwner)
	copy(h[tarSizeOffset:], tarOctalField(int64(size), tarSizeWidth))
	copy(h[tarMTimeOffset:], tarOctalField(modTime, tarSizeWidth))
	h[tarTypeOffset] = tarRegularFile
	copy(h[tarMagicOffset:], tarUSTAR)

	// A reader works the checksum out over the whole header with this field
	// held as spaces, so it is filled with spaces before the sum is taken. What
	// goes back is seven digits and then a null, which is the form every reader
	// accepts and the one CyberChef writes.
	for i := tarChecksumOffset; i < tarChecksumOffset+tarChecksumWidth; i++ {
		h[i] = ' '
	}
	copy(h[tarChecksumOffset:], tarOctalField(tarChecksum(h), tarChecksumDigits))
	h[tarChecksumOffset+tarChecksumDigits] = 0
	return h, nil
}

// tarChecksum sums every byte of a header, which is how a reader tells a header
// from anything else that happens to be there.
func tarChecksum(header []byte) int64 {
	sum := int64(0)
	for _, b := range header {
		sum += int64(b)
	}
	return sum
}

// tarOctalField renders a number as the zero-padded octal a header field holds.
func tarOctalField(v int64, digits int) string {
	return fmt.Sprintf("%0*o", digits, v)
}

// tarFits reports whether a number can be written in the octal digits its field
// gives it.
func tarFits(v int64, digits int) bool {
	return v >= 0 && v < 1<<(tarOctalBits*digits)
}

// Untar unpacks a tarball into the files it holds.
type Untar struct{}

// Meta returns the operation metadata.
func (Untar) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Untar",
		Module:      "Compression",
		Description: "Unpacks a tarball and displays it per file.",
		InfoURL:     "https://wikipedia.org/wiki/Tar_(computing)",
		InputType:   core.TypeByteArray,
		OutputType:  core.TypeFileList,
	}
}

// Args returns the argument definitions.
func (Untar) Args() []core.ArgDef { return nil }

// Run unpacks the archive.
func (Untar) Run(in *core.Dish, _ []any) (*core.Dish, error) {
	files, err := tarUnpack(in.Bytes())
	if err != nil {
		return nil, err
	}
	return core.NewFileListDish(files), nil
}

// tarUnpack takes every regular file out of an archive, in the order it holds
// them. Directories and links have no contents of their own and are passed over.
func tarUnpack(data []byte) ([]core.NamedFile, error) {
	r := tar.NewReader(bytes.NewReader(data))
	var files []core.NamedFile
	for {
		header, err := r.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		// CyberChef leaves its own archives ending mid-block, so a reader that
		// insists on whole ones runs out partway through the closing zeroes.
		// Everything read before that point is still sound.
		if errors.Is(err, io.ErrUnexpectedEOF) && len(files) > 0 {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		body, err := io.ReadAll(r)
		if err != nil {
			return nil, err
		}
		files = append(files, core.NamedFile{Name: header.Name, Data: body})
	}
	return files, nil
}

func init() {
	core.Register(Tar{})
	core.Register(Untar{})
}
