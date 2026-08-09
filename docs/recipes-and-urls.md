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

Provide the recipe inline with `-e/--expr`, from a file with `-r/--recipe` (or
`-r -` to read it from stdin), or from a CyberChef share link with
`--from-url`. Input is resolved the same way as for any operation (stdin, `-i`,
positional, or `--in-file`).

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
`-e`/`-r`/`--from-url` recipe flags as `bake`.

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

### Reading a share URL

`--from-url` is the other direction: it takes a CyberChef link and reads the
recipe out of it. Nothing is fetched — the recipe travels in the link itself, so
this works offline. `bake`, `recipe convert` and `url` all accept it.

```bash
cchef bake --from-url "https://gchq.github.io/CyberChef/#recipe=ROT13()&input=aGVsbG8"
```

Output:

```
uryyb
```

A link built by CyberChef's **Save recipe** carries the input it was made with,
which is what the example above ran on. An input given on the command line wins
over it:

```bash
cchef bake --from-url "https://gchq.github.io/CyberChef/#recipe=ROT13()&input=aGVsbG8" world
```

Output:

```
jbeyq
```

Point a colleague's public link at your own instance by reading it and writing
it back out:

```bash
cchef url --from-url "https://gchq.github.io/CyberChef/#recipe=ROT13()&input=aGVsbG8" --base-url https://cyberchef.internal/
```

Output:

```
https://cyberchef.internal/#recipe=ROT13()&input=aGVsbG8
```

A whole URL, a bare `#recipe=...` fragment, or a bare `recipe=...` parameter
string are all accepted. Settings that only a browser can act on — `theme`,
`ienc`, `oenc`, `ieol`, `oeol` — are ignored. A link naming no recipe is an
error rather than a recipe that does nothing. `--in-dir` ignores a link's input,
since the directory is the input.

Only one recipe source may be given: `-e`, `-r` and `--from-url` together are
refused rather than resolved by precedence.

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

Reads a recipe (from `-e`, `-r` or `--from-url`) and prints it in the other
format. Use `--to` to force a target of `json` or `chef`.

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
| `cchef recipe load` | Replace the staged recipe with one read from elsewhere |
| `cchef recipe show` | Print the staged recipe, numbered |
| `cchef recipe rm <N>...` | Remove steps by number |
| `cchef recipe move <from> <to>` | Reorder a step |
| `cchef recipe toggle <N>...` | Turn steps off (and back on) without removing them |
| `cchef recipe clear` | Discard the staged recipe |

`bake`, `url`, and `recipe convert` all use the staged recipe when given no
recipe of their own.

**Example**

```bash
cchef recipe add rot13
cchef recipe add to-base64
cchef recipe add "To_Hex('Colon')"
cchef recipe show
```

Output:

```
0  [X] ROT13 (true, true, false, 13)
1  [X] To Base64 ("A-Za-z0-9+/=")
2  [X] To Hex ("Colon")
```

Each step carries `[X]` when it runs and `[ ]` when it is disabled, and a step
that stops a bake early is marked `(breakpoint)`. At a terminal the markers are
colored green and red, and the breakpoint yellow; the marker itself carries the
meaning, so the listing reads the same when piped, under
[`NO_COLOR`](https://no-color.org/), or to anyone who cannot tell the two colors
apart. `--ansi always` keeps the color through a pipe and `--ansi never` drops
it at a terminal.

Turn a step off to see what changes, then run what is left:

```bash
cchef recipe toggle 1
cchef recipe show
```

Output:

```
0  [X] ROT13 (true, true, false, 13)
1  [ ] To Base64 ("A-Za-z0-9+/=")
2  [X] To Hex ("Colon")
```

The disabled step is skipped when the recipe runs:

```bash
echo -n hello | cchef bake
```

Output:

```
75:72:79:79:62
```

The numbers in `recipe show` are the ones every other command takes, and each
number refers to the recipe as it was before the command — so `cchef recipe rm 0 2`
removes the steps shown as 0 and 2, not whatever shifts into those positions.

### Loading a recipe into the stage

`recipe add` appends a step; `recipe load` replaces the whole staged recipe with
one from somewhere else — a string, a file, stdin, or a CyberChef share link:

```bash
cchef recipe load -e "To_Base64()To_Hex('Space')"
cchef recipe load -r recipe.json
cat recipe.json | cchef recipe load -r -
cchef recipe load --from-url "https://gchq.github.io/CyberChef/#recipe=ROT13()"
```

Exactly one source is given. The recipe is read and checked before anything is
written, so a load that fails — a malformed recipe, a missing file, an operation
cchef does not have — leaves the recipe already staged untouched. A recipe with
no operations in it is an error; use `cchef recipe clear` to empty the stage.

When a share link also carries an input, the recipe is staged and a note says
where the input went, since the stage holds only a recipe.

### Exporting the staged recipe

`recipe convert` is the way back out: given no recipe of its own, it reads the
staged one, so it doubles as the export.

```bash
cchef recipe convert --to json > recipe.json
cchef recipe convert --to chef
```

Output:

```
ROT13(true,true,false,13)To_Base64('A-Za-z0-9+/=')To_Hex('Colon')
```

## `cchef list` — discover operations

Lists every available operation and its subcommand name, grouped by module.

```bash
cchef list
```

`--json` gives the same listing in a form a script can read: one object per
subcommand, sorted by name, carrying the subcommand, the CyberChef operation
name, the one-line summary and the categories it is filed under.

```bash
cchef list --json | jq -r '.[] | select(.categories[] == "Hashing") | .command'
```
