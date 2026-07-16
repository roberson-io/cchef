package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeTree creates each file (relative path -> content) under a fresh temp dir
// and returns the dir root.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// execRootCapture runs the root command and returns its combined output plus any
// error, resetting shared flag state first (mirrors execRoot).
func execRootCapture(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetIOFlags()
	flagRecipeExpr, flagRecipeFile, flagConvertTo = "", "", ""

	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetIn(strings.NewReader(""))
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return buf.String(), err
}

// These tests use the argument-free Atbash op (a<->z, b<->y) so no per-operation
// flag state can leak in from other cases in the suite.

// TestInDirStdoutHeaders runs an op over a flat directory and checks the
// per-file `==> name <==` headers on stdout.
func TestInDirStdoutHeaders(t *testing.T) {
	dir := writeTree(t, map[string]string{"a.txt": "a", "b.txt": "ab"})
	out := execRoot(t, "atbash-cipher", "--in-dir", dir)
	// atbash("a")="z", atbash("ab")="zy"; files are visited in sorted order.
	want := "==> a.txt <==\nz\n==> b.txt <==\nzy\n"
	if out != want {
		t.Fatalf("got %q\nwant %q", out, want)
	}
}

// TestInDirFlatSkipsSubdirs confirms the default (non-recursive) mode processes
// only top-level files.
func TestInDirFlatSkipsSubdirs(t *testing.T) {
	dir := writeTree(t, map[string]string{"top.txt": "a", "sub/deep.txt": "b"})
	out := execRoot(t, "atbash-cipher", "--in-dir", dir)
	if !strings.Contains(out, "==> top.txt <==") {
		t.Fatalf("missing top-level file in output: %q", out)
	}
	if strings.Contains(out, "deep.txt") {
		t.Fatalf("subdirectory file should be skipped without --recursive: %q", out)
	}
}

// TestInDirRecursive walks nested directories and labels files by their path
// relative to the root.
func TestInDirRecursive(t *testing.T) {
	dir := writeTree(t, map[string]string{"top.txt": "a", "sub/deep.txt": "b"})
	out := execRoot(t, "atbash-cipher", "--in-dir", dir, "--recursive")
	if !strings.Contains(out, "==> top.txt <==") {
		t.Fatalf("missing top-level file: %q", out)
	}
	if !strings.Contains(out, "==> "+filepath.Join("sub", "deep.txt")+" <==") {
		t.Fatalf("missing recursive file with relative path: %q", out)
	}
}

// TestInDirOutDir mirrors output into --out-dir, one file per input, preserving
// the relative tree.
func TestInDirOutDir(t *testing.T) {
	dir := writeTree(t, map[string]string{"a.txt": "a", "sub/b.txt": "b"})
	outDir := filepath.Join(t.TempDir(), "results")
	out := execRoot(t, "atbash-cipher", "--in-dir", dir, "--out-dir", outDir, "--recursive")
	if out != "" {
		t.Fatalf("expected no stdout with --out-dir, got %q", out)
	}
	got, err := os.ReadFile(filepath.Join(outDir, "a.txt"))
	if err != nil {
		t.Fatalf("reading mirrored output: %v", err)
	}
	if string(got) != "z" {
		t.Fatalf("a.txt: got %q, want %q", got, "z")
	}
	got2, err := os.ReadFile(filepath.Join(outDir, "sub", "b.txt"))
	if err != nil {
		t.Fatalf("reading nested mirrored output: %v", err)
	}
	if string(got2) != "y" {
		t.Fatalf("sub/b.txt: got %q, want %q", got2, "y")
	}
}

// TestOutDirRequiresInDir rejects --out-dir without --in-dir.
func TestOutDirRequiresInDir(t *testing.T) {
	err := execRootErr(t, "atbash-cipher", "--out-dir", t.TempDir(), "hello")
	if err == nil {
		t.Fatal("expected error, got none")
	}
	if !strings.Contains(err.Error(), "--out-dir requires --in-dir") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestInDirPartialFailure reports a failing file to stderr, keeps going, and
// exits non-zero. The delimiter is passed explicitly so the case is independent
// of any leaked flag state.
func TestInDirPartialFailure(t *testing.T) {
	dir := writeTree(t, map[string]string{"good.txt": "1 2 3", "zbad.txt": "99"})
	out, err := execRootCapture(t, "a1z26-cipher-decode", "--delimiter", "Space", "--in-dir", dir)
	if err == nil {
		t.Fatal("expected non-zero exit, got nil error")
	}
	if !strings.Contains(out, "==> good.txt <==\nabc") {
		t.Fatalf("good file not processed: %q", out)
	}
	if !strings.Contains(out, "zbad.txt:") {
		t.Fatalf("failing file not reported to stderr: %q", out)
	}
	if !strings.Contains(err.Error(), "1 of 2") {
		t.Fatalf("summary error not returned: %v", err)
	}
}

// TestInDirBake confirms directory mode also works for full recipes via bake.
func TestInDirBake(t *testing.T) {
	dir := writeTree(t, map[string]string{"a.txt": "a"})
	out := execRoot(t, "bake", "-e", "Atbash_Cipher()", "--in-dir", dir)
	if out != "==> a.txt <==\nz\n" {
		t.Fatalf("bake over dir: got %q", out)
	}
}

// TestInDirMissing errors when --in-dir does not exist (non-recursive and
// recursive both surface the directory-listing error).
func TestInDirMissing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	if err := execRootErr(t, "atbash-cipher", "--in-dir", missing); err == nil {
		t.Fatal("expected error for missing --in-dir (flat)")
	}
	if err := execRootErr(t, "atbash-cipher", "--in-dir", missing, "--recursive"); err == nil {
		t.Fatal("expected error for missing --in-dir (recursive)")
	}
}

