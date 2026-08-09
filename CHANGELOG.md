# Changelog

All notable changes to cchef are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

cchef's version tracks CyberChef's, one component down: a CyberChef minor
release maps to a cchef minor, a CyberChef major to a cchef major, and the
patch component is cchef's own (a CyberChef patch or a cchef-only fix). Each
entry names the CyberChef release it is aligned with.

## [Unreleased]

## [1.0.1]

Aligned with CyberChef v11.3.0.

Recipes gained a share-link reader and a `load` command, and running two
community collections of CyberChef recipes through cchef turned up a batch of
divergences from upstream — most of them silent corruption of non-ASCII input.

### Added

- **`cchef recipe load`** replaces the staged recipe from a string (`-e`), a
  file (`-r`), stdin (`-r -`) or a share link (`--from-url`). It parses and
  validates before writing, so a failed load leaves the existing recipe intact.
  `recipe convert` is the matching export.
- **`--from-url` reads a CyberChef share link, offline**, on `bake`,
  `recipe convert`, `recipe load` and `url`. The input a link carries is used
  unless input is given explicitly. `cchef url` still writes them, so a recipe
  now round-trips in both directions.
- **`recipe show` marks each step** `[X]` when it runs and `[ ]` when it is
  disabled, and flags a step carrying a breakpoint. `--ansi` colors the markers,
  but the ASCII marker carries the meaning, so `NO_COLOR` and piped output read
  the same.
- **`-r -` reads a recipe from stdin** on `bake` and `recipe convert`.
- **A Scoop package for Windows** (`scoop bucket add roberson-io
  https://github.com/roberson-io/scoop-bucket`), and PowerShell completions now
  ship in the release archives.
- **[docs/install.md](docs/install.md)** — a per-platform install guide covering
  what each package installs where, uninstalling, and verifying a release. The
  README's install commands no longer name a version, so they keep working
  across releases.
- Library: `core.ParseURL`, `core.DecodeURIFragment` and
  `core.MarshalRecipeJSON`, the inverses of the existing URL builder and
  `ParseRecipeConfig`.

### Changed

- **`--version` reports the version the binary was actually installed from.**
  With no release stamp it now reads the module version the toolchain recorded,
  so `go install …@latest` reports that release rather than a placeholder, and a
  build from a checkout reports a pseudo-version naming its commit.
- **`ToggleString` serializes `option` before `string`**, matching CyberChef, so
  a recipe's spelling is stable across a round trip.
- **A `List<File>` operation chains into the next one.** Its files' contents
  feed the following operation concatenated in order (CyberChef's
  `List<File>`→`ArrayBuffer`), so `Unzip | Extract URLs` works instead of
  erroring.
- **A user-supplied regular expression falls back to a JavaScript-compatible
  engine.** The pattern is compiled with RE2 first, and only when RE2 rejects it
  — lookahead, lookbehind, backreferences, `(?<name>)` — is it retried in
  ECMAScript mode, bounded by a match timeout. The six operations taking a user
  regex share the path.
- The `.rpm` recommends `tesseract`; only the `.deb` recommends Debian's
  `tesseract-ocr`. Previously both named the Debian package.
- Dependencies updated: `golang.org/x/arch`, `evanw/esbuild`, `cloudflare/circl`,
  `dlclark/regexp2/v2`, `tidwall/gjson` and `tidwall/match`.

### Fixed

- **Non-ASCII input was corrupted by the operations that convert between bytes
  and text**, which used UTF-8 where CyberChef uses Latin-1. Reverse,
  Substitute, the case conversions, Escape string, Format MAC addresses, RAKE,
  Remove diacritics, To Braille and others silently mangled or dropped bytes.
- **RAKE mangled any input holding a non-ASCII character.** The text was split
  on delimiter matches by indexing the string with rune offsets, which cut
  multi-byte characters in half, so "café" came back as two broken fragments and
  the output was not valid UTF-8. Keywords now come back whole, matching
  CyberChef byte for byte.
- **A trailing newline was written even when output was not a terminal**, which
  showed up as a stray `%` after `cchef recipe show` in zsh.
- **`Parse DateTime` mis-read a format beginning with `X` or `x`** (UNIX
  seconds and milliseconds) when further tokens followed it, as in the Squid
  log format `X.SSS`.
- **`recipe add` accepted an unknown operation inside an expression** —
  `"To_Hex()No_Such_Op()"` staged a recipe that could never run.
- **`recipe show` printed toggle-string arguments as Go maps**
  (`map[option:Hex string:ff]`).
- **A staged breakpoint was honored but never displayed**, so a bake could stop
  early with nothing on screen explaining why.
- **Indented recipe files were rejected**, a Chef-format step naming no
  operation was accepted, and JSON operation names were not trimmed. Each is now
  a regression test and a fuzz corpus entry.
- **`make fuzz` had been passing without running anything.** It iterated package
  paths that no longer existed, printed "directory not found" and still exited
  0, so only the YARA target had run since the packages moved. It now fails when
  a package yields no fuzz targets — and the three parser bugs above came
  straight out of the targets that started running.
- Syntax highlighter's documented examples pin their HTML with `--ansi never`,
  so each `Output:` block is exactly what the command above it prints.

## [1.0.0]

Initial public release, aligned with CyberChef v11.3.0.

A curated set of 505 operation subcommands covering every CyberChef operation,
a recipe engine (`bake`, `url`, interactive `recipe` staging), and the CLI
around them. See [README.md](README.md) for what the tool does and
[docs/](docs/README.md) for per-operation documentation.

[Unreleased]: https://github.com/roberson-io/cchef/compare/v1.0.1...HEAD
[1.0.1]: https://github.com/roberson-io/cchef/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/roberson-io/cchef/releases/tag/v1.0.0
