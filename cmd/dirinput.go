package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"

	"github.com/roberson-io/cchef/internal/core"
)

// runRecipeIO executes recipe against the resolved input and writes the result.
// With --in-dir it fans the recipe out over the files in a directory (one run
// per file); otherwise it runs once over the single resolved input.
func runRecipeIO(cmd *cobra.Command, posArgs []string, recipe core.Recipe) error {
	if flagOutDir != "" && flagInDir == "" {
		return fmt.Errorf("--out-dir requires --in-dir")
	}
	if flagInDir != "" {
		return runRecipeDir(cmd, recipe)
	}

	in, err := resolveInput(cmd, posArgs)
	if err != nil {
		return err
	}
	out, err := recipe.Execute(core.NewDish(in, core.TypeString))
	if err != nil {
		return err
	}
	return writeOutput(cmd, out.Bytes())
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
func runRecipeDir(cmd *cobra.Command, recipe core.Recipe) error {
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
		data, err := os.ReadFile(f.path) // #nosec G304 -- reads files under a user-specified input directory by design
		if err != nil {
			_, _ = fmt.Fprintf(errOut, "cchef: %s: %v\n", f.rel, err)
			failures++
			continue
		}
		res, err := recipe.Execute(core.NewDish(data, core.TypeString))
		if err != nil {
			_, _ = fmt.Fprintf(errOut, "cchef: %s: %v\n", f.rel, err)
			failures++
			continue
		}
		if flagOutDir != "" {
			if err := writeMirroredOutput(f.rel, res.Bytes()); err != nil {
				_, _ = fmt.Fprintf(errOut, "cchef: %s: %v\n", f.rel, err)
				failures++
			}
			continue
		}
		if _, err := fmt.Fprintf(out, "==> %s <==\n", f.rel); err != nil {
			return err
		}
		b := res.Bytes()
		if _, err := out.Write(b); err != nil {
			return err
		}
		if len(b) == 0 || b[len(b)-1] != '\n' {
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
		}
	}
	if failures > 0 {
		return fmt.Errorf("%d of %d file(s) failed", failures, len(files))
	}
	return nil
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
