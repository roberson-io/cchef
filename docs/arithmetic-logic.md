# Arithmetic / Logic

Reductions over lists of numbers, and operations over sets.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| Cartesian Product | `cartesian-product` | [Cartesian product](https://wikipedia.org/wiki/Cartesian_product) |
| Divide | `divide` | [Division](https://wikipedia.org/wiki/Division_(mathematics)) |
| Mean | `mean` | [Arithmetic mean](https://wikipedia.org/wiki/Arithmetic_mean) |
| Median | `median` | [Median](https://wikipedia.org/wiki/Median) |
| Multiply | `multiply` | [Multiplication](https://wikipedia.org/wiki/Multiplication) |
| Power Set | `power-set` | [Power set](https://wikipedia.org/wiki/Power_set) |
| Set Difference | `set-difference` | [Relative complement](https://wikipedia.org/wiki/Complement_(set_theory)#Relative_complement) |
| Set Intersection | `set-intersection` | [Intersection](https://wikipedia.org/wiki/Intersection_(set_theory)) |
| Set Union | `set-union` | [Union](https://wikipedia.org/wiki/Union_(set_theory)) |
| Standard Deviation | `standard-deviation` | [Standard deviation](https://wikipedia.org/wiki/Standard_deviation) |
| Subtract | `subtract` | [Subtraction](https://wikipedia.org/wiki/Subtraction) |
| Sum | `sum` | [Summation](https://wikipedia.org/wiki/Summation) |
| Symmetric Difference | `symmetric-difference` | [Symmetric difference](https://wikipedia.org/wiki/Symmetric_difference) |

## Number-list operations

`divide`, `mean`, `median`, `multiply`, `standard-deviation`, `subtract` and
`sum` each read a delimited list of numbers and fold it to a single value,
sharing these conventions with CyberChef:

- **Number parsing.** Each item is parsed like a JavaScript `BigNumber`: decimals
  (`8`, `.5`, `-3.2`), scientific notation (`1e3`), the prefixes `0x` / `0o` /
  `0b` (`0x0a` is `10`), and `Infinity` / `-Infinity`. Any item that is not a
  valid number is silently excluded from the list.
- **Arbitrary precision.** Addition, subtraction and multiplication are exact.
  Division, mean and standard deviation are computed to 20 decimal places
  (rounding half away from zero), just like CyberChef's `bignumber.js`.
- **Empty result.** If no valid numbers remain, the result is `NaN`.
- **Delimiter.** The `--delimiter` flag selects how items are separated. The
  default is `Line feed`.

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Line feed` | Item separator: `Line feed`, `Space`, `Comma`, `Semi-colon`, `Colon`, `CRLF`. |

## Set operations

`set-union`, `set-intersection`, `set-difference` and `symmetric-difference` take
**two** sets; `cartesian-product` takes **two or more**; `power-set` takes one.
Sets are separated by the `--sample-delimiter` (default `\n\n`) and items within a
set by the `--item-delimiter` (default `,`). Both delimiters accept backslash
escapes (`\n`, `\t`, …). Giving the wrong number of sets is an error.

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--sample-delimiter` | string | `\n\n` | Separator between sets (`power-set` has no sample delimiter). |
| `--item-delimiter` | string | `,` | Separator between items within a set. |

---

## Cartesian Product

Returns every combination drawing one item from each of two or more sets. Each
combination is formatted as `(a,b,…)` and the combinations are joined by the item
delimiter.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--sample-delimiter` | string | `\n\n` | Separator between sets. |
| `--item-delimiter` | string | `,` | Separator between items and between combinations. |

**Simple example**

```bash
printf '1,2\n\na,b' | cchef cartesian-product
```

Output:

```
(1,a),(1,b),(2,a),(2,b)
```

---

## Divide

Divides the list left-to-right (`a / b / c / …`). Non-numeric items are excluded.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Line feed` | Item separator (see shared options). |

**Simple example**

```bash
cchef divide --delimiter Space -i '0x0a 8 .5'
```

Output:

```
2.5
```

**Non-terminating result (20 decimal places)**

```bash
cchef divide --delimiter Space -i '1 3'
```

Output:

```
0.33333333333333333333
```

---

## Mean

Computes the mean (average) of the list. Non-numeric items are excluded.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Line feed` | Item separator (see shared options). |

**Simple example**

```bash
cchef mean --delimiter Space -i '0x0a 8 .5 .5'
```

Output:

```
4.75
```

---

## Median

Computes the median of the list, sorting it first and averaging the two middle
values for an even-length list. Non-numeric items are excluded.

> CyberChef fixed a bug in mid-2026
> ([PR #2284](https://github.com/gchq/CyberChef/pull/2284)) where odd-length lists
> were not sorted before the median was taken. `cchef` matches the corrected
> behaviour (a true numeric median for all list lengths).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Line feed` | Item separator (see shared options). |

**Simple example (odd length)**

```bash
cchef median --delimiter Space -i '10 1 2'
```

Output:

```
2
```

**Even length (mean of the two middle values)**

```bash
cchef median --delimiter Space -i '10 1 2 5'
```

Output:

```
3.5
```

---

## Multiply

Multiplies the list together (`a * b * c * …`). Non-numeric items are excluded.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Line feed` | Item separator (see shared options). |

**Simple example**

```bash
cchef multiply --delimiter Space -i '0x0a 8 .5'
```

Output:

```
40
```

---

## Power Set

Returns every subset of a single set (its power set). Each subset is joined by the
item delimiter, subsets are ordered by their length, and each is followed by a
newline (so the first line — the empty subset — is blank). Empty items are
ignored.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--item-delimiter` | string | `,` | Separator between items. |

**Simple example**

```bash
printf 'a,b,c' | cchef power-set
```

Output:

```
c
b
a
b,c
a,c
a,b
a,b,c
```

---

## Set Difference

Returns the items of the first set that are not in the second (the relative
complement). Duplicates in the first set are removed; order is preserved.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--sample-delimiter` | string | `\n\n` | Separator between the two sets. |
| `--item-delimiter` | string | `,` | Separator between items. |

**Simple example**

```bash
printf '1,2,3,4\n\n3,4' | cchef set-difference
```

Output:

```
1,2
```

---

## Set Intersection

Returns the items present in both sets, in the order they appear in the first set.
Duplicates in the first set are removed.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--sample-delimiter` | string | `\n\n` | Separator between the two sets. |
| `--item-delimiter` | string | `,` | Separator between items. |

**Simple example**

```bash
printf '1,2,3\n\n2,3,4' | cchef set-intersection
```

Output:

```
2,3
```

---

## Set Union

Returns the items that appear in either set, deduplicated. Matching CyberChef's
implementation (a JavaScript object used as a hash set), integer-like items are
emitted first in ascending numeric order, then the remaining items in the order
first seen.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--sample-delimiter` | string | `\n\n` | Separator between the two sets. |
| `--item-delimiter` | string | `,` | Separator between items. |

**Simple example**

```bash
printf '1,2,3\n\n3,4,5' | cchef set-union
```

Output:

```
1,2,3,4,5
```

**Integer ordering quirk**

```bash
printf '3,1,2\n\n5,4' | cchef set-union
```

Output:

```
1,2,3,4,5
```

---

## Standard Deviation

Computes the population standard deviation of the list. Non-numeric items are
excluded. The result is computed to 20 decimal places.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Line feed` | Item separator (see shared options). |

**Simple example**

```bash
cchef standard-deviation --delimiter Space -i '0x0a 8 .5'
```

Output:

```
4.08928138212843238213
```

---

## Subtract

Subtracts the list left-to-right (`a - b - c - …`). Non-numeric items are
excluded.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Line feed` | Item separator (see shared options). |

**Simple example**

```bash
cchef subtract --delimiter Comma -i '321,123,test'
```

Output:

```
198
```

---

## Sum

Adds the list together. Non-numeric items are excluded.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--delimiter` | option | `Line feed` | Item separator (see shared options). |

**Simple example**

The default delimiter is a line feed, so a newline-separated list works with no
flags:

```bash
printf '10\n8\n0.5' | cchef sum
```

Output:

```
18.5
```

**Mixed radices on one line**

```bash
cchef sum --delimiter Space -i '0x0a 8 .5'
```

Output:

```
18.5
```

---

## Symmetric Difference

Returns the items that appear in exactly one of the two sets (the first set's
extras followed by the second set's extras). Unlike the other set operations, it
preserves duplicates within each side.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--sample-delimiter` | string | `\n\n` | Separator between the two sets. |
| `--item-delimiter` | string | `,` | Separator between items. |

**Simple example**

```bash
printf 'a,b,c\n\nb,c,d' | cchef symmetric-difference
```

Output:

```
a,d
```
