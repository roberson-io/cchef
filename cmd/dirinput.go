package cmd

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/core"
)

// runRecipeIO executes recipe against the resolved input and writes the result.
// With --in-dir it fans the recipe out over the files in a directory (one run
// per file); otherwise it runs once over the single resolved input.
func runRecipeIO(cmd *cobra.Command, posArgs []string, recipe core.Recipe) error {
	fileList, err := recipeFileListOutput(recipe)
	if err != nil {
		return err
	}
	// Resolved up front so an unusable --color value is reported before the
	// recipe runs, whatever the recipe is.
	color, err := wantANSI(cmd)
	if err != nil {
		return err
	}
	switch {
	case fileList && flagOutDir == "":
		return fmt.Errorf("this recipe produces several files; use --out-dir to write them")
	case !fileList && flagOutDir != "" && flagInDir == "":
		return fmt.Errorf("--out-dir requires --in-dir")
	}
	if flagInDir != "" {
		// Per-file results go to --out-dir as files, which take the HTML.
		return runRecipeDir(cmd, recipe, color && flagOutDir == "")
	}

	in, err := resolveInput(cmd, posArgs)
	if err != nil {
		return err
	}
	out, err := recipe.Execute(core.NewDish(in, core.TypeString))
	if err != nil {
		return err
	}
	if out.Type() == core.TypeFileList {
		return writeFileList(flagOutDir, out.Files())
	}
	data := out.Bytes()
	if color && recipeHighlights(recipe) {
		data = []byte(ansiFromHLJS(string(data)))
	}
	return writeOutput(cmd, data)
}

// recipeFileListOutput reports whether the recipe's last enabled step produces a
// file list, which needs --out-dir rather than a single output stream. Knowing
// this before running lets the flag combination be rejected up front. A file
// list anywhere earlier is an error: it holds several files and so cannot feed
// the following step.
func recipeFileListOutput(recipe core.Recipe) (bool, error) {
	var last bool
	for i, step := range recipe {
		if step.Disabled {
			continue
		}
		op, ok := core.Default.Get(step.Op)
		if !ok {
			return false, nil // reported by the recipe run itself
		}
		emits := op.Meta().OutputType == core.TypeFileList
		if emits && hasEnabledStepAfter(recipe, i) {
			return false, fmt.Errorf("%s produces several files and so cannot be chained; it must be the last step", step.Op)
		}
		last = emits
	}
	return last, nil
}

// hasEnabledStepAfter reports whether any step after i is enabled.
func hasEnabledStepAfter(recipe core.Recipe, i int) bool {
	for _, step := range recipe[i+1:] {
		if !step.Disabled {
			return true
		}
	}
	return false
}

// writeFileList writes each named file into dir, creating it if needed.
func writeFileList(dir string, files []core.NamedFile) error {
	if err := os.MkdirAll(dir, 0o755); err != nil { // #nosec G301 -- 0755 is conventional for an output directory created on the user's behalf
		return err
	}
	root := filepath.Clean(dir)
	for _, f := range files {
		dest, err := safeOutputPath(root, f.Name)
		if err != nil {
			return err
		}
		// A name may carry a path — an archive's entries usually do — so the
		// directories it names have to exist first.
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil { // #nosec G301 -- 0755 is conventional for an output directory created on the user's behalf
			return err
		}
		if err := os.WriteFile(dest, f.Data, 0o644); err != nil { // #nosec G306 -- 0644 is conventional for CLI output files
			return err
		}
	}
	return nil
}

// safeOutputPath resolves a file's name under root, refusing any name that
// would put it somewhere else. Names reach here from the data being read — the
// entries of an archive, say — so one may well be trying to climb out.
func safeOutputPath(root, name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	dest := filepath.Join(root, clean)
	escapes := filepath.IsAbs(clean) ||
		clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) ||
		!strings.HasPrefix(dest, root+string(filepath.Separator))
	if escapes {
		return "", fmt.Errorf("refusing to write %q outside the output directory", name)
	}
	return dest, nil
}

// dirFile is one input file discovered under --in-dir: path is where to read it,
// rel is its path relative to the directory root (used for the stdout header
// and, with --out-dir, the mirrored output path).
type dirFile struct {
	path string
	rel  string
}

