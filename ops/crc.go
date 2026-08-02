package ops

import (
	"errors"
	"math/big"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(CRCChecksum{})
}

// CRCChecksum computes a Cyclic Redundancy Check. It follows CyberChef's
// CRCChecksum: ~170 named CRC variants plus a fully custom width/polynomial/
// initialisation/reflection/xor configuration, using the Rocksoft parameterised
// model computed bit-by-bit over big integers (so arbitrary widths up to 82 bits
// work).
type CRCChecksum struct{}

// crcParam holds the Rocksoft CRC parameters for a named algorithm. poly, init
// and xor are hex strings; width is in bits.
type crcParam struct {
	width         int
	poly, init    string
	refIn, refOut bool
	xor           string
}

// crcParams maps each named CRC algorithm to its parameters (transcribed from
// CyberChef's CRCChecksum run() switch).
var crcParams = map[string]crcParam{
	"CRC-3/GSM":                {3, "3", "0", false, false, "7"},
	"CRC-3/ROHC":               {3, "3", "7", true, true, "0"},
	"CRC-4/G-704":              {4, "3", "0", true, true, "0"},
	"CRC-4/INTERLAKEN":         {4, "3", "f", false, false, "f"},
	"CRC-4/ITU":                {4, "3", "0", true, true, "0"},
	"CRC-5/EPC":                {5, "09", "09", false, false, "00"},
	"CRC-5/EPC-C1G2":           {5, "09", "09", false, false, "00"},
	"CRC-5/G-704":              {5, "15", "00", true, true, "00"},
	"CRC-5/ITU":                {5, "15", "00", true, true, "00"},
	"CRC-5/USB":                {5, "05", "1f", true, true, "1f"},
	"CRC-6/CDMA2000-A":         {6, "27", "3f", false, false, "00"},
	"CRC-6/CDMA2000-B":         {6, "07", "3f", false, false, "00"},
	"CRC-6/DARC":               {6, "19", "00", true, true, "00"},
	"CRC-6/G-704":              {6, "03", "00", true, true, "00"},
	"CRC-6/GSM":                {6, "2f", "00", false, false, "3f"},
	"CRC-6/ITU":                {6, "03", "00", true, true, "00"},
	"CRC-7/MMC":                {7, "09", "00", false, false, "00"},
	"CRC-7/ROHC":               {7, "4f", "7f", true, true, "00"},
	"CRC-7/UMTS":               {7, "45", "00", false, false, "00"},
	"CRC-8":                    {8, "07", "00", false, false, "00"},
	"CRC-8/8H2F":               {8, "2f", "ff", false, false, "ff"},
	"CRC-8/AES":                {8, "1d", "ff", true, true, "00"},
	"CRC-8/AUTOSAR":            {8, "2f", "ff", false, false, "ff"},
	"CRC-8/BLUETOOTH":          {8, "a7", "00", true, true, "00"},
	"CRC-8/CDMA2000":           {8, "9b", "ff", false, false, "00"},
	"CRC-8/DARC":               {8, "39", "00", true, true, "00"},
	"CRC-8/DVB-S2":             {8, "d5", "00", false, false, "00"},
	"CRC-8/EBU":                {8, "1d", "ff", true, true, "00"},
	"CRC-8/GSM-A":              {8, "1d", "00", false, false, "00"},
	"CRC-8/GSM-B":              {8, "49", "00", false, false, "ff"},
	"CRC-8/HITAG":              {8, "1d", "ff", false, false, "00"},
	"CRC-8/I-432-1":            {8, "07", "00", false, false, "55"},
	"CRC-8/I-CODE":             {8, "1d", "fd", false, false, "00"},
	"CRC-8/ITU":                {8, "07", "00", false, false, "55"},
	"CRC-8/LTE":                {8, "9b", "00", false, false, "00"},
	"CRC-8/MAXIM":              {8, "31", "00", true, true, "00"},
	"CRC-8/MAXIM-DOW":          {8, "31", "00", true, true, "00"},
	"CRC-8/MIFARE-MAD":         {8, "1d", "c7", false, false, "00"},
	"CRC-8/NRSC-5":             {8, "31", "ff", false, false, "00"},
	"CRC-8/OPENSAFETY":         {8, "2f", "00", false, false, "00"},
	"CRC-8/ROHC":               {8, "07", "ff", true, true, "00"},
	"CRC-8/SAE-J1850":          {8, "1d", "ff", false, false, "ff"},
	"CRC-8/SAE-J1850-ZERO":     {8, "1d", "00", false, false, "00"},
	"CRC-8/SMBUS":              {8, "07", "00", false, false, "00"},
	"CRC-8/TECH-3250":          {8, "1d", "ff", true, true, "00"},
	"CRC-8/WCDMA":              {8, "9b", "00", true, true, "00"},
	"CRC-10/ATM":               {10, "233", "000", false, false, "000"},
	"CRC-10/CDMA2000":          {10, "3d9", "3ff", false, false, "000"},
	"CRC-10/GSM":               {10, "175", "000", false, false, "3ff"},
	"CRC-10/I-610":             {10, "233", "000", false, false, "000"},
	"CRC-11/FLEXRAY":           {11, "385", "01a", false, false, "000"},
	"CRC-11/UMTS":              {11, "307", "000", false, false, "000"},
	"CRC-12/3GPP":              {12, "80f", "000", false, true, "000"},
	"CRC-12/CDMA2000":          {12, "f13", "fff", false, false, "000"},
	"CRC-12/DECT":              {12, "80f", "000", false, false, "000"},
	"CRC-12/GSM":               {12, "d31", "000", false, false, "fff"},
	"CRC-12/UMTS":              {12, "80f", "000", false, true, "000"},
	"CRC-13/BBC":               {13, "1cf5", "0000", false, false, "0000"},
	"CRC-14/DARC":              {14, "0805", "0000", true, true, "0000"},
	"CRC-14/GSM":               {14, "202d", "0000", false, false, "3fff"},
	"CRC-15/CAN":               {15, "4599", "0000", false, false, "0000"},
	"CRC-15/MPT1327":           {15, "6815", "0000", false, false, "0001"},
	"CRC-16":                   {16, "8005", "0000", true, true, "0000"},
	"CRC-16/A":                 {16, "1021", "c6c6", true, true, "0000"},
	"CRC-16/ACORN":             {16, "1021", "0000", false, false, "0000"},
	"CRC-16/ARC":               {16, "8005", "0000", true, true, "0000"},
	"CRC-16/AUG-CCITT":         {16, "1021", "1d0f", false, false, "0000"},
	"CRC-16/AUTOSAR":           {16, "1021", "ffff", false, false, "0000"},
	"CRC-16/B":                 {16, "1021", "ffff", true, true, "ffff"},
	"CRC-16/BLUETOOTH":         {16, "1021", "0000", true, true, "0000"},
	"CRC-16/BUYPASS":           {16, "8005", "0000", false, false, "0000"},
	"CRC-16/CCITT":             {16, "1021", "0000", true, true, "0000"},
	"CRC-16/CCITT-FALSE":       {16, "1021", "ffff", false, false, "0000"},
	"CRC-16/CCITT-TRUE":        {16, "1021", "0000", true, true, "0000"},
	"CRC-16/CCITT-ZERO":        {16, "1021", "0000", false, false, "0000"},
	"CRC-16/CDMA2000":          {16, "c867", "ffff", false, false, "0000"},
	"CRC-16/CMS":               {16, "8005", "ffff", false, false, "0000"},
	"CRC-16/DARC":              {16, "1021", "ffff", false, false, "ffff"},
	"CRC-16/DDS-110":           {16, "8005", "800d", false, false, "0000"},
	"CRC-16/DECT-R":            {16, "0589", "0000", false, false, "0001"},
	"CRC-16/DECT-X":            {16, "0589", "0000", false, false, "0000"},
	"CRC-16/DNP":               {16, "3d65", "0000", true, true, "ffff"},
	"CRC-16/EN-13757":          {16, "3d65", "0000", false, false, "ffff"},
	"CRC-16/EPC":               {16, "1021", "ffff", false, false, "ffff"},
	"CRC-16/EPC-C1G2":          {16, "1021", "ffff", false, false, "ffff"},
	"CRC-16/GENIBUS":           {16, "1021", "ffff", false, false, "ffff"},
	"CRC-16/GSM":               {16, "1021", "0000", false, false, "ffff"},
	"CRC-16/I-CODE":            {16, "1021", "ffff", false, false, "ffff"},
	"CRC-16/IBM":               {16, "8005", "0000", true, true, "0000"},
	"CRC-16/IBM-3740":          {16, "1021", "ffff", false, false, "0000"},
	"CRC-16/IBM-SDLC":          {16, "1021", "ffff", true, true, "ffff"},
	"CRC-16/IEC-61158-2":       {16, "1dcf", "ffff", false, false, "ffff"},
	"CRC-16/ISO-HDLC":          {16, "1021", "ffff", true, true, "ffff"},
	"CRC-16/ISO-IEC-14443-3-A": {16, "1021", "c6c6", true, true, "0000"},
	"CRC-16/ISO-IEC-14443-3-B": {16, "1021", "ffff", true, true, "ffff"},
	"CRC-16/KERMIT":            {16, "1021", "0000", true, true, "0000"},
	"CRC-16/LHA":               {16, "8005", "0000", true, true, "0000"},
	"CRC-16/LJ1200":            {16, "6f63", "0000", false, false, "0000"},
	"CRC-16/LTE":               {16, "1021", "0000", false, false, "0000"},
	"CRC-16/M17":               {16, "5935", "ffff", false, false, "0000"},
	"CRC-16/MAXIM":             {16, "8005", "0000", true, true, "ffff"},
	"CRC-16/MAXIM-DOW":         {16, "8005", "0000", true, true, "ffff"},
	"CRC-16/MCRF4XX":           {16, "1021", "ffff", true, true, "0000"},
	"CRC-16/MODBUS":            {16, "8005", "ffff", true, true, "0000"},
	"CRC-16/NRSC-5":            {16, "080b", "ffff", true, true, "0000"},
	"CRC-16/OPENSAFETY-A":      {16, "5935", "0000", false, false, "0000"},
	"CRC-16/OPENSAFETY-B":      {16, "755b", "0000", false, false, "0000"},
	"CRC-16/PROFIBUS":          {16, "1dcf", "ffff", false, false, "ffff"},
	"CRC-16/RIELLO":            {16, "1021", "b2aa", true, true, "0000"},
	"CRC-16/SPI-FUJITSU":       {16, "1021", "1d0f", false, false, "0000"},
	"CRC-16/T10-DIF":           {16, "8bb7", "0000", false, false, "0000"},
	"CRC-16/TELEDISK":          {16, "a097", "0000", false, false, "0000"},
	"CRC-16/TMS37157":          {16, "1021", "89ec", true, true, "0000"},
	"CRC-16/UMTS":              {16, "8005", "0000", false, false, "0000"},
	"CRC-16/USB":               {16, "8005", "ffff", true, true, "ffff"},
	"CRC-16/V-41-LSB":          {16, "1021", "0000", true, true, "0000"},
	"CRC-16/V-41-MSB":          {16, "1021", "0000", false, false, "0000"},
	"CRC-16/VERIFONE":          {16, "8005", "0000", false, false, "0000"},
	"CRC-16/X-25":              {16, "1021", "ffff", true, true, "ffff"},
	"CRC-16/XMODEM":            {16, "1021", "0000", false, false, "0000"},
	"CRC-16/ZMODEM":            {16, "1021", "0000", false, false, "0000"},
	"CRC-17/CAN-FD":            {17, "1685b", "00000", false, false, "00000"},
	"CRC-21/CAN-FD":            {21, "102899", "000000", false, false, "000000"},
	"CRC-24/BLE":               {24, "00065b", "555555", true, true, "000000"},
	"CRC-24/FLEXRAY-A":         {24, "5d6dcb", "fedcba", false, false, "000000"},
	"CRC-24/FLEXRAY-B":         {24, "5d6dcb", "abcdef", false, false, "000000"},
	"CRC-24/INTERLAKEN":        {24, "328b63", "ffffff", false, false, "ffffff"},
	"CRC-24/LTE-A":             {24, "864cfb", "000000", false, false, "000000"},
	"CRC-24/LTE-B":             {24, "800063", "000000", false, false, "000000"},
	"CRC-24/OPENPGP":           {24, "864cfb", "b704ce", false, false, "000000"},
	"CRC-24/OS-9":              {24, "800063", "ffffff", false, false, "ffffff"},
	"CRC-30/CDMA":              {30, "2030b9c7", "3fffffff", false, false, "3fffffff"},
	"CRC-31/PHILIPS":           {31, "04c11db7", "7fffffff", false, false, "7fffffff"},
	"CRC-32":                   {32, "04c11db7", "ffffffff", true, true, "ffffffff"},
	"CRC-32/AAL5":              {32, "04c11db7", "ffffffff", false, false, "ffffffff"},
	"CRC-32/ADCCP":             {32, "04c11db7", "ffffffff", true, true, "ffffffff"},
	"CRC-32/AIXM":              {32, "814141ab", "00000000", false, false, "00000000"},
	"CRC-32/AUTOSAR":           {32, "f4acfb13", "ffffffff", true, true, "ffffffff"},
	"CRC-32/BASE91-C":          {32, "1edc6f41", "ffffffff", true, true, "ffffffff"},
	"CRC-32/BASE91-D":          {32, "a833982b", "ffffffff", true, true, "ffffffff"},
	"CRC-32/BZIP2":             {32, "04c11db7", "ffffffff", false, false, "ffffffff"},
	"CRC-32/C":                 {32, "1edc6f41", "ffffffff", true, true, "ffffffff"},
	"CRC-32/CASTAGNOLI":        {32, "1edc6f41", "ffffffff", true, true, "ffffffff"},
	"CRC-32/CD-ROM-EDC":        {32, "8001801b", "00000000", true, true, "00000000"},
	"CRC-32/CKSUM":             {32, "04c11db7", "00000000", false, false, "ffffffff"},
	"CRC-32/D":                 {32, "a833982b", "ffffffff", true, true, "ffffffff"},
	"CRC-32/DECT-B":            {32, "04c11db7", "ffffffff", false, false, "ffffffff"},
	"CRC-32/INTERLAKEN":        {32, "1edc6f41", "ffffffff", true, true, "ffffffff"},
	"CRC-32/ISCSI":             {32, "1edc6f41", "ffffffff", true, true, "ffffffff"},
	"CRC-32/ISO-HDLC":          {32, "04c11db7", "ffffffff", true, true, "ffffffff"},
	"CRC-32/JAMCRC":            {32, "04c11db7", "ffffffff", true, true, "00000000"},
	"CRC-32/MEF":               {32, "741b8cd7", "ffffffff", true, true, "00000000"},
	"CRC-32/MPEG-2":            {32, "04c11db7", "ffffffff", false, false, "00000000"},
	"CRC-32/NVME":              {32, "1edc6f41", "ffffffff", true, true, "ffffffff"},
	"CRC-32/PKZIP":             {32, "04c11db7", "ffffffff", true, true, "ffffffff"},
	"CRC-32/POSIX":             {32, "04c11db7", "00000000", false, false, "ffffffff"},
	"CRC-32/Q":                 {32, "814141ab", "00000000", false, false, "00000000"},
	"CRC-32/SATA":              {32, "04c11db7", "52325032", false, false, "00000000"},
	"CRC-32/V-42":              {32, "04c11db7", "ffffffff", true, true, "ffffffff"},
	"CRC-32/XFER":              {32, "000000af", "00000000", false, false, "00000000"},
	"CRC-32/XZ":                {32, "04c11db7", "ffffffff", true, true, "ffffffff"},
	"CRC-40/GSM":               {40, "0004820009", "0000000000", false, false, "ffffffffff"},
	"CRC-64/ECMA-182":          {64, "42f0e1eba9ea3693", "0000000000000000", false, false, "0000000000000000"},
	"CRC-64/GO-ECMA":           {64, "42f0e1eba9ea3693", "ffffffffffffffff", true, true, "ffffffffffffffff"},
	"CRC-64/GO-ISO":            {64, "000000000000001b", "ffffffffffffffff", true, true, "ffffffffffffffff"},
	"CRC-64/MS":                {64, "259c84cba6426349", "ffffffffffffffff", true, true, "0000000000000000"},
	"CRC-64/NVME":              {64, "ad93d23594c93659", "ffffffffffffffff", true, true, "ffffffffffffffff"},
	"CRC-64/REDIS":             {64, "ad93d23594c935a9", "0000000000000000", true, true, "0000000000000000"},
	"CRC-64/WE":                {64, "42f0e1eba9ea3693", "ffffffffffffffff", false, false, "ffffffffffffffff"},
	"CRC-64/XZ":                {64, "42f0e1eba9ea3693", "ffffffffffffffff", true, true, "ffffffffffffffff"},
	"CRC-82/DARC":              {82, "0308c0111011401440411", "000000000000000000000", true, true, "000000000000000000000"},
}