// TestInDirEmpty produces no output and no error for a directory with no files.
func TestInDirEmpty(t *testing.T) {
	out, err := execRootCapture(t, "atbash-cipher", "--in-dir", t.TempDir())
	if err != nil {
		t.Fatalf("empty dir should not error: %v", err)
	}
	if out != "" {
		t.Fatalf("empty dir should produce no output, got %q", out)
	}
}

// TestInDirReadError reports a per-file read failure and exits non-zero. Skipped
// when running as root, where file permissions do not restrict reads.
func TestInDirReadError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("read-permission checks do not apply to root")
	}
	dir := writeTree(t, map[string]string{"locked.txt": "a"})
	if err := os.Chmod(filepath.Join(dir, "locked.txt"), 0o000); err != nil {
		t.Fatal(err)
	}
	out, err := execRootCapture(t, "atbash-cipher", "--in-dir", dir)
	if err == nil {
		t.Fatal("expected non-zero exit from unreadable file")
	}
	if !strings.Contains(out, "locked.txt:") {
		t.Fatalf("read error not reported: %q", out)
	}
}

// TestOutDirMkdirFails surfaces the error when --out-dir cannot be created
// because a path component is an existing file.
func TestOutDirMkdirFails(t *testing.T) {
	dir := writeTree(t, map[string]string{"a.txt": "a"})
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	// blocker is a file, so creating blocker/out beneath it must fail.
	err := execRootErr(t, "atbash-cipher", "--in-dir", dir, "--out-dir", filepath.Join(blocker, "out"))
	if err == nil {
		t.Fatal("expected --out-dir creation to fail")
	}
}

// runDirWithWriter runs atbash over dir writing to w, returning the error. Used
// to drive the stdout write-error guards.
func runDirWithWriter(dir string, w *failOnNth) error {
	resetIOFlags()
	flagRecipeExpr, flagRecipeFile, flagConvertTo = "", "", ""
	rootCmd.SetOut(w)
	rootCmd.SetErr(&bytes.Buffer{})
	rootCmd.SetIn(strings.NewReader(""))
	rootCmd.SetArgs([]string{"atbash-cipher", "--in-dir", dir})
	err := rootCmd.Execute()
	rootCmd.SetOut(nil) // restore default for later tests
	return err
}

// failOnNth writes succeed until the nth call, which fails. This isolates each
// of the three sequential stdout writes per file (header, body, trailing NL).
type failOnNth struct {
	n     int
	count int
}

func (f *failOnNth) Write(p []byte) (int, error) {
	f.count++
	if f.count == f.n {
		return 0, fmt.Errorf("write failed on call %d", f.n)
	}
	return len(p), nil
}

// TestInDirWriteError covers the three per-file stdout write guards: the header
// write (call 1), the result-body write (call 2) and the trailing-newline write
// (call 3) each propagate their error.
func TestInDirWriteError(t *testing.T) {
	dir := writeTree(t, map[string]string{"a.txt": "a"})
	for n := 1; n <= 3; n++ {
		if err := runDirWithWriter(dir, &failOnNth{n: n}); err == nil {
			t.Fatalf("write failure on call %d should propagate", n)
		}
	}
}

// TestOutDirPerFileWriteError covers the per-file mirror-write failure path: the
// destination path already exists as a directory, so writing the file fails but
// the run continues and exits non-zero.
func TestOutDirPerFileWriteError(t *testing.T) {
	dir := writeTree(t, map[string]string{"a.txt": "a"})
	outDir := filepath.Join(t.TempDir(), "results")
	// Pre-create results/a.txt as a directory so WriteFile to it fails.
	if err := os.MkdirAll(filepath.Join(outDir, "a.txt"), 0o750); err != nil {
		t.Fatal(err)
	}
	out, err := execRootCapture(t, "atbash-cipher", "--in-dir", dir, "--out-dir", outDir)
	if err == nil {
		t.Fatal("expected non-zero exit from mirror-write failure")
	}
	if !strings.Contains(out, "a.txt:") {
		t.Fatalf("mirror-write error not reported: %q", out)
	}
}

// TestOutDirNestedMkdirError covers writeMirroredOutput's parent-directory
// creation failure: a path component under --out-dir is an existing file.
func TestOutDirNestedMkdirError(t *testing.T) {
	dir := writeTree(t, map[string]string{"sub/b.txt": "b"})
	outDir := filepath.Join(t.TempDir(), "results")
	// Pre-create results/sub as a FILE so MkdirAll(results/sub) fails.
	if err := os.MkdirAll(outDir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outDir, "sub"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	out, err := execRootCapture(t, "atbash-cipher", "--in-dir", dir, "--out-dir", outDir, "--recursive")
	if err == nil {
		t.Fatal("expected non-zero exit from nested mkdir failure")
	}
	if !strings.Contains(out, filepath.Join("sub", "b.txt")+":") {
		t.Fatalf("nested mkdir error not reported: %q", out)
	}
}

// TestInDirRecursiveWalkError covers the WalkDir callback error path: an
// unreadable subdirectory encountered mid-walk. Skipped as root.
func TestInDirRecursiveWalkError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("directory-permission checks do not apply to root")
	}
	dir := writeTree(t, map[string]string{"sub/deep.txt": "b"})
	locked := filepath.Join(dir, "sub")
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(locked, 0o750) //nolint:errcheck // best-effort restore so t.TempDir cleanup can remove it
	if err := execRootErr(t, "atbash-cipher", "--in-dir", dir, "--recursive"); err == nil {
		t.Fatal("expected error walking an unreadable subdirectory")
	}
}
