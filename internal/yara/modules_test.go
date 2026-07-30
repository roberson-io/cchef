package yara

import (
	"strings"
	"testing"
)

// moduleHolds reports whether a rule using a module held, over given data.
func moduleHolds(t *testing.T, imports, cond, data string) bool {
	t.Helper()
	src := `import "` + imports + `" rule R { condition: ` + cond + ` }`
	return len(scan(t, src, data)) == 1
}

// TestHashModule covers the digests a rule may take, over a stretch of the data
// or over text the rule wrote out itself. The values are the ones CyberChef
// gave for the same input, two of which are its own fixture.
func TestHashModule(t *testing.T) {
	cases := []struct{ cond, data string }{
		{`hash.md5(0, filesize) == "5eb63bbbe01eeed093cb22bb8f5acdc3"`, "hello world"},
		{`hash.sha1(0, filesize) == "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed"`, "hello world"},
		{`hash.sha256(0, filesize) == ` +
			`"b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"`, "hello world"},
		{`hash.crc32(0, filesize) == 0x0d4a1185`, "hello world"},
		{`hash.checksum32(0, filesize) == 1116`, "hello world"},
		// CyberChef's own fixture, over "Hello World!".
		{`hash.md5(0, filesize) == "ed076287532e86365e841e92bfc50d8c"`, "Hello World!"},
		{`hash.sha256(0, filesize) == ` +
			`"7f83b1657ff1fc53b92dc18148a1d65dfc2d4b1fa3d677284addd200126d9069"`, "Hello World!"},
		// Part of the data rather than all of it.
		{`hash.md5(0, 5) == "5d41402abc4b2a76b9719d911017c592"`, "hello world"},
		// Text the rule wrote out, which need not be in the data at all.
		{`hash.md5("hello") == "5d41402abc4b2a76b9719d911017c592"`, "hello world"},
		{`hash.crc32("hello") == 907060870`, "hello world"},
		{`hash.checksum32("hello") == 532`, "hello world"},
		{`hash.sha1("") == "da39a3ee5e6b4b0d3255bfef95601890afd80709"`, "hello world"},
	}
	for _, c := range cases {
		t.Run(c.cond, func(t *testing.T) {
			if !moduleHolds(t, "hash", c.cond, c.data) {
				t.Error("the digest did not come out as CyberChef's")
			}
		})
	}
}

// TestHashModuleBeyondTheData covers a stretch that runs past the end, which is
// cut short rather than coming to nothing, and one that starts before the
// beginning, which has no answer.
func TestHashModuleBeyondTheData(t *testing.T) {
	t.Run("a length past the end is cut short", func(t *testing.T) {
		if !moduleHolds(t, "hash", `hash.md5(0, 100) == hash.md5(0, filesize)`, "hello world") {
			t.Error("a stretch running past the end did not stop at it")
		}
	})
	for _, cond := range []string{
		`defined hash.md5(-1, 1)`,
		`defined hash.md5(0, -1)`,
		`defined hash.crc32(-1, 1)`,
		`defined hash.checksum32(0, -1)`,
	} {
		t.Run(cond, func(t *testing.T) {
			if moduleHolds(t, "hash", cond, "hello world") {
				t.Error("took a digest over data that is not there")
			}
		})
	}
}