// crcAlgorithms is the ordered algorithm selector (Custom first, then every
// named variant), matching CyberChef's dropdown.
var crcAlgorithms = []string{
	"Custom",
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
	"CRC-40/GSM",
	"CRC-64/ECMA-182",
	"CRC-64/GO-ECMA",
	"CRC-64/GO-ISO",
	"CRC-64/MS",
	"CRC-64/NVME",
	"CRC-64/REDIS",
	"CRC-64/WE",
	"CRC-64/XZ",
	"CRC-82/DARC",
}

// errCustomCRC is CyberChef's error for malformed custom CRC parameters.
var errCustomCRC = errors.New("Invalid custom CRC arguments") //nolint:staticcheck,revive // verbatim CyberChef text

// Meta returns the operation metadata.
func (CRCChecksum) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "CRC Checksum",
		Module:      "Default",
		Description: "A Cyclic Redundancy Check (CRC) is an error-detecting code commonly used in digital networks and storage devices to detect accidental changes to raw data.",
		InfoURL:     "https://wikipedia.org/wiki/Cyclic_redundancy_check",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (CRCChecksum) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Algorithm", Type: core.ArgOption, Value: crcAlgorithms},
		{Name: "Width (bits)", Type: core.ArgToggleString, Value: "0", ToggleValues: []string{"Decimal"}},
		{Name: "Polynomial", Type: core.ArgToggleString, Value: "0", ToggleValues: []string{"Hex"}},
		{Name: "Initialization", Type: core.ArgToggleString, Value: "0", ToggleValues: []string{"Hex"}},
		{Name: "Reflect input", Type: core.ArgOption, Value: []string{"True", "False"}},
		{Name: "Reflect output", Type: core.ArgOption, Value: []string{"True", "False"}},
		{Name: "Xor Output", Type: core.ArgToggleString, Value: "0", ToggleValues: []string{"Hex"}},
	}
}

