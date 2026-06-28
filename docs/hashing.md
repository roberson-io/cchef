# Hashing

Cryptographic hash functions. Each operation takes input and outputs the
lower-case hexadecimal digest. None of these operations take options.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| MD5 | `md5` | [MD5](https://wikipedia.org/wiki/MD5) |
| SHA1 | `sha1` | [SHA-1](https://wikipedia.org/wiki/SHA-1) |
| SHA256 | `sha256` | [SHA-2](https://wikipedia.org/wiki/SHA-2) |
| SHA512 | `sha512` | [SHA-2](https://wikipedia.org/wiki/SHA-2) |

> **Note:** MD5 and SHA1 are not collision resistant and should not be used for
> security-sensitive purposes such as signatures or certificates. They remain
> useful for checksums and interoperability.

---

## MD5

```bash
$ cchef md5 -i 'Hello, World!'
65a8e27d8879283831b664bd8b7f0ad4
```

## SHA1

```bash
$ cchef sha1 -i 'Hello, World!'
0a0a9f2a6772942557ab5355d76af442f8f65e01
```

## SHA256

```bash
$ cchef sha256 -i 'Hello, World!'
dffd6021bb2bd5b0af676290809ec3a53191dd81c7f70a4b28688a362182986f
```

## SHA512

```bash
$ cchef sha512 -i 'Hello, World!'
374d794a95cdcfd8b35993185fef9ba368f160d8daf432d08ba9f1ed1e5abe6cc69291e0fa2fe0006a52570ef18c19def4e617c33ce52ef0a6e5fbe318cb0387
```

---

### Hashing a file

Because input can come from a file, hashing a file's contents is straightforward:

```bash
$ cchef sha256 --in-file ./document.pdf
```
