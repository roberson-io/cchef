package ops

import (
	"errors"

	"github.com/roberson-io/cchef/core"
)

func init() {
	core.Register(Entropy{})
}

// Entropy measures how evenly the bytes of the input are spread. Ported from
// CyberChef's Entropy.
type Entropy struct{}

// Meta returns the operation metadata.
func (Entropy) Meta() core.OpMeta {
	return core.OpMeta{
		Name:        "Entropy",
		Module:      "Default",
		Description: "Shannon Entropy, in the context of information theory, is a measure of the rate at which information is produced by a source of data. It can be used, as a heuristic, to detect encryption or compression.",
		InputType:   core.TypeArrayBuffer,
		OutputType:  core.TypeString,
	}
}

// Args returns the argument definitions.
func (Entropy) Args() []core.ArgDef {
	return []core.ArgDef{
		{
			Name: "Visualisation", Type: core.ArgOption,
			Value: []string{"Shannon scale", "Histogram (Bar)", "Histogram (Line)", "Curve", "Image"},
		},
	}
}

// The visualisations the operation offers.
const (
	entropyShannonScale      = "Shannon scale"
	entropyBarHistogramView  = "Histogram (Bar)"
	entropyLineHistogramView = "Histogram (Line)"
	entropyCurveView         = "Curve"
	entropyImageView         = "Image"
)

// Run measures the entropy. Each visualisation measures something different:
// the scale reads the whole input at once, the histograms count byte values,
// and the curve and image walk the input in blocks.
func (Entropy) Run(in *core.Dish, args []any) (*core.Dish, error) {
	data := in.Bytes()

	switch args[0].(string) {
	case entropyShannonScale:
		// CyberChef draws a scale bar beside the figure, which needs a browser:
		// the canvas is sized from the page and drawn by a script.
		return core.NewDish(
			[]byte("Shannon entropy: "+jsNumberString(shannonEntropy(data))),
			core.TypeString,
		), nil
	case entropyBarHistogramView:
		return core.NewDish([]byte(entropyBarHistogram(byteFrequency(data))), core.TypeString), nil
	case entropyLineHistogramView:
		return core.NewDish([]byte(entropyLineHistogram(byteFrequency(data))), core.TypeString), nil
	case entropyCurveView:
		return core.NewDish([]byte(entropyScanningCurve(scanningEntropy(data))), core.TypeString), nil
	case entropyImageView:
		return core.NewDish([]byte(entropyImage(scanningEntropy(data))), core.TypeString), nil
	}
	return nil, errors.New("unsupported visualisation")
}