// TestMathModule covers the numbers a rule may work out about the data. Each
// answer is the one CyberChef gave for the same condition.
func TestMathModule(t *testing.T) {
	cases := []struct {
		name string
		cond string
		data string
		want bool
	}{
		{"entropy of text", `math.entropy(0, filesize) > 2.0`, "hello world", true},
		{
			"entropy of one byte over and over", `math.entropy(0, filesize) == 0.0`,
			strings.Repeat("a", 50), true,
		},
		{"entropy of every byte", `math.entropy(0, filesize) == 8.0`, allBytes(), true},
		{"entropy of text written out", `math.entropy("aaaa") == 0.0`, "x", true},
		{
			"mean", `math.mean(0, filesize) > 90.0 and math.mean(0, filesize) < 120.0`,
			"hello world", true,
		},
		{"mean of one byte", `math.mean(0, filesize) == 97.0`, "aaaa", true},
		{"mean of text written out", `math.mean("aaaa") == 97.0`, "x", true},
		{"the average byte there could be", `math.MEAN_BYTES == 127.5`, "x", true},
		{"the smaller of two", `math.min(3, 5) == 3`, "x", true},
		{"the larger of two", `math.max(3, 5) == 5`, "x", true},
		{"the smaller when equal", `math.min(4, 4) == 4`, "x", true},
		// libyara compares as if the numbers had no sign, so a negative number
		// counts as a very large one.
		{"the smaller when one is negative", `math.min(-1, 1) == 1`, "x", true},
		{"the larger when one is negative", `math.max(-1, 1) == -1`, "x", true},
		{
			"how far from a middle the rule names",
			`math.deviation(0, filesize, 97.0) == 0.0`, "aaaa", true,
		},
		{
			"how far from the middle, over text written out",
			`math.deviation("aaaa", 97.0) == 0.0`, "x", true,
		},
		{
			"how much each byte follows the last",
			`math.serial_correlation(0, filesize) == -100000.0`, "aaaa", true,
		},
		{
			"how much each byte follows the last, over text",
			`math.serial_correlation(0, filesize) < 1.0`, "hello world", true,
		},
		{"within a stretch", `math.in_range(5.0, 1.0, 10.0)`, "x", true},
		{"outside a stretch", `math.in_range(50.0, 1.0, 10.0)`, "x", false},
		{"the size of a number", `math.abs(-5) == 5 and math.abs(5) == 5`, "x", true},
		{"yes as a number", `math.to_number(true) == 1`, "x", true},
		{"no as a number", `math.to_number(false) == 0`, "x", true},
		{"how often a byte turns up", `math.count(108) == 3`, "hello world", true},
		{
			"how often a byte turns up in a stretch", `math.count(108, 0, 5) == 2`,
			"hello world", true,
		},
		// A byte that cannot be was never there, so it is counted no times.
		{"how often a byte that cannot be turns up", `math.count(300) == 0`, "hello world", true},
		{"how much of the data one byte is", `math.percentage(108) > 0.27`, "hello world", true},
		{"the commonest byte", `math.mode() == 108`, "hello world", true},
		{"the commonest byte of a stretch", `math.mode(0, 5) == 108`, "hello world", true},
		{
			"how far the data is off being random",
			`math.monte_carlo_pi(0, filesize) > 0.0`, strings.Repeat("hello world", 10), true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := moduleHolds(t, "math", c.cond, c.data); got != c.want {
				t.Errorf("got %v, want %v", got, c.want)
			}
		})
	}
}

// allBytes is every value a byte can take, whose entropy is exactly eight bits.
func allBytes() string {
	b := make([]byte, byteValues)
	for i := range b {
		b[i] = byte(i)
	}
	return string(b)
}

// TestMathModuleWithoutAnAnswer covers the sums that cannot be done: a stretch
// of data that is not there, and a division by no data at all.
func TestMathModuleWithoutAnAnswer(t *testing.T) {
	for _, cond := range []string{
		`defined math.entropy(-1, 5)`,
		`defined math.mean(-1, 5)`,
		`defined math.mean(0, -1)`,
		`defined math.mean(0, 0)`,
		`defined math.deviation(0, 0, 1.0)`,
		`defined math.mode(-1, 5)`,
		`defined math.count(108, -1, 5)`,
		`defined math.percentage(108, -1, 5)`,
		`defined math.monte_carlo_pi(0, 3)`,
	} {
		t.Run(cond, func(t *testing.T) {
			if moduleHolds(t, "math", cond, "hello world") {
				t.Error("worked out a sum it had no answer for")
			}
		})
	}
}

