# Utils

General-purpose text utilities.

> Operations are listed alphabetically.

| Operation | Subcommand |
| --- | --- |
| Reverse | `reverse` |
| To Lower case | `to-lower-case` |
| To Upper case | `to-upper-case` |

---

## Reverse

Reverses the input by byte, character (UTF-8 rune), or line.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--by` | option | `Character` | `Byte` (raw bytes), `Character` (UTF-8 runes, keeps multi-byte sequences intact), or `Line` (reverses line order). |

**Simple example**

```bash
$ cchef reverse -i 'Hello, World!'
!dlroW ,olleH
```

**Reverse line order**

```bash
$ printf 'one\ntwo\nthree' | cchef reverse --by Line
three
two
one
```

---

## To Lower case

Converts every character in the input to lower case. This operation takes no
options.

**Simple example**

```bash
$ cchef to-lower-case -i 'Hello, World!'
hello, world!
```

---

## To Upper case

Converts the input to upper case, optionally limiting the scope to the first
character of each word, sentence, or paragraph.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--scope` | option | `All` | `All`, `Word`, `Sentence`, or `Paragraph`. |

**Simple example**

```bash
$ cchef to-upper-case -i 'Hello, World!'
HELLO, WORLD!
```

**Title-case each word**

```bash
$ cchef to-upper-case --scope Word -i 'hello there world'
Hello There World
```
