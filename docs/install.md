# Installing cchef

cchef is a single static binary with no cgo and no runtime dependencies. Any of
the methods below give you a working `cchef`; they differ in whether you also get
the man page, shell completions, and automatic upgrades.

`tesseract` is the one optional dependency, needed only by the Optical Character
Recognition operation. Every other operation is self-contained.

| Method | Platforms | Man page | Completions | Upgrades |
| --- | --- | --- | --- | --- |
| [Homebrew](#homebrew-macos-and-linux) | macOS, Linux | yes | bash, zsh, fish | `brew upgrade` |
| [`.deb`](#debian-and-ubuntu) | Debian, Ubuntu | yes | bash, zsh, fish | re-install |
| [`.rpm`](#fedora-rhel-and-opensuse) | Fedora, RHEL, openSUSE | yes | bash, zsh, fish | re-install |
| [Archive](#prebuilt-archive-any-platform) | all | in the archive | in the archive | re-download |
| [Scoop](#scoop-windows) | Windows | no | PowerShell | `scoop update` |
| [Windows zip](#windows-zip) | Windows | in the archive | PowerShell | re-download |
| [`go install`](#go-install) | all | no | no | re-run |
| [From source](#from-source) | all | no | no | `git pull` |

Release artifacts are named `cchef_<version>_<os>_<arch>` — for example
`cchef_1.0.0_linux_amd64.deb`. The architecture is `amd64` or `arm64` in every
artifact name, including the `.rpm` (not `x86_64`/`aarch64`).

Because the name carries the version, `releases/latest/download/<name>` cannot be
used: the URL would name one release forever. The commands below ask the API for
the latest release instead, so nothing has to be edited when a new one ships.

## Homebrew (macOS and Linux)

```bash
brew install roberson-io/tap/cchef
```

This installs the binary, the man page and the bash, zsh and fish completions.
On macOS it also clears the quarantine attribute, so the binary runs without
being cleared in System Settings first.

Upgrade and uninstall in the usual way:

```bash
brew upgrade cchef
brew uninstall cchef
```

## Debian and Ubuntu

Fetch the latest `.deb` for your architecture and install it:

```bash
curl -LO "$(curl -fsSL https://api.github.com/repos/roberson-io/cchef/releases/latest \
  | grep -o 'https://[^"]*linux_amd64\.deb"' | tr -d '"')"
sudo dpkg -i cchef_*_linux_amd64.deb
```

Use `linux_arm64` instead on 64-bit Arm. With the [`gh`](https://cli.github.com/)
CLI the same thing is shorter:

```bash
gh release download --repo roberson-io/cchef --pattern '*linux_amd64.deb'
sudo dpkg -i cchef_*_linux_amd64.deb
```

To pin a particular version, take the URL from the
[releases page](https://github.com/roberson-io/cchef/releases) instead.

The package installs:

| Path | What |
| --- | --- |
| `/usr/bin/cchef` | the binary |
| `/usr/share/man/man1/cchef.1` | the man page |
| `/usr/share/bash-completion/completions/cchef` | bash completion |
| `/usr/share/zsh/site-functions/_cchef` | zsh completion |
| `/usr/share/fish/vendor_completions.d/cchef.fish` | fish completion |
| `/usr/share/doc/cchef/` | `LICENSE` and `NOTICE` |

It recommends `tesseract-ocr`, so `apt` offers it alongside unless you install
with `--no-install-recommends`. Remove with `sudo apt remove cchef`.

## Fedora, RHEL and openSUSE

```bash
curl -LO "$(curl -fsSL https://api.github.com/repos/roberson-io/cchef/releases/latest \
  | grep -o 'https://[^"]*linux_amd64\.rpm"' | tr -d '"')"
sudo rpm -i cchef_*_linux_amd64.rpm
```

`sudo dnf install ./cchef_*_linux_amd64.rpm` works too, and pulls in the
recommended `tesseract` package. The installed paths are the same as the `.deb`
above. Remove with `sudo rpm -e cchef`.

> The architecture in the file name is `amd64`/`arm64`, matching the other
> artifacts, rather than the `x86_64`/`aarch64` an rpm usually carries.

## Scoop (Windows)

[Scoop](https://scoop.sh/) installs and upgrades cchef in place:

```powershell
scoop bucket add roberson-io https://github.com/roberson-io/scoop-bucket
scoop install cchef
```

Upgrade with `scoop update cchef`, remove with `scoop uninstall cchef`.

## Windows zip

Download `cchef_<version>_windows_amd64.zip` (or `windows_arm64.zip`) from the
[latest release](https://github.com/roberson-io/cchef/releases/latest), unzip it,
and put `cchef.exe` somewhere on your `PATH`.

```powershell
$Url = (Invoke-RestMethod https://api.github.com/repos/roberson-io/cchef/releases/latest).assets |
  Where-Object name -like '*windows_amd64.zip' |
  Select-Object -ExpandProperty browser_download_url
Invoke-WebRequest -Uri $Url -OutFile cchef.zip
Expand-Archive cchef.zip -DestinationPath $env:LOCALAPPDATA\cchef
$env:PATH += ";$env:LOCALAPPDATA\cchef"
```

That asks the API for the latest release, so there is no version to keep up to
date. Use `*windows_arm64.zip` on 64-bit Arm.

Add the directory to your `PATH` permanently through *Settings → System → About →
Advanced system settings → Environment Variables*, or with `setx`.

The archive carries the man page, the docs and `completions/cchef.ps1`. To load
completions for the session, dot-source that file; to load them every session,
add it to your profile:

```powershell
. $env:LOCALAPPDATA\cchef\completions\cchef.ps1
```

There is no `.msi`, winget or Chocolatey package. Scoop and this zip are the
supported routes on Windows.

## Prebuilt archive (any platform)

```bash
curl -LO "$(curl -fsSL https://api.github.com/repos/roberson-io/cchef/releases/latest \
  | grep -o 'https://[^"]*darwin_arm64\.tar\.gz"' | tr -d '"')"
tar xzf cchef_*_darwin_arm64.tar.gz
sudo install -m 0755 cchef /usr/local/bin/cchef
```

Replace `darwin_arm64` with `darwin_amd64`, `linux_amd64` or `linux_arm64`. Every
archive also contains `README.md`, `CHANGELOG.md`, `LICENSE`, `NOTICE`,
`SECURITY.md`, the `docs/` pages, `man/cchef.1` and the `completions/` scripts —
install the man page and completion for your shell by hand if you want them.

## `go install`

```bash
go install github.com/roberson-io/cchef@latest
```

Requires Go 1.26 or newer. `cchef --version` reports the module version you
installed — `go install ...@latest` on the 1.0.0 release reports `1.0.0` — since
the binary reads it from the build information the toolchain records. No man page
or completions are installed, though: `go install` only ever produces the binary.

## From source

```bash
git clone https://github.com/roberson-io/cchef
cd cchef
make build      # produces ./dist/cchef
```

Requires Go 1.26+. `make all` runs the full check suite (format, vet, test,
build, lint, security scans).

A binary built from a checkout reports a pseudo-version naming the commit it came
from, with `+dirty` when the tree had uncommitted changes — for example
`1.0.1-0.20260809100557-a68f68f08e55+dirty`, meaning a build from a commit after
the 1.0.0 release. Built with no version control information available, it
reports `dev`.

## Shell completion

`cchef completion <shell>` prints a completion script for `bash`, `zsh`, `fish`
or `powershell`. Homebrew and the deb/rpm packages install the bash, zsh and fish
scripts for you; PowerShell has no system-wide completion directory, so its
script ships in the archives instead.

```bash
# bash — current session
source <(cchef completion bash)

# zsh — install for the current user
cchef completion zsh > "${fpath[1]}/_cchef"

# fish
cchef completion fish > ~/.config/fish/completions/cchef.fish
```

```powershell
# PowerShell — add to $PROFILE to load every session
cchef completion powershell | Out-String | Invoke-Expression
```

## Verifying a release

Release archives are checksummed, the checksums file is signed with
[Sigstore cosign](https://www.sigstore.dev/) (keyless), and every artifact
carries [SLSA build provenance](https://slsa.dev/). To verify a download:

```bash
# 1. Provenance — that the artifact was built by this repo's release workflow:
gh attestation verify cchef_1.0.0_linux_amd64.tar.gz --repo roberson-io/cchef

# 2. Checksum — that your download matches what was released:
sha256sum -c checksums.txt --ignore-missing

# 3. Signature — that the checksums file itself is authentic:
cosign verify-blob checksums.txt \
  --signature checksums.txt.sig \
  --certificate checksums.txt.pem \
  --certificate-identity-regexp '^https://github.com/roberson-io/cchef' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

A CycloneDX SBOM is published alongside each archive, named after it with a
`.cyclonedx.sbom.json` suffix.

## Uninstalling

| Installed with | Remove with |
| --- | --- |
| Homebrew | `brew uninstall cchef` |
| Scoop | `scoop uninstall cchef` |
| `.deb` | `sudo apt remove cchef` (or `sudo dpkg -r cchef`) |
| `.rpm` | `sudo dnf remove cchef` (or `sudo rpm -e cchef`) |
| archive / `go install` / source | delete the binary (`which cchef` finds it) |

The staged recipe file `.cchef-recipe.json` lives in whatever directory you
created it in, and the optional config file at
`$XDG_CONFIG_HOME/cchef/config.yaml` — remove those by hand if you used them.
