package ops

import (
	"github.com/dlclark/regexp2"

	"github.com/roberson-io/cchef/core"
)

// The two shapes a path can take. A Windows path starts at a drive letter and
// walks backslash-separated names, ending in an optional extension; a UNIX path
// is a run of slash-separated names. The UNIX shape is loose enough to match
// ordinary text that happens to contain slashes, which is what the operation's
// description warns about.
const (
	pathWinDrive = `[A-Z]:\\`
	pathWinName  = `[A-Z\d][A-Z\d\- '_\(\)~]{0,61}`
	pathWinExt   = `[A-Z\d]{1,6}`
	pathWindows  = pathWinDrive + `(?:` + pathWinName + `\\?)*` + pathWinName +
		`(?:\.` + pathWinExt + `)?`

	pathUnix = `(?:/[A-Z\d.][A-Z\d\-.]{0,61})+`
)

var (
	windowsPathRegexp = jsRegex(pathWindows, regexp2.IgnoreCase)
	unixPathRegexp    = jsRegex(pathUnix, regexp2.IgnoreCase)
	anyPathRegexp     = jsRegex(pathWindows+`|`+pathUnix, regexp2.IgnoreCase)
)

// ExtractFilePaths pulls anything shaped like a file path out of the input.
type ExtractFilePaths struct{}

// Meta returns the operation metadata.
func (ExtractFilePaths) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Extract file paths",
		Module: "Regex",
		Description: "Extracts anything that looks like a Windows or UNIX file path." +
			"<br><br>Note that if UNIX is selected, there will likely be a lot of false " +
			"positives.",
		InputType:  core.TypeString,
		OutputType: core.TypeString,
	}
}

// Args returns the argument definitions.
func (ExtractFilePaths) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Windows", Type: core.ArgBoolean, Value: true},
		{Name: "UNIX", Type: core.ArgBoolean, Value: true},
		{Name: "Display total", Type: core.ArgBoolean, Value: false},
		{Name: "Sort", Type: core.ArgBoolean, Value: false},
		{Name: "Unique", Type: core.ArgBoolean, Value: false},
	}
}

// Run finds the paths. With neither shape asked for the result is empty, and the
// total is not shown either.
func (ExtractFilePaths) Run(in *core.Dish, args []any) (*core.Dish, error) {
	windows, _ := args[0].(bool)
	unix, _ := args[1].(bool)
	displayTotal, _ := args[2].(bool)
	sortResults, _ := args[3].(bool)
	unique, _ := args[4].(bool)

	re := filePathPattern(windows, unix)
	if re == nil {
		return core.NewDish(nil, core.TypeString), nil
	}
	var less func(a, b string) bool
	if sortResults {
		less = caseInsensitiveLess
	}

	found := extractSearch(in.String(), re, nil, less, unique)
	return core.NewDish([]byte(extractResult(found, displayTotal)), core.TypeString), nil
}

// filePathPattern returns the pattern for the shapes asked for, or nil when
// neither was.
func filePathPattern(windows, unix bool) *regexp2.Regexp {
	switch {
	case windows && unix:
		return anyPathRegexp
	case windows:
		return windowsPathRegexp
	case unix:
		return unixPathRegexp
	}
	return nil
}

func init() { core.Register(ExtractFilePaths{}) }
