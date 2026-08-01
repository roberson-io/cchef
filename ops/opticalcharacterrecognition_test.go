package ops

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ocrArgs builds the operation's two arguments.
func ocrArgs(showConfidence bool, oem string) []any {
	return []any{showConfidence, oem}
}

// stubOCR replaces the tesseract invocation for the duration of a test.
func stubOCR(t *testing.T, fn func(image []byte, oem int) (ocrOutput, error)) {
	t.Helper()
	previous := runOCR
	runOCR = fn
	t.Cleanup(func() { runOCR = previous })
}

// A tiny valid PNG, so the file-type check passes without needing a fixture.
func ocrTestImage(t *testing.T) string {
	t.Helper()
	return loadPNGBytes(t, "resize_input.png")
}

// With confidence shown, the mean word confidence precedes the text, matching
// CyberChef's `Confidence: N%\n\n<text>`.
func TestOCRShowsConfidence(t *testing.T) {
	stubOCR(t, func([]byte, int) (ocrOutput, error) {
		return ocrOutput{
			text: "Hello World\n",
			tsv: "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\tleft\ttop\twidth\theight\tconf\ttext\n" +
				"4\t1\t1\t1\t1\t0\t24\t19\t239\t36\t-1\t\n" +
				"5\t1\t1\t1\t1\t1\t24\t19\t103\t36\t95.493790\tHello\n" +
				"5\t1\t1\t1\t1\t2\t143\t19\t120\t36\t96.146400\tWorld\n",
		}, nil
	})

	got, err := runOp(t, "Optical Character Recognition", ocrTestImage(t), ocrArgs(true, "LSTM only")...)
	if err != nil {
		t.Fatal(err)
	}
	if want := "Confidence: 96%\n\nHello World\n"; got != want {
		t.Errorf("output = %q, want %q", got, want)
	}
}

// With confidence hidden, only the recognised text is returned.
func TestOCRHidesConfidence(t *testing.T) {
	stubOCR(t, func([]byte, int) (ocrOutput, error) {
		return ocrOutput{text: "Hello World\n"}, nil
	})
	got, err := runOp(t, "Optical Character Recognition", ocrTestImage(t), ocrArgs(false, "LSTM only")...)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Hello World\n" {
		t.Errorf("output = %q, want just the text", got)
	}
}

// The engine mode option maps onto tesseract's --oem values in CyberChef's order.
func TestOCREngineModeMapping(t *testing.T) {
	for _, tc := range []struct {
		mode string
		want int
	}{
		{"Tesseract only", 0},
		{"LSTM only", 1},
		{"Tesseract/LSTM Combined", 2},
	} {
		var got int
		stubOCR(t, func(_ []byte, oem int) (ocrOutput, error) {
			got = oem
			return ocrOutput{text: ""}, nil
		})
		if _, err := runOp(t, "Optical Character Recognition", ocrTestImage(t), ocrArgs(false, tc.mode)...); err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("%s mapped to --oem %d, want %d", tc.mode, got, tc.want)
		}
	}
}

// Non-image input is rejected before tesseract is ever invoked.
func TestOCRRejectsNonImage(t *testing.T) {
	stubOCR(t, func([]byte, int) (ocrOutput, error) {
		t.Error("tesseract was invoked for a non-image input")
		return ocrOutput{}, nil
	})
	_, err := runOp(t, "Optical Character Recognition", "not an image", ocrArgs(true, "LSTM only")...)
	want := "Unsupported file type (supported: jpg,png,pbm,bmp) or no file provided"
	if err == nil || err.Error() != want {
		t.Errorf("error = %v, want %q", err, want)
	}
}

// A failure from tesseract is surfaced with CyberChef's wrapper text.
func TestOCRWrapsFailure(t *testing.T) {
	stubOCR(t, func([]byte, int) (ocrOutput, error) {
		return ocrOutput{}, errors.New("legacy engine not present")
	})
	_, err := runOp(t, "Optical Character Recognition", ocrTestImage(t), ocrArgs(true, "Tesseract only")...)
	if err == nil || !strings.Contains(err.Error(), "Error performing OCR on image.") {
		t.Errorf("error = %v, want CyberChef's wrapper", err)
	}
	if err != nil && !strings.Contains(err.Error(), "legacy engine not present") {
		t.Errorf("error = %v, want the underlying cause included", err)
	}
}

// ocrTSVHeader is the header tesseract writes; conf is the eleventh column.
const ocrTSVHeader = "level\tpage_num\tblock_num\tpar_num\tline_num\tword_num\t" +
	"left\ttop\twidth\theight\tconf\ttext\n"

