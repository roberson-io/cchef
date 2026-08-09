package cmd

import (
	"strings"
	"testing"
)

// shareURL is a link of the kind CyberChef's "Save recipe" produces.
const shareURL = "https://gchq.github.io/CyberChef/#recipe=ROT13()&input=aGVsbG8"

// TestBakeFromURL runs a shared link straight through, using the input the link
// carries so nothing else has to be given.
func TestBakeFromURL(t *testing.T) {
	stageIn(t)
	out, err := execRootCapture(t, "bake", "--from-url", shareURL)
	if err != nil {
		t.Fatalf("bake --from-url: %v", err)
	}
	if strings.TrimSpace(out) != "uryyb" {
		t.Errorf("got %q, want uryyb", out)
	}
}

// TestBakeFromURLExplicitInputWins checks the precedence: an input named on the
// command line beats the one the link carries.
func TestBakeFromURLExplicitInputWins(t *testing.T) {
	stageIn(t)
	out, err := execRootCapture(t, "bake", "--from-url", shareURL, "world")
	if err != nil {
		t.Fatalf("bake --from-url with input: %v", err)
	}
	if strings.TrimSpace(out) != "jbeyq" {
		t.Errorf("got %q, want jbeyq (the positional input, not the URL's)", out)
	}
}

// TestBakeFromURLNoInput covers a link sharing only a recipe: the input then
// comes from the command line as usual.
func TestBakeFromURLNoInput(t *testing.T) {
	stageIn(t)
	out, err := execRootCapture(t, "bake", "--from-url",
		"https://gchq.github.io/CyberChef/#recipe=ROT13()", "hello")
	if err != nil {
		t.Fatalf("bake --from-url: %v", err)
	}
	if strings.TrimSpace(out) != "uryyb" {
		t.Errorf("got %q, want uryyb", out)
	}
}

// TestBakeFromURLErrors covers links that cannot be run.
func TestBakeFromURLErrors(t *testing.T) {
	stageIn(t)
	for name, u := range map[string]string{
		"no recipe":   "https://gchq.github.io/CyberChef/#input=aGVsbG8",
		"unparseable": "https://gchq.github.io/CyberChef/#recipe=To_Hex(",
		"bad input":   "https://gchq.github.io/CyberChef/#recipe=ROT13()&input=%21%21%21",
	} {
		if err := execRootErr(t, "bake", "--from-url", u); err == nil {
			t.Errorf("%s: expected %q to be refused", name, u)
		}
	}
}

// TestConvertFromURL reads a link and prints the recipe in the chosen format.
func TestConvertFromURL(t *testing.T) {
	stageIn(t)
	out, err := execRootCapture(t, "recipe", "convert", "--from-url", shareURL, "--to", "chef")
	if err != nil {
		t.Fatalf("recipe convert --from-url: %v", err)
	}
	if strings.TrimSpace(out) != "ROT13()" {
		t.Errorf("got %q, want ROT13()", out)
	}
}

// TestURLFromURLRepointsInstance covers reading a public link and writing it
// back out against a self-hosted instance, keeping the recipe and the input.
func TestURLFromURLRepointsInstance(t *testing.T) {
	stageIn(t)
	out, err := execRootCapture(t, "url", "--from-url", shareURL,
		"--base-url", "https://cyberchef.internal/")
	if err != nil {
		t.Fatalf("url --from-url: %v", err)
	}
	if got, want := strings.TrimSpace(out),
		"https://cyberchef.internal/#recipe=ROT13()&input=aGVsbG8"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestURLRoundTripThroughCLI is the end-to-end proof that the two directions
// agree: a link built by `cchef url` runs unchanged through `bake --from-url`.
func TestURLRoundTripThroughCLI(t *testing.T) {
	stageIn(t)
	built, err := execRootCapture(t, "url", "-e", "ROT13()", "-i", "hello")
	if err != nil {
		t.Fatalf("url: %v", err)
	}
	out, err := execRootCapture(t, "bake", "--from-url", strings.TrimSpace(built))
	if err != nil {
		t.Fatalf("bake --from-url %q: %v", strings.TrimSpace(built), err)
	}
	if strings.TrimSpace(out) != "uryyb" {
		t.Errorf("round trip gave %q, want uryyb", out)
	}
}

// TestBakeRecipeFromStdin covers `-r -`, so a recipe can be piped in the way
// every other file argument accepts "-".
func TestBakeRecipeFromStdin(t *testing.T) {
	stageIn(t)
	out, err := execRootStdin(t, "To_Hex('Colon')", "bake", "-r", "-", "-i", "hi")
	if err != nil {
		t.Fatalf("bake -r -: %v", err)
	}
	if strings.TrimSpace(out) != "68:69" {
		t.Errorf("got %q, want 68:69", out)
	}
}

// TestRecipeSourceConflictCLI refuses two recipe sources at the command line.
func TestRecipeSourceConflictCLI(t *testing.T) {
	stageIn(t)
	err := execRootErr(t, "bake", "-e", "ROT13()", "--from-url", shareURL, "-i", "x")
	if err == nil {
		t.Fatal("expected two recipe sources to be refused")
	}
	for _, want := range []string{"-e", "--from-url"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
}