// Run computes the selected CRC over the input bytes.
func (CRCChecksum) Run(in *core.Dish, args []any) (*core.Dish, error) {
	out, err := crcChecksumValue(args[0].(string), args, in.Bytes())
	if err != nil {
		return nil, err
	}
	return core.NewDish([]byte(out), core.TypeString), nil
}

// crcChecksumValue dispatches on the algorithm name, handling Custom mode and the
// named-variant table.
func crcChecksumValue(algo string, args []any, data []byte) (string, error) {
	if algo == "Custom" {
		return crcCustom(args, data)
	}
	p, ok := crcParams[algo]
	if !ok {
		//nolint:staticcheck,revive // verbatim CyberChef OperationError text
		return "", errors.New("Unknown checksum algorithm")
	}
	return crcCompute(p.width, data, bigFromHex(p.poly), bigFromHex(p.init), p.refIn, p.refOut, bigFromHex(p.xor)), nil
}

// crcCustom parses the custom width/polynomial/initialisation/reflection/xor
// arguments and computes the CRC, mirroring CyberChef's error handling (width is
// decimal; polynomial, initialisation and xor are hex; any malformed value is
// rejected).
func crcCustom(args []any, data []byte) (string, error) {
	widthTS := args[1].(core.ToggleString)
	polyTS := args[2].(core.ToggleString)
	initTS := args[3].(core.ToggleString)
	refIn := args[4].(string) == "True"
	refOut := args[5].(string) == "True"
	xorTS := args[6].(core.ToggleString)

	width, okW := new(big.Int).SetString(widthTS.Value, 10)
	if !okW || width.Sign() < 1 {
		return "", errCustomCRC
	}
	poly, okP := new(big.Int).SetString(polyTS.Value, 16)
	init, okI := new(big.Int).SetString(initTS.Value, 16)
	xor, okX := new(big.Int).SetString(xorTS.Value, 16)
	if !okP || !okI || !okX {
		return "", errCustomCRC
	}
	return crcCompute(int(width.Int64()), data, poly, init, refIn, refOut, xor), nil
}

