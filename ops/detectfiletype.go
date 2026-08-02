package ops

import (
	"strings"

	"github.com/roberson-io/cchef/core"
	"github.com/roberson-io/cchef/internal/filesig"
)

func init() {
	core.Register(DetectFileType{})
}

// detectFileTypeCats lists the signature categories in CyberChef's declared
// order; each becomes a boolean argument (all default on).
var detectFileTypeCats = []string{"Images", "Video", "Audio", "Documents", "Applications", "Archives", "Miscellaneous"}

// DetectFileType guesses the MIME type of the input from its magic bytes.
type DetectFileType struct{}

// Meta returns the operation metadata.
func (DetectFileType) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Detect File Type",
		Module:      "Default",
		Description: "Attempts to guess the MIME type of the data based on 'magic bytes'.",
		InfoURL:     "https://wikipedia.org/wiki/List_of_file_signatures",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns one boolean per signature category (all enabled by default).
func (DetectFileType) Args() []core.ArgDef {
	args := make([]core.ArgDef, len(detectFileTypeCats))
	for i, cat := range detectFileTypeCats {
		args[i] = core.ArgDef{Name: cat, Type: core.ArgBoolean, Value: true}
	}
	return args
}

// Run reports the detected file types.
func (DetectFileType) Run(in *core.Dish, args []any) (*core.Dish, error) {
	categories := []string{}
	for i, cat := range detectFileTypeCats {
		if args[i].(bool) {
			categories = append(categories, cat)
		}
	}

	types := filesig.Detect(in.Bytes(), categories)
	if len(types) == 0 {
		return core.NewDish([]byte("Unknown file type. Have you tried checking the entropy of this data to determine whether it might be encrypted or compressed?"), core.TypeString), nil
	}

	results := make([]string, len(types))
	for i, t := range types {
		out := "File type:   " + t.Name + "\n" +
			"Extension:   " + t.Extension + "\n" +
			"MIME type:   " + t.MIME + "\n"
		if t.Description != "" {
			out += "Description: " + t.Description + "\n"
		}
		results[i] = out
	}
	return core.NewDish([]byte(strings.Join(results, "\n")), core.TypeString), nil
}
