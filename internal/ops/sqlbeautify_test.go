package ops

import (
	"testing"

	"github.com/roberson-io/cchef/internal/core"
)

// Fixtures transcribed from ../CyberChef/tests/operations/tests/SQLBeautify.mjs.
// CyberChef wraps the sql-formatter npm library with a fixed config (MySQL
// dialect, "standard" indent style, keywordCase preserve) and a bind-variable
// placeholder shuffle; cchef reimplements that formatter from scratch.
func TestSQLBeautifyFixtures(t *testing.T) {
	runCases(t, []opCase{
		{
			"SQL Beautify - basic",
			"SELECT MONTH, ID, RAIN_I, TEMP_F FROM STATS;",
			"SELECT\n  MONTH,\n  ID,\n  RAIN_I,\n  TEMP_F\nFROM\n  STATS;",
			core.Recipe{{Op: "SQL Beautify", Args: []any{"  "}}},
		},
		{
			"SQL Beautify - upsert",
			"INSERT INTO Table1 SELECT * FROM (SELECT :Bind1 as Field1, :Bind2 as Field2, :id as id) as new_data ON DUPLICATE KEY UPDATE Field1 = new_data.Field1, Field2 = new_data.Field2;",
			"INSERT INTO\n  Table1\nSELECT\n  *\nFROM\n  (\n    SELECT\n      :Bind1 as Field1,\n      :Bind2 as Field2,\n      :id as id\n  ) as new_data\nON DUPLICATE KEY UPDATE\n  Field1 = new_data.Field1,\n  Field2 = new_data.Field2;",
			core.Recipe{{Op: "SQL Beautify", Args: []any{"  "}}},
		},
	})
}

// Additional oracle-verified cases (2-space indent) locking the layout rules:
// AND-newline, JOIN, subquery block expansion, CASE, known-vs-unknown function
// spacing, inline LIMIT comma, oneline UPDATE, negative-number tokenization and
// parameterized data types.
func TestSQLBeautifyMore(t *testing.T) {
	rec := core.Recipe{{Op: "SQL Beautify", Args: []any{"  "}}}
	runCases(t, []opCase{
		{
			"where and order", "SELECT * FROM t WHERE a>1 AND b=2 ORDER BY name",
			"SELECT\n  *\nFROM\n  t\nWHERE\n  a > 1\n  AND b = 2\nORDER BY\n  name", rec,
		},
		{
			"join", "SELECT a FROM t1 JOIN t2 ON t1.id=t2.id",
			"SELECT\n  a\nFROM\n  t1\n  JOIN t2 ON t1.id = t2.id", rec,
		},
		{
			"subquery expands", "SELECT * FROM (SELECT a FROM t WHERE a>1) sub",
			"SELECT\n  *\nFROM\n  (\n    SELECT\n      a\n    FROM\n      t\n    WHERE\n      a > 1\n  ) sub", rec,
		},
		{
			"case", "SELECT CASE WHEN x>1 THEN 1 ELSE 0 END FROM t",
			"SELECT\n  CASE\n    WHEN x > 1 THEN 1\n    ELSE 0\n  END\nFROM\n  t", rec,
		},
		{
			"known vs unknown function", "SELECT COUNT(*), myfunc(a) FROM t",
			"SELECT\n  COUNT(*),\n  myfunc (a)\nFROM\n  t", rec,
		},
		{
			"limit inline comma", "SELECT a FROM t LIMIT 5,10",
			"SELECT\n  a\nFROM\n  t\nLIMIT\n  5, 10", rec,
		},
		{
			"update oneline", "UPDATE t SET a=1 WHERE id=2",
			"UPDATE t\nSET\n  a = 1\nWHERE\n  id = 2", rec,
		},
		{
			"negative number", "SELECT a-1 FROM t",
			"SELECT\n  a -1\nFROM\n  t", rec,
		},
		{
			"parameterized data type", "CREATE TABLE t (id INT, n VARCHAR(255))",
			"CREATE TABLE t (id INT, n VARCHAR(255))", rec,
		},
		{
			"string and backtick", "SELECT 'it''s', `col` FROM t",
			"SELECT\n  'it''s',\n  `col`\nFROM\n  t", rec,
		},
		{
			"hex and scientific", "SELECT 0xFF, 3.14e2 FROM t",
			"SELECT\n  0xFF,\n  3.14e2\nFROM\n  t", rec,
		},
		{
			"line comment", "SELECT a -- hi\nFROM t",
			"SELECT\n  a -- hi\nFROM\n  t", rec,
		},
		{
			"standalone block comment", "SELECT /* b */ a FROM t",
			"SELECT\n  /* b */\n  a\nFROM\n  t", rec,
		},
		{
			"multiple statements", "SELECT 1;SELECT 2",
			"SELECT\n  1;\n\nSELECT\n  2", rec,
		},
	})
}

// TestSQLBeautifyIndents covers the indent-unit handling and the '#'/':=' tokens.
func TestSQLBeautifyIndents(t *testing.T) {
	runCases(t, []opCase{
		{
			"tab indent", "SELECT a,b FROM t", "SELECT\n\ta,\n\tb\nFROM\n\tt",
			core.Recipe{{Op: "SQL Beautify", Args: []any{`\t`}}},
		},
		{
			"empty indent is four spaces", "SELECT a,b FROM t",
			"SELECT\n    a,\n    b\nFROM\n    t",
			core.Recipe{{Op: "SQL Beautify", Args: []any{""}}},
		},
		{
			"hash comment", "SELECT a # hi\nFROM t", "SELECT\n  a # hi\nFROM\n  t",
			core.Recipe{{Op: "SQL Beautify", Args: []any{"  "}}},
		},
		{
			"assign operator", "SELECT @x:=5 FROM t", "SELECT\n  @x := 5\nFROM\n  t",
			core.Recipe{{Op: "SQL Beautify", Args: []any{"  "}}},
		},
		// Regression: @/$-prefixed variables must not stall the tokenizer.
		{
			"session and user variables", "SELECT @x:=5, $var FROM t",
			"SELECT\n  @x := 5,\n  $var\nFROM\n  t",
			core.Recipe{{Op: "SQL Beautify", Args: []any{"  "}}},
		},
		{
			"between keeps its and inline", "SELECT * FROM t WHERE x BETWEEN 1 AND 10",
			"SELECT\n  *\nFROM\n  t\nWHERE\n  x BETWEEN 1 AND 10",
			core.Recipe{{Op: "SQL Beautify", Args: []any{"  "}}},
		},
		{
			"set operation", "SELECT a FROM t1 UNION SELECT b FROM t2",
			"SELECT\n  a\nFROM\n  t1\nUNION\nSELECT\n  b\nFROM\n  t2",
			core.Recipe{{Op: "SQL Beautify", Args: []any{"  "}}},
		},
		{
			"nested case", "SELECT CASE WHEN x THEN CASE WHEN y THEN 1 END END FROM t",
			"SELECT\n  CASE\n    WHEN x THEN CASE\n      WHEN y THEN 1\n    END\n  END\nFROM\n  t",
			core.Recipe{{Op: "SQL Beautify", Args: []any{"  "}}},
		},
		{
			"negative float and exponent", "SELECT -1.5, 1e-5 FROM t",
			"SELECT\n  -1.5,\n  1e-5\nFROM\n  t",
			core.Recipe{{Op: "SQL Beautify", Args: []any{"  "}}},
		},
		{
			"empty statement", "SELECT 1;;SELECT 2", "SELECT\n  1;\n\n;\n\nSELECT\n  2",
			core.Recipe{{Op: "SQL Beautify", Args: []any{"  "}}},
		},
	})
}
