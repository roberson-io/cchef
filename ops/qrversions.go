package ops

// The tables the QR reader identifies a code by, generated from jsQR's own.

// qrECBlock is a group of blocks holding the same number of data codewords, and
// qrECLevel the whole layout of one error correction level.
type (
	qrECBlock struct{ count, dataCodewords int }
	qrECLevel struct {
		ecCodewordsPerBlock int
		blocks              []qrECBlock
	}
)

// qrVersionInfo describes one version: the bits identifying it, where its
// alignment patterns sit, and how each correction level divides its codewords.
type qrReaderVersion struct {
	infoBits  int
	number    int
	alignment []int
	levels    [4]qrECLevel
}

// qrReaderVersions is the version table, generated from jsQR's own.
var qrReaderVersions = [40]qrReaderVersion{
	{0, 1, []int{}, [4]qrECLevel{{7, []qrECBlock{{1, 19}}}, {10, []qrECBlock{{1, 16}}}, {13, []qrECBlock{{1, 13}}}, {17, []qrECBlock{{1, 9}}}}},
	{0, 2, []int{6, 18}, [4]qrECLevel{{10, []qrECBlock{{1, 34}}}, {16, []qrECBlock{{1, 28}}}, {22, []qrECBlock{{1, 22}}}, {28, []qrECBlock{{1, 16}}}}},
	{0, 3, []int{6, 22}, [4]qrECLevel{{15, []qrECBlock{{1, 55}}}, {26, []qrECBlock{{1, 44}}}, {18, []qrECBlock{{2, 17}}}, {22, []qrECBlock{{2, 13}}}}},
	{0, 4, []int{6, 26}, [4]qrECLevel{{20, []qrECBlock{{1, 80}}}, {18, []qrECBlock{{2, 32}}}, {26, []qrECBlock{{2, 24}}}, {16, []qrECBlock{{4, 9}}}}},
	{0, 5, []int{6, 30}, [4]qrECLevel{{26, []qrECBlock{{1, 108}}}, {24, []qrECBlock{{2, 43}}}, {18, []qrECBlock{{2, 15}, {2, 16}}}, {22, []qrECBlock{{2, 11}, {2, 12}}}}},
	{0, 6, []int{6, 34}, [4]qrECLevel{{18, []qrECBlock{{2, 68}}}, {16, []qrECBlock{{4, 27}}}, {24, []qrECBlock{{4, 19}}}, {28, []qrECBlock{{4, 15}}}}},
	{0x7C94, 7, []int{6, 22, 38}, [4]qrECLevel{{20, []qrECBlock{{2, 78}}}, {18, []qrECBlock{{4, 31}}}, {18, []qrECBlock{{2, 14}, {4, 15}}}, {26, []qrECBlock{{4, 13}, {1, 14}}}}},
	{0x85BC, 8, []int{6, 24, 42}, [4]qrECLevel{{24, []qrECBlock{{2, 97}}}, {22, []qrECBlock{{2, 38}, {2, 39}}}, {22, []qrECBlock{{4, 18}, {2, 19}}}, {26, []qrECBlock{{4, 14}, {2, 15}}}}},
	{0x9A99, 9, []int{6, 26, 46}, [4]qrECLevel{{30, []qrECBlock{{2, 116}}}, {22, []qrECBlock{{3, 36}, {2, 37}}}, {20, []qrECBlock{{4, 16}, {4, 17}}}, {24, []qrECBlock{{4, 12}, {4, 13}}}}},
	{0xA4D3, 10, []int{6, 28, 50}, [4]qrECLevel{{18, []qrECBlock{{2, 68}, {2, 69}}}, {26, []qrECBlock{{4, 43}, {1, 44}}}, {24, []qrECBlock{{6, 19}, {2, 20}}}, {28, []qrECBlock{{6, 15}, {2, 16}}}}},
	{0xBBF6, 11, []int{6, 30, 54}, [4]qrECLevel{{20, []qrECBlock{{4, 81}}}, {30, []qrECBlock{{1, 50}, {4, 51}}}, {28, []qrECBlock{{4, 22}, {4, 23}}}, {24, []qrECBlock{{3, 12}, {8, 13}}}}},
	{0xC762, 12, []int{6, 32, 58}, [4]qrECLevel{{24, []qrECBlock{{2, 92}, {2, 93}}}, {22, []qrECBlock{{6, 36}, {2, 37}}}, {26, []qrECBlock{{4, 20}, {6, 21}}}, {28, []qrECBlock{{7, 14}, {4, 15}}}}},
	{0xD847, 13, []int{6, 34, 62}, [4]qrECLevel{{26, []qrECBlock{{4, 107}}}, {22, []qrECBlock{{8, 37}, {1, 38}}}, {24, []qrECBlock{{8, 20}, {4, 21}}}, {22, []qrECBlock{{12, 11}, {4, 12}}}}},
	{0xE60D, 14, []int{6, 26, 46, 66}, [4]qrECLevel{{30, []qrECBlock{{3, 115}, {1, 116}}}, {24, []qrECBlock{{4, 40}, {5, 41}}}, {20, []qrECBlock{{11, 16}, {5, 17}}}, {24, []qrECBlock{{11, 12}, {5, 13}}}}},
	{0xF928, 15, []int{6, 26, 48, 70}, [4]qrECLevel{{22, []qrECBlock{{5, 87}, {1, 88}}}, {24, []qrECBlock{{5, 41}, {5, 42}}}, {30, []qrECBlock{{5, 24}, {7, 25}}}, {24, []qrECBlock{{11, 12}, {7, 13}}}}},
	{0x10B78, 16, []int{6, 26, 50, 74}, [4]qrECLevel{{24, []qrECBlock{{5, 98}, {1, 99}}}, {28, []qrECBlock{{7, 45}, {3, 46}}}, {24, []qrECBlock{{15, 19}, {2, 20}}}, {30, []qrECBlock{{3, 15}, {13, 16}}}}},
	{0x1145D, 17, []int{6, 30, 54, 78}, [4]qrECLevel{{28, []qrECBlock{{1, 107}, {5, 108}}}, {28, []qrECBlock{{10, 46}, {1, 47}}}, {28, []qrECBlock{{1, 22}, {15, 23}}}, {28, []qrECBlock{{2, 14}, {17, 15}}}}},
	{0x12A17, 18, []int{6, 30, 56, 82}, [4]qrECLevel{{30, []qrECBlock{{5, 120}, {1, 121}}}, {26, []qrECBlock{{9, 43}, {4, 44}}}, {28, []qrECBlock{{17, 22}, {1, 23}}}, {28, []qrECBlock{{2, 14}, {19, 15}}}}},
	{0x13532, 19, []int{6, 30, 58, 86}, [4]qrECLevel{{28, []qrECBlock{{3, 113}, {4, 114}}}, {26, []qrECBlock{{3, 44}, {11, 45}}}, {26, []qrECBlock{{17, 21}, {4, 22}}}, {26, []qrECBlock{{9, 13}, {16, 14}}}}},
	{0x149A6, 20, []int{6, 34, 62, 90}, [4]qrECLevel{{28, []qrECBlock{{3, 107}, {5, 108}}}, {26, []qrECBlock{{3, 41}, {13, 42}}}, {30, []qrECBlock{{15, 24}, {5, 25}}}, {28, []qrECBlock{{15, 15}, {10, 16}}}}},
	{0x15683, 21, []int{6, 28, 50, 72, 94}, [4]qrECLevel{{28, []qrECBlock{{4, 116}, {4, 117}}}, {26, []qrECBlock{{17, 42}}}, {28, []qrECBlock{{17, 22}, {6, 23}}}, {30, []qrECBlock{{19, 16}, {6, 17}}}}},
	{0x168C9, 22, []int{6, 26, 50, 74, 98}, [4]qrECLevel{{28, []qrECBlock{{2, 111}, {7, 112}}}, {28, []qrECBlock{{17, 46}}}, {30, []qrECBlock{{7, 24}, {16, 25}}}, {24, []qrECBlock{{34, 13}}}}},
	{0x177EC, 23, []int{6, 30, 54, 74, 102}, [4]qrECLevel{{30, []qrECBlock{{4, 121}, {5, 122}}}, {28, []qrECBlock{{4, 47}, {14, 48}}}, {30, []qrECBlock{{11, 24}, {14, 25}}}, {30, []qrECBlock{{16, 15}, {14, 16}}}}},
	{0x18EC4, 24, []int{6, 28, 54, 80, 106}, [4]qrECLevel{{30, []qrECBlock{{6, 117}, {4, 118}}}, {28, []qrECBlock{{6, 45}, {14, 46}}}, {30, []qrECBlock{{11, 24}, {16, 25}}}, {30, []qrECBlock{{30, 16}, {2, 17}}}}},
	{0x191E1, 25, []int{6, 32, 58, 84, 110}, [4]qrECLevel{{26, []qrECBlock{{8, 106}, {4, 107}}}, {28, []qrECBlock{{8, 47}, {13, 48}}}, {30, []qrECBlock{{7, 24}, {22, 25}}}, {30, []qrECBlock{{22, 15}, {13, 16}}}}},
	{0x1AFAB, 26, []int{6, 30, 58, 86, 114}, [4]qrECLevel{{28, []qrECBlock{{10, 114}, {2, 115}}}, {28, []qrECBlock{{19, 46}, {4, 47}}}, {28, []qrECBlock{{28, 22}, {6, 23}}}, {30, []qrECBlock{{33, 16}, {4, 17}}}}},
	{0x1B08E, 27, []int{6, 34, 62, 90, 118}, [4]qrECLevel{{30, []qrECBlock{{8, 122}, {4, 123}}}, {28, []qrECBlock{{22, 45}, {3, 46}}}, {30, []qrECBlock{{8, 23}, {26, 24}}}, {30, []qrECBlock{{12, 15}, {28, 16}}}}},
	{0x1CC1A, 28, []int{6, 26, 50, 74, 98, 122}, [4]qrECLevel{{30, []qrECBlock{{3, 117}, {10, 118}}}, {28, []qrECBlock{{3, 45}, {23, 46}}}, {30, []qrECBlock{{4, 24}, {31, 25}}}, {30, []qrECBlock{{11, 15}, {31, 16}}}}},
	{0x1D33F, 29, []int{6, 30, 54, 78, 102, 126}, [4]qrECLevel{{30, []qrECBlock{{7, 116}, {7, 117}}}, {28, []qrECBlock{{21, 45}, {7, 46}}}, {30, []qrECBlock{{1, 23}, {37, 24}}}, {30, []qrECBlock{{19, 15}, {26, 16}}}}},
	{0x1ED75, 30, []int{6, 26, 52, 78, 104, 130}, [4]qrECLevel{{30, []qrECBlock{{5, 115}, {10, 116}}}, {28, []qrECBlock{{19, 47}, {10, 48}}}, {30, []qrECBlock{{15, 24}, {25, 25}}}, {30, []qrECBlock{{23, 15}, {25, 16}}}}},
	{0x1F250, 31, []int{6, 30, 56, 82, 108, 134}, [4]qrECLevel{{30, []qrECBlock{{13, 115}, {3, 116}}}, {28, []qrECBlock{{2, 46}, {29, 47}}}, {30, []qrECBlock{{42, 24}, {1, 25}}}, {30, []qrECBlock{{23, 15}, {28, 16}}}}},
	{0x209D5, 32, []int{6, 34, 60, 86, 112, 138}, [4]qrECLevel{{30, []qrECBlock{{17, 115}}}, {28, []qrECBlock{{10, 46}, {23, 47}}}, {30, []qrECBlock{{10, 24}, {35, 25}}}, {30, []qrECBlock{{19, 15}, {35, 16}}}}},
	{0x216F0, 33, []int{6, 30, 58, 86, 114, 142}, [4]qrECLevel{{30, []qrECBlock{{17, 115}, {1, 116}}}, {28, []qrECBlock{{14, 46}, {21, 47}}}, {30, []qrECBlock{{29, 24}, {19, 25}}}, {30, []qrECBlock{{11, 15}, {46, 16}}}}},
	{0x228BA, 34, []int{6, 34, 62, 90, 118, 146}, [4]qrECLevel{{30, []qrECBlock{{13, 115}, {6, 116}}}, {28, []qrECBlock{{14, 46}, {23, 47}}}, {30, []qrECBlock{{44, 24}, {7, 25}}}, {30, []qrECBlock{{59, 16}, {1, 17}}}}},
	{0x2379F, 35, []int{6, 30, 54, 78, 102, 126, 150}, [4]qrECLevel{{30, []qrECBlock{{12, 121}, {7, 122}}}, {28, []qrECBlock{{12, 47}, {26, 48}}}, {30, []qrECBlock{{39, 24}, {14, 25}}}, {30, []qrECBlock{{22, 15}, {41, 16}}}}},
	{0x24B0B, 36, []int{6, 24, 50, 76, 102, 128, 154}, [4]qrECLevel{{30, []qrECBlock{{6, 121}, {14, 122}}}, {28, []qrECBlock{{6, 47}, {34, 48}}}, {30, []qrECBlock{{46, 24}, {10, 25}}}, {30, []qrECBlock{{2, 15}, {64, 16}}}}},
	{0x2542E, 37, []int{6, 28, 54, 80, 106, 132, 158}, [4]qrECLevel{{30, []qrECBlock{{17, 122}, {4, 123}}}, {28, []qrECBlock{{29, 46}, {14, 47}}}, {30, []qrECBlock{{49, 24}, {10, 25}}}, {30, []qrECBlock{{24, 15}, {46, 16}}}}},
	{0x26A64, 38, []int{6, 32, 58, 84, 110, 136, 162}, [4]qrECLevel{{30, []qrECBlock{{4, 122}, {18, 123}}}, {28, []qrECBlock{{13, 46}, {32, 47}}}, {30, []qrECBlock{{48, 24}, {14, 25}}}, {30, []qrECBlock{{42, 15}, {32, 16}}}}},
	{0x27541, 39, []int{6, 26, 54, 82, 110, 138, 166}, [4]qrECLevel{{30, []qrECBlock{{20, 117}, {4, 118}}}, {28, []qrECBlock{{40, 47}, {7, 48}}}, {30, []qrECBlock{{43, 24}, {22, 25}}}, {30, []qrECBlock{{10, 15}, {67, 16}}}}},
	{0x28C69, 40, []int{6, 30, 58, 86, 114, 142, 170}, [4]qrECLevel{{30, []qrECBlock{{19, 118}, {6, 119}}}, {28, []qrECBlock{{18, 47}, {31, 48}}}, {30, []qrECBlock{{34, 24}, {34, 25}}}, {30, []qrECBlock{{20, 15}, {61, 16}}}}},
}

