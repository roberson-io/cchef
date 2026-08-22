package ops

import (
	"errors"
	"fmt"
	"slices"
)

// blockApplyPadding pads message to a whole number of blockSize-byte blocks for
// the chosen scheme (PKCS5/ZERO/RANDOM/BIT). PKCS5 always adds a full block when
// the input is already aligned; NO padding errors on a partial block. This is the
// padding CyberChef's PRESENT and RC6 block ciphers share.
func blockApplyPadding(message []byte, padding string, blockSize int) ([]byte, error) {
	remainder := len(message) % blockSize
	nPadding := 0
	if remainder != 0 {
		nPadding = blockSize - remainder
	}
	if padding == "PKCS5" && remainder == 0 {
		nPadding = blockSize
	}
	if nPadding == 0 {
		return append([]byte{}, message...), nil
	}
	padded := append([]byte{}, message...)
	switch padding {
	case "PKCS5":
		for range nPadding {
			padded = append(padded, byte(nPadding))
		}
		return padded, nil
	case "ZERO":
		return append(padded, make([]byte, nPadding)...), nil
	case "RANDOM":
		for range nPadding {
			padded = append(padded, byte(randInt(256))) // #nosec G115 -- randInt(256) is always in [0,255]
		}
		return padded, nil
	case "BIT":
		padded = append(padded, 0x80)
		return append(padded, make([]byte, nPadding-1)...), nil
	}
	// The only remaining option is "NO", which cannot pad a partial block.
	//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
	return nil, fmt.Errorf("No padding requested but input is not a %d-byte multiple.", blockSize)
}

// blockRemovePadding strips padding after decryption. NO/ZERO/RANDOM leave the
// message unchanged (they cannot be reliably removed).
func blockRemovePadding(message []byte, padding string, blockSize int) ([]byte, error) {
	if len(message) == 0 {
		return message, nil
	}
	switch padding {
	case "PKCS5":
		padByte := int(message[len(message)-1])
		if padByte > 0 && padByte <= blockSize {
			for i := range padByte {
				if message[len(message)-1-i] != byte(padByte) { // #nosec G115 -- padByte is int(a byte), always 0-255
					//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
					return nil, errors.New("Invalid PKCS#5 padding.")
				}
			}
			return message[:len(message)-padByte], nil
		}
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Invalid PKCS#5 padding.")
	case "BIT":
		for i, m := range slices.Backward(message) {
			if m == 0x80 {
				return message[:i], nil
			} else if m != 0 {
				//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
				return nil, errors.New("Invalid BIT padding.")
			}
		}
		//nolint:staticcheck,revive // CyberChef's verbatim OperationError text
		return nil, errors.New("Invalid BIT padding.")
	}
	return message, nil // NO / ZERO / RANDOM
}
