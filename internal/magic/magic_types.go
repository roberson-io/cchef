package magic

// magicCheck is one operation's claim that it can decode data of a given shape:
// the data must match the pattern and fall inside the entropy range, if either
// is given. Output describes what the result must look like for the claim to
// hold once the operation has actually run.
type magicCheck struct {
	Op           string
	Pattern      string
	Args         []any
	Useful       bool
	EntropyRange []float64
	Output       *magicOutputCheck
}

// magicOutputCheck is what an operation's result must look like for its check
// to count.
type magicOutputCheck struct {
	Pattern      string
	EntropyRange []float64
	Mime         string
}
