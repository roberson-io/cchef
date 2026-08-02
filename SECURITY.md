# Security Policy

## Reporting a vulnerability

Please report suspected vulnerabilities privately through GitHub's
[security advisories](https://github.com/roberson-io/cchef/security/advisories/new)
for this repository ("Report a vulnerability"). Do not open a public issue for
anything you believe is exploitable.

A useful report includes the exact command or recipe, the input (or a way to
generate it), what happened, and why it matters. You should receive an
acknowledgement within a week.

## Scope

cchef is a data-transformation tool, and — as the README notes — it is not
recommended for production or critical-infrastructure use. Reports most worth
making:

- Memory-safety or resource-exhaustion issues reachable through operation
  inputs (the parsers and decoders are fuzzed, but coverage is never
  complete).
- A cryptographic operation producing wrong output for valid parameters.
- Anything that makes `cchef` read or write files the command line did not
  name.

Two things are design decisions rather than vulnerabilities:

- **CyberChef parity.** Operations reproduce CyberChef's behavior, including
  its legacy or weak algorithms (MD5, SHA-1, RC4, DES, the classical
  ciphers). Their presence is the point of the tool, not an oversight.
- **Secrets on the command line** are visible to the local machine; use the
  `--<flag>-file` companions documented in
  [docs/README.md](docs/README.md#reading-argument-values-from-files) to keep
  keys and passphrases out of shell history and process listings.

## Supported versions

Only the latest release is supported with fixes.
