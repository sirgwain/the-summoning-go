package content

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Grid is a level layer: Grid[x][y] for cell (x, y). In JSON each row is one string of
// space-separated values, with "." for NoFrame, so a level diffs well.
type Grid [][]int

const gridNone = "."

func (g Grid) MarshalJSON() ([]byte, error) {
	rows := make([]string, len(g))
	for x, row := range g {
		tokens := make([]string, len(row))
		for y, v := range row {
			if v == NoFrame {
				tokens[y] = gridNone
			} else {
				tokens[y] = strconv.Itoa(v)
			}
		}
		rows[x] = strings.Join(tokens, " ")
	}
	return json.Marshal(rows)
}

func (g *Grid) UnmarshalJSON(data []byte) error {
	var rows []string
	if err := json.Unmarshal(data, &rows); err != nil {
		return err
	}
	*g = make(Grid, len(rows))
	for x, row := range rows {
		tokens := strings.Fields(row)
		(*g)[x] = make([]int, len(tokens))
		for y, t := range tokens {
			if t == gridNone {
				(*g)[x][y] = NoFrame
				continue
			}
			v, err := strconv.Atoi(t)
			if err != nil || v < 0 {
				return fmt.Errorf("grid row %d, column %d: bad value %q", x, y, t)
			}
			(*g)[x][y] = v
		}
	}
	return nil
}
