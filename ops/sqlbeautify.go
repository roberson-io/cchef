package ops

import (
	"slices"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/jsnum"
	"github.com/roberson-io/cchef/internal/opsutil"
)

func init() {
	core.Register(SQLBeautify{})
}

// SQLBeautify indents and prettifies SQL. Ported from CyberChef SQLBeautify.mjs,
// which wraps the sql-formatter npm library with a fixed configuration (MySQL
// dialect, "standard" indent style, keywordCase "preserve") and a bind-variable
// placeholder shuffle. cchef reimplements sql-formatter's tokenizer and layout
// formatter (the WS/Layout/Indentation engine follows sql-formatter's);
// keyword categories that drive the layout are curated for the MySQL dialect.
type SQLBeautify struct{}

// Meta returns the operation metadata.
func (SQLBeautify) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "SQL Beautify",
		Module:      "Code",
		Description: "Indents and prettifies Structured Query Language (SQL) code.",
		InfoURL:     "https://wikipedia.org/wiki/SQL",
		InputType:   core.TypeString,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (SQLBeautify) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Indent string", Type: core.ArgString, Value: `\t`},
	}
}

// Run formats the SQL input.
func (SQLBeautify) Run(in *core.Dish, args []any) (*core.Dish, error) {
	indentStr := opsutil.ParseEscapedChars(args[0].(string))
	// The .mjs passes useTabs=(indentStr==="\t") and tabWidth=indentStr.length||4
	// to sql-formatter, whose indent unit is a tab or that many spaces.
	indent := "\t"
	if indentStr != "\t" {
		width := len([]rune(indentStr))
		if width == 0 {
			width = 4
		}
		indent = strings.Repeat(" ", width)
	}
	// Match the .mjs: extract bind variables (:name) to placeholders, format,
	// then restore them, so sql-formatter treats them as plain identifiers.
	placeholder, binds := extractSQLBinds(in.String())
	formatted := formatSQL(placeholder, indent)
	for ph, orig := range binds {
		formatted = strings.ReplaceAll(formatted, ph, orig)
	}
	return core.NewDish([]byte(formatted), core.TypeString), nil
}

// ===== bind-variable extraction (mirrors the .mjs :\w+ -> __BIND_N__ shuffle) =====