// qrFormatInfo is what the format information carries: the error correction
// level, and which of the eight masks was applied.
type qrFormatInfo struct{ level, mask int }

// qrFormatTable maps every valid format field to what it means. The fields are
// spread far enough apart that a few wrong bits still identify one.
var qrFormatTable = map[int]qrFormatInfo{
	0x5412: {1, 0}, 0x5125: {1, 1}, 0x5E7C: {1, 2}, 0x5B4B: {1, 3},
	0x45F9: {1, 4}, 0x40CE: {1, 5}, 0x4F97: {1, 6}, 0x4AA0: {1, 7},
	0x77C4: {0, 0}, 0x72F3: {0, 1}, 0x7DAA: {0, 2}, 0x789D: {0, 3},
	0x662F: {0, 4}, 0x6318: {0, 5}, 0x6C41: {0, 6}, 0x6976: {0, 7},
	0x1689: {3, 0}, 0x13BE: {3, 1}, 0x1CE7: {3, 2}, 0x19D0: {3, 3},
	0x0762: {3, 4}, 0x0255: {3, 5}, 0x0D0C: {3, 6}, 0x083B: {3, 7},
	0x355F: {2, 0}, 0x3068: {2, 1}, 0x3F31: {2, 2}, 0x3A06: {2, 3},
	0x24B4: {2, 4}, 0x2183: {2, 5}, 0x2EDA: {2, 6}, 0x2BED: {2, 7},
}

