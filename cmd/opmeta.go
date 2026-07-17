package cmd

import (
	"slices"
	"sort"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

// maxSummaryLen bounds a derived one-line summary so `cchef list` and command
// help do not wrap in a standard-width terminal.
const maxSummaryLen = 60

// summaryOf returns the concise one-line summary shown for an operation in
// `cchef list` and as its cobra Short. It prefers a curated entry in
// opSummaries and otherwise derives one from the (often multi-sentence)
// CyberChef description.
func summaryOf(meta core.OpMeta) string {
	if s, ok := opSummaries[meta.Name]; ok && s != "" {
		return s
	}
	return deriveSummary(meta.Description)
}

// summaryAbbrevs are lowercase abbreviations whose trailing period must not be
// treated as a sentence boundary when deriving a summary.
var summaryAbbrevs = map[string]bool{
	"e.g": true, "i.e": true, "etc": true, "vs": true, "no": true,
	"approx": true, "incl": true, "al": true,
}

// deriveSummary compresses a description to a single short clause: the first
// sentence, trimmed of its trailing period, truncated at a word boundary with
// an ellipsis if still too long.
func deriveSummary(desc string) string {
	desc = strings.TrimSpace(desc)
	desc = strings.TrimSpace(firstSentence(desc))
	if len(desc) <= maxSummaryLen {
		return desc
	}
	cut := desc[:maxSummaryLen]
	if sp := strings.LastIndex(cut, " "); sp > 0 {
		cut = cut[:sp]
	}
	return strings.TrimRight(cut, " ,;:") + "…"
}

// firstSentence returns s up to the first sentence-terminating ., ! or ? that
// is followed by whitespace (or end of string) and not part of an abbreviation
// such as "e.g.". If none is found it returns all of s.
func firstSentence(s string) string {
	for i := 0; i < len(s); i++ {
		if c := s[i]; c == '.' || c == '!' || c == '?' {
			if i+1 < len(s) && s[i+1] != ' ' && s[i+1] != '\n' && s[i+1] != '\t' {
				continue
			}
			if summaryAbbrevs[strings.ToLower(lastWord(s[:i]))] {
				continue
			}
			return s[:i]
		}
	}
	return s
}

// lastWord returns the trailing run of letters and dots in s (so "up to (e.g"
// yields "e.g"), used to detect abbreviations before a period.
func lastWord(s string) string {
	i := len(s)
	for i > 0 {
		c := s[i-1]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '.' {
			i--
			continue
		}
		break
	}
	return s[i:]
}

// CyberChef operation categories, used as the values in opCategories and as the
// grouping headings in `cchef list`. Defined as constants so a mistyped category
// is a compile error rather than a silent new group.
const (
	catArithmeticLogic    = "Arithmetic / Logic"
	catCodeTidy           = "Code tidy"
	catDataFormat         = "Data format"
	catDateTime           = "Date / Time"
	catEncryptionEncoding = "Encryption / Encoding"
	catHashing            = "Hashing"
	catLanguage           = "Language"
	catNetworking         = "Networking"
	catPublicKey          = "Public Key"
	catUtils              = "Utils"
)

// opCategories maps each operation's display name to the CyberChef categories it
// belongs to, mirroring the master table in docs/README.md. It is the source of
// truth for grouping `cchef list`. A few operations (URL Decode/Encode) belong
// to more than one category. TestOpCategoriesMatchRegistry keeps this table in
// exact sync with the registered operations.
var opCategories = map[string][]string{
	"A1Z26 Cipher Decode":                {catEncryptionEncoding},
	"A1Z26 Cipher Encode":                {catEncryptionEncoding},
	"ADD":                                {catEncryptionEncoding},
	"AES Decrypt":                        {catEncryptionEncoding},
	"AES Encrypt":                        {catEncryptionEncoding},
	"AES Key Unwrap":                     {catEncryptionEncoding},
	"AES Key Wrap":                       {catEncryptionEncoding},
	"AMF Decode":                         {catDataFormat},
	"AMF Encode":                         {catDataFormat},
	"AND":                                {catEncryptionEncoding},
	"Add line numbers":                   {catUtils},
	"Adler-32 Checksum":                  {catHashing},
	"Affine Cipher Decode":               {catEncryptionEncoding},
	"Affine Cipher Encode":               {catEncryptionEncoding},
	"Alternating Caps":                   {catUtils},
	"Ascon Decrypt":                      {catEncryptionEncoding},
	"Ascon Encrypt":                      {catEncryptionEncoding},
	"Atbash Cipher":                      {catEncryptionEncoding},
	"Avro to JSON":                       {catDataFormat},
	"Bacon Cipher Decode":                {catEncryptionEncoding},
	"Bacon Cipher Encode":                {catEncryptionEncoding},
	"Bcrypt":                             {catEncryptionEncoding, catHashing},
	"Bcrypt compare":                     {catHashing},
	"Bcrypt parse":                       {catHashing},
	"Bifid Cipher Decode":                {catEncryptionEncoding},
	"Bifid Cipher Encode":                {catEncryptionEncoding},
	"Bit shift left":                     {catEncryptionEncoding},
	"Bit shift right":                    {catEncryptionEncoding},
	"Blowfish Decrypt":                   {catEncryptionEncoding},
	"Blowfish Encrypt":                   {catEncryptionEncoding},
	"Bombe":                              {catEncryptionEncoding},
	"Caesar Box Cipher":                  {catEncryptionEncoding},
	"Caret/M-decode":                     {catDataFormat},
	"Cartesian Product":                  {catArithmeticLogic},
	"CBOR Decode":                        {catDataFormat},
	"CBOR Encode":                        {catDataFormat},
	"Cetacean Cipher Decode":             {catEncryptionEncoding},
	"Cetacean Cipher Encode":             {catEncryptionEncoding},
	"ChaCha":                             {catEncryptionEncoding},
	"Change IP format":                   {catNetworking},
	"CipherSaber2 Decrypt":               {catEncryptionEncoding},
	"CipherSaber2 Encrypt":               {catEncryptionEncoding},
	"Citrix CTX1 Decode":                 {catEncryptionEncoding},
	"Citrix CTX1 Encode":                 {catEncryptionEncoding},
	"Colossus":                           {catEncryptionEncoding},
	"Convert area":                       {catUtils},
	"Convert co-ordinate format":         {catUtils},
	"Convert data units":                 {catUtils},
	"Convert distance":                   {catUtils},
	"Convert mass":                       {catUtils},
	"Convert speed":                      {catUtils},
	"Count occurrences":                  {catUtils},
	"CSV to JSON":                        {catDataFormat},
	"DNS over HTTPS":                     {catNetworking},
	"DateTime Delta":                     {catDateTime},
	"Dechunk HTTP response":              {catNetworking},
	"Decode NetBIOS Name":                {catNetworking},
	"Decode text":                        {catDataFormat, catLanguage},
	"Defang IP Addresses":                {catNetworking},
	"Defang URL":                         {catNetworking},
	"Derive EVP key":                     {catEncryptionEncoding},
	"Derive HKDF key":                    {catEncryptionEncoding},
	"Derive PBKDF2 key":                  {catEncryptionEncoding},
	"DES Decrypt":                        {catEncryptionEncoding},
	"DES Encrypt":                        {catEncryptionEncoding},
	"Diff":                               {catUtils, catCodeTidy},
	"Divide":                             {catArithmeticLogic},
	"Drop bytes":                         {catUtils},
	"Drop nth bytes":                     {catUtils},
	"Encode NetBIOS Name":                {catNetworking},
	"Encode text":                        {catDataFormat, catLanguage},
	"Enigma":                             {catEncryptionEncoding},
	"Escape Smart Characters":            {catDataFormat},
	"Escape Unicode Characters":          {catDataFormat},
	"Escape string":                      {catUtils},
	"Expand alphabet range":              {catUtils},
	"Extract dates":                      {catDateTime},
	"Fang URL":                           {catNetworking},
	"File Tree":                          {catUtils},
	"Filter":                             {catUtils},
	"Find / Replace":                     {catUtils},
	"Format MAC addresses":               {catNetworking},
	"From Base":                          {catDataFormat},
	"From Base32":                        {catDataFormat},
	"From Base45":                        {catDataFormat},
	"From Base58":                        {catDataFormat},
	"From Base62":                        {catDataFormat},
	"From Base64":                        {catDataFormat},
	"From Base85":                        {catDataFormat},
	"From Base92":                        {catDataFormat},
	"From BCD":                           {catDataFormat},
	"From Bech32":                        {catDataFormat},
	"From Binary":                        {catDataFormat},
	"From Braille":                       {catDataFormat},
	"From Case Insensitive Regex":        {catUtils},
	"From Charcode":                      {catDataFormat},
	"From Decimal":                       {catDataFormat},
	"From Float":                         {catDataFormat},
	"From HTML Entity":                   {catDataFormat},
	"From Hex":                           {catDataFormat},
	"From Hex Content":                   {catDataFormat},
	"From Hexdump":                       {catDataFormat},
	"From MessagePack":                   {catDataFormat, catCodeTidy},
	"From Modhex":                        {catDataFormat},
	"From Octal":                         {catDataFormat},
	"From Punycode":                      {catDataFormat},
	"From Quoted Printable":              {catDataFormat},
	"From UNIX Timestamp":                {catDateTime},
	"Fuzzy Match":                        {catUtils},
	"Get All Casings":                    {catUtils},
	"Get Time":                           {catDateTime},
	"Group IP addresses":                 {catNetworking},
	"HAS-160":                            {catHashing},
	"HASSH Client Fingerprint":           {catNetworking},
	"HASSH Server Fingerprint":           {catNetworking},
	"HMAC":                               {catHashing},
	"HTTP request":                       {catNetworking},
	"Hamming Distance":                   {catUtils},
	"Head":                               {catUtils},
	"Hex to PEM":                         {catDataFormat, catPublicKey},
	"IPv6 Transition Addresses":          {catNetworking},
	"JA3 Fingerprint":                    {catNetworking},
	"JA3S Fingerprint":                   {catNetworking},
	"JA4 Fingerprint":                    {catNetworking},
	"JA4Server Fingerprint":              {catNetworking},
	"JSON to CSV":                        {catDataFormat},
	"JSON to YAML":                       {catDataFormat},
	"Keccak":                             {catHashing},
	"Levenshtein Distance":               {catUtils},
	"Lorenz":                             {catEncryptionEncoding},
	"MD2":                                {catHashing},
	"MD4":                                {catHashing},
	"MD5":                                {catHashing},
	"Mean":                               {catArithmeticLogic},
	"Median":                             {catArithmeticLogic},
	"MIME Decoding":                      {catDataFormat},
	"Multiple Bombe":                     {catEncryptionEncoding},
	"Multiply":                           {catArithmeticLogic},
	"Normalise Unicode":                  {catDataFormat},
	"NOT":                                {catEncryptionEncoding},
	"OR":                                 {catEncryptionEncoding},
	"Offset checker":                     {catUtils},
	"PEM to Hex":                         {catDataFormat, catPublicKey},
	"Pad lines":                          {catUtils},
	"Parse ASN.1 hex string":             {catDataFormat, catPublicKey},
	"Parse DateTime":                     {catDateTime},
	"Parse Ethernet frame":               {catNetworking},
	"Parse IP range":                     {catNetworking},
	"Parse IPv4 header":                  {catNetworking},
	"Parse IPv6 address":                 {catNetworking},
	"Parse ObjectID timestamp":           {catUtils},
	"Parse SSH Host Key":                 {catNetworking, catPublicKey},
	"Parse TCP":                          {catNetworking},
	"Parse TLS record":                   {catNetworking},
	"Parse TLV":                          {catDataFormat},
	"Parse UDP":                          {catNetworking},
	"Parse UNIX file permissions":        {catUtils},
	"Parse URI":                          {catNetworking},
	"Parse User Agent":                   {catNetworking},
	"Parse colour code":                  {catUtils},
	"Power Set":                          {catArithmeticLogic},
	"Protobuf Decode":                    {catNetworking},
	"Protobuf Encode":                    {catNetworking},
	"Pseudo-Random Number Generator":     {catUtils},
	"ROR13":                              {catEncryptionEncoding},
	"RIPEMD":                             {catHashing},
	"ROT13":                              {catEncryptionEncoding},
	"ROT47":                              {catEncryptionEncoding},
	"ROT8000":                            {catEncryptionEncoding},
	"Regular expression":                 {catUtils},
	"Remove ANSI Escape Codes":           {catUtils},
	"Remove line numbers":                {catUtils},
	"Remove null bytes":                  {catUtils},
	"Remove whitespace":                  {catUtils},
	"Reverse":                            {catUtils},
	"Rison Decode":                       {catDataFormat},
	"Rison Encode":                       {catDataFormat},
	"Rotate left":                        {catEncryptionEncoding},
	"Rotate right":                       {catEncryptionEncoding},
	"SHA0":                               {catHashing},
	"SHA1":                               {catHashing},
	"SHA224":                             {catHashing},
	"SHA256":                             {catHashing},
	"SHA3":                               {catHashing},
	"SHA384":                             {catHashing},
	"SHA512":                             {catHashing},
	"SUB":                                {catEncryptionEncoding},
	"Set Difference":                     {catArithmeticLogic},
	"Set Intersection":                   {catArithmeticLogic},
	"Set Union":                          {catArithmeticLogic},
	"Show Base64 offsets":                {catDataFormat},
	"Show on map":                        {catUtils},
	"Shuffle":                            {catUtils},
	"Sleep":                              {catUtils},
	"Snefru":                             {catHashing},
	"Sort":                               {catUtils},
	"Split":                              {catUtils},
	"Standard Deviation":                 {catArithmeticLogic},
	"Strip HTTP headers":                 {catNetworking},
	"Strip IPv4 header":                  {catNetworking},
	"Strip TCP header":                   {catNetworking},
	"Strip UDP header":                   {catNetworking},
	"Subtract":                           {catArithmeticLogic},
	"Sum":                                {catArithmeticLogic},
	"Swap case":                          {catUtils},
	"Swap endianness":                    {catDataFormat},
	"Symmetric Difference":               {catArithmeticLogic},
	"Tail":                               {catUtils},
	"Take bytes":                         {catUtils},
	"Take nth bytes":                     {catUtils},
	"Text Encoding Brute Force":          {catDataFormat},
	"Text-Integer Conversion":            {catDataFormat},
	"To Base":                            {catDataFormat},
	"To Base32":                          {catDataFormat},
	"To Base45":                          {catDataFormat},
	"To Base58":                          {catDataFormat},
	"To Base62":                          {catDataFormat},
	"To Base64":                          {catDataFormat},
	"To Base85":                          {catDataFormat},
	"To Base92":                          {catDataFormat},
	"To BCD":                             {catDataFormat},
	"To Bech32":                          {catDataFormat},
	"To Binary":                          {catDataFormat},
	"To Braille":                         {catDataFormat},
	"To Case Insensitive Regex":          {catUtils},
	"To Charcode":                        {catDataFormat},
	"To Decimal":                         {catDataFormat},
	"To Float":                           {catDataFormat},
	"To HTML Entity":                     {catDataFormat},
	"To Hex":                             {catDataFormat},
	"To Hex Content":                     {catDataFormat},
	"To Hexdump":                         {catDataFormat},
	"To Lower case":                      {catUtils},
	"To MessagePack":                     {catDataFormat, catCodeTidy},
	"To Modhex":                          {catDataFormat},
	"To Octal":                           {catDataFormat},
	"To Punycode":                        {catDataFormat},
	"To Quoted Printable":                {catDataFormat},
	"To Table":                           {catUtils},
	"To UNIX Timestamp":                  {catDateTime},
	"To Upper case":                      {catUtils},
	"Translate DateTime Format":          {catDateTime},
	"Triple DES Decrypt":                 {catEncryptionEncoding},
	"Triple DES Encrypt":                 {catEncryptionEncoding},
	"UNIX Timestamp to Windows Filetime": {catDateTime},
	"URL Decode":                         {catDataFormat, catNetworking},
	"Unescape Unicode Characters":        {catDataFormat, catLanguage},
	"URL Encode":                         {catDataFormat, catNetworking},
	"Unescape string":                    {catUtils},
	"Unique":                             {catUtils},
	"VarInt Decode":                      {catNetworking},
	"VarInt Encode":                      {catNetworking},
	"Whirlpool":                          {catHashing},
	"Windows Filetime to UNIX Timestamp": {catDateTime},
	"Wrap":                               {catUtils},
	"XOR":                                {catEncryptionEncoding},
	"XOR Brute Force":                    {catEncryptionEncoding},
	"YAML to JSON":                       {catDataFormat},
}

// categoriesOf returns the categories an operation belongs to (sorted), or a
// single "Uncategorized" bucket if the name is absent (which the reconciliation
// test prevents for registered operations).
func categoriesOf(name string) []string {
	cats, ok := opCategories[name]
	if !ok || len(cats) == 0 {
		return []string{"Uncategorized"}
	}
	out := slices.Clone(cats)
	sort.Strings(out)
	return out
}
