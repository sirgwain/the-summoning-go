package summoning

import (
	"bytes"
	"testing"
)

func TestReadContainer(t *testing.T) {
	data := container([]byte("one"), nil, []byte("three"))
	recs, err := ReadContainer(data)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"one", "", "three"}
	if len(recs) != len(want) {
		t.Fatalf("got %d records, want %d", len(recs), len(want))
	}
	for i, w := range want {
		if !bytes.Equal(recs[i], []byte(w)) {
			t.Errorf("record %d = %q, want %q", i, recs[i], w)
		}
	}
}

func TestReadContainerRejects(t *testing.T) {
	tests := map[string][]byte{
		"empty":             nil,
		"short directory":   fixture{}.u32(3, 16),
		"offset in header":  fixture{}.u32(1, 2).str("ab"),
		"offset past end":   fixture{}.u32(1, 99).str("ab"),
		"offsets backwards": fixture{}.u32(2, 14, 12).str("abcd"),
	}
	for name, data := range tests {
		if _, err := ReadContainer(data); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}
