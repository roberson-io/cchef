# Recipes & URLs

A **recipe** is an ordered list of operations. Instead of piping several `cchef`
commands together, you can express the whole chain as one recipe and run it with
`cchef bake`, share it as a CyberChef URL with `cchef url`, or convert it between
formats with `cchef recipe convert`.

## Recipe formats

cchef understands the same two recipe formats as CyberChef, and auto-detects
which one you provide (a leading `[` means JSON).

### Chef format (compact)

Operation names with spaces replaced by underscores, arguments in single quotes:

```
To_Base64('A-Za-z0-9+/=')To_Hex('Space')
```

### JSON format

An array of `{op, args}` objects:

```json
[
  { "op": "To Base64", "args": ["A-Za-z0-9+/="] },
  { "op": "To Hex", "args": ["Space"] }
]
```

In both formats, `args` is the ordered list of an operation's argument values.
Operations may also carry `disabled` / `breakpoint` flags (`/disabled`,
`/breakpoint` in Chef format).

---

## `cchef bake` — run a recipe

Provide the recipe inline with `-e/--expr` or from a file with `-r/--recipe`.
Input is resolved the same way as for any operation (stdin, `-i`, positional,
or `--in-file`).

**Inline Chef format**

```bash
cchef bake -i 'hello' -e "To_Base64()To_Hex()"
```

Output:

```
61 47 56 73 62 47 38 3d
```

**Recipe from a JSON file**

```bash
cat recipe.json
```

Output:

```
[{"op":"To Base64","args":["A-Za-z0-9+/="]}]
```

```bash
cchef bake -i 'hello' -r recipe.json
```

Output:

```
aGVsbG8=
```

> Arguments may be omitted to use an operation's defaults: `To_Base64()` is
> equivalent to `To_Base64('A-Za-z0-9+/=')`.

---

## `cchef url` — share a recipe

Builds a `https://gchq.github.io/CyberChef/` link that opens the recipe in the
CyberChef web app, with the input pre-loaded if one is supplied. Takes the same
`-e`/`-r` recipe flags as `bake`.

```bash
cchef url -e "ROT13()To_Hex()" -i 'Hello'
```

Output:

```
https://gchq.github.io/CyberChef/#recipe=ROT13()To_Hex()&input=SGVsbG8
```

Open the URL to continue editing the recipe interactively in CyberChef.

---

## `cchef recipe convert` — convert between formats

Reads a recipe (from `-e` or `-r`) and prints it in the other format. Use `--to`
to force a target of `json` or `chef`.

**Chef → JSON**

```bash
cchef recipe convert -e "To_Base64('A-Za-z0-9+/=')To_Hex('Space')" --to json
```

Output:

```
[
  {
    "op": "To Base64",
    "args": [
      "A-Za-z0-9+/="
    ]
  },
  {
    "op": "To Hex",
    "args": [
      "Space"
    ]
  }
]
```

**JSON → Chef**

```bash
cchef recipe convert -r recipe.json --to chef
```

Output:

```
To_Base64('A-Za-z0-9+/=')
```

---

## `cchef list` — discover operations

Lists every available operation and its subcommand name, grouped by module.

```bash
cchef list
```
