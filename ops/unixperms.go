package ops

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(ParseUNIXFilePermissions{})
}

// unixPerms holds the parsed permission bits.
type unixPerms struct {
	d, sl, np, s, cd, bd, dr bool // file types
	sb, su, sg               bool // sticky bit, setuid, setgid
	ru, wu, eu               bool // user read/write/execute
	rg, wg, eg               bool // group
	ro, wo, eo               bool // other
}

var (
	reOctalPerms   = regexp.MustCompile(`^\s*([0-7]{1,4})\s*`)
	reTextualPerms = regexp.MustCompile(`^\s*([dlpcbDrwxsStT-]{1,10})\s*`)
)

// ParseUNIXFilePermissions describes a UNIX permission string (octal or textual).
type ParseUNIXFilePermissions struct{}

// Meta returns the operation metadata.
func (ParseUNIXFilePermissions) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Parse UNIX file permissions",
		Module:      "Default",
		Description: "Parses a UNIX file permission string (octal e.g. 755, or textual e.g. drwxr-xr-x) and describes it.",
		InfoURL:     "https://wikipedia.org/wiki/File_system_permissions#Traditional_Unix_permissions",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (ParseUNIXFilePermissions) Args() []core.ArgDef { return nil }

// Run parses the permissions.
func (ParseUNIXFilePermissions) Run(in *core.Dish, args []any) (*core.Dish, error) {
	input := in.String()
	var p unixPerms
	textual := false

	switch {
	case reOctalPerms.MatchString(input):
		p = parseOctalPerms(reOctalPerms.FindStringSubmatch(input)[1])
	case reTextualPerms.MatchString(input):
		p = parseTextualPerms(reTextualPerms.FindStringSubmatch(input)[1])
		textual = true
	default:
		return nil, fmt.Errorf("invalid input format; enter permissions in octal (e.g. 755) or textual (e.g. drwxr-xr-x) format")
	}

	return core.NewDish([]byte(describePerms(p, textual)), core.TypeString), nil
}

// parseOctalPerms parses an octal permission string (3 or 4 digits) into the
// permission bits; each digit's bits are read/write/execute (4/2/1).
func parseOctalPerms(octal string) unixPerms {
	var p unixPerms
	var d, u, g, o int
	if len(octal) == 4 {
		d, u, g, o = octDigit(octal[0]), octDigit(octal[1]), octDigit(octal[2]), octDigit(octal[3])
	} else {
		if len(octal) > 0 {
			u = octDigit(octal[0])
		}
		if len(octal) > 1 {
			g = octDigit(octal[1])
		}
		if len(octal) > 2 {
			o = octDigit(octal[2])
		}
	}
	p.su, p.sg, p.sb = d>>2&1 == 1, d>>1&1 == 1, d&1 == 1
	p.ru, p.wu, p.eu = u>>2&1 == 1, u>>1&1 == 1, u&1 == 1
	p.rg, p.wg, p.eg = g>>2&1 == 1, g>>1&1 == 1, g&1 == 1
	p.ro, p.wo, p.eo = o>>2&1 == 1, o>>1&1 == 1, o&1 == 1
	return p
}

// applyPermTypeChar sets the file-type bit for the leading character of a
// textual permission string. A regular-file '-' (or anything else) sets nothing.
func applyPermTypeChar(p *unixPerms, c byte) {
	switch c {
	case 'd':
		p.d = true
	case 'l':
		p.sl = true
	case 'p':
		p.np = true
	case 's':
		p.s = true
	case 'c':
		p.cd = true
	case 'b':
		p.bd = true
	case 'D':
		p.dr = true
	}
}

// parseTextualPerms parses a textual permission string (e.g. drwxr-sr-x) into
// the permission bits, including the type character and the setuid/setgid/sticky
// markers in the execute slots.
func parseTextualPerms(t string) unixPerms {
	var p unixPerms
	at := func(i int) byte {
		if i < len(t) {
			return t[i]
		}
		return 0
	}
	applyPermTypeChar(&p, at(0))
	p.ru, p.wu = at(1) == 'r', at(2) == 'w'
	switch at(3) {
	case 'x':
		p.eu = true
	case 's':
		p.eu, p.su = true, true
	case 'S':
		p.su = true
	}
	p.rg, p.wg = at(4) == 'r', at(5) == 'w'
	switch at(6) {
	case 'x':
		p.eg = true
	case 's':
		p.eg, p.sg = true, true
	case 'S':
		p.sg = true
	}
	p.ro, p.wo = at(7) == 'r', at(8) == 'w'
	switch at(9) {
	case 'x':
		p.eo = true
	case 't':
		p.eo, p.sb = true, true
	case 'T':
		p.sb = true
	}
	return p
}