// TestMathModuleOverEmptyData covers the sums that do have an answer over no
// data at all, which are not the same as the ones that do not.
func TestMathModuleOverEmptyData(t *testing.T) {
	for _, cond := range []string{
		`math.entropy(0, 0) == 0.0`,
		`math.serial_correlation(0, 0) == -100000.0`,
		`math.mode(0, 0) == 0`,
		`math.count(108, 0, 0) == 0`,
	} {
		t.Run(cond, func(t *testing.T) {
			if !moduleHolds(t, "math", cond, "hello world") {
				t.Error("no data at all left the sum with no answer")
			}
		})
	}
}

// TestConsoleModule covers the notes a rule can leave for the scan to report,
// which are always true so that adding one does not change the answer.
func TestConsoleModule(t *testing.T) {
	cases := []struct {
		name string
		cond string
		want []string
	}{
		{"a message", `console.log("saw it")`, []string{"saw it"}},
		{"a message and a number", `console.log("n=", 42)`, []string{"n=42"}},
		{
			"a number in hex, as in CyberChef's own fixture",
			`console.hex("log rule b: int8(0)=", int8(0))`,
			[]string{"log rule b: int8(0)=0x43"},
		},
		{"a fraction", `console.log("x=", 1.5)`, []string{"x=1.500000"}},
		{"two of them", `console.log("one") and console.log("two")`, []string{"one", "two"}},
		{"a number on its own", `console.log(42)`, []string{"42"}},
		{"a number in hex on its own", `console.hex(255)`, []string{"0xff"}},
		// Text is noted as printable characters, and anything else as its
		// value, so that a note stays readable whatever it is given.
		{
			"text with a byte that cannot be printed",
			`console.log("a\x00b\xa9")`,
			[]string{`a\x00b\xa9`},
		},
		{
			"only what is being noted is written that way, not what names it",
			`console.log("\xa9 says: ", "\xa9")`,
			[]string{"\xa9 says: \\xa9"},
		},
		{
			"text of nothing at all",
			`console.log("")`,
			[]string{""},
		},
	}
	runConsoleCases(t, cases)
}

// TestConsoleModuleUnanswerable covers a note asked to say something that has
// no value. There is nothing to say, so nothing is noted and the note itself
// has no answer, rather than half a line being left behind.
func TestConsoleModuleUnanswerable(t *testing.T) {
	for _, cond := range []string{
		`console.log("n=", pe.number_of_sections)`,
		`console.log(pe.number_of_sections)`,
		`console.hex("n=", pe.number_of_sections)`,
	} {
		t.Run(cond, func(t *testing.T) {
			src := `import "console" import "pe" rule R { condition: ` + cond + ` }`
			set, err := Parse(src)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			rules, err := Compile(set)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			// The data is not a PE file, so there is no count to say.
			results, logs, err := rules.Scan([]byte("CyberChef Yara"))
			if err != nil {
				t.Fatalf("scan: %v", err)
			}
			if len(logs) != 0 {
				t.Errorf("noted %q, want nothing noted", logs)
			}
			if len(results) != 0 {
				t.Error("the rule held, want it to have no answer")
			}
		})
	}
}

// runConsoleCases scans each case and checks what it noted.
func runConsoleCases(t *testing.T, cases []struct {
	name string
	cond string
	want []string
},
) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			set, err := Parse(`import "console" rule R { condition: ` + c.cond + ` }`)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			rules, err := Compile(set)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			results, logs, err := rules.Scan([]byte("CyberChef Yara"))
			if err != nil {
				t.Fatalf("scan: %v", err)
			}
			if len(results) != 1 {
				t.Error("a note changed whether the rule held")
			}
			if strings.Join(logs, "\n") != strings.Join(c.want, "\n") {
				t.Errorf("noted %q, want %q", logs, c.want)
			}
		})
	}
}

