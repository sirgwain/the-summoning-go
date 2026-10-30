package summoning

import "encoding/binary"

// fixture builds little-endian test data.
type fixture []byte

func (f fixture) u8(v ...uint8) fixture { return append(f, v...) }

func (f fixture) u16(v ...uint16) fixture {
	for _, x := range v {
		f = binary.LittleEndian.AppendUint16(f, x)
	}
	return f
}

func (f fixture) u32(v ...uint32) fixture {
	for _, x := range v {
		f = binary.LittleEndian.AppendUint32(f, x)
	}
	return f
}

func (f fixture) str(s string) fixture { return append(f, s...) }

func (f fixture) zeros(n int) fixture { return append(f, make([]byte, n)...) }

// container builds an indexed container holding records.
func container(records ...[]byte) []byte {
	f := fixture{}.u32(uint32(len(records)))
	off := 4 + 4*len(records)
	for _, r := range records {
		f = f.u32(uint32(off))
		off += len(r)
	}
	for _, r := range records {
		f = append(f, r...)
	}
	return f
}
