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