// qrFormatOrder is the order the table is searched in, so the nearest match is
// found the same way every time.
var qrFormatOrder = []int{
	0x5412, 0x5125, 0x5E7C, 0x5B4B, 0x45F9, 0x40CE, 0x4F97, 0x4AA0,
	0x77C4, 0x72F3, 0x7DAA, 0x789D, 0x662F, 0x6318, 0x6C41, 0x6976,
	0x1689, 0x13BE, 0x1CE7, 0x19D0, 0x0762, 0x0255, 0x0D0C, 0x083B,
	0x355F, 0x3068, 0x3F31, 0x3A06, 0x24B4, 0x2183, 0x2EDA, 0x2BED,
}

// qrReaderMasks are the eight patterns one of which was applied to the data.
var qrReaderMasks = [8]func(x, y int) bool{
	func(x, y int) bool { return (y+x)%2 == 0 },
	func(x, y int) bool { return y%2 == 0 },
	func(x, y int) bool { return x%3 == 0 },
	func(x, y int) bool { return (y+x)%3 == 0 },
	func(x, y int) bool { return (y/2+x/3)%2 == 0 },
	func(x, y int) bool { return (x*y)%2+(x*y)%3 == 0 },
	func(x, y int) bool { return ((y*x)%2+(y*x)%3)%2 == 0 },
	func(x, y int) bool { return ((y+x)%2+(y*x)%3)%2 == 0 },
}

// The number of wrong bits a version or format field may carry and still be
// identified, given how far apart the valid fields are.
const qrMaxFieldErrors = 3
