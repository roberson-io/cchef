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

### Pointing at another CyberChef instance

Links go to the public GCHQ instance by default. A self-hosted or air-gapped
deployment is named in one of three places, most specific first:

| Source | Example |
| --- | --- |
| `--base-url` | `cchef url --base-url https://cyberchef.internal/ -e "ROT13()"` |
| `$CCHEF_BASE_URL` | `CCHEF_BASE_URL=https://cyberchef.internal/ cchef url -e "ROT13()"` |
| the config file | `base-url: https://cyberchef.internal/` |

```bash
cchef url --base-url https://cyberchef.internal/ -e "ROT13()" -i 'Hello'
```

Output:

```
https://cyberchef.internal/#recipe=ROT13()&input=SGVsbG8
```

An address that is not an `http://` or `https://` URL is refused, naming
whichever of the three supplied it, rather than printing a link that goes
nowhere.

Only link generation is affected. `bake` and `recipe convert` never fetch
anything, so they behave the same wherever the instance lives.

---

## Configuration file

cchef reads a config file for settings that belong to the machine rather than to
a recipe. It is optional, and there is one setting so far:

```yaml
# ~/.config/cchef/config.yaml
base-url: https://cyberchef.internal/
```

The file is looked for at `$XDG_CONFIG_HOME/cchef/config.yaml`, with
`$XDG_CONFIG_HOME` defaulting to `~/.config` as the XDG Base Directory
Specification says. The same layout is used on every platform, so the path is
the same one everywhere. `$CCHEF_CONFIG` names the file outright and skips the
search.

Having no config file is the normal case. One that exists but cannot be parsed
is an error naming the file, so a typo is reported rather than ignored.

Recipes and their arguments never come from here: a recipe has to mean the same
thing wherever it is run.

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

## `cchef recipe add` and friends — build a recipe step by step

Rather than writing a whole recipe in one go, you can stage it an operation at a
time, the way a commit is built up with `git add`, then run it. The staged
recipe lives in `.cchef-recipe.json` in the working directory, so different
projects keep different recipes; set `CCHEF_RECIPE` to put it somewhere else.

| Command | Does |
| --- | --- |
| `cchef recipe add <operation>` | Stage an operation with its default arguments |
| `cchef recipe add "<recipe>"` | Stage a recipe expression exactly (JSON or Chef) |
| `cchef recipe add <op> --at N` | Insert at position *N* instead of appending |
| `cchef recipe show` | Print the staged recipe, numbered |
| `cchef recipe rm <N>...` | Remove steps by number |
| `cchef recipe move <from> <to>` | Reorder a step |
| `cchef recipe toggle <N>...` | Turn steps off (and back on) without removing them |
| `cchef recipe clear` | Discard the staged recipe |

`bake`, `url`, and `recipe convert` all use the staged recipe when given no
`-e` or `-r` of their own.

**Example**

```bash
cchef recipe add rot13
cchef recipe add to-base64
cchef recipe add "To_Hex('Colon')"
cchef recipe show
```

Output:

```
0  ROT13 (true, true, false, 13)
1  To Base64 ("A-Za-z0-9+/=")
2  To Hex ("Colon")
```

Turn a step off to see what changes, then run what is left:

```bash
cchef recipe toggle 1
echo -n hello | cchef bake
```

Output:

```
75:72:79:79:62
```

The numbers in `recipe show` are the ones every other command takes, and each
number refers to the recipe as it was before the command — so `cchef recipe rm 0 2`
removes the steps shown as 0 and 2, not whatever shifts into those positions.

## `cchef list` — discover operations

Lists every available operation and its subcommand name, grouped by module.

```bash
cchef list
```
