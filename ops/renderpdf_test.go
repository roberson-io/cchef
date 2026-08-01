package ops

import (
	"encoding/base64"
	"testing"
)

const pdfSample = "%PDF-1.4\nsome pdf bytes"

func TestRenderPDFRawPassthrough(t *testing.T) {
	if out, err := runOp(t, "Render PDF", pdfSample, "Raw"); err != nil || out != pdfSample {
		t.Errorf("Raw/Raw = %q, %v; want %q", out, err, pdfSample)
	}
	b64 := base64.StdEncoding.EncodeToString([]byte(pdfSample))
	if out, err := runOp(t, "Render PDF", b64, "Base64"); err != nil || out != pdfSample {
		t.Errorf("Base64/Raw = %q, %v; want %q", out, err, pdfSample)
	}
}

func TestRenderPDFEmpty(t *testing.T) {
	if out, err := runOp(t, "Render PDF", "", "Raw"); err != nil || out != "" {
		t.Errorf("empty = %q, %v; want empty", out, err)
	}
}

// Transcribed from CyberChef tests/operations/tests/RenderPDF.mjs.
func TestRenderPDFInvalid(t *testing.T) {
	for _, in := range []string{"Not a PDF", "%PD"} { // wrong magic; too short
		if _, err := runOp(t, "Render PDF", in, "Raw"); err == nil {
			t.Errorf("expected error for %q", in)
		}
	}
}
