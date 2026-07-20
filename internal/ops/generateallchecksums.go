package ops

import (
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(GenerateAllChecksums{})
}

// GenerateAllChecksums runs every built-in checksum over the input and lists the
// results. A faithful port of CyberChef's GenerateAllChecksums: the same ordered
// set of CRC variants plus the Fletcher and Adler-32 checksums, optionally
// filtered by bit width and labelled with aligned names.
type GenerateAllChecksums struct{}

// genChecksumOrder is the ordered list of checksum names exactly as CyberChef
// lists them (CRC variants interleaved with Fletcher-8/16/32/64 and Adler-32 at
// their respective widths).
var genChecksumOrder = []string{
	"CRC-3/GSM",
	"CRC-3/ROHC",
	"CRC-4/G-704",
	"CRC-4/INTERLAKEN",
	"CRC-4/ITU",
	"CRC-5/EPC",
	"CRC-5/EPC-C1G2",
	"CRC-5/G-704",
	"CRC-5/ITU",
	"CRC-5/USB",
	"CRC-6/CDMA2000-A",
	"CRC-6/CDMA2000-B",
	"CRC-6/DARC",
	"CRC-6/G-704",
	"CRC-6/GSM",
	"CRC-6/ITU",
	"CRC-7/MMC",
	"CRC-7/ROHC",
	"CRC-7/UMTS",
	"CRC-8",
	"CRC-8/8H2F",
	"CRC-8/AES",
	"CRC-8/AUTOSAR",
	"CRC-8/BLUETOOTH",
	"CRC-8/CDMA2000",
	"CRC-8/DARC",
	"CRC-8/DVB-S2",
	"CRC-8/EBU",
	"CRC-8/GSM-A",
	"CRC-8/GSM-B",
	"CRC-8/HITAG",
	"CRC-8/I-432-1",
	"CRC-8/I-CODE",
	"CRC-8/ITU",
	"CRC-8/LTE",
	"CRC-8/MAXIM",
	"CRC-8/MAXIM-DOW",
	"CRC-8/MIFARE-MAD",
	"CRC-8/NRSC-5",
	"CRC-8/OPENSAFETY",
	"CRC-8/ROHC",
	"CRC-8/SAE-J1850",
	"CRC-8/SAE-J1850-ZERO",
	"CRC-8/SMBUS",
	"CRC-8/TECH-3250",
	"CRC-8/WCDMA",
	"Fletcher-8",
	"CRC-10/ATM",
	"CRC-10/CDMA2000",
	"CRC-10/GSM",
	"CRC-10/I-610",
	"CRC-11/FLEXRAY",
	"CRC-11/UMTS",
	"CRC-12/3GPP",
	"CRC-12/CDMA2000",
	"CRC-12/DECT",
	"CRC-12/GSM",
	"CRC-12/UMTS",
	"CRC-13/BBC",
	"CRC-14/DARC",
	"CRC-14/GSM",
	"CRC-15/CAN",
	"CRC-15/MPT1327",
	"CRC-16",
	"CRC-16/A",
	"CRC-16/ACORN",
	"CRC-16/ARC",
	"CRC-16/AUG-CCITT",
	"CRC-16/AUTOSAR",
	"CRC-16/B",
	"CRC-16/BLUETOOTH",
	"CRC-16/BUYPASS",
	"CRC-16/CCITT",
	"CRC-16/CCITT-FALSE",
	"CRC-16/CCITT-TRUE",
	"CRC-16/CCITT-ZERO",
	"CRC-16/CDMA2000",
	"CRC-16/CMS",
	"CRC-16/DARC",
	"CRC-16/DDS-110",
	"CRC-16/DECT-R",
	"CRC-16/DECT-X",
	"CRC-16/DNP",
	"CRC-16/EN-13757",
	"CRC-16/EPC",
	"CRC-16/EPC-C1G2",
	"CRC-16/GENIBUS",
	"CRC-16/GSM",
	"CRC-16/I-CODE",
	"CRC-16/IBM",
	"CRC-16/IBM-3740",
	"CRC-16/IBM-SDLC",
	"CRC-16/IEC-61158-2",
	"CRC-16/ISO-HDLC",
	"CRC-16/ISO-IEC-14443-3-A",
	"CRC-16/ISO-IEC-14443-3-B",
	"CRC-16/KERMIT",
	"CRC-16/LHA",
	"CRC-16/LJ1200",
	"CRC-16/LTE",
	"CRC-16/M17",
	"CRC-16/MAXIM",
	"CRC-16/MAXIM-DOW",
	"CRC-16/MCRF4XX",
	"CRC-16/MODBUS",
	"CRC-16/NRSC-5",
	"CRC-16/OPENSAFETY-A",
	"CRC-16/OPENSAFETY-B",
	"CRC-16/PROFIBUS",
	"CRC-16/RIELLO",
	"CRC-16/SPI-FUJITSU",
	"CRC-16/T10-DIF",
	"CRC-16/TELEDISK",
	"CRC-16/TMS37157",
	"CRC-16/UMTS",
	"CRC-16/USB",
	"CRC-16/V-41-LSB",
	"CRC-16/V-41-MSB",
	"CRC-16/VERIFONE",
	"CRC-16/X-25",
	"CRC-16/XMODEM",
	"CRC-16/ZMODEM",
	"Fletcher-16",
	"CRC-17/CAN-FD",
	"CRC-21/CAN-FD",
	"CRC-24/BLE",
	"CRC-24/FLEXRAY-A",
	"CRC-24/FLEXRAY-B",
	"CRC-24/INTERLAKEN",
	"CRC-24/LTE-A",
	"CRC-24/LTE-B",
	"CRC-24/OPENPGP",
	"CRC-24/OS-9",
	"CRC-30/CDMA",
	"CRC-31/PHILIPS",
	"Adler-32",
	"CRC-32",
	"CRC-32/AAL5",
	"CRC-32/ADCCP",
	"CRC-32/AIXM",
	"CRC-32/AUTOSAR",
	"CRC-32/BASE91-C",
	"CRC-32/BASE91-D",
	"CRC-32/BZIP2",
	"CRC-32/C",
	"CRC-32/CASTAGNOLI",
	"CRC-32/CD-ROM-EDC",
	"CRC-32/CKSUM",
	"CRC-32/D",
	"CRC-32/DECT-B",
	"CRC-32/INTERLAKEN",
	"CRC-32/ISCSI",
	"CRC-32/ISO-HDLC",
	"CRC-32/JAMCRC",
	"CRC-32/MEF",
	"CRC-32/MPEG-2",
	"CRC-32/NVME",
	"CRC-32/PKZIP",
	"CRC-32/POSIX",
	"CRC-32/Q",
	"CRC-32/SATA",
	"CRC-32/V-42",
	"CRC-32/XFER",
	"CRC-32/XZ",
	"Fletcher-32",
	"CRC-40/GSM",
	"CRC-64/ECMA-182",
	"CRC-64/GO-ECMA",
	"CRC-64/GO-ISO",
	"CRC-64/MS",
	"CRC-64/NVME",
	"CRC-64/REDIS",
	"CRC-64/WE",
	"CRC-64/XZ",
	"Fletcher-64",
	"CRC-82/DARC",
}

