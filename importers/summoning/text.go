package summoning

import (
	"fmt"

	"golang.org/x/text/encoding/charmap"
)

// textKey is XORed into every byte of a TEXT string.
const textKey = 0xDA

// ReadText reads TEXT: every string by record, decoded to UTF-8. The last record, END, is
// dropped.
func ReadText(data []byte) ([]string, error) {
	records, err := ReadContainer(data)
	if err != nil {
		return nil, fmt.Errorf("text: %w", err)
	}
	if len(records) > 0 {
		records = records[:len(records)-1]
	}
	out := make([]string, len(records))
	for i, rec := range records {
		b := make([]byte, 0, len(rec))
		for _, c := range rec {
			if c == 0 {
				break
			}
			b = append(b, c^textKey)
		}
		out[i] = decode437(b)
	}
	return out, nil
}

// decode437 converts code page 437 text to UTF-8.
func decode437(b []byte) string {
	s, err := charmap.CodePage437.NewDecoder().Bytes(b)
	if err != nil {
		return string(b)
	}
	return string(s)
}
