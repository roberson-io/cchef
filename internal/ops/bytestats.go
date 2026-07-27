package ops

import "math"

// The counting the byte-statistics operations share. Entropy, Frequency
// distribution and Chi Square all begin by tallying how often each byte value
// occurs; Index of Coincidence counts letters instead.

// byteCounts tallies how often each byte value occurs.
func byteCounts(data []byte) [256]int {
	var counts [256]int
	for _, b := range data {
		counts[b]++
	}
	return counts
}

// shannonEntropy measures how many bits of information each byte carries, from
// zero where every byte is the same to eight where they are evenly spread.
func shannonEntropy(data []byte) float64 {
	counts := byteCounts(data)

	entropy := 0.0
	for _, count := range counts {
		if count == 0 {
			continue
		}
		p := float64(count) / float64(len(data))
		entropy += p * math.Log(p) / math.Log(2)
	}
	return -entropy
}

// The block sizes the scanning entropy measures over: a short input is divided
// finely so it still yields several readings.
const (
	scanningBinSmall = 8
	scanningBinLarge = 256
)

// scanningEntropy measures the entropy of each block of the input in turn, so a
// stretch of encrypted or compressed data stands out from the rest.
func scanningEntropy(data []byte) []float64 {
	binWidth := scanningBinLarge
	if len(data) < scanningBinLarge {
		binWidth = scanningBinSmall
	}

	var out []float64
	for at := 0; at < len(data); at += binWidth {
		out = append(out, shannonEntropy(data[at:min(at+binWidth, len(data))]))
	}
	return out
}

// byteFrequency gives the proportion of the input each byte value makes up.
func byteFrequency(data []byte) []float64 {
	freq := make([]float64, 256)
	if len(data) == 0 {
		return freq
	}
	for value, count := range byteCounts(data) {
		freq[value] = float64(count) / float64(len(data))
	}
	return freq
}