// describePerms renders the permission report: textual and octal forms, the file
// type (textual input only), any special-flag notes, and the permission matrix.
func describePerms(p unixPerms, textual bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Textual representation: %s", permsToStr(p))
	fmt.Fprintf(&b, "\nOctal representation:   %s", permsToOctal(p))
	if textual {
		fmt.Fprintf(&b, "\nFile type: %s", ftFromPerms(p))
	}
	if p.su {
		b.WriteString("\nThe setuid flag is set")
	}
	if p.sg {
		b.WriteString("\nThe setgid flag is set")
	}
	if p.sb {
		b.WriteString("\nThe sticky bit is set")
	}
	b.WriteString(permsMatrix(p))
	return b.String()
}

func octDigit(c byte) int { return int(c - '0') }

func mark(set bool) string {
	if set {
		return "X"
	}
	return " "
}

// permsToStr renders the textual (rwx) representation.
func permsToStr(p unixPerms) string {
	typ := "-"
	switch {
	case p.d:
		typ = "d"
	case p.sl:
		typ = "l"
	case p.np:
		typ = "p"
	case p.s:
		typ = "s"
	case p.cd:
		typ = "c"
	case p.bd:
		typ = "b"
	case p.dr:
		typ = "D"
	}
	triad := func(r, w, e, special bool, lo, hi byte) string {
		s := "-"
		if r {
			s = "r"
		}
		out := s
		if w {
			out += "w"
		} else {
			out += "-"
		}
		switch {
		case e && special:
			out += string(lo)
		case special:
			out += string(hi)
		case e:
			out += "x"
		default:
			out += "-"
		}
		return out
	}
	return typ +
		triad(p.ru, p.wu, p.eu, p.su, 's', 'S') +
		triad(p.rg, p.wg, p.eg, p.sg, 's', 'S') +
		triad(p.ro, p.wo, p.eo, p.sb, 't', 'T')
}

// permsToOctal renders the 4-digit octal representation.
func permsToOctal(p unixPerms) string {
	bit := func(b bool, v int) int {
		if b {
			return v
		}
		return 0
	}
	d := bit(p.su, 4) + bit(p.sg, 2) + bit(p.sb, 1)
	u := bit(p.ru, 4) + bit(p.wu, 2) + bit(p.eu, 1)
	g := bit(p.rg, 4) + bit(p.wg, 2) + bit(p.eg, 1)
	o := bit(p.ro, 4) + bit(p.wo, 2) + bit(p.eo, 1)
	return fmt.Sprintf("%d%d%d%d", d, u, g, o)
}

// ftFromPerms returns the file type description.
func ftFromPerms(p unixPerms) string {
	switch {
	case p.d:
		return "Directory"
	case p.sl:
		return "Symbolic link"
	case p.np:
		return "Named pipe"
	case p.s:
		return "Socket"
	case p.cd:
		return "Character device"
	case p.bd:
		return "Block device"
	case p.dr:
		return "Door"
	default:
		return "Regular file"
	}
}

// permsMatrix renders the read/write/execute matrix.
func permsMatrix(p unixPerms) string {
	return fmt.Sprintf(`

 +---------+-------+-------+-------+
 |         | User  | Group | Other |
 +---------+-------+-------+-------+
 |    Read |   %s   |   %s   |   %s   |
 +---------+-------+-------+-------+
 |   Write |   %s   |   %s   |   %s   |
 +---------+-------+-------+-------+
 | Execute |   %s   |   %s   |   %s   |
 +---------+-------+-------+-------+`,
		mark(p.ru), mark(p.rg), mark(p.ro),
		mark(p.wu), mark(p.wg), mark(p.wo),
		mark(p.eu), mark(p.eg), mark(p.eo))
}
