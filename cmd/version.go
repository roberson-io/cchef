package cmd

// version is the cchef version string. It defaults to a development value and
// can be overridden at build time with:
//
//	go build -ldflags "-X github.com/roberson-io/cchef/cmd.version=1.2.3"
var version = "0.1.0-dev"

// alignedCyberChef is the CyberChef release the operations are aligned with.
// Minor and major versions of cchef track CyberChef's one component down
// (CyberChef v11.4.0 -> cchef v1.1.0); the patch component is cchef's own.
const alignedCyberChef = "11.3.0"
