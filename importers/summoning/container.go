package summoning

import (
	"encoding/binary"
	"fmt"
)

// ReadContainer splits an indexed container (RESOURCE, TEXT, LEVELS) into its records. The
// records share data's backing array.
func ReadContainer(data []byte) ([][]byte, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("container: %d bytes, too short for a count", len(data))
	}
	count := uint64(binary.LittleEndian.Uint32(data))
	dirEnd := 4 + 4*count
	if dirEnd > uint64(len(data)) {
		return nil, fmt.Errorf("container: directory of %d records runs past the end of the file", count)
	}

	offsets := make([]int, count+1)
	for i := range count {
		offsets[i] = int(binary.LittleEndian.Uint32(data[4+4*i:]))
	}
	offsets[count] = len(data)

	records := make([][]byte, count)
	for i := range count {
		start, end := offsets[i], offsets[i+1]
		if uint64(start) < dirEnd || start > len(data) {
			return nil, fmt.Errorf("container: record %d offset %d outside the data", i, start)
		}
		if end < start {
			return nil, fmt.Errorf("container: record %d offset %d is after the next record's %d", i, start, end)
		}
		records[i] = data[start:end:end]
	}
	return records, nil
}