// TestTimeModule covers reading the clock, which only has to be a sensible
// number of seconds.
func TestTimeModule(t *testing.T) {
	// Any moment after the start of 2020, which the clock has long passed.
	if !moduleHolds(t, "time", `time.now() > 1577836800`, "x") {
		t.Error("the clock reads before 2020")
	}
}

// TestUnknownModuleAtScanTime covers a module the checks would refuse, reached
// directly so that the last line of defence is covered too.
func TestUnknownModuleAtScanTime(t *testing.T) {
	e := &evaluator{vars: map[string]int64{}, matched: map[string]bool{}}
	if _, err := e.moduleRoot("nonsense"); err == nil {
		t.Fatal("built a module that does not exist")
	}
}

// TestModuleFunctionsGuardTheirArguments covers each function handed something
// it cannot work with. The checks refuse these before a scan, so they are
// called directly: whatever they are given, they have to come to nothing rather
// than read past what is there.
func TestModuleFunctionsGuardTheirArguments(t *testing.T) {
	e := &evaluator{
		buf:  newBuffer([]byte("hello world")),
		vars: map[string]int64{}, matched: map[string]bool{},
	}
	cases := []struct {
		name string
		call func(*evaluator, []value) (value, error)
		args []value
	}{
		{"how far from a middle, given nothing", deviation, nil},
		{
			"how far from a middle that is not a number", deviation,
			[]value{intValue(0), intValue(1), stringValue("x")},
		},
		{
			"how far from a middle, over data that is not there", deviation,
			[]value{intValue(-1), intValue(1), floatValue(1)},
		},
		{"a stretch given too few bounds", inRange, []value{floatValue(1), floatValue(2)}},
		{
			"a stretch whose bounds are not numbers", inRange,
			[]value{floatValue(1), stringValue("x"), floatValue(2)},
		},
		{"the smaller of one thing", smaller, []value{intValue(1)}},
		{
			"the smaller of things that are not numbers", smaller,
			[]value{stringValue("a"), stringValue("b")},
		},
		{
			"the larger of things that are not numbers", larger,
			[]value{stringValue("a"), stringValue("b")},
		},
		{"a yes or no that is not there", toNumber, []value{undefined}},
		{"a yes or no, given nothing", toNumber, nil},
		{"the size of something that is not a number", absolute, []value{stringValue("x")}},
		{"the size of nothing at all", absolute, nil},
		{
			"how often something that is not a byte turns up", countByte,
			[]value{stringValue("x")},
		},
		{"how often a byte turns up, given nothing", countByte, nil},
		{"how much of the data a byte is, given nothing", percentageOfByte, nil},
		{
			"how often a byte turns up over data that is not there", countByte,
			[]value{intValue(1), intValue(-1), intValue(1)},
		},
		{
			"the commonest byte of data that is not there", commonestByte,
			[]value{intValue(-1), intValue(1)},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := c.call(e, c.args)
			if err != nil {
				t.Fatalf("refused rather than coming to nothing: %v", err)
			}
			if got.kind != valueUndefined {
				t.Errorf("came to %v, want nothing at all", got.kind)
			}
		})
	}
}

// TestDataArgumentBeyondTheData covers pointing a module at a stretch that
// starts past the end, which is nothing to work on rather than a shorter
// stretch.
func TestDataArgumentBeyondTheData(t *testing.T) {
	e := &evaluator{buf: newBuffer([]byte("hello")), vars: map[string]int64{}}
	if _, ok := e.dataArgument([]value{intValue(99), intValue(1)}); ok {
		t.Error("found data past the end")
	}
	if _, ok := e.dataArgument([]value{intValue(0)}); ok {
		t.Error("read one number as a stretch of data")
	}
	if _, ok := e.dataArgument([]value{stringValue("a"), stringValue("b")}); ok {
		t.Error("read two pieces of text as a stretch of data")
	}
	got, ok := e.dataArgument([]value{intValue(2), intValue(99)})
	if !ok || string(got) != "llo" {
		t.Errorf("a stretch running past the end read as %q %v, want %q", got, ok, "llo")
	}
}
