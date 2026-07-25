package ops

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/roberson-io/cchef/internal/core"
)

func init() {
	core.Register(OpticalCharacterRecognition{})
}

// OpticalCharacterRecognition reads text out of an image. Ported from
// CyberChef's Optical Character Recognition, which runs Tesseract compiled to
// WebAssembly in the browser. cchef drives the locally installed `tesseract`
// binary instead: the only cgo-free Go bindings run Tesseract under a
// WebAssembly runtime, and the one library doing so is unmaintained, so shelling
// out is what keeps cchef a single static binary. Same engine and language data,
// so the recognised text matches.
type OpticalCharacterRecognition struct{}

// Meta returns the operation metadata.
func (OpticalCharacterRecognition) Meta() core.OpMeta {
	return core.OpMeta{
		Name:   "Optical Character Recognition",
		Module: "OCR",
		Description: "Optical character recognition or optical character reader (OCR) is the mechanical " +
			"or electronic conversion of images of typed, handwritten or printed text into " +
			"machine-encoded text.<br><br>Supported image formats: png, jpg, bmp, pbm.",
		InfoURL:    "https://wikipedia.org/wiki/Optical_character_recognition",
		InputType:  core.TypeArrayBuffer,
		OutputType: core.TypeString,
	}
}

// ocrEngineModes are CyberChef's engine choices; their order is tesseract's
// --oem numbering.
var ocrEngineModes = []string{"Tesseract only", "LSTM only", "Tesseract/LSTM Combined"}

// ocrDefaultEngineMode selects "LSTM only", as CyberChef does.
const ocrDefaultEngineMode = 1

// Args returns the argument definitions.
func (OpticalCharacterRecognition) Args() []core.ArgDef {
	return []core.ArgDef{
		{Name: "Show confidence", Type: core.ArgBoolean, Value: true},
		{Name: "OCR Engine Mode", Type: core.ArgOption, Value: ocrEngineModes, DefaultIndex: ocrDefaultEngineMode},
	}
}

// ocrOutput is what one tesseract run produced: the recognised text, and the
// per-word TSV the confidence is derived from.
type ocrOutput struct {
	text string
	tsv  string
}

// runOCR invokes tesseract. It is a variable so tests can exercise the operation
// without the binary installed.
var runOCR = runTesseract

// Run recognises text in the image.
func (OpticalCharacterRecognition) Run(in *core.Dish, args []any) (*core.Dish, error) {
	showConfidence := args[0].(bool)
	oem := slices.Index(ocrEngineModes, args[1].(string))

	if isImage(in.Bytes()) == "" {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Unsupported file type (supported: jpg,png,pbm,bmp) or no file provided")
	}

	out, err := runOCR(in.Bytes(), oem)
	if err != nil {
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, fmt.Errorf("Error performing OCR on image. (%w)", err)
	}

	text := out.text
	if showConfidence {
		text = "Confidence: " + strconv.Itoa(meanWordConfidence(out.tsv)) + "%\n\n" + text
	}
	return core.NewDish([]byte(text), core.TypeString), nil
}

// tesseractBinary is the OCR engine cchef drives.
const tesseractBinary = "tesseract"

// runTesseract runs tesseract over the image bytes, asking for both plain text
// and the per-word TSV in a single pass.
func runTesseract(image []byte, oem int) (ocrOutput, error) {
	if _, err := exec.LookPath(tesseractBinary); err != nil {
		return ocrOutput{}, fmt.Errorf(
			"%s is not installed or not on PATH; install it to use this operation "+
				"(macOS: brew install tesseract, Debian/Ubuntu: apt install tesseract-ocr)",
			tesseractBinary)
	}

	dir, err := os.MkdirTemp("", "cchef-ocr-")
	if err != nil {
		return ocrOutput{}, err
	}
	defer func() { _ = os.RemoveAll(dir) }()

	// tesseract reads the image from stdin and writes <base>.txt and <base>.tsv.
	base := filepath.Join(dir, "out")
	cmd := exec.Command(tesseractBinary, "-", base, "--oem", strconv.Itoa(oem), "-l", "eng", "txt", "tsv") // #nosec G204 -- the binary name is a constant and the only variable argument is a bounded integer
	cmd.Stdin = bytes.NewReader(image)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return ocrOutput{}, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return readOCROutputs(base)
}

// readOCROutputs collects the files tesseract wrote for one run.
func readOCROutputs(base string) (ocrOutput, error) {
	text, err := os.ReadFile(base + ".txt") // #nosec G304 -- base is a path this process just created under the temp directory
	if err != nil {
		return ocrOutput{}, err
	}
	// The TSV is only needed for the confidence figure, so a missing one is not
	// worth failing the whole run over.
	tsv, _ := os.ReadFile(base + ".tsv") // #nosec G304 -- as above
	return ocrOutput{text: string(text), tsv: string(tsv)}, nil
}

// TSV columns tesseract emits; only these two are needed here.
const (
	ocrTSVConfColumn = 10
	ocrTSVMinColumns = ocrTSVConfColumn + 1
)

// meanWordConfidence averages the confidence of the recognised words, which is
// what Tesseract's mean text confidence — the figure CyberChef reports —
// measures. Rows describing structure rather than a word carry a confidence of
// -1 and are skipped.
func meanWordConfidence(tsv string) int {
	total, count := 0.0, 0
	for line := range strings.SplitSeq(tsv, "\n") {
		columns := strings.Split(line, "\t")
		if len(columns) < ocrTSVMinColumns {
			continue
		}
		conf, err := strconv.ParseFloat(columns[ocrTSVConfColumn], 64)
		if err != nil || conf < 0 {
			continue
		}
		total += conf
		count++
	}
	if count == 0 {
		return 0
	}
	return int(math.Round(total / float64(count)))
}