// ocrTSVRow builds one row in tesseract's twelve-column layout.
func ocrTSVRow(level, conf, text string) string {
	return strings.Join([]string{level, "1", "1", "1", "1", "1", "0", "0", "1", "1", conf, text}, "\t") + "\n"
}

// meanWordConfidence averages only the word rows, ignoring the structural rows
// tesseract marks with a confidence of -1.
func TestMeanWordConfidence(t *testing.T) {
	for _, tc := range []struct {
		name string
		rows string
		want int
	}{
		{"single word", ocrTSVRow("5", "90.0", "a"), 90},
		{"rounds to nearest", ocrTSVRow("5", "95.49379", "a") + ocrTSVRow("5", "96.1464", "b"), 96},
		{
			"ignores structural rows",
			ocrTSVRow("4", "-1", "") + ocrTSVRow("5", "80.0", "a") + ocrTSVRow("5", "60.0", "b"),
			70,
		},
		{"no words at all", ocrTSVRow("4", "-1", ""), 0},
		{"empty input", "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := tc.rows
			if in != "" {
				in = ocrTSVHeader + tc.rows
			}
			if got := meanWordConfidence(in); got != tc.want {
				t.Errorf("meanWordConfidence = %d, want %d", got, tc.want)
			}
		})
	}
}

// Malformed TSV rows are skipped rather than crashing or skewing the mean.
func TestMeanWordConfidenceMalformed(t *testing.T) {
	tsv := ocrTSVHeader +
		ocrTSVRow("5", "notanumber", "a") + // unparseable confidence
		"5\t1\t1\n" + // too few columns
		"\n" + // blank line
		ocrTSVRow("5", "50.0", "b")
	if got := meanWordConfidence(tsv); got != 50 {
		t.Errorf("meanWordConfidence = %d, want 50 from the one usable row", got)
	}
}

// A missing tesseract binary is reported as something the user can act on.
func TestOCRMissingBinary(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := runTesseract([]byte("ignored"), 1)
	if err == nil {
		t.Fatal("expected an error when tesseract is not installed")
	}
	if !strings.Contains(err.Error(), "tesseract") || !strings.Contains(err.Error(), "install") {
		t.Errorf("error = %v, want it to name tesseract and how to install it", err)
	}
}

// End-to-end against the real binary, skipped where it is not installed. The
// image is produced by cchef's own Add Text To Image, so the whole path from
// rendering to recognition is covered.
func TestOCRAgainstRealTesseract(t *testing.T) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("tesseract is not installed")
	}
	image, err := os.ReadFile("testdata/ocr_hello.png")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	got, err := runOp(t, "Optical Character Recognition", string(image), ocrArgs(true, "LSTM only")...)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "Hello World") {
		t.Errorf("output = %q, want it to contain the rendered text", got)
	}
	if !strings.HasPrefix(got, "Confidence: ") {
		t.Errorf("output = %q, want a leading confidence line", got)
	}
}

// Losing the recognised-text file is reported rather than yielding empty output.
func TestReadOCROutputsMissing(t *testing.T) {
	if _, err := readOCROutputs(filepath.Join(t.TempDir(), "absent")); err == nil {
		t.Error("expected an error when tesseract's output is missing")
	}
}

// A missing TSV only costs the confidence figure, not the run.
func TestReadOCROutputsWithoutTSV(t *testing.T) {
	base := filepath.Join(t.TempDir(), "out")
	if err := os.WriteFile(base+".txt", []byte("text only\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readOCROutputs(base)
	if err != nil {
		t.Fatal(err)
	}
	if got.text != "text only\n" || got.tsv != "" {
		t.Errorf("got %+v, want the text with no TSV", got)
	}
}

// A temp directory that cannot be created fails the run.
func TestRunTesseractTempDirFailure(t *testing.T) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("tesseract is not installed")
	}
	t.Setenv("TMPDIR", filepath.Join(t.TempDir(), "does", "not", "exist"))
	if _, err := runTesseract([]byte("ignored"), 1); err == nil {
		t.Error("expected an error when the temp directory cannot be created")
	}
}

// tesseract's own failures carry its stderr through, so the cause is visible.
func TestRunTesseractReportsEngineFailure(t *testing.T) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		t.Skip("tesseract is not installed")
	}
	_, err := runTesseract([]byte("this is not an image"), 1)
	if err == nil {
		t.Fatal("expected an error for input tesseract cannot read")
	}
	if err.Error() == "" {
		t.Error("expected the engine's own message to be included")
	}
}
