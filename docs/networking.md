# Networking

Reformat IP and MAC addresses, decode chunked HTTP, strip protocol headers,
encode/decode NetBIOS names, encode VarInts, and defang/fang indicators of
compromise.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Change IP format | `change-ip-format` | [IPv4](https://wikipedia.org/wiki/IPv4) |
| Dechunk HTTP response | `dechunk-http-response` | [Chunked transfer encoding](https://wikipedia.org/wiki/Chunked_transfer_encoding) |
| Decode NetBIOS Name | `decode-netbios-name` | [NetBIOS](https://wikipedia.org/wiki/NetBIOS) |
| Defang IP Addresses | `defang-ip-addresses` | [Defanging](https://isc.sans.edu/forums/diary/Defang+all+the+things/22744/) |
| Defang URL | `defang-url` | [Defanging](https://isc.sans.edu/forums/diary/Defang+all+the+things/22744/) |
| Encode NetBIOS Name | `encode-netbios-name` | [NetBIOS](https://wikipedia.org/wiki/NetBIOS) |
| Fang URL | `fang-url` | [Defanging](https://isc.sans.edu/forums/diary/Defang+all+the+things/22744/) |
| Format MAC addresses | `format-mac-addresses` | [MAC address](https://wikipedia.org/wiki/MAC_address) |
| Group IP addresses | `group-ip-addresses` | [Subnetwork](https://wikipedia.org/wiki/Subnetwork) |
| IPv6 Transition Addresses | `ipv6-transition-addresses` | [IPv6 transition mechanism](https://wikipedia.org/wiki/IPv6_transition_mechanism) |
| Parse Ethernet frame | `parse-ethernet-frame` | [Ethernet frame](https://wikipedia.org/wiki/Ethernet_frame) |
| Parse IP range | `parse-ip-range` | [Subnetwork](https://wikipedia.org/wiki/Subnetwork) |
| Parse IPv4 header | `parse-ipv4-header` | [IPv4 header](https://wikipedia.org/wiki/IPv4#Header) |
| Parse IPv6 address | `parse-ipv6-address` | [IPv6 address](https://wikipedia.org/wiki/IPv6_address) |
| Parse SSH Host Key | `parse-ssh-host-key` | [Secure Shell](https://wikipedia.org/wiki/Secure_Shell) |
| Parse TCP | `parse-tcp` | [TCP](https://wikipedia.org/wiki/Transmission_Control_Protocol) |
| Parse TLS record | `parse-tls-record` | [TLS](https://wikipedia.org/wiki/Transport_Layer_Security) |
| Parse UDP | `parse-udp` | [UDP](https://wikipedia.org/wiki/User_Datagram_Protocol) |
| Parse URI | `parse-uri` | [URI](https://wikipedia.org/wiki/Uniform_Resource_Identifier) |
| Parse User Agent | `parse-user-agent` | [User agent](https://wikipedia.org/wiki/User_agent) |
| Strip HTTP headers | `strip-http-headers` | [HTTP headers](https://wikipedia.org/wiki/List_of_HTTP_header_fields) |
| Strip IPv4 header | `strip-ipv4-header` | [IPv4](https://wikipedia.org/wiki/IPv4) |
| Strip TCP header | `strip-tcp-header` | [TCP](https://wikipedia.org/wiki/Transmission_Control_Protocol) |
| Strip UDP header | `strip-udp-header` | [UDP](https://wikipedia.org/wiki/User_Datagram_Protocol) |
| VarInt Decode | `varint-decode` | [Varints](https://developers.google.com/protocol-buffers/docs/encoding#varints) |
| VarInt Encode | `varint-encode` | [Varints](https://developers.google.com/protocol-buffers/docs/encoding#varints) |

---

## Change IP format

Converts an IPv4 address between dotted-decimal, decimal, octal and hex. Multiple
addresses (one per line) are converted independently.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `Dotted Decimal` | Input format: `Dotted Decimal`, `Decimal`, `Octal`, `Hex`. |
| `--output-format` | option | `Dotted Decimal` | Output format (same choices). |

**Simple example**

```bash
$ printf '192.168.1.1' | cchef change-ip-format --input-format 'Dotted Decimal' --output-format Hex
c0a80101
```

---

## Dechunk HTTP response

Reassembles the body of an HTTP response sent with `Transfer-Encoding: chunked`,
discarding chunk sizes and trailing headers. Both `\n` and `\r\n` line endings
are handled.

**Simple example**

```bash
$ printf '7\r\nMozilla\r\n0\r\n\r\n' | cchef dechunk-http-response
Mozilla
```

---

## Decode NetBIOS Name

Reverses the first-level NetBIOS name encoding (see RFC 1001), mapping each pair
of characters back to a byte using the offset.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--offset` | number | `65` | Character offset used during encoding (`A` = 65). |

**Simple example**

```bash
$ printf 'FEGIGFCAEOGFHEECEJEPFDCAGOGBGNGF' | cchef decode-netbios-name
The NetBIOS name
```

---

## Defang IP Addresses

Makes IPv4 and IPv6 addresses safe to share by wrapping their separators, so they
are no longer clickable / auto-linked.

**Simple example**

```bash
$ printf '192.168.1.1' | cchef defang-ip-addresses
192[.]168[.]1[.]1
```

---

## Defang URL

Neutralises URLs (and, by default, bare domains) so they cannot be accidentally
clicked. Escaping of dots, the scheme and `://` are individually toggleable.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--escape-dots` | bool | `true` | Replace `.` with `[.]`. |
| `--escape-http` | bool | `true` | Replace `http` with `hxxp`. |
| `--escape` | bool | `true` | Replace `://` with `[://]` (the "Escape ://" option). |
| `--process` | option | `Valid domains and full URLs` | What to match: `Valid domains and full URLs`, `Only full URLs`, `Everything`. |

**Simple example**

```bash
$ printf 'Visit http://evil.example.com/x' | cchef defang-url
Visit hxxp[://]evil[.]example[.]com/x
```

---

## Encode NetBIOS Name

Applies the first-level NetBIOS name encoding (RFC 1001): the name is padded to
16 bytes with spaces and each byte split into two characters offset by 65.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--offset` | number | `65` | Character offset to add (`A` = 65). |

**Simple example**

```bash
$ printf 'The NetBIOS name' | cchef encode-netbios-name
FEGIGFCAEOGFHEECEJEPFDCAGOGBGNGF
```

---

## Fang URL

Reverses Defang URL, restoring `[.]` → `.`, `hxxp` → `http` and `[://]` → `://`.
Each restoration is individually toggleable.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--restore` | bool | `true` | Restore `[.]` → `.` (the "Restore [.]" option). |
| `--restore-hxxp` | bool | `true` | Restore `hxxp` → `http`. |
| `--restore-2` | bool | `true` | Restore `[://]` → `://` (the "Restore ://" option). |

**Simple example**

```bash
$ printf 'hxxp[://]evil[.]example[.]com/x' | cchef fang-url
http://evil.example.com/x
```

---

## Format MAC addresses

Reformats MAC addresses into several delimiter styles. Input may contain multiple
addresses separated by newlines, spaces or commas. There are no validity checks.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--output-case` | option | `Both` | `Both`, `Upper only` or `Lower only`. |
| `--no-delimiter` | bool | `true` | Emit `0011...` (no delimiter). |
| `--dash-delimiter` | bool | `true` | Emit `00-11-...`. |
| `--colon-delimiter` | bool | `true` | Emit `00:11:...`. |
| `--cisco-style` | bool | `false` | Emit `0011.2233...`. |
| `--ipv6-interface-id` | bool | `false` | Emit the EUI-64 IPv6 interface ID. |

**Simple example**

```bash
$ printf '00:11:22:33:44:55' | cchef format-mac-addresses --output-case 'Lower only'
001122334455
00-11-22-33-44-55
00:11:22:33:44:55
```

---

## Group IP addresses

Groups a list of IPv4 and/or IPv6 addresses into subnets. IPv4 subnets are printed
in ascending order; IPv6 subnets in first-seen order (matching CyberChef).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Line feed` | Input delimiter: `Line feed`, `CRLF`, `Space`, `Comma`, `Semi-colon`. |
| `--subnet-cidr` | number | `24` | CIDR prefix length (< 32 for IPv4, < 128 for IPv6). |
| `--only-show-the-subnets` | bool | `false` | Print only the subnet lines, not their members. |

**Simple example**

```bash
$ printf '192.168.1.5\n192.168.1.200\n10.0.0.1' | cchef group-ip-addresses
10.0.0.0/24
  10.0.0.1

192.168.1.0/24
  192.168.1.5
  192.168.1.200
```

---

## IPv6 Transition Addresses

Converts an IPv4 address (or /24 range) into its 6to4, mapped, translated and
NAT64 IPv6 transition addresses, converts those back to IPv4, and converts a MAC
address into an EUI-64 interface ID.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--ignore-ranges` | bool | `true` | Skip input lines containing a `/` (ranges). |
| `--remove-headers` | bool | `false` | Omit the descriptive labels from each line. |

**Simple example**

```bash
$ printf '198.51.100.7' | cchef ipv6-transition-addresses
6to4: 2002:c633:6407::/48
IPv4 Mapped: ::ffff:c633:6407
IPv4 Translated: ::ffff:0:c633:6407
Nat 64: 64:ff9b::c633:6407
```

---

## Parse Ethernet frame

Parses an Ethernet II frame — source/destination MAC, any VLAN tags, and the
payload.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-type` | option | `Raw` | Input encoding: `Raw` or `Hex`. |
| `--return-type` | option | `Text output` | `Text output`, `Packet data`, or `Packet data (hex)`. |

**Simple example**

```bash
$ printf '000000000000ffffffffffff08004500' | cchef parse-ethernet-frame --input-type Hex
Source MAC: ff:ff:ff:ff:ff:ff
Destination MAC: 00:00:00:00:00:00
Data:
45 00
```

---

## Parse IP range

Given a CIDR range, a hyphenated range, or a list of IPs/CIDRs (IPv4 or IPv6),
prints network information and enumerates the addresses (IPv4 only).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--include-network-info` | bool | `true` | Print network/mask/range summary. |
| `--enumerate-ip-addresses` | bool | `true` | List each address (IPv4). |
| `--allow-large-queries` | bool | `false` | Permit enumerating large ranges. |

**Simple example**

```bash
$ printf '10.0.0.0/30' | cchef parse-ip-range
Network: 10.0.0.0
CIDR: 30
Mask: 255.255.255.252
Range: 10.0.0.0 - 10.0.0.3
Total addresses in range: 4

10.0.0.0
10.0.0.1
10.0.0.2
10.0.0.3
```

---

## Parse IPv4 header

Parses an IPv4 header, either as a field table or by extracting the payload.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `Hex` | Input encoding: `Hex` or `Raw`. |
| `--output-format` | option | `Table` | `Table` (HTML), `Data (hex)`, or `Data (raw)`. |

**Simple example**

```bash
$ printf '450000140005400080060000c0a80001c0a80002cafe' | cchef parse-ipv4-header --output-format 'Data (hex)'
ca fe
```

---

## Parse IPv6 address

Displays the longhand and shorthand forms of an IPv6 address, its type (loopback,
multicast, 6to4, Teredo, unique-local, …), and any embedded IPv4 or MAC address.

**Simple example**

```bash
$ printf '::1' | cchef parse-ipv6-address
Longhand:  0000:0000:0000:0000:0000:0000:0000:0001
Shorthand: ::1

Loopback address to the local host corresponding to 127.0.0.1/8 in IPv4.
Loopback addresses range: ::1/128
```

---

## Parse SSH Host Key

Extracts the key type and parameters from an SSH host key (RSA, DSA, ECDSA,
Ed25519).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `Auto` | `Auto`, `Base64`, or `Hex`. |

**Simple example**

```bash
$ printf 'AAAAC3NzaC1lZDI1NTE5AAAAIBOF6r99IkvqGu1kwZrHHIqjpTB5w79bpv67B/Aw3+WJ' | cchef parse-ssh-host-key
Key type: ssh-ed25519
x: 0x1385eabf7d224bea1aed64c19ac71c8aa3a53079c3bf5ba6febb07f030dfe589
```

---

## Parse TCP

Parses a TCP header (and options) into JSON.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `Hex` | Input encoding: `Hex` or `Raw`. |

**Simple example**

```bash
$ printf 'c2eb0050a138132e70dc9fb9501804025ea70000' | cchef parse-tcp
{"Source port":49899,"Destination port":80,"Sequence number":"2704806702","Acknowledgement number":1893507001,"Data offset":"5 (20 bytes)","Flags":{"Reserved":"000","NS":0,"CWR":0,"ECE":0,"URG":0,"ACK":1,"PSH":1,"RST":0,"SYN":0,"FIN":0},"Window size":"1026 (Scaled: 1026)","Checksum":"0x5ea7","Urgent pointer":"0x0000"}
```

---

## Parse TLS record

Parses one or more raw TLS records (change-cipher-spec, alert, handshake,
application-data) into a JSON array. Handshake messages (ClientHello, ServerHello,
Certificate, …) are parsed in detail.

**Simple example**

```bash
$ printf '140303000101' | cchef from-hex --delimiter None | cchef parse-tls-record
[{"type":"change_cipher_spec","version":"0x0303","length":1,"value":"0x01"}]
```

---

## Parse UDP

Parses a UDP header (and payload, if present) into JSON.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `Hex` | Input encoding: `Hex` or `Raw`. |

**Simple example**

```bash
$ printf '04 89 00 35 00 2c 01 01' | cchef parse-udp
{"Source port":1161,"Destination port":53,"Length":44,"Checksum":"0x0101"}
```

---

## Parse URI

Pretty-prints the components of a URI (protocol, auth, host, port, path, query
arguments, fragment).

**Simple example**

```bash
$ printf 'https://user:pass@example.com:8080/p?q=1&r=2#frag' | cchef parse-uri
Protocol:	https:
Auth:		user:pass
Hostname:	example.com
Port:		8080
Path name:	/p
Arguments:
	q = 1
	r = 2
Hash:		#frag
```

> Uses Go's `net/url`, which approximates Node's `url.parse` (matched on the
> common cases; exotic/malformed URIs may differ).

---

## Parse User Agent

Identifies the browser, device, engine, OS and CPU from a User-Agent string. This
is a faithful port of `ua-parser-js` 2.0.10's default detection ruleset (the exact
version CyberChef uses) — the rule tables in `useragent_rules.go` are generated
from that library.

**Simple example**

```bash
$ printf 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36' | cchef parse-user-agent
Browser
    Name: Chrome
    Version: 120.0.0.0
Device
    Model: unknown
    Type: unknown
    Vendor: unknown
Engine
    Name: Blink
    Version: 120.0.0.0
OS
    Name: Windows
    Version: 10
CPU
    Architecture: amd64
```

---

## Strip HTTP headers

Removes the header block from an HTTP request or response by cutting everything up
to and including the first blank line (`\r\n\r\n` or `\n\n`).

**Simple example**

```bash
$ printf 'HTTP/1.1 200 OK\r\nServer: x\r\n\r\n<html>' | cchef strip-http-headers
<html>
```

---

## Strip IPv4 header

Removes the IPv4 header (using its IHL field) from a packet, leaving the payload.
Input is raw bytes, so pipe packet data in via `from-hex`.

**Simple example**

```bash
$ printf '450000140005400080060000c0a80001c0a80002cafe' \
    | cchef from-hex --delimiter None | cchef strip-ipv4-header | cchef to-hex --delimiter None
cafe
```

---

## Strip TCP header

Removes the TCP header (using its data-offset field) from a segment, leaving the
payload.

**Simple example**

```bash
$ printf '7f900050000fa4b2000cb2a45010bff100000000cafe' \
    | cchef from-hex --delimiter None | cchef strip-tcp-header | cchef to-hex --delimiter None
cafe
```

---

## Strip UDP header

Removes the fixed 8-byte UDP header from a datagram, leaving the payload.

**Simple example**

```bash
$ printf '8111003500080000cafe' \
    | cchef from-hex --delimiter None | cchef strip-udp-header | cchef to-hex --delimiter None
cafe
```

---

## VarInt Decode

Decodes a Protobuf-style LEB128 variable-length integer (raw bytes in) to its
decimal value. Arbitrary precision is supported.

**Simple example**

```bash
$ printf 'ac02' | cchef from-hex --delimiter None | cchef varint-decode
300
```

---

## VarInt Encode

Encodes a non-negative decimal integer as an LEB128 variable-length integer.

**Simple example**

```bash
$ printf '300' | cchef varint-encode | cchef to-hex --delimiter None
ac02
```
