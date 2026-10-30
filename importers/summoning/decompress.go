package summoning

import "fmt"

// Sprite bank compression modes.
const (
	CompressionRaw  = 0
	CompressionRLE  = 1
	CompressionLZSS = 3
)

// decompress expands src into exactly n bytes with the given compression mode.
func decompress(mode uint8, src []byte, n int) ([]byte, error) {
	switch mode {
	case CompressionRaw:
		if len(src) < n {
			return nil, fmt.Errorf("raw: %d bytes, want %d", len(src), n)
		}
		return src[:n:n], nil
	case CompressionRLE:
		return decompressRLE(src, n)
	case CompressionLZSS:
		return decompressLZSS(src, n)
	default:
		return nil, fmt.Errorf("unsupported compression mode %d", mode)
	}
}

func decompressRLE(src []byte, n int) ([]byte, error) {
	out := make([]byte, 0, n)
	i := 0
	for len(out) < n {
		if i >= len(src) {
			return nil, fmt.Errorf("rle: input ends after %d of %d bytes", len(out), n)
		}
		c := int8(src[i])
		i++
		if c >= 0 {
			run := min(int(c)+1, n-len(out))
			if i+run > len(src) {
				return nil, fmt.Errorf("rle: literal run past the end of the input")
			}
			out = append(out, src[i:i+run]...)
			i += run
		} else {
			if i >= len(src) {
				return nil, fmt.Errorf("rle: repeat run past the end of the input")
			}
			b := src[i]
			i++
			for range min(1-int(c), n-len(out)) {
				out = append(out, b)
			}
		}
	}
	return out, nil
}

const (
	lzssRingSize  = 4096
	lzssRingStart = 0xFEE
	lzssFill      = 0xFE
	lzssMinMatch  = 3
)

func decompressLZSS(src []byte, n int) ([]byte, error) {
	var ring [lzssRingSize]byte
	for i := range lzssRingStart {
		ring[i] = lzssFill
	}
	r := lzssRingStart

	out := make([]byte, 0, n)
	put := func(b byte) {
		out = append(out, b)
		ring[r] = b
		r = (r + 1) % lzssRingSize
	}

	i := 0
	for len(out) < n {
		if i >= len(src) {
			return nil, fmt.Errorf("lzss: input ends after %d of %d bytes", len(out), n)
		}
		flags := src[i]
		i++
		for bit := 0; bit < 8 && len(out) < n; bit++ {
			if flags&(1<<bit) != 0 {
				if i >= len(src) {
					return nil, fmt.Errorf("lzss: input ends after %d of %d bytes", len(out), n)
				}
				put(src[i])
				i++
				continue
			}
			if i+2 > len(src) {
				return nil, fmt.Errorf("lzss: input ends after %d of %d bytes", len(out), n)
			}
			a, b := int(src[i]), int(src[i+1])
			i += 2
			pos := a | (b&0xF0)<<4
			length := b&0x0F + lzssMinMatch
			for k := 0; k < length && len(out) < n; k++ {
				put(ring[(pos+k)%lzssRingSize])
			}
		}
	}
	return out, nil
}
