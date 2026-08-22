# Networking

Reformat IP and MAC addresses, decode chunked HTTP, strip protocol headers,
encode/decode NetBIOS names, encode VarInts, and defang/fang indicators of
compromise. Fingerprint TLS and SSH clients and servers (JA3/JA3S, JA4/JA4S,
HASSH), make live HTTP and DNS-over-HTTPS requests, and encode/decode Protobuf.

> Operations are listed alphabetically. Two of them are documented under
> [Data format](data-format.md), which is where they are chiefly used:
> [URL Decode](data-format.md#url-decode) and
> [URL Encode](data-format.md#url-encode).
>
> Every string flag has a `--<flag>-file` companion that reads the value from a
> file, keeping keys and passphrases out of shell history — see
> [Reading argument values from files](README.md#reading-argument-values-from-files).

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Change IP format | `change-ip-format` | [IPv4](https://wikipedia.org/wiki/IPv4) |
| Dechunk HTTP response | `dechunk-http-response` | [Chunked transfer encoding](https://wikipedia.org/wiki/Chunked_transfer_encoding) |
| Decode NetBIOS Name | `decode-netbios-name` | [NetBIOS](https://wikipedia.org/wiki/NetBIOS) |
| Defang IP Addresses | `defang-ip-addresses` | [Defanging](https://isc.sans.edu/forums/diary/Defang+all+the+things/22744/) |
| Defang URL | `defang-url` | [Defanging](https://isc.sans.edu/forums/diary/Defang+all+the+things/22744/) |
| DNS over HTTPS | `dns-over-https` | [DNS over HTTPS](https://wikipedia.org/wiki/DNS_over_HTTPS) |
| Encode NetBIOS Name | `encode-netbios-name` | [NetBIOS](https://wikipedia.org/wiki/NetBIOS) |
| Fang URL | `fang-url` | [Defanging](https://isc.sans.edu/forums/diary/Defang+all+the+things/22744/) |
| Format MAC addresses | `format-mac-addresses` | [MAC address](https://wikipedia.org/wiki/MAC_address) |
| Group IP addresses | `group-ip-addresses` | [Subnetwork](https://wikipedia.org/wiki/Subnetwork) |
| HASSH Client Fingerprint | `hassh-client-fingerprint` | [HASSH](https://engineering.salesforce.com/open-sourcing-hassh-abed3ae5044c) |
| HASSH Server Fingerprint | `hassh-server-fingerprint` | [HASSH](https://engineering.salesforce.com/open-sourcing-hassh-abed3ae5044c) |
| HTTP request | `http-request` | [HTTP request fields](https://wikipedia.org/wiki/List_of_HTTP_header_fields#Request_fields) |
| IPv6 Transition Addresses | `ipv6-transition-addresses` | [IPv6 transition mechanism](https://wikipedia.org/wiki/IPv6_transition_mechanism) |
| JA3 Fingerprint | `ja3-fingerprint` | [JA3](https://engineering.salesforce.com/tls-fingerprinting-with-ja3-and-ja3s-247362855967) |
| JA3S Fingerprint | `ja3s-fingerprint` | [JA3S](https://engineering.salesforce.com/tls-fingerprinting-with-ja3-and-ja3s-247362855967) |
| JA4 Fingerprint | `ja4-fingerprint` | [JA4](https://medium.com/foxio/ja4-network-fingerprinting-9376fe9ca637) |
| JA4Server Fingerprint | `ja4server-fingerprint` | [JA4](https://medium.com/foxio/ja4-network-fingerprinting-9376fe9ca637) |
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
| Protobuf Decode | `protobuf-decode` | [Protocol Buffers](https://wikipedia.org/wiki/Protocol_Buffers) |
| Protobuf Encode | `protobuf-encode` | [Protocol Buffers](https://wikipedia.org/wiki/Protocol_Buffers) |
| Strip HTTP headers | `strip-http-headers` | [HTTP headers](https://wikipedia.org/wiki/List_of_HTTP_header_fields) |
| Strip IPv4 header | `strip-ipv4-header` | [IPv4](https://wikipedia.org/wiki/IPv4) |
| Strip TCP header | `strip-tcp-header` | [TCP](https://wikipedia.org/wiki/Transmission_Control_Protocol) |
| Strip UDP header | `strip-udp-header` | [UDP](https://wikipedia.org/wiki/User_Datagram_Protocol) |
| URL Decode | `url-decode` | [Data format](data-format.md#url-decode) |
| URL Encode | `url-encode` | [Data format](data-format.md#url-encode) |
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
cchef change-ip-format -i '192.168.1.1' --input-format 'Dotted Decimal' --output-format Hex
```

Output:

```
c0a80101
```

---

## Dechunk HTTP response

Reassembles the body of an HTTP response sent with `Transfer-Encoding: chunked`,
discarding chunk sizes and trailing headers. Both `\n` and `\r\n` line endings
are handled.

**Simple example**

```bash
printf '7\r\nMozilla\r\n0\r\n\r\n' | cchef dechunk-http-response
```

Output:

```
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
cchef decode-netbios-name -i 'FEGIGFCAEOGFHEECEJEPFDCAGOGBGNGF'
```

Output:

```
The NetBIOS name
```

---

## Defang IP Addresses

Makes IPv4 and IPv6 addresses safe to share by wrapping their separators, so they
are no longer clickable / auto-linked.

**Simple example**

```bash
cchef defang-ip-addresses -i '192.168.1.1'
```

Output:

```
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
cchef defang-url -i 'Visit http://evil.example.com/x'
```

Output:

```
Visit hxxp[://]evil[.]example[.]com/x
```

---

## DNS over HTTPS

Resolves a domain name by sending a [DNS-over-HTTPS](https://wikipedia.org/wiki/DNS_over_HTTPS)
query (RFC 8484 JSON) to a resolver and returning the response. **Makes a live
network request**, so the output varies with current DNS records.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--resolver` | editableOption | `https://dns.google.com/resolve` | DoH resolver endpoint (e.g. Cloudflare `https://cloudflare-dns.com/dns-query`). |
| `--request-type` | option | `A` | Record type: `A`, `AAAA`, `CNAME`, `MX`, `TXT`, `NS`, … `ANY`. |
| `--answer-data-only` | boolean | `false` | Return only the answer record data (as a JSON array) instead of the full response. |
| `--disable-dnssec-validation` | boolean | `false` | Set the `cd` (checking disabled) flag to skip DNSSEC validation. |

**Simple example**

```bash
cchef dns-over-https -i 'example.com' --answer-data-only
```

Output:

```
["104.20.23.154","172.66.147.243"]
```

> Output is fetched live from the resolver and will differ from run to run.

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
cchef encode-netbios-name -i 'The NetBIOS name'
```

Output:

```
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
cchef fang-url -i 'hxxp[://]evil[.]example[.]com/x'
```

Output:

```
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
cchef format-mac-addresses -i '00:11:22:33:44:55' --output-case 'Lower only'
```

Output:

```
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
printf '192.168.1.5\n192.168.1.200\n10.0.0.1' | cchef group-ip-addresses
```

Output:

```
10.0.0.0/24
  10.0.0.1

192.168.1.0/24
  192.168.1.5
  192.168.1.200
```

---

## HASSH Client Fingerprint

Generates a [HASSH](https://engineering.salesforce.com/open-sourcing-hassh-abed3ae5044c)
fingerprint from an SSH client's `SSH_MSG_KEXINIT` packet — an MD5 of the client's
key-exchange, encryption, MAC and compression algorithm lists (client-to-server),
joined with `;`.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `Hex` | Input encoding: `Hex`, `Base64` or `Raw`. |
| `--output-format` | option | `Hash digest` | `Hash digest`, `HASSH algorithms string`, or `Full details`. |

**Simple example**

```bash
cchef hassh-client-fingerprint -i '0000018404140000000000000000000000000000000000000042637572766532353531392d7368613235362c656364682d736861322d6e697374703235362c6469666669652d68656c6c6d616e2d67726f757031342d736861323536000000257373682d656432353531392c7273612d736861322d3531322c7273612d736861322d3235360000003363686163686132302d706f6c7931333035406f70656e7373682e636f6d2c6165733132382d6374722c6165733235362d6374720000003363686163686132302d706f6c7931333035406f70656e7373682e636f6d2c6165733132382d6374722c6165733235362d63747200000025756d61632d36342d65746d406f70656e7373682e636f6d2c686d61632d736861322d32353600000025756d61632d36342d65746d406f70656e7373682e636f6d2c686d61632d736861322d323536000000156e6f6e652c7a6c6962406f70656e7373682e636f6d000000156e6f6e652c7a6c6962406f70656e7373682e636f6d0000000000000000000000000000000000'
```

Output:

```
6559ab006495e3044da5a8821704047e
```

**Complex example** — the same packet with `--output-format 'HASSH algorithms
string'` shows the four component lists (kex;encryption;mac;compression) that are
hashed:

```
curve25519-sha256,ecdh-sha2-nistp256,diffie-hellman-group14-sha256;chacha20-poly1305@openssh.com,aes128-ctr,aes256-ctr;umac-64-etm@openssh.com,hmac-sha2-256;none,zlib@openssh.com
```

---

## HASSH Server Fingerprint

Generates a [HASSH](https://engineering.salesforce.com/open-sourcing-hassh-abed3ae5044c)
fingerprint from an SSH server's `SSH_MSG_KEXINIT` packet — the server-to-server
(server-to-client) counterpart of HASSH Client Fingerprint.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `Hex` | Input encoding: `Hex`, `Base64` or `Raw`. |
| `--output-format` | option | `Hash digest` | `Hash digest`, `HASSH algorithms string`, or `Full details`. |

**Simple example**

```bash
cchef hassh-server-fingerprint -i '0000012206141111111111111111111111111111111100000024637572766532353531392d7368613235362c656364682d736861322d6e69737470323536000000187373682d656432353531392c7273612d736861322d353132000000156165733132382d6374722c6165733235362d6374720000003463686163686132302d706f6c7931333035406f70656e7373682e636f6d2c6165733132382d67636d406f70656e7373682e636f6d0000000d686d61632d736861322d32353600000032686d61632d736861322d3531322d65746d406f70656e7373682e636f6d2c756d61632d313238406f70656e7373682e636f6d000000046e6f6e65000000156e6f6e652c7a6c6962406f70656e7373682e636f6d00000000000000000000000000000000000000'
```

Output:

```
95dc0a7fbfb0eb627394c0e4240dc213
```

---

## HTTP request

Makes an HTTP request to a URL and returns the response body (optionally prefixed
with the status and exposed response headers). **Makes a live network request.**

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--method` | option | `GET` | HTTP method: `GET`, `POST`, `HEAD`, `PUT`, `PATCH`, `DELETE`, `CONNECT`, `TRACE`, `OPTIONS`. |
| `--url` | string | | Request URL. |
| `--headers` | string | | Extra request headers, one `Name: value` per line. |
| `--mode` | option | `Cross-Origin Resource Sharing` | Present for CyberChef parity; has no effect on the CLI. |
| `--show-response-metadata` | boolean | `false` | Prepend the response status and exposed headers to the body. |

The request body is read from stdin (ignored for `GET`/`HEAD`).

**Simple example**

```bash
cchef http-request -i '' --url 'https://example.com'
```

Output:

```
<!doctype html><html lang="en"><head><title>Example Domain</title>...
```

> Output is fetched live and depends on the server's current response.

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
cchef ipv6-transition-addresses -i '198.51.100.7'
```

Output:

```
6to4: 2002:c633:6407::/48
IPv4 Mapped: ::ffff:c633:6407
IPv4 Translated: ::ffff:0:c633:6407
Nat 64: 64:ff9b::c633:6407
```

---

## JA3 Fingerprint

Generates a [JA3](https://engineering.salesforce.com/tls-fingerprinting-with-ja3-and-ja3s-247362855967)
fingerprint from a TLS `ClientHello` — an MD5 of the TLS version, cipher suites,
extensions, elliptic curves and curve point formats (with GREASE values removed).

**Fidelity.** A record that runs out mid-field is refused rather than crashing.
CyberChef skips the record version, the client random and the session ID by
moving the stream, which throws past the end of the buffer — and unlike the
length checks around it, that throw is not an operation error, so a truncated
record takes CyberChef down instead of reporting anything. Here it is an
ordinary error carrying CyberChef's own message, `Cannot move to position 3 in
stream. Out of bounds.`

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `Hex` | Input encoding: `Hex`, `Base64` or `Raw`. |
| `--output-format` | option | `Hash digest` | `Hash digest`, `JA3 string`, or `Full details`. |

**Simple example**

```bash
cchef ja3-fingerprint -i '16030100a4010000a00301543dd2dd48f517ca9a93b1e599f019fdece704a23e86c1dcac588427abbaddf200005cc014c00a0039003800880087c00fc00500350084c012c00800160013c00dc003000ac013c00900330032009a009900450044c00ec004002f009600410007c011c007c00cc002000500040015001200090014001100080006000300ff0100001b000b000403000102000a000600040018001700230000000f000101'
```

Output:

```
503053a0c5b2bd9b9334bf7f3d3b8852
```

**Complex example** — the same `ClientHello` with `--output-format 'JA3 string'`
shows the underlying `version,ciphers,extensions,curves,pointFormats` string that
is hashed:

```
769,49172-49162-57-56-136-135-49167-49157-53-132-49170-49160-22-19-49165-49155-10-49171-49161-51-50-154-153-69-68-49166-49156-47-150-65-7-49169-49159-49164-49154-5-4-21-18-9-20-17-8-6-3-255,11-10-35-15,24-23,0-1-2
```

---

## JA3S Fingerprint

Generates a [JA3S](https://engineering.salesforce.com/tls-fingerprinting-with-ja3-and-ja3s-247362855967)
fingerprint from a TLS `ServerHello` — an MD5 of the TLS version, the single
selected cipher suite, and the extensions (server extensions are *not* GREASE-filtered).

**Fidelity.** A record that runs out mid-field is refused rather than crashing.
CyberChef skips the record version, the client random and the session ID by
moving the stream, which throws past the end of the buffer — and unlike the
length checks around it, that throw is not an operation error, so a truncated
record takes CyberChef down instead of reporting anything. Here it is an
ordinary error carrying CyberChef's own message, `Cannot move to position 3 in
stream. Out of bounds.`

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `Hex` | Input encoding: `Hex`, `Base64` or `Raw`. |
| `--output-format` | option | `Hash digest` | `Hash digest`, `JA3S string`, or `Full details`. |

**Simple example**

```bash
cchef ja3s-fingerprint -i '160301003d020000390301543dd2ddedbfe33895bd6bc676a3fa6b9fe5773a6e04d5476d1af3bcbc1dcbbb00c011000011ff01000100000b00040300010200230000'
```

Output:

```
bed95e1b525d2f41db3a6d68fac5b566
```

---

## JA4 Fingerprint

Generates a [JA4](https://medium.com/foxio/ja4-network-fingerprinting-9376fe9ca637)
fingerprint from a TLS `ClientHello`. JA4 is a structured, human-readable
fingerprint (`t13d1516h2_…`) built from the TLS version, SNI presence, cipher and
extension counts, ALPN, and truncated SHA-256 hashes of the sorted cipher and
extension lists.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `Hex` | Input encoding: `Hex`, `Base64` or `Raw`. |
| `--output-format` | option | `JA4` | `JA4`, `JA4 Original Rendering`, `JA4 Raw`, `JA4 Raw Original Rendering`, or `All`. |

**Simple example**

```bash
cchef ja4-fingerprint -i '1603010200010001fc0303b2c03e7ba990ef540c316a665d4d925f8e9079ac4b15687e587dc99016e75a6c20d0b0099243c9296a0c84153ea4ada7d87ad017f4211c2ea1350b0b3cc5514d5f00205a5a130113021303c02bc02fc02cc030cca9cca8c013c014009c009d002f003501000193fafa000000000024002200001f636f6e74656e742d6175746f66696c6c2e676f6f676c65617069732e636f6d0033002b00293a3a000100001d0020fb2cd8ef3d605b96ab03119ec4f30a6e2088cb1af86c41a81feace8706068c50000d001200100403080404010503080505010806060100230000000b00020100ff01000100000a000a00083a3a001d00170018001b000302000244690005000302683200120000002d000201010010000e000c02683208687474702f312e31000500050100000000002b0007060a0a03040303001700001a1a000100001500b800000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000'
```

Output:

```
t13d1516h2_8daaf6152771_e5627efa2ab1
```

**Complex example** — `--output-format All` emits every rendering, including the
raw (un-hashed) cipher and extension lists:

```
JA4:    t13d1516h2_8daaf6152771_e5627efa2ab1
JA4_o:  t13d1516h2_acb858a92679_5276cb03a33b
JA4_r:  t13d1516h2_002f,0035,009c,009d,1301,1302,1303,c013,c014,c02b,c02c,c02f,c030,cca8,cca9_0005,000a,000b,000d,0012,0015,0017,001b,0023,002b,002d,0033,4469,ff01_0403,0804,0401,0503,0805,0501,0806,0601
JA4_ro: t13d1516h2_1301,1302,1303,c02b,c02f,c02c,c030,cca9,cca8,c013,c014,009c,009d,002f,0035_0000,0033,000d,0023,000b,ff01,000a,001b,4469,0012,002d,0010,0005,002b,0017,0015_0403,0804,0401,0503,0805,0501,0806,0601
```

---

## JA4Server Fingerprint

Generates a [JA4S](https://medium.com/foxio/ja4-network-fingerprinting-9376fe9ca637)
fingerprint from a TLS `ServerHello` — the server-side counterpart of JA4
(`t1204h2_…`), built from the TLS version, extension count, ALPN, selected cipher,
and a truncated SHA-256 hash of the server extensions.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format` | option | `Hex` | Input encoding: `Hex`, `Base64` or `Raw`. |
| `--output-format` | option | `JA4S` | `JA4S`, `JA4S Raw`, or `Both`. |

**Simple example**

```bash
cchef ja4server-fingerprint -i '16030300640200006003035f0236c07f47bfb12dc2da706ecb3fe7f9eeac9968cc2ddf444f574e4752440120b89ff1ab695278c69b8a73f76242ef755e0b13dc6d459aaaa784fec9c2dfce34cca900001800000000ff01000100000b00020100001000050003026832'
```

Output:

```
t1204h2_cca9_1428ce7b4018
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
cchef parse-ethernet-frame -i '000000000000ffffffffffff08004500' --input-type Hex
```

Output:

```
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
cchef parse-ip-range -i '10.0.0.0/30'
```

Output:

```
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
cchef parse-ipv4-header -i '450000140005400080060000c0a80001c0a80002cafe' --output-format 'Data (hex)'
```

Output:

```
ca fe
```

---

## Parse IPv6 address

Displays the longhand and shorthand forms of an IPv6 address, its type (loopback,
multicast, 6to4, Teredo, unique-local, …), and any embedded IPv4 or MAC address.

**Simple example**

```bash
cchef parse-ipv6-address -i '::1'
```

Output:

```
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
cchef parse-ssh-host-key -i 'AAAAC3NzaC1lZDI1NTE5AAAAIBOF6r99IkvqGu1kwZrHHIqjpTB5w79bpv67B/Aw3+WJ'
```

Output:

```
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
cchef parse-tcp -i 'c2eb0050a138132e70dc9fb9501804025ea70000'
```

Output:

```
{"Source port":49899,"Destination port":80,"Sequence number":"2704806702","Acknowledgement number":1893507001,"Data offset":"5 (20 bytes)","Flags":{"Reserved":"000","NS":0,"CWR":0,"ECE":0,"URG":0,"ACK":1,"PSH":1,"RST":0,"SYN":0,"FIN":0},"Window size":"1026 (Scaled: 1026)","Checksum":"0x5ea7","Urgent pointer":"0x0000"}
```

---

## Parse TLS record

Parses one or more raw TLS records (change-cipher-spec, alert, handshake,
application-data) into a JSON array. Handshake messages (ClientHello, ServerHello,
Certificate, …) are parsed in detail.

**Simple example**

```bash
printf '140303000101' | cchef from-hex --delimiter None | cchef parse-tls-record
```

Output:

```
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
cchef parse-udp -i '04 89 00 35 00 2c 01 01'
```

Output:

```
{"Source port":1161,"Destination port":53,"Length":44,"Checksum":"0x0101"}
```

---

## Parse URI

Pretty-prints the components of a URI (protocol, auth, host, port, path, query
arguments, fragment).

**Simple example**

```bash
cchef parse-uri -i 'https://user:pass@example.com:8080/p?q=1&r=2#frag'
```

Output:

```
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

Identifies the browser, device, engine, OS and CPU from a User-Agent string. The
detection rules are cchef's own, derived from the structure of real user-agent
strings and verified to give the same answers as CyberChef across a broad corpus
of real-world agents; very obscure agents may classify differently.

**Simple example**

```bash
cchef parse-user-agent -i 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36'
```

Output:

```
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

## Protobuf Decode

Decodes [Protocol Buffers](https://wikipedia.org/wiki/Protocol_Buffers) wire data
into JSON. Without a schema the raw wire structure is decoded with field-number
keys; with a `.proto` schema the fields are named and typed (following protobufjs
`toObject` conventions — repeated/map fields first, defaults included, bytes as
base64, enums as names).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--schema-proto-text` | string | | Optional `.proto` schema text. If empty, the raw wire structure is decoded. |
| `--show-unknown-fields` | boolean | `false` | With a schema, also report wire fields absent from the schema under `Unknown Fields`. |
| `--show-types` | boolean | `false` | Annotate each field with its wire type (raw) or `.proto` type (schema). |

**Simple example** — raw decode (no schema):

```bash
printf '089601120774657374696e671a02082a1205616761696e' | cchef from-hex --delimiter None | cchef protobuf-decode
```

Output:

```
{"1":150,"2":["testing","again"],"3":{"1":42}}
```

**Complex example** — with a schema the fields are named:

```bash
printf '0896011202686918011802' | cchef from-hex --delimiter None | cchef protobuf-decode --schema-proto-text 'syntax="proto3"; message Test { int32 a=1; string b=2; repeated int32 c=3; }'
```

Output:

```
{"c":[1,2],"a":150,"b":"hi"}
```

---

## Protobuf Encode

Encodes a JSON object into [Protocol Buffers](https://wikipedia.org/wiki/Protocol_Buffers)
wire data using a `.proto` schema. Input keys are matched to schema field names;
unknown keys are ignored and values are coerced protobufjs-style (numeric strings,
`bool`↔number, etc.). Repeated scalars are emitted unpacked, matching protobufjs.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--schema-proto-text` | string | | `.proto` schema text (required; the first message defined is the root). |

**Simple example**

```bash
printf '{"a":150,"b":"hi","c":[1,2]}' | cchef protobuf-encode --schema-proto-text 'syntax="proto3"; message Test { int32 a=1; string b=2; repeated int32 c=3; }' | cchef to-hex --delimiter None
```

Output:

```
0896011202686918011802
```

---

## Strip HTTP headers

Removes the header block from an HTTP request or response by cutting everything up
to and including the first blank line (`\r\n\r\n` or `\n\n`).

**Simple example**

```bash
printf 'HTTP/1.1 200 OK\r\nServer: x\r\n\r\n<html>' | cchef strip-http-headers
```

Output:

```
<html>
```

---

## Strip IPv4 header

Removes the IPv4 header (using its IHL field) from a packet, leaving the payload.
Input is raw bytes, so pipe packet data in via `from-hex`.

**Simple example**

```bash
printf '450000140005400080060000c0a80001c0a80002cafe' \
    | cchef from-hex --delimiter None | cchef strip-ipv4-header | cchef to-hex --delimiter None
```

Output:

```
cafe
```

---

## Strip TCP header

Removes the TCP header (using its data-offset field) from a segment, leaving the
payload.

**Simple example**

```bash
printf '7f900050000fa4b2000cb2a45010bff100000000cafe' \
    | cchef from-hex --delimiter None | cchef strip-tcp-header | cchef to-hex --delimiter None
```

Output:

```
cafe
```

---

## Strip UDP header

Removes the fixed 8-byte UDP header from a datagram, leaving the payload.

**Simple example**

```bash
printf '8111003500080000cafe' \
    | cchef from-hex --delimiter None | cchef strip-udp-header | cchef to-hex --delimiter None
```

Output:

```
cafe
```

---

## VarInt Decode

Decodes a Protobuf-style LEB128 variable-length integer (raw bytes in) to its
decimal value. Arbitrary precision is supported.

**Simple example**

```bash
printf 'ac02' | cchef from-hex --delimiter None | cchef varint-decode
```

Output:

```
300
```

---

## VarInt Encode

Encodes a non-negative decimal integer as an LEB128 variable-length integer.

**Simple example**

```bash
printf '300' | cchef varint-encode | cchef to-hex --delimiter None
```

Output:

```
ac02
```
