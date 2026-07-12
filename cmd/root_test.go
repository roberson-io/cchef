package cmd

import (
	"bytes"
	"strings"
	"testing"
)

// TestExecuteWrapper covers the exported Execute() wrapper on its success path
// (rootCmd.Execute returns nil, so os.Exit is not called).
func TestExecuteWrapper(t *testing.T) {
	resetIOFlags()
	var buf bytes.Buffer
	rootCmd.SetOut(&buf)
	rootCmd.SetErr(&buf)
	rootCmd.SetIn(strings.NewReader(""))
	rootCmd.SetArgs([]string{"--version"})
	Execute()
	if !strings.Contains(buf.String(), version) {
		t.Fatalf("Execute --version output %q missing version", buf.String())
	}
}