// crcCompute runs the parameterised CRC bit-by-bit, returning the result as hex
// padded to ceil(width/4) digits.
func crcCompute(width int, data []byte, poly, init *big.Int, refIn, refOut bool, xor *big.Int) string {
	one := big.NewInt(1)
	topBit := new(big.Int).Lsh(one, uint(width-1))
	mask := new(big.Int).Sub(new(big.Int).Lsh(one, uint(width)), one)
	rem := new(big.Int).Set(init)

	for _, by := range data {
		b := big.NewInt(int64(by))
		if refIn {
			b = crcReflect(b, 8)
		}
		for i := 7; i >= 0; i-- {
			bit := new(big.Int).And(rem, topBit)
			rem.And(rem.Lsh(rem, 1), mask)
			if b.Bit(i) == 1 {
				bit.Xor(bit, topBit)
			}
			if bit.Sign() != 0 {
				rem.Xor(rem, poly)
			}
		}
	}
	if refOut {
		rem = crcReflect(rem, width)
	}
	rem.Xor(rem, xor)

	s := rem.Text(16)
	for digits := (width + 3) / 4; len(s) < digits; {
		s = "0" + s
	}
	return s
}

// crcReflect reverses the low nbits bits of data.
func crcReflect(data *big.Int, nbits int) *big.Int {
	value := new(big.Int)
	d := new(big.Int).Set(data)
	for bit := range nbits {
		if d.Bit(0) == 1 {
			value.SetBit(value, nbits-1-bit, 1)
		}
		d.Rsh(d, 1)
	}
	return value
}
