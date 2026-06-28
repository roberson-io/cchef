# cchef Documentation

`cchef` is a command-line port of [CyberChef](https://gchq.github.io/CyberChef/),
the "Cyber Swiss Army Knife". Every operation is a subcommand that reads input and
writes output, so operations chain together through Unix pipes or as a single
recipe.

> **Scope:** This is a curated starter set of 18 operations that exercises the
> whole architecture. More operations are added over time against the same
> interfaces. Run `cchef list` to see everything currently available.

## Installing / building

```bash
make build      # produces ./dist/cchef
```

## How input and output work

Each operation resolves its input from the first source available, in this order:

1. `--in-file <path>` — read input from a file
2. `-i, --input <string>` — input given directly on the command line
3. **positional argument** — `cchef rot13 "Have a nice day."`
4. **stdin** — `echo hello | cchef rot13` (this is what makes pipes work)

Output goes to stdout, or to a file with `-o, --output <path>`. When writing to a
terminal, a trailing newline is added for readability; when piped or redirected,
output stays byte-exact so operations chain cleanly.

```bash
# All four input styles are equivalent here:
cchef to-base64 -i hello
cchef to-base64 hello
echo -n hello | cchef to-base64
cchef to-base64 --in-file ./greeting.txt
```

## Chaining operations

Two ways to combine operations:

```bash
# 1. Unix pipes — one subcommand per operation
echo -n hello | cchef to-base64 | cchef to-hex

# 2. A recipe — multiple operations in one command (see Recipes & URLs)
echo -n hello | cchef bake -e "To_Base64()To_Hex()"
```

## Operation categories

Operations are grouped using the same categories as the original CyberChef tool.

Within each category, operations are listed alphabetically.

| Category | Operations |
| --- | --- |
| [Data format](data-format.md) | AMF Decode, AMF Encode, From Base, From Base32, From Base45, From Base58, From Base62, From Base64, From Base85, From Base92, From Binary, From Charcode, From Decimal, From Hex, From Octal, Swap endianness, To Base, To Base32, To Base45, To Base58, To Base62, To Base64, To Base85, To Base92, To Binary, To Charcode, To Decimal, To Hex, To Octal, URL Decode, URL Encode |
| [Encryption / Encoding](encryption-encoding.md) | ROT13, ROT47, XOR |
| [Hashing](hashing.md) | Adler-32 Checksum, HMAC, Keccak, MD5, SHA1, SHA224, SHA256, SHA3, SHA384, SHA512 |
| [Utils](utils.md) | Filter, Find / Replace, Pad lines, Remove null bytes, Remove whitespace, Reverse, Sort, Swap case, To Lower case, To Upper case, Unique |

## Recipes, URLs, and tooling

- [Recipes & URLs](recipes-and-urls.md) — `bake`, `url`, `recipe convert`, recipe
  formats (JSON and Chef), and generating CyberChef share links.

## Command reference

| Command | Purpose |
| --- | --- |
| `cchef <operation> [flags]` | Run a single operation |
| `cchef bake -e/-r <recipe>` | Run a multi-operation recipe (JSON or Chef format) |
| `cchef url -e/-r <recipe>` | Print a CyberChef share URL for a recipe |
| `cchef recipe convert` | Convert a recipe between JSON and Chef formats |
| `cchef list` | List available operations grouped by module |
