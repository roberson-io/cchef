package ops

import (
	"os"
	"strings"
	"testing"
)

func TestExtractEXIFEmpty(t *testing.T) {
	if out, err := runOp(t, "Extract EXIF", ""); err != nil || out != "Found 0 tags.\n" {
		t.Errorf("empty = %q, %v; want %q", out, err, "Found 0 tags.\n")
	}
}

func TestExtractEXIFNotImage(t *testing.T) {
	_, err := runOp(t, "Extract EXIF", "hello world")
	if err == nil {
		t.Fatal("expected error for non-image input")
	}
	want := "Could not extract EXIF data from image: Error: Invalid JPEG section offset"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}

// Transcribed from CyberChef tests/operations/tests/Image.mjs (meerkat jpeg).
func TestExtractEXIFMeerkat(t *testing.T) {
	data, err := os.ReadFile("testdata/exif_meerkat.jpg")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	want := strings.Join([]string{
		"Found 28 tags.",
		"",
		"Make: SONY",
		"Model: DSC-H5",
		"XResolution: 70",
		"YResolution: 70",
		"ResolutionUnit: 2",
		"Software: Pictomio 1.2.31.0",
		"ModifyDate: 1278286273",
		"ExposureTime: 0.008",
		"FNumber: 3.7",
		"ExposureProgram: 3",
		"ISO: 200",
		"DateTimeOriginal: 1220275486",
		"CreateDate: 1220275486",
		"ShutterSpeedValue: 6.965784",
		"ApertureValue: 3.775051",
		"ExposureCompensation: 0.3",
		"MaxApertureValue: 3",
		"MeteringMode: 5",
		"LightSource: 10",
		"Flash: 16",
		"FocalLength: 72",
		"CustomRendered: 0",
		"ExposureMode: 1",
		"WhiteBalance: 1",
		"SceneCaptureType: 0",
		"Contrast: 0",
		"Saturation: 0",
		"Sharpness: 0",
	}, "\n")
	if out, err := runOp(t, "Extract EXIF", string(data)); err != nil || out != want {
		t.Errorf("meerkat mismatch:\n got %q\nwant %q\nerr %v", out, want, err)
	}
}

// exif_gps.jpg is a hand-built big-endian TIFF exercising GPS coordinate casting,
// the Interop and IFD1 sub-directories, and inline vs. offset values. The expected
// output was captured from the CyberChef-server oracle.
func TestExtractEXIFGPS(t *testing.T) {
	data, err := os.ReadFile("testdata/exif_gps.jpg")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	want := strings.Join([]string{
		"Found 9 tags.",
		"",
		"Make: cchef",
		"Orientation: 1",
		"XResolution: 72",
		"GPSLatitudeRef: N",
		"GPSLatitude: 51.5",
		"GPSLongitudeRef: E",
		"GPSLongitude: 0.125",
		"ExposureTime: 0.01",
		"InteropIndex: ",
	}, "\n")
	if out, err := runOp(t, "Extract EXIF", string(data)); err != nil || out != want {
		t.Errorf("gps mismatch:\n got %q\nwant %q\nerr %v", out, want, err)
	}
}

// An APP1 segment that is not EXIF (e.g. XMP) is skipped, leaving no tags.
func TestExtractEXIFNonExifApp1(t *testing.T) {
	// SOI | APP1 (len 8) "AAAAAA" | SOS
	in := string(mustHex(t, "ffd8ffe10008414141414141ffda000200"))
	if out, err := runOp(t, "Extract EXIF", in); err != nil || out != "Found 0 tags.\n" {
		t.Errorf("non-exif APP1 = %q, %v; want %q", out, err, "Found 0 tags.\n")
	}
}

func TestExtractEXIFNoEXIF(t *testing.T) {
	data, err := os.ReadFile("testdata/exif_noexif.jpg")
	if err != nil {
		t.Fatalf("read testdata: %v", err)
	}
	if out, err := runOp(t, "Extract EXIF", string(data)); err != nil || out != "Found 0 tags.\n" {
		t.Errorf("no-exif = %q, %v; want %q", out, err, "Found 0 tags.\n")
	}
}
