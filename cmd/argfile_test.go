package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeSecret writes content to a file in a test temp dir and returns its path.
func writeSecret(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestArgFileToggleString covers reading a toggleString value from a file: the
// XOR key comes from --key-file, the --key-type flag still applies, and one
// trailing newline is stripped so a normal text file works as-is.
func TestArgFileToggleString(t *testing.T) {
	key := writeSecret(t, "key.txt", "0a\n")
	out := execRoot(t, "xor", "--key-file", key, "--key-type", "Hex", "abc")
	if out != "khi" {
		t.Errorf("file-sourced key gave %q, want %q", out, "khi")
	}
}

// TestArgFileString covers a plain string argument (Vigenère's key), and that
// only a single trailing newline is stripped — inner content is untouched.
func TestArgFileString(t *testing.T) {
	key := writeSecret(t, "key.txt", "lemon\r\n")
	out := execRoot(t, "vigenere-encode", "--key-file", key, "attackatdawn")
	if out != "lxfopvefrnhr" {
		t.Errorf("file-sourced key gave %q, want %q", out, "lxfopvefrnhr")
	}
}

// TestArgFileBothSetErrors covers the conflict: naming both the inline flag and
// its file variant is ambiguous and must be an error, not a silent preference.
func TestArgFileBothSetErrors(t *testing.T) {
	key := writeSecret(t, "key.txt", "lemon")
	err := execRootErr(t, "vigenere-encode", "--key", "other", "--key-file", key, "attackatdawn")
	if err == nil || !strings.Contains(err.Error(), "--key") {
		t.Fatalf("both --key and --key-file: got %v, want an error naming the flags", err)
	}
}

// TestArgFileMissingFileErrors covers an unreadable file path.
func TestArgFileMissingFileErrors(t *testing.T) {
	err := execRootErr(t, "vigenere-encode", "--key-file", "/nonexistent/key.txt", "attackatdawn")
	if err == nil {
		t.Fatal("missing key file: expected an error")
	}
}

// TestArgFileInHelp covers discoverability: the op help lists the file variant.
func TestArgFileInHelp(t *testing.T) {
	out := execRoot(t, "vigenere-encode", "--help")
	if !strings.Contains(out, "--key-file") {
		t.Errorf("help does not mention --key-file:\n%s", out)
	}
}