func extractSQLBinds(s string) (string, map[string]string) {
	binds := map[string]string{}
	var b strings.Builder
	n := 0
	for i := 0; i < len(s); {
		if s[i] == ':' && i+1 < len(s) && isSQLWord(s[i+1]) {
			j := i + 1
			for j < len(s) && isSQLWord(s[j]) {
				j++
			}
			ph := "__BIND_" + strconv.Itoa(n) + "__"
			binds[ph] = s[i:j]
			b.WriteString(ph)
			n++
			i = j
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String(), binds
}

func isSQLWord(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

// ===== tokenizer =====

type sqlTokKind int

const (
	tkWord     sqlTokKind = iota // identifier or keyword
	tkString                     // quoted string / identifier
	tkNumber                     // numeric literal
	tkOperator                   // = < > + - . etc.
	tkPunct                      // ( ) , ;
	tkComment                    // -- ... or /* ... */
)

type sqlTok struct {
	kind        sqlTokKind
	text        string // original text
	up          string // uppercased (word matching)
	spaceBefore bool   // whitespace (or start) preceded this token
}

// tokenizeSQL splits SQL into tokens, recording whether whitespace preceded each.
func tokenizeSQL(s string) []sqlTok {
	var toks []sqlTok
	i, n := 0, len(s)
	space := true
	for i < n {
		if isSQLSpace(s[i]) {
			space = true
			i++
			continue
		}
		kind, end := scanSQLToken(s, i)
		text := s[i:end]
		toks = append(toks, sqlTok{kind: kind, text: text, up: strings.ToUpper(text), spaceBefore: space})
		space = false
		i = end
	}
	return toks
}

func isSQLSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\f'
}

// scanSQLToken returns the kind and end index of the token at s[i].
func scanSQLToken(s string, i int) (sqlTokKind, int) {
	if end, ok := scanSQLComment(s, i); ok {
		return tkComment, end
	}
	c := s[i]
	switch {
	case c == '\'' || c == '"' || c == '`':
		return tkString, sqlScanQuoted(s, i, c)
	case isSQLNumberStart(s, i):
		return tkNumber, sqlScanNumber(s, i)
	case isSQLWordStart(c):
		return tkWord, sqlScanWord(s, i)
	case c == '(' || c == ')' || c == ',' || c == ';':
		return tkPunct, i + 1
	default:
		return tkOperator, sqlScanOperator(s, i)
	}
}

// scanSQLComment returns the end index of a -- / # line comment or /* */ block
// comment at s[i], and whether one was found.
func scanSQLComment(s string, i int) (int, bool) {
	if strings.HasPrefix(s[i:], "--") || s[i] == '#' {
		j := i
		for j < len(s) && s[j] != '\n' {
			j++
		}
		return j, true
	}
	if strings.HasPrefix(s[i:], "/*") {
		if k := strings.Index(s[i+2:], "*/"); k >= 0 {
			return i + 2 + k + 2, true
		}
		return len(s), true
	}
	return 0, false
}

func isSQLNumberStart(s string, i int) bool {
	c := s[i]
	return c >= '0' && c <= '9' ||
		(c == '.' && i+1 < len(s) && s[i+1] >= '0' && s[i+1] <= '9') ||
		sqlNegativeNumber(s, i)
}

func sqlScanWord(s string, i int) int {
	j := i + 1 // the start char (which may be @ or $) is already consumed
	for j < len(s) && isSQLWord(s[j]) {
		j++
	}
	return j
}

func isSQLWordStart(c byte) bool {
	return c == '_' || c == '$' || c == '@' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func sqlScanQuoted(s string, i int, q byte) int {
	n := len(s)
	j := i + 1
	for j < n {
		if s[j] == '\\' && j+1 < n {
			j += 2
			continue
		}
		if s[j] == q {
			if j+1 < n && s[j+1] == q { // doubled quote escape
				j += 2
				continue
			}
			return j + 1
		}
		j++
	}
	return n
}

// sqlNegativeNumber reports whether s[i] is a '-' that introduces a negative
// number literal (immediately followed by a digit or ".digit"), which
// sql-formatter tokenizes as a single number rather than a binary operator.
func sqlNegativeNumber(s string, i int) bool {
	if s[i] != '-' || i+1 >= len(s) {
		return false
	}
	c := s[i+1]
	if c >= '0' && c <= '9' {
		return true
	}
	return c == '.' && i+2 < len(s) && s[i+2] >= '0' && s[i+2] <= '9'
}

func sqlScanNumber(s string, i int) int {
	n := len(s)
	j := i
	if s[j] == '-' {
		j++
	}
	if strings.HasPrefix(strings.ToLower(s[j:]), "0x") {
		j += 2
		for j < n && jsnum.IsHexDigit(s[j]) {
			j++
		}
		return j
	}
	for j < n && (s[j] >= '0' && s[j] <= '9' || s[j] == '.') {
		j++
	}
	return sqlScanExponent(s, j)
}

// sqlScanExponent consumes an optional e[+/-]digits suffix at s[j].
func sqlScanExponent(s string, j int) int {
	n := len(s)
	if j >= n || (s[j] != 'e' && s[j] != 'E') {
		return j
	}
	j++
	if j < n && (s[j] == '+' || s[j] == '-') {
		j++
	}
	for j < n && s[j] >= '0' && s[j] <= '9' {
		j++
	}
	return j
}

// sqlScanOperator consumes a maximal multi-char operator.
func sqlScanOperator(s string, i int) int {
	twos := []string{"<=>", "->>", "->", "<=", ">=", "<>", "!=", "||", "&&", "<<", ">>", ":=", "::"}
	for _, op := range twos {
		if strings.HasPrefix(s[i:], op) {
			return i + len(op)
		}
	}
	return i + 1
}

// ===== keyword categories (MySQL, layout-relevant only; keywordCase is preserve) =====

// set membership uses uppercased, single-space-joined phrases.
func sqlSet(phrases ...string) map[string]bool {
	m := make(map[string]bool, len(phrases))
	for _, p := range phrases {
		m[p] = true
	}
	return m
}

// indentedClauses format their contents on indented following lines.
var sqlIndentedClauses = sqlSet(
	"SELECT", "SELECT ALL", "SELECT DISTINCT", "SELECT DISTINCTROW",
	"WITH", "WITH RECURSIVE", "FROM", "WHERE", "GROUP BY", "HAVING", "WINDOW",
	"PARTITION BY", "ORDER BY", "LIMIT", "OFFSET", "VALUES", "SET",
	"ON DUPLICATE KEY UPDATE",
)

// onelineClauses keep the clause keyword and its content on one line (the union
// of sql-formatter's standard- and tabular-oneline clauses, which are all oneline
// in the standard indent style).
var sqlOnelineClauses = sqlSet(
	"CREATE TABLE", "CREATE TEMPORARY TABLE",
	"UPDATE", "DELETE FROM", "ALTER TABLE", "TRUNCATE TABLE", "TRUNCATE",
	"ADD COLUMN", "ADD", "DROP COLUMN", "DROP TABLE", "RENAME", "RENAME TO",
	"RENAME AS", "RENAME COLUMN",
)

var sqlSetOps = sqlSet(
	"UNION", "UNION ALL", "UNION DISTINCT", "EXCEPT", "EXCEPT ALL", "EXCEPT DISTINCT",
	"INTERSECT", "INTERSECT ALL", "INTERSECT DISTINCT", "MINUS", "MINUS ALL",
)

var sqlJoins = sqlSet(
	"JOIN", "INNER JOIN", "LEFT JOIN", "LEFT OUTER JOIN", "RIGHT JOIN",
	"RIGHT OUTER JOIN", "FULL JOIN", "FULL OUTER JOIN", "CROSS JOIN",
	"NATURAL JOIN", "NATURAL INNER JOIN", "NATURAL LEFT JOIN",
	"NATURAL LEFT OUTER JOIN", "NATURAL RIGHT JOIN", "NATURAL RIGHT OUTER JOIN",
	"STRAIGHT_JOIN",
)

var sqlLogical = sqlSet("AND", "OR", "XOR")

// sqlInsertModifiers are the words INSERT/REPLACE may be followed by in a clause.
var sqlInsertModifiers = sqlSet("LOW_PRIORITY", "DELAYED", "HIGH_PRIORITY", "IGNORE", "INTO")

const sqlMaxPhraseWords = 4

// matchSQLPhrase greedily matches the longest phrase from set starting at word
// token index i, returning the original (space-normalized) text and word count.
func matchSQLPhrase(toks []sqlTok, i int, set map[string]bool) (string, int) {
	best, bestWords := "", 0
	var up, orig []string
	for w := 0; w < sqlMaxPhraseWords && i+w < len(toks); w++ {
		t := toks[i+w]
		if t.kind != tkWord {
			break
		}
		up = append(up, t.up)
		orig = append(orig, t.text)
		if set[strings.Join(up, " ")] {
			best = strings.Join(orig, " ")
			bestWords = w + 1
		}
	}
	return best, bestWords
}

// matchInsertClause matches INSERT/REPLACE plus its modifier words.
func matchInsertClause(toks []sqlTok, i int) (string, int) {
	if toks[i].kind != tkWord || (toks[i].up != "INSERT" && toks[i].up != "REPLACE") {
		return "", 0
	}
	orig := []string{toks[i].text}
	w := 1
	for i+w < len(toks) && toks[i+w].kind == tkWord && sqlInsertModifiers[toks[i+w].up] {
		orig = append(orig, toks[i+w].text)
		w++
	}
	return strings.Join(orig, " "), w
}

// ===== whitespace layout engine (ported from sql-formatter Layout/Indentation) =====

type wsMark int8

const (
	mSpace wsMark = iota
	mNoSpace
	mNoNewline
	mNewline
	mMandNewline
	mIndent
	mSingleIndent
)

type litem struct {
	str   string
	ws    wsMark
	isStr bool
}

func li(s string) litem { return litem{str: s, isStr: true} }
func lw(m wsMark) litem { return litem{ws: m} }

// sqlIndent manages a stack of top-level / block-level indents.
type sqlIndent struct {
	unit  string
	types []bool // true = top-level, false = block-level
}

func (s *sqlIndent) level() int { return len(s.types) }
func (s *sqlIndent) incTop()    { s.types = append(s.types, true) }
func (s *sqlIndent) incBlock()  { s.types = append(s.types, false) }
func (s *sqlIndent) decTop() {
	if n := len(s.types); n > 0 && s.types[n-1] {
		s.types = s.types[:n-1]
	}
}

func (s *sqlIndent) decBlock() {
	for len(s.types) > 0 {
		top := s.types[len(s.types)-1]
		s.types = s.types[:len(s.types)-1]
		if !top {
			break
		}
	}
}

type sqlLayout struct {
	ind           *sqlIndent
	items         []litem
	inline        bool
	width         int
	length        int
	trailingSpace bool
	overflow      bool
	limitMode     bool // LIMIT/OFFSET clause: commas stay inline
}

func (l *sqlLayout) add(items ...litem) {
	for _, it := range items {
		if l.inline {
			l.addToLength(it)
			if l.overflow || l.length > l.width {
				l.overflow = true
				return
			}
		}
		if it.isStr {
			l.items = append(l.items, it)
			continue
		}
		switch it.ws {
		case mSpace:
			l.items = append(l.items, it)
		case mNoSpace:
			l.trimHorizontal()
		case mNoNewline:
			l.trimRemovable()
		case mNewline:
			l.trimHorizontal()
			l.addNewline(mNewline)
		case mMandNewline:
			l.trimHorizontal()
			l.addNewline(mMandNewline)
		case mIndent:
			for i := 0; i < l.ind.level(); i++ {
				l.items = append(l.items, lw(mSingleIndent))
			}
		case mSingleIndent:
			l.items = append(l.items, it)
		}
	}
}

func (l *sqlLayout) addToLength(it litem) {
	if it.isStr {
		l.length += len(it.str)
		l.trailingSpace = false
		return
	}
	switch it.ws {
	case mNewline, mMandNewline:
		l.overflow = true
	case mIndent, mSingleIndent, mSpace:
		if !l.trailingSpace {
			l.length++
			l.trailingSpace = true
		}
	case mNoNewline, mNoSpace:
		if l.trailingSpace {
			l.trailingSpace = false
			l.length--
		}
	}
}

func (l *sqlLayout) lastIsHorizontal() bool {
	if len(l.items) == 0 {
		return false
	}
	last := l.items[len(l.items)-1]
	return !last.isStr && (last.ws == mSpace || last.ws == mSingleIndent)
}

func (l *sqlLayout) trimHorizontal() {
	for l.lastIsHorizontal() {
		l.items = l.items[:len(l.items)-1]
	}
}

func (l *sqlLayout) trimRemovable() {
	for len(l.items) > 0 {
		last := l.items[len(l.items)-1]
		if last.isStr || (last.ws != mSpace && last.ws != mSingleIndent && last.ws != mNewline) {
			break
		}
		l.items = l.items[:len(l.items)-1]
	}
}

func (l *sqlLayout) addNewline(nl wsMark) {
	if len(l.items) == 0 {
		return
	}
	last := l.items[len(l.items)-1]
	if !last.isStr && last.ws == mNewline {
		l.items[len(l.items)-1] = lw(nl)
	} else if !last.isStr && last.ws == mMandNewline {
		// keep mandatory newline
	} else {
		l.items = append(l.items, lw(nl))
	}
}

func (l *sqlLayout) String() string {
	var b strings.Builder
	for _, it := range l.items {
		if it.isStr {
			b.WriteString(it.str)
			continue
		}
		switch it.ws {
		case mSpace:
			b.WriteByte(' ')
		case mNewline, mMandNewline:
			b.WriteByte('\n')
		case mSingleIndent:
			b.WriteString(l.ind.unit)
		}
	}
	return b.String()
}

// ===== parser (tokens -> AST) =====

type sqlNodeKind int

const (
	snClause  sqlNodeKind = iota // text = clause keyword, children
	snOneline                    // oneline clause (keyword + inline content)
	snSetOp                      // set operation
	snJoin                       // JOIN keyword
	snLogical                    // AND / OR / XOR
	snIdent                      // identifier / keyword / literal
	snOperator
	snDot // .
	snComma
	snParen // ( children )
	snFunc  // function call: text = name, children = arguments
	snCase  // CASE ... END: text = CASE kw, text2 = END kw, children = clauses
	snComment
)

type sqlNode struct {
	kind     sqlNodeKind
	text     string
	text2    string // END keyword text for snCase
	children []sqlNode
}

// splitSQLStatements splits the token stream on top-level ';'.
type sqlStatement struct {
	toks         []sqlTok
	hasSemicolon bool
}

func splitSQLStatements(toks []sqlTok) []sqlStatement {
	var stmts []sqlStatement
	depth, start := 0, 0
	for i, t := range toks {
		if t.kind == tkPunct {
			switch t.text {
			case "(":
				depth++
			case ")":
				depth--
			case ";":
				if depth == 0 {
					stmts = append(stmts, sqlStatement{toks: toks[start:i], hasSemicolon: true})
					start = i + 1
				}
			}
		}
	}
	if start < len(toks) {
		stmts = append(stmts, sqlStatement{toks: toks[start:], hasSemicolon: false})
	}
	return stmts
}

// matchClauseAt matches a clause keyword phrase at word index i.
func matchClauseAt(toks []sqlTok, i int) (name string, words int, oneline bool) {
	if name, words = matchInsertClause(toks, i); words > 0 {
		return name, words, false
	}
	if name, words = matchSQLPhrase(toks, i, sqlOnelineClauses); words > 0 {
		return name, words, true
	}
	if name, words = matchSQLPhrase(toks, i, sqlIndentedClauses); words > 0 {
		return name, words, false
	}
	return "", 0, false
}

// nextClauseBoundary returns the index where the current clause's children end:
// the next clause/set-operation keyword at paren depth 0, or the end.
func nextClauseBoundary(toks []sqlTok, start int) int {
	depth := 0
	for i := start; i < len(toks); i++ {
		t := toks[i]
		if t.kind == tkPunct {
			switch t.text {
			case "(":
				depth++
			case ")":
				depth--
			}
			continue
		}
		if depth == 0 && t.kind == tkWord {
			if _, w, _ := matchClauseAt(toks, i); w > 0 {
				return i
			}
			if _, w := matchSQLPhrase(toks, i, sqlSetOps); w > 0 {
				return i
			}
		}
	}
	return len(toks)
}

// matchCaseEnd returns the index of the END keyword matching the CASE at caseIdx.
func matchCaseEnd(toks []sqlTok, caseIdx int) int {
	depth := 0
	for i := caseIdx + 1; i < len(toks); i++ {
		if toks[i].kind == tkWord {
			switch toks[i].up {
			case "CASE":
				depth++
			case "END":
				if depth == 0 {
					return i
				}
				depth--
			}
		}
	}
	return len(toks)
}

// matchParen returns the index of the ')' matching the '(' at open.
func matchParen(toks []sqlTok, open int) int {
	depth := 0
	for i := open; i < len(toks); i++ {
		if toks[i].kind == tkPunct {
			switch toks[i].text {
			case "(":
				depth++
			case ")":
				depth--
				if depth == 0 {
					return i
				}
			}
		}
	}
	return len(toks)
}

func parseSequence(toks []sqlTok) []sqlNode {
	var nodes []sqlNode
	i := 0
	for i < len(toks) {
		if name, w, oneline := matchClauseAt(toks, i); w > 0 {
			end := nextClauseBoundary(toks, i+w)
			kind := snClause
			if oneline {
				kind = snOneline
			}
			nodes = append(nodes, sqlNode{kind: kind, text: name, children: parseExpression(toks[i+w : end])})
			i = end
		} else if name, w := matchSQLPhrase(toks, i, sqlSetOps); w > 0 {
			end := nextClauseBoundary(toks, i+w)
			nodes = append(nodes, sqlNode{kind: snSetOp, text: name, children: parseExpression(toks[i+w : end])})
			i = end
		} else {
			// Leading expression before any clause (e.g. a parenthesized query).
			// Scan from i so parens before the next clause keyword are respected.
			end := nextClauseBoundary(toks, i)
			if end <= i {
				end = i + 1
			}
			nodes = append(nodes, parseExpression(toks[i:end])...)
			i = end
		}
	}
	return nodes
}

func parseExpression(toks []sqlTok) []sqlNode {
	p := &sqlExprParser{toks: toks}
	for i := 0; i < len(toks); {
		i = p.step(i)
	}
	return p.nodes
}

type sqlExprParser struct {
	toks           []sqlTok
	nodes          []sqlNode
	betweenPending bool
}

func (p *sqlExprParser) push(n sqlNode) { p.nodes = append(p.nodes, n) }

func (p *sqlExprParser) step(i int) int {
	t := p.toks[i]
	switch t.kind {
	case tkPunct:
		return p.parsePunct(i)
	case tkWord:
		return p.parseWord(i)
	case tkOperator:
		if t.text == "." {
			p.push(sqlNode{kind: snDot})
		} else {
			p.push(sqlNode{kind: snOperator, text: t.text})
		}
	case tkComment:
		p.push(sqlNode{kind: snComment, text: t.text})
	default: // tkString, tkNumber
		p.push(sqlNode{kind: snIdent, text: t.text})
	}
	return i + 1
}

func (p *sqlExprParser) parsePunct(i int) int {
	t := p.toks[i]
	switch t.text {
	case "(":
		end := matchParen(p.toks, i)
		inner := parseSequence(p.toks[i+1 : end])
		if n := len(p.nodes); n > 0 && p.nodes[n-1].kind == snIdent && !t.spaceBefore && isSQLFuncName(p.nodes[n-1].text) {
			p.nodes[n-1] = sqlNode{kind: snFunc, text: p.nodes[n-1].text, children: inner}
		} else {
			p.push(sqlNode{kind: snParen, children: inner})
		}
		return end + 1
	case ",":
		p.push(sqlNode{kind: snComma})
	}
	return i + 1
}

func (p *sqlExprParser) parseWord(i int) int {
	t := p.toks[i]
	if t.up == "CASE" {
		end := matchCaseEnd(p.toks, i)
		endText := "END"
		if end < len(p.toks) {
			endText = p.toks[end].text
		}
		p.push(sqlNode{kind: snCase, text: t.text, text2: endText, children: parseExpression(p.toks[i+1 : end])})
		return end + 1
	}
	if name, w := matchSQLPhrase(p.toks, i, sqlJoins); w > 0 {
		p.push(sqlNode{kind: snJoin, text: name})
		return i + w
	}
	if sqlLogical[t.up] {
		if p.betweenPending && t.up == "AND" {
			p.push(sqlNode{kind: snIdent, text: t.text})
			p.betweenPending = false
		} else {
			p.push(sqlNode{kind: snLogical, text: t.text})
		}
		return i + 1
	}
	if t.up == "BETWEEN" {
		p.betweenPending = true
	}
	p.push(sqlNode{kind: snIdent, text: t.text})
	return i + 1
}

// sqlFunctions is the MySQL function-name set. A name in this set that is
// immediately followed by '(' is a function call (no space before the paren);
// any other identifier followed by '(' keeps a space.
var sqlFunctions = map[string]bool{
	"ABS": true, "ACOS": true, "ADDDATE": true, "ADDTIME": true, "AES_DECRYPT": true, "AES_ENCRYPT": true, "ANY_VALUE": true, "ASCII": true,
	"ASIN": true, "ATAN": true, "ATAN2": true, "AVG": true, "BENCHMARK": true, "BIN": true, "BINARY": true, "BIN_TO_UUID": true,
	"BIT_AND": true, "BIT_COUNT": true, "BIT_LENGTH": true, "BIT_OR": true, "BIT_XOR": true, "CAN_ACCESS_COLUMN": true, "CAN_ACCESS_DATABASE": true, "CAN_ACCESS_TABLE": true,
	"CAN_ACCESS_USER": true, "CAN_ACCESS_VIEW": true, "CAST": true, "CEIL": true, "CEILING": true, "CHAR": true, "CHARACTER_LENGTH": true, "CHARSET": true,
	"CHAR_LENGTH": true, "COALESCE": true, "COERCIBILITY": true, "COLLATION": true, "COMPRESS": true, "CONCAT": true, "CONCAT_WS": true, "CONNECTION_ID": true,
	"CONV": true, "CONVERT": true, "CONVERT_TZ": true, "COS": true, "COT": true, "COUNT": true, "CRC32": true, "CUME_DIST": true,
	"CURDATE": true, "CURRENT_DATE": true, "CURRENT_ROLE": true, "CURRENT_TIME": true, "CURRENT_TIMESTAMP": true, "CURRENT_USER": true, "CURTIME": true, "DATABASE": true,
	"DATE": true, "DATEDIFF": true, "DATE_ADD": true, "DATE_FORMAT": true, "DATE_SUB": true, "DAY": true, "DAYNAME": true, "DAYOFMONTH": true,
	"DAYOFWEEK": true, "DAYOFYEAR": true, "DEFAULT": true, "DEGREES": true, "DENSE_RANK": true, "DIV": true, "ELT": true, "EXP": true,
	"EXPORT_SET": true, "EXTRACT": true, "EXTRACTVALUE": true, "FIELD": true, "FIND_IN_SET": true, "FIRST_VALUE": true, "FLOOR": true, "FORMAT": true,
	"FORMAT_BYTES": true, "FORMAT_PICO_TIME": true, "FOUND_ROWS": true, "FROM_BASE64": true, "FROM_DAYS": true, "FROM_UNIXTIME": true, "GEOMCOLLECTION": true, "GEOMETRYCOLLECTION": true,
	"GET_DD_COLUMN_PRIVILEGES": true, "GET_DD_CREATE_OPTIONS": true, "GET_DD_INDEX_SUB_PART_LENGTH": true, "GET_FORMAT": true, "GET_LOCK": true, "GREATEST": true, "GROUPING": true, "GROUP_CONCAT": true,
	"GTID_SUBSET": true, "GTID_SUBTRACT": true, "HEX": true, "HOUR": true, "ICU_VERSION": true, "IF": true, "IFNULL": true, "INET6_ATON": true,
	"INET6_NTOA": true, "INET_ATON": true, "INET_NTOA": true, "INSERT": true, "INSTR": true, "INTERNAL_AUTO_INCREMENT": true, "INTERNAL_AVG_ROW_LENGTH": true, "INTERNAL_CHECKSUM": true,
	"INTERNAL_CHECK_TIME": true, "INTERNAL_DATA_FREE": true, "INTERNAL_DATA_LENGTH": true, "INTERNAL_DD_CHAR_LENGTH": true, "INTERNAL_GET_COMMENT_OR_ERROR": true, "INTERNAL_GET_ENABLED_ROLE_JSON": true, "INTERNAL_GET_HOSTNAME": true, "INTERNAL_GET_USERNAME": true,
	"INTERNAL_GET_VIEW_WARNING_OR_ERROR": true, "INTERNAL_INDEX_COLUMN_CARDINALITY": true, "INTERNAL_INDEX_LENGTH": true, "INTERNAL_IS_ENABLED_ROLE": true, "INTERNAL_IS_MANDATORY_ROLE": true, "INTERNAL_KEYS_DISABLED": true, "INTERNAL_MAX_DATA_LENGTH": true, "INTERNAL_TABLE_ROWS": true,
	"INTERNAL_UPDATE_TIME": true, "INTERVAL": true, "IS": true, "ISNULL": true, "IS_FREE_LOCK": true, "IS_IPV4": true, "IS_IPV4_COMPAT": true, "IS_IPV4_MAPPED": true,
	"IS_IPV6": true, "IS_USED_LOCK": true, "IS_UUID": true, "JSON_ARRAY": true, "JSON_ARRAYAGG": true, "JSON_ARRAY_APPEND": true, "JSON_ARRAY_INSERT": true, "JSON_CONTAINS": true,
	"JSON_CONTAINS_PATH": true, "JSON_DEPTH": true, "JSON_EXTRACT": true, "JSON_INSERT": true, "JSON_KEYS": true, "JSON_LENGTH": true, "JSON_MERGE": true, "JSON_MERGE_PATCH": true,
	"JSON_MERGE_PRESERVE": true, "JSON_OBJECT": true, "JSON_OBJECTAGG": true, "JSON_OVERLAPS": true, "JSON_PRETTY": true, "JSON_QUOTE": true, "JSON_REMOVE": true, "JSON_REPLACE": true,
	"JSON_SCHEMA_VALID": true, "JSON_SCHEMA_VALIDATION_REPORT": true, "JSON_SEARCH": true, "JSON_SET": true, "JSON_STORAGE_FREE": true, "JSON_STORAGE_SIZE": true, "JSON_TABLE": true, "JSON_TYPE": true,
	"JSON_UNQUOTE": true, "JSON_VALID": true, "JSON_VALUE": true, "LAG": true, "LAST_DAY": true, "LAST_INSERT_ID": true, "LAST_VALUE": true, "LCASE": true,
	"LEAD": true, "LEAST": true, "LEFT": true, "LENGTH": true, "LIKE": true, "LINESTRING": true, "LN": true, "LOAD_FILE": true,
	"LOCALTIME": true, "LOCALTIMESTAMP": true, "LOCATE": true, "LOG": true, "LOG10": true, "LOG2": true, "LOWER": true, "LPAD": true,
	"LTRIM": true, "MAKEDATE": true, "MAKETIME": true, "MAKE_SET": true, "MASTER_POS_WAIT": true, "MATCH": true, "MAX": true, "MBRCONTAINS": true,
	"MBRCOVEREDBY": true, "MBRCOVERS": true, "MBRDISJOINT": true, "MBREQUALS": true, "MBRINTERSECTS": true, "MBROVERLAPS": true, "MBRTOUCHES": true, "MBRWITHIN": true,
	"MD5": true, "MICROSECOND": true, "MID": true, "MIN": true, "MINUTE": true, "MOD": true, "MONTH": true, "MONTHNAME": true,
	"MULTILINESTRING": true, "MULTIPOINT": true, "MULTIPOLYGON": true, "NAME_CONST": true, "NOW": true, "NTH_VALUE": true, "NTILE": true, "NULLIF": true,
	"OCT": true, "OCTET_LENGTH": true, "ORD": true, "PERCENT_RANK": true, "PERIOD_ADD": true, "PERIOD_DIFF": true, "PI": true, "POINT": true,
	"POLYGON": true, "POSITION": true, "POW": true, "POWER": true, "PS_CURRENT_THREAD_ID": true, "PS_THREAD_ID": true, "QUARTER": true, "QUOTE": true,
	"RADIANS": true, "RAND": true, "RANDOM_BYTES": true, "RANK": true, "REGEXP": true, "REGEXP_INSTR": true, "REGEXP_LIKE": true, "REGEXP_REPLACE": true,
	"REGEXP_SUBSTR": true, "RELEASE_ALL_LOCKS": true, "RELEASE_LOCK": true, "REPEAT": true, "REPLACE": true, "REVERSE": true, "RIGHT": true, "RLIKE": true,
	"ROLES_GRAPHML": true, "ROUND": true, "ROW_COUNT": true, "ROW_NUMBER": true, "RPAD": true, "RTRIM": true, "SCHEMA": true, "SECOND": true,
	"SEC_TO_TIME": true, "SESSION_USER": true, "SHA1": true, "SHA2": true, "SIGN": true, "SIN": true, "SLEEP": true, "SOUNDEX": true,
	"SOURCE_POS_WAIT": true, "SPACE": true, "SQRT": true, "STATEMENT_DIGEST": true, "STATEMENT_DIGEST_TEXT": true, "STD": true, "STDDEV": true, "STDDEV_POP": true,
	"STDDEV_SAMP": true, "STRCMP": true, "STR_TO_DATE": true, "ST_AREA": true, "ST_ASBINARY": true, "ST_ASGEOJSON": true, "ST_ASTEXT": true, "ST_BUFFER": true,
	"ST_BUFFER_STRATEGY": true, "ST_CENTROID": true, "ST_COLLECT": true, "ST_CONTAINS": true, "ST_CONVEXHULL": true, "ST_CROSSES": true, "ST_DIFFERENCE": true, "ST_DIMENSION": true,
	"ST_DISJOINT": true, "ST_DISTANCE": true, "ST_DISTANCE_SPHERE": true, "ST_ENDPOINT": true, "ST_ENVELOPE": true, "ST_EQUALS": true, "ST_EXTERIORRING": true, "ST_FRECHETDISTANCE": true,
	"ST_GEOHASH": true, "ST_GEOMCOLLFROMTEXT": true, "ST_GEOMCOLLFROMWKB": true, "ST_GEOMETRYN": true, "ST_GEOMETRYTYPE": true, "ST_GEOMFROMGEOJSON": true, "ST_GEOMFROMTEXT": true, "ST_GEOMFROMWKB": true,
	"ST_HAUSDORFFDISTANCE": true, "ST_INTERIORRINGN": true, "ST_INTERSECTION": true, "ST_INTERSECTS": true, "ST_ISCLOSED": true, "ST_ISEMPTY": true, "ST_ISSIMPLE": true, "ST_ISVALID": true,
	"ST_LATFROMGEOHASH": true, "ST_LATITUDE": true, "ST_LENGTH": true, "ST_LINEFROMTEXT": true, "ST_LINEFROMWKB": true, "ST_LINEINTERPOLATEPOINT": true, "ST_LINEINTERPOLATEPOINTS": true, "ST_LONGFROMGEOHASH": true,
	"ST_LONGITUDE": true, "ST_MAKEENVELOPE": true, "ST_MLINEFROMTEXT": true, "ST_MLINEFROMWKB": true, "ST_MPOINTFROMTEXT": true, "ST_MPOINTFROMWKB": true, "ST_MPOLYFROMTEXT": true, "ST_MPOLYFROMWKB": true,
	"ST_NUMGEOMETRIES": true, "ST_NUMINTERIORRING": true, "ST_NUMPOINTS": true, "ST_OVERLAPS": true, "ST_POINTATDISTANCE": true, "ST_POINTFROMGEOHASH": true, "ST_POINTFROMTEXT": true, "ST_POINTFROMWKB": true,
	"ST_POINTN": true, "ST_POLYFROMTEXT": true, "ST_POLYFROMWKB": true, "ST_SIMPLIFY": true, "ST_SRID": true, "ST_STARTPOINT": true, "ST_SWAPXY": true, "ST_SYMDIFFERENCE": true,
	"ST_TOUCHES": true, "ST_TRANSFORM": true, "ST_UNION": true, "ST_VALIDATE": true, "ST_WITHIN": true, "ST_X": true, "ST_Y": true, "SUBDATE": true,
	"SUBSTR": true, "SUBSTRING": true, "SUBSTRING_INDEX": true, "SUBTIME": true, "SUM": true, "SYSDATE": true, "SYSTEM_USER": true, "TAN": true,
	"TIME": true, "TIMEDIFF": true, "TIMESTAMP": true, "TIMESTAMPADD": true, "TIMESTAMPDIFF": true, "TIME_FORMAT": true, "TIME_TO_SEC": true, "TO_BASE64": true,
	"TO_DAYS": true, "TO_SECONDS": true, "TRIM": true, "TRUNCATE": true, "UCASE": true, "UNCOMPRESS": true, "UNCOMPRESSED_LENGTH": true, "UNHEX": true,
	"UNIX_TIMESTAMP": true, "UPDATEXML": true, "UPPER": true, "USER": true, "UTC_DATE": true, "UTC_TIME": true, "UTC_TIMESTAMP": true, "UUID": true,
	"UUID_SHORT": true, "UUID_TO_BIN": true, "VALIDATE_PASSWORD_STRENGTH": true, "VALUES": true, "VARIANCE": true, "VAR_POP": true, "VAR_SAMP": true, "VERSION": true,
	"WAIT_FOR_EXECUTED_GTID_SET": true, "WAIT_UNTIL_SQL_THREAD_AFTER_GTIDS": true, "WEEK": true, "WEEKDAY": true, "WEEKOFYEAR": true, "WEIGHT_STRING": true, "YEAR": true, "YEARWEEK": true,
}

// sqlDataTypes are MySQL data-type names; like functions, a following '(' (as
// in VARCHAR(255)) is attached without a space.
var sqlDataTypes = map[string]bool{
	"BIGINT": true, "BINARY": true, "BIT": true, "BLOB": true, "BOOL": true, "BOOLEAN": true, "CHAR": true, "CHARACTER": true,
	"DATE": true, "DATETIME": true, "DEC": true, "DECIMAL": true, "DOUBLE": true, "ENUM": true, "FIXED": true, "FLOAT": true,
	"FLOAT4": true, "FLOAT8": true, "INT": true, "INT1": true, "INT2": true, "INT3": true, "INT4": true, "INT8": true,
	"INTEGER": true, "LONGBLOB": true, "LONGTEXT": true, "MEDIUMBLOB": true, "MEDIUMINT": true, "MEDIUMTEXT": true, "MIDDLEINT": true, "NUMERIC": true,
	"PRECISION": true, "REAL": true, "SET": true, "SMALLINT": true, "TEXT": true, "TIME": true, "TIMESTAMP": true, "TINYBLOB": true,
	"TINYINT": true, "TINYTEXT": true, "VARBINARY": true, "VARCHAR": true, "VARCHARACTER": true, "VARYING": true, "YEAR": true,
}

// isSQLFuncName reports whether a name is a known MySQL function or data type, in
// which case an immediately following '(' is attached without a space.
func isSQLFuncName(name string) bool {
	up := strings.ToUpper(name)
	return sqlFunctions[up] || sqlDataTypes[up]
}

// ===== formatter (AST -> layout, reproducing sql-formatter's standard style) =====

const sqlExpressionWidth = 50

func formatSQL(input, indentUnit string) string {
	toks := tokenizeSQL(input)
	var parts []string
	for _, st := range splitSQLStatements(toks) {
		if len(st.toks) == 0 && !st.hasSemicolon {
			continue
		}
		nodes := parseSequence(st.toks)
		l := &sqlLayout{ind: &sqlIndent{unit: indentUnit}, width: sqlExpressionWidth}
		formatNodes(nodes, l)
		if st.hasSemicolon {
			l.add(lw(mNoNewline), li(";"))
		}
		parts = append(parts, l.String())
	}
	return strings.TrimRight(strings.Join(parts, "\n\n"), " \t\r\n")
}

func formatNodes(nodes []sqlNode, l *sqlLayout) {
	for i := range nodes {
		formatNode(nodes[i], l)
		if l.inline && l.overflow {
			return
		}
	}
}

func formatNode(nd sqlNode, l *sqlLayout) {
	switch nd.kind {
	case snClause:
		l.add(lw(mNewline), lw(mIndent), li(nd.text), lw(mNewline))
		l.ind.incTop()
		l.add(lw(mIndent))
		limit := strings.EqualFold(nd.text, "LIMIT") || strings.EqualFold(nd.text, "OFFSET")
		prev := l.limitMode
		l.limitMode = limit
		formatNodes(nd.children, l)
		l.limitMode = prev
		l.ind.decTop()
	case snOneline:
		l.add(lw(mNewline), lw(mIndent), li(nd.text), lw(mSpace))
		formatNodes(nd.children, l)
	case snSetOp:
		l.add(lw(mNewline), lw(mIndent), li(nd.text), lw(mNewline), lw(mIndent))
		formatNodes(nd.children, l)
	case snJoin, snLogical:
		l.add(lw(mNewline), lw(mIndent), li(nd.text), lw(mSpace))
	case snIdent, snOperator:
		l.add(li(nd.text), lw(mSpace))
	case snDot:
		l.add(lw(mNoSpace), li("."))
	case snComma:
		if l.inline || l.limitMode {
			l.add(lw(mNoSpace), li(","), lw(mSpace))
		} else {
			l.add(lw(mNoSpace), li(","), lw(mNewline), lw(mIndent))
		}
	case snParen:
		formatParenChildren(nd.children, l)
	case snFunc:
		l.add(li(nd.text))
		formatParenChildren(nd.children, l)
	case snCase:
		formatSQLCase(nd, l)
	case snComment:
		formatSQLComment(nd.text, l)
	}
}

// formatSQLCase lays out a CASE ... WHEN ... THEN ... ELSE ... END expression.
func formatSQLCase(nd sqlNode, l *sqlLayout) {
	l.add(li(nd.text), lw(mSpace))
	l.ind.incBlock()
	for i := range nd.children {
		c := nd.children[i]
		if c.kind == snIdent {
			switch strings.ToUpper(c.text) {
			case "WHEN", "ELSE":
				l.add(lw(mNewline), lw(mIndent), li(c.text), lw(mSpace))
				continue
			case "THEN":
				l.add(li(c.text), lw(mSpace))
				continue
			}
		}
		formatNode(c, l)
	}
	l.ind.decBlock()
	l.add(lw(mNewline), lw(mIndent), li(nd.text2), lw(mSpace))
}

func formatParenChildren(children []sqlNode, l *sqlLayout) {
	if items, ok := formatSQLInline(children); ok {
		l.add(li("("))
		l.add(items...)
		l.add(lw(mNoSpace), li(")"), lw(mSpace))
		return
	}
	l.add(li("("), lw(mNewline))
	l.ind.incBlock()
	l.add(lw(mIndent))
	formatNodes(children, l)
	l.ind.decBlock()
	l.add(lw(mNewline), lw(mIndent), li(")"), lw(mSpace))
}

// formatSQLInline attempts to lay out nodes on a single line within the
// expression width; returns (items, false) when it doesn't fit.
func formatSQLInline(nodes []sqlNode) ([]litem, bool) {
	il := &sqlLayout{ind: &sqlIndent{unit: ""}, inline: true, width: sqlExpressionWidth}
	formatNodes(nodes, il)
	if il.overflow {
		return nil, false
	}
	return il.items, true
}

func formatSQLComment(text string, l *sqlLayout) {
	if strings.HasPrefix(text, "--") || strings.HasPrefix(text, "#") {
		if len(l.items) == 0 {
			l.add(li(text), lw(mMandNewline), lw(mIndent))
		} else {
			l.add(lw(mNoNewline), lw(mSpace), li(text), lw(mMandNewline), lw(mIndent))
		}
		return
	}
	// Block comment: standalone (on its own line) when multi-line or at the start
	// of a line, otherwise inline.
	if strings.Contains(text, "\n") || l.isAtStartOfLine() {
		for line := range strings.SplitSeq(text, "\n") {
			l.add(lw(mNewline), lw(mIndent), li(strings.TrimLeft(line, " \t")))
		}
		l.add(lw(mNewline), lw(mIndent))
		return
	}
	l.add(li(text), lw(mSpace))
}

func (l *sqlLayout) isAtStartOfLine() bool {
	for _, it := range slices.Backward(l.items) {
		if !it.isStr && it.ws == mSingleIndent {
			continue
		}
		return !it.isStr && (it.ws == mNewline || it.ws == mMandNewline)
	}
	return false
}
