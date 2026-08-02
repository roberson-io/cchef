package xmldom

import (
	"fmt"
	"strconv"
	"strings"
)

// parsePseudo translates a :pseudo-class (s[i] is the colon) into an XPath
// predicate. name is the compound's element name test (for *-of-type pseudos).
func parsePseudo(s string, i int, name string) (string, int, error) {
	i++ // ':'
	if i < len(s) && s[i] == ':' {
		i++ // pseudo-element "::"; treated like a pseudo-class for translation
	}
	start := i
	for i < len(s) && isCSSNameChar(s[i]) {
		i++
	}
	kind := strings.ToLower(s[start:i])
	arg := ""
	if i < len(s) && s[i] == '(' {
		depth, j := 0, i
		for j < len(s) {
			if s[j] == '(' {
				depth++
			} else if s[j] == ')' {
				depth--
				if depth == 0 {
					break
				}
			}
			j++
		}
		if depth != 0 {
			return "", i, fmt.Errorf("unterminated pseudo-class argument")
		}
		arg = s[i+1 : j]
		i = j + 1
	}
	pred, err := pseudoPredicate(kind, arg, name)
	if err != nil {
		return "", i, err
	}
	return pred, i, nil
}

func pseudoPredicate(kind, arg, name string) (string, error) {
	switch kind {
	case "first-child":
		return "[not(preceding-sibling::*)]", nil
	case "last-child":
		return "[not(following-sibling::*)]", nil
	case "only-child":
		return "[not(preceding-sibling::*) and not(following-sibling::*)]", nil
	case "first-of-type":
		return "[not(preceding-sibling::" + name + ")]", nil
	case "last-of-type":
		return "[not(following-sibling::" + name + ")]", nil
	case "only-of-type":
		return "[not(preceding-sibling::" + name + ") and not(following-sibling::" + name + ")]", nil
	case "empty":
		// :empty ignores comments, CDATA and PIs (nwmatcher keys on child
		// elements and text nodes only).
		return "[not(*) and not(text())]", nil
	case "root":
		return "[not(parent::*)]", nil
	case "link", "visited", "target", "active", "focus", "hover",
		"checked", "disabled", "enabled", "selected":
		// nwmatcher's dynamic pseudo-classes depend on rendering/interaction or
		// DOM properties that xmldom nodes lack, so they match nothing.
		return "[1=0]", nil
	case "nth-child":
		return nthPredicate("preceding-sibling::*", arg)
	case "nth-last-child":
		return nthPredicate("following-sibling::*", arg)
	case "nth-of-type":
		return nthPredicate("preceding-sibling::"+name, arg)
	case "nth-last-of-type":
		return nthPredicate("following-sibling::"+name, arg)
	case "not":
		return notPredicate(arg)
	}
	return "", fmt.Errorf("unsupported pseudo-class :%s", kind)
}

func nthPredicate(axis, arg string) (string, error) {
	a, b, err := parseNth(arg)
	if err != nil {
		return "", err
	}
	pos := "(count(" + axis + ")+1)"
	if a == 0 {
		return "[" + pos + "=" + strconv.Itoa(b) + "]", nil
	}
	diff := "(" + pos + "-" + strconv.Itoa(b) + ")"
	return "[" + diff + " mod " + strconv.Itoa(a) + "=0 and " + diff + " div " + strconv.Itoa(a) + ">=0]", nil
}

// parseNth parses an An+B expression (also "odd"/"even").
func parseNth(arg string) (int, int, error) {
	e := strings.ToLower(strings.ReplaceAll(arg, " ", ""))
	switch e {
	case "odd":
		return 2, 1, nil
	case "even":
		return 2, 0, nil
	case "":
		return 0, 0, fmt.Errorf("empty nth argument")
	}
	if before, after, ok := strings.Cut(e, "n"); ok {
		a, err := nthCoef(before)
		if err != nil {
			return 0, 0, err
		}
		b := 0
		if rest := after; rest != "" {
			b, err = strconv.Atoi(rest)
			if err != nil {
				return 0, 0, err
			}
		}
		return a, b, nil
	}
	b, err := strconv.Atoi(e)
	if err != nil {
		return 0, 0, err
	}
	return 0, b, nil
}

func nthCoef(s string) (int, error) {
	switch s {
	case "", "+":
		return 1, nil
	case "-":
		return -1, nil
	}
	return strconv.Atoi(s)
}

// notPredicate translates :not(simple) into a negated predicate.
func notPredicate(arg string) (string, error) {
	c, err := parseCompound(strings.TrimSpace(arg))
	if err != nil {
		return "", err
	}
	var conds []string
	if c.name != "*" {
		conds = append(conds, "self::"+c.name)
	}
	for _, p := range c.preds {
		conds = append(conds, strings.TrimSuffix(strings.TrimPrefix(p, "["), "]"))
	}
	if len(conds) == 0 {
		conds = append(conds, "self::*")
	}
	return "[not(" + strings.Join(conds, " and ") + ")]", nil
}
