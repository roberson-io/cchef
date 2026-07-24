package ops

import "testing"

func TestImageHueSaturationLightness(t *testing.T) {
	assertImageGolden(t, "Image Hue/Saturation/Lightness", "hsl.png", float64(40), float64(30), float64(20))
}

// A negative hue exercises the spin wrap-around branch.
func TestImageHueSaturationLightnessNegativeHue(t *testing.T) {
	assertImageGolden(t, "Image Hue/Saturation/Lightness", "hsl_negative_hue.png", float64(-40), float64(0), float64(0))
}

func TestImageHueSaturationLightnessInvalid(t *testing.T) {
	if _, err := runOp(t, "Image Hue/Saturation/Lightness", "not an image", float64(0), float64(0), float64(0)); err == nil {
		t.Error("expected error for non-image input")
	}
}
