package cmd

import (
	"runtime/debug"
	"strings"
)

// devVersion is what a binary reports when it carries neither a release stamp
// nor a module version — a build with no version control information to derive
// one from, such as one made with -buildvcs=false or from an extracted archive.
// A build from a checkout gets a pseudo-version naming the commit instead.
const devVersion = "dev"

// version is stamped in at release time and is empty in every other build:
//
//	go build -ldflags "-X github.com/roberson-io/cchef/cmd.version=1.2.3"
var version string

// buildVersion is the version this binary reports.
var buildVersion = resolveVersion(version, moduleVersion())

// resolveVersion picks the most authoritative version available. A release
// stamp is exact, so it wins. Without one, the module version the toolchain
// recorded is what `go install module@version` was built from — the version the
// user asked for — and reporting a placeholder instead would name a release
// they did not install.
func resolveVersion(ldflag, module string) string {
	if ldflag != "" {
		return ldflag
	}
	// "(devel)" is what the toolchain records for a build that is not from a
	// published module version, which carries no more information than the
	// placeholder does.
	if module != "" && module != "(devel)" {
		return strings.TrimPrefix(module, "v")
	}
	return devVersion
}

// moduleVersion returns the module version the toolchain recorded in this
// binary, or "" when it recorded none.
func moduleVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return bi.Main.Version
}

// alignedCyberChef is the CyberChef release the operations are aligned with.
// Minor and major versions of cchef track CyberChef's one component down
// (CyberChef v11.4.0 -> cchef v1.1.0); the patch component is cchef's own.
const alignedCyberChef = "11.3.0"