// runRecipeDir runs recipe once per regular file under --in-dir. Results go to
// --out-dir (one file each, mirroring the input tree) when set, otherwise to
// stdout with a `==> name <==` header per file. A file whose read or recipe run
// fails is reported to stderr and skipped; the command still exits non-zero.
func runRecipeDir(cmd *cobra.Command, recipe core.Recipe, color bool) error {
	color = color && recipeHighlights(recipe)
	files, err := listDirFiles(flagInDir, flagRecursive)
	if err != nil {
		return err
	}
	if flagOutDir != "" {
		if err := os.MkdirAll(flagOutDir, 0o755); err != nil { // #nosec G301 -- 0755 is conventional for an output directory created on the user's behalf
			return err
		}
	}

	out := cmd.OutOrStdout()
	errOut := cmd.ErrOrStderr()
	var failures int
	for _, f := range files {
		res, err := runRecipeOnFile(recipe, f)
		if err != nil {
			_, _ = fmt.Fprintf(errOut, "cchef: %s: %v\n", f.rel, err)
			failures++
			continue
		}
		if color {
			res = core.NewDish([]byte(ansiFromHLJS(res.String())), core.TypeString)
		}
		fatal, err := writeDirResult(out, f, res)
		if err == nil {
			continue
		}
		if fatal {
			return err
		}
		_, _ = fmt.Fprintf(errOut, "cchef: %s: %v\n", f.rel, err)
		failures++
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d file(s) failed", failures, len(files))
	}
	return nil
}

// runRecipeOnFile reads one input file and runs the recipe over it.
func runRecipeOnFile(recipe core.Recipe, f dirFile) (*core.Dish, error) {
	data, err := os.ReadFile(f.path) // #nosec G304 -- reads files under a user-specified input directory by design
	if err != nil {
		return nil, err
	}
	return recipe.Execute(core.NewDish(data, core.TypeString))
}

// writeDirResult writes one input's result: into --out-dir when set, otherwise
// to stdout under a `==> name <==` header. The bool reports whether a failure is
// fatal — a broken stdout stops the whole run, a per-file write failure does not.
func writeDirResult(out io.Writer, f dirFile, res *core.Dish) (bool, error) {
	if res.Type() == core.TypeFileList {
		// Several files per input: keep each input's results in their own
		// directory, named after the input, so they cannot collide.
		stem := strings.TrimSuffix(f.rel, filepath.Ext(f.rel))
		return false, writeFileList(filepath.Join(flagOutDir, stem), res.Files())
	}
	if flagOutDir != "" {
		return false, writeMirroredOutput(f.rel, res.Bytes())
	}
	if _, err := fmt.Fprintf(out, "==> %s <==\n", f.rel); err != nil {
		return true, err
	}
	b := res.Bytes()
	if _, err := out.Write(b); err != nil {
		return true, err
	}
	if len(b) == 0 || b[len(b)-1] != '\n' {
		if _, err := fmt.Fprintln(out); err != nil {
			return true, err
		}
	}
	return false, nil
}

// writeMirroredOutput writes one output file under --out-dir at rel (the input's
// path relative to the root), creating parent directories as needed.
func writeMirroredOutput(rel string, data []byte) error {
	dest := filepath.Join(flagOutDir, rel)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil { // #nosec G301 -- 0755 is conventional for an output directory created on the user's behalf
		return err
	}
	// rel is a cleaned relative path from the directory listing (WalkDir yields
	// only descendants of the root, so filepath.Rel never produces "..", and
	// symlinks are skipped), so dest stays within the user-specified --out-dir.
	return os.WriteFile(dest, data, 0o644) // #nosec G306,G703 -- 0644 is conventional for CLI output; dest is confined to the user-specified --out-dir (rel has no traversal)
}

// listDirFiles returns the regular files directly under root (or, when
// recursive, anywhere beneath it), sorted by their path relative to root.
// Non-regular entries (directories, symlinks, devices) are skipped.
func listDirFiles(root string, recursive bool) ([]dirFile, error) {
	var files []dirFile
	if recursive {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !d.Type().IsRegular() {
				return nil
			}
			rel, err := filepath.Rel(root, p)
			if err != nil {
				return err
			}
			files = append(files, dirFile{path: p, rel: rel})
			return nil
		})
		if err != nil {
			return nil, err
		}
	} else {
		entries, err := os.ReadDir(root)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() || !e.Type().IsRegular() {
				continue
			}
			files = append(files, dirFile{path: filepath.Join(root, e.Name()), rel: e.Name()})
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].rel < files[j].rel })
	return files, nil
}
