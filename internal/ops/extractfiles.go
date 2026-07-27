package ops

import (
	"errors"
	"fmt"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

// extractFilesMinSize is the size below which a carved file is taken to be a
// false positive rather than something worth keeping.
const extractFilesMinSize = 100.0

// ExtractFiles carves embedded files out of a buffer.
type ExtractFiles struct{}

// Meta returns the operation metadata.
func (ExtractFiles) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Extract Files",
		Module: "Default",
		Description: "Performs file carving to attempt to extract files from the input." +
			"<br><br>This operation is currently capable of carving out the following " +
			"formats:<br><ul><li>" + strings.Join(carvableExtensions(), "</li><li>") +
			"</li></ul>Minimum File Size can be used to prune small false positives.",
		InfoURL:    "https://forensics.wiki/file_carving",
		InputType:  core.TypeArrayBuffer,
		OutputType: core.TypeFileList,
	}
}

// carvableExtensions lists, in upper case, the extensions of every signature
// that can be carved out, without repeats. It is what the description advertises
// the operation as able to recover. A signature naming several extensions
// contributes all of them as one entry, as CyberChef's description does — so
// the run of names a portable executable can go by stays together rather than
// being reduced to "EXE".
func carvableExtensions() []string {
	var out []string
	seen := map[string]bool{}
	for _, cat := range fileSignatures {
		for _, ft := range cat.types {
			if ft.carver == "" {
				continue
			}
			ext := strings.ToUpper(ft.extension)
			if !seen[ext] {
				seen[ext] = true
				out = append(out, ext)
			}
		}
	}
	return out
}

// Args returns the argument definitions: one switch per signature category,
// then whether to keep quiet about carves that fail and the size floor.
func (ExtractFiles) Args() []core.ArgDef {
	args := make([]core.ArgDef, 0, len(fileSignatures)+2)
	for _, cat := range fileSignatures {
		args = append(args, core.ArgDef{
			Name: cat.name,
			Type: core.ArgBoolean,
			// The catch-all category is off by default: its signatures are
			// short and match often by chance.
			Value: cat.name != "Miscellaneous",
		})
	}
	return append(args,
		core.ArgDef{Name: "Ignore failed extractions", Type: core.ArgBoolean, Value: true},
		core.ArgDef{Name: "Minimum File Size", Type: core.ArgNumber, Value: extractFilesMinSize},
	)
}

// Run carves out every file it can find.
func (ExtractFiles) Run(in *core.Dish, args []any) (*core.Dish, error) {
	categories, ignoreFailures, minSize := extractFilesSettings(args)

	var files []core.NamedFile
	var failures []string
	for _, found := range scanForFileTypes(in.Bytes(), categories) {
		file, err := extractFile(in.Bytes(), found.details, found.offset)
		if err != nil {
			if note := extractFilesNote(err, found); note != "" && !ignoreFailures {
				failures = append(failures, note)
			}
			continue
		}
		if len(file.Data) >= minSize {
			files = append(files, file)
		}
	}

	if len(failures) > 0 {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New(strings.Join(failures, "\n\n"))
	}
	return core.NewFileListDish(files), nil
}

// extractFilesSettings reads the arguments: which categories to search, whether
// to keep quiet about carves that fail, and the smallest file worth keeping.
func extractFilesSettings(args []any) (categories []string, ignoreFailures bool, minSize int) {
	categories = []string{}
	for i, cat := range fileSignatures {
		if i < len(args) {
			if on, _ := args[i].(bool); on {
				categories = append(categories, cat.name)
			}
		}
	}
	if len(args) >= len(fileSignatures)+2 {
		ignoreFailures, _ = args[len(fileSignatures)].(bool)
		size, _ := args[len(fileSignatures)+1].(float64)
		minSize = int(size)
	}
	return categories, ignoreFailures, minSize
}

// extractFilesNote describes a carve that failed, or returns the empty string
// when there is nothing worth saying. A type that simply has no algorithm is not
// a failure — most of the signature table is in that position — so only a carve
// that was attempted and went wrong is reported.
func extractFilesNote(err error, found foundFile) string {
	if strings.HasPrefix(err.Error(), "No extraction algorithm available") {
		return ""
	}
	return fmt.Sprintf("Error while attempting to extract %s at offset %d:\n%s",
		found.details.name, found.offset, err.Error())
}

func init() { core.Register(ExtractFiles{}) }
