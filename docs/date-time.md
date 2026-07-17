# Date / Time

Convert between timestamps, Windows Filetimes and human-readable datetimes, and
reformat datetime strings.

> Operations are listed alphabetically.

| Operation | Subcommand | Reference |
| --- | --- | --- |
| DateTime Delta | `datetime-delta` | [moment.js format](https://momentjs.com/docs/#/parsing/string-format/) |
| Extract dates | `extract-dates` | [ISO 8601](https://wikipedia.org/wiki/ISO_8601) |
| From UNIX Timestamp | `from-unix-timestamp` | [UNIX time](https://wikipedia.org/wiki/Unix_time) |
| Get Time | `get-time` | [UNIX time](https://wikipedia.org/wiki/Unix_time) |
| Parse DateTime | `parse-datetime` | [moment.js format](https://momentjs.com/docs/#/parsing/string-format/) |
| To UNIX Timestamp | `to-unix-timestamp` | [UNIX time](https://wikipedia.org/wiki/Unix_time) |
| Translate DateTime Format | `translate-datetime-format` | [moment.js format](https://momentjs.com/docs/#/parsing/string-format/) |
| UNIX Timestamp to Windows Filetime | `unix-timestamp-to-windows-filetime` | [Windows Filetime](https://learn.microsoft.com/en-us/windows/win32/api/minwinbase/ns-minwinbase-filetime) |
| Windows Filetime to UNIX Timestamp | `windows-filetime-to-unix-timestamp` | [Windows Filetime](https://learn.microsoft.com/en-us/windows/win32/api/minwinbase/ns-minwinbase-filetime) |

**Format strings.** `datetime-delta`, `parse-datetime` and
`translate-datetime-format` take [moment.js format tokens](https://momentjs.com/docs/#/displaying/format/)
(e.g. `DD/MM/YYYY HH:mm:ss`, `dddd Do MMMM YYYY`). cchef reimplements the
formatter and parser rather than depending on moment, and reproduces CyberChef's
output for the common tokens. Known fidelity gaps, matching or approximating
CyberChef where noted:

- Timezone names are resolved via the system tz database, so the `z` abbreviation
  and `Z`/`ZZ` offsets are only as complete as the host's zoneinfo.
- Week-year tokens (`gg`/`GG`) approximate to the calendar year.
- A bare `W` token is emitted literally (as moment does); use `WW`/`Wo` for the
  ISO week number.
- Parsing uses a fixed set of ISO-like layouts for the **Automatic** format;
  unusual input shapes moment would still accept may be rejected.

Each of the three format operations also accepts a leading `--built-in-formats`
argument (the CyberChef "Built in formats" dropdown). It is accepted for recipe
compatibility but ignored — set the format directly with the format-string flag.

---

## DateTime Delta

Adds or subtracts a days/hours/minutes/seconds delta from a datetime, keeping the
input format. Parsing is done in UTC.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format-string` | string | `DD/MM/YYYY HH:mm:ss` | Format of the input (and output) datetime. |
| `--time-operation` | option | `Add` | `Add` or `Subtract` the delta. |
| `--days` | number | `0` | Days to add/subtract. |
| `--hours` | number | `0` | Hours to add/subtract. |
| `--minutes` | number | `0` | Minutes to add/subtract. |
| `--seconds` | number | `0` | Seconds to add/subtract. |

**Simple example**

```bash
printf '20/02/2024 13:36:00' | cchef datetime-delta --time-operation Add --minutes 1
```

Output:

```
20/02/2024 13:37:00
```

---

## Extract dates

Extracts dates in `yyyy-mm-dd`, `dd/mm/yyyy` and `mm/dd/yyyy` shapes (separators
`-`, `/` or `.`), one per line.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--display-total` | bool | `false` | Prefix the output with a total count. |

**Simple example**

```bash
printf 'Due 2024-02-20, ship 01/04/1999.' | cchef extract-dates --display-total
```

Output:

```
Total found: 2

2024-02-20
01/04/1999
```

---

## From UNIX Timestamp

Renders a UNIX timestamp as a human-readable UTC datetime.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--units` | option | `Seconds (s)` | Input units: `Seconds (s)`, `Milliseconds (ms)`, `Microseconds (μs)`, `Nanoseconds (ns)`. |

**Simple example**

```bash
printf '1276263039' | cchef from-unix-timestamp
```

Output:

```
Fri 11 June 2010 13:30:39 UTC
```

---

## Get Time

Emits the current time since the UNIX epoch in the chosen unit. The value depends
on when it runs.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--granularity` | option | `Seconds (s)` | Output units: `Seconds (s)`, `Milliseconds (ms)`, `Microseconds (μs)`, `Nanoseconds (ns)`. |

**Simple example**

```bash
cchef get-time --granularity 'Milliseconds (ms)'
```

Output:

```
1783118250086
```

---

## Parse DateTime

Parses a datetime in the given format and timezone and reports its components.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format-string` | string | `DD/MM/YYYY HH:mm:ss` | moment.js format of the input. |
| `--input-timezone` | string | `UTC` | Timezone the input is in (any tz database name). |

**Simple example**

```bash
printf '01/04/1999 22:33:01' | cchef parse-datetime
```

Output:

```
Date: Thursday 1st April 1999
Time: 22:33:01
Period: PM
Timezone: UTC
UTC offset: +0000

Daylight Saving Time: false
Leap year: false
Days in this month: 30

Day of year: 91
Week number: 14
Quarter: 2
```

---

## To UNIX Timestamp

Parses a datetime string and returns the corresponding UNIX timestamp. The input
is parsed leniently (ISO 8601 and close relatives).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--units` | option | `Seconds (s)` | Output units: `Seconds (s)`, `Milliseconds (ms)`, `Microseconds (μs)`, `Nanoseconds (ns)`. |
| `--treat-as-utc` | bool | `true` | Interpret the input as UTC (otherwise the local timezone). |
| `--show-parsed-datetime` | bool | `true` | Append the parsed datetime for confirmation. |

**Simple example**

```bash
printf '2013-02-04 22:33:01' | cchef to-unix-timestamp
```

Output:

```
1360017181 (Mon 4 February 2013 22:33:01 UTC)
```

**Just the number**

```bash
printf '2013-02-04 22:33:01' | cchef to-unix-timestamp --show-parsed-datetime=false
```

Output:

```
1360017181
```

---

## Translate DateTime Format

Parses a datetime in one format/timezone and re-writes it in another. Returns
`Invalid format.` if the input does not match the input format.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-format-string` | string | `DD/MM/YYYY HH:mm:ss` | Format of the input (empty = automatic). |
| `--input-timezone` | string | `UTC` | Timezone the input is in. |
| `--output-format-string` | string | `dddd Do MMMM YYYY HH:mm:ss Z z` | Format to write. |
| `--output-timezone` | string | `UTC` | Timezone to convert to. |

**Simple example**

```bash
printf '01/04/1999 22:33:01' | cchef translate-datetime-format
```

Output:

```
Thursday 1st April 1999 22:33:01 +00:00 UTC
```

**Convert timezone**

```bash
printf '01/04/1999 22:33:01' | cchef translate-datetime-format --output-timezone US/Eastern
```

Output:

```
Thursday 1st April 1999 17:33:01 -05:00 EST
```

---

## UNIX Timestamp to Windows Filetime

Converts a UNIX timestamp to a Windows Filetime (100 ns intervals since
1601-01-01 UTC).

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--input-units` | option | `Seconds (s)` | Units of the input timestamp. |
| `--output-format` | option | `Decimal` | Output radix: `Decimal`, `Hex (big endian)`, `Hex (little endian)`. |

**Simple example**

```bash
printf '1276263039' | cchef unix-timestamp-to-windows-filetime --input-units 'Seconds (s)'
```

Output:

```
129207366390000000
```

---

## Windows Filetime to UNIX Timestamp

Converts a Windows Filetime back to a UNIX timestamp in the chosen unit.

**Options**

| Flag | Type | Default | Description |
| --- | --- | --- | --- |
| `--output-units` | option | `Seconds (s)` | Units of the output timestamp. |
| `--input-format` | option | `Decimal` | Input radix: `Decimal`, `Hex (big endian)`, `Hex (little endian)`. |

**Simple example**

```bash
printf '129207366395297693' | cchef windows-filetime-to-unix-timestamp --output-units 'Nanoseconds (ns)'
```

Output:

```
1276263039529769300
```