// genChecksumLengths are the width options offered by the operation.
var genChecksumLengths = []string{
	"All", "3", "4", "5", "6", "7", "8", "10", "11", "12", "13", "14", "15",
	"16", "17", "21", "24", "30", "31", "32", "40", "64", "82",
}

// genLengthRe extracts a checksum's bit width from its name: the first "-<digits>"
// group followed by "/" or the end of the name (e.g. CRC-16/A → 16, Adler-32 → 32).
var genLengthRe = regexp.MustCompile(`-(\d{1,2})(/|$)`)

// genNamePad is the column the checksum value is aligned to when names are shown.
const genNamePad = 25

// Meta returns the operation metadata.
func (GenerateAllChecksums) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Generate all checksums",
		Module:      "Crypto",
		Description: "Generates all available checksums for the input.",
		InfoURL:     "https://wikipedia.org/wiki/Checksum",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (GenerateAllChecksums) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Length (bits)", Type: core.ArgOption, Value: genChecksumLengths},
		{Name: "Include names", Type: core.ArgBoolean, Value: true},
	}
}

// Run generates the selected checksums.
func (GenerateAllChecksums) Run(in *core.Dish, args []any) (*core.Dish, error) {
	length := args[0].(string)
	includeNames := args[1].(bool)
	data := in.Bytes()

	var out strings.Builder
	for _, name := range genChecksumOrder {
		if length != "All" && length != genLengthRe.FindStringSubmatch(name)[1] {
			continue
		}
		value := genChecksumValue(name, data)
		if includeNames {
			out.WriteString(name + ":" + strings.Repeat(" ", genNamePad-len(name)) + value + "\n")
		} else {
			out.WriteString(value + "\n")
		}
	}
	return core.NewDish([]byte(out.String()), core.TypeString), nil
}

// genChecksumValue computes one checksum by name, delegating to the corresponding
// operation. The named CRC variants and these fixed-parameter checksums never
// error, so the errors are discarded.
func genChecksumValue(name string, data []byte) string {
	switch name {
	case "Fletcher-8":
		return subChecksum(Fletcher8Checksum{}, data)
	case "Fletcher-16":
		return subChecksum(Fletcher16Checksum{}, data)
	case "Fletcher-32":
		return subChecksum(Fletcher32Checksum{}, data)
	case "Fletcher-64":
		return subChecksum(Fletcher64Checksum{}, data)
	case "Adler-32":
		return subChecksum(Adler32{}, data)
	default:
		v, _ := crcChecksumValue(name, nil, data)
		return v
	}
}

// subChecksum runs a fixed-parameter checksum operation over data and returns its
// string result.
func subChecksum(op core.Operation, data []byte) string {
	d, _ := op.Run(core.NewDish(data, core.TypeArrayBuffer), nil)
	return d.String()
}
