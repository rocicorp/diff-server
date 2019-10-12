package tbl

import (
	"fmt"
	"io"
	"strconv"
)

// Table implements a simple two-column ascii table.
type Table struct {
	rows []row
}

type row struct {
	key   string
	value string
}

// Add a row to the table.
func (t *Table) Add(key, value string) *Table {
	t.rows = append(t.rows, row{key, value})
	return t
}

// WriteTo writes the table out.
func (t *Table) WriteTo(w io.Writer) (int64, error) {
	keyWidth := 0
	for _, r := range t.rows {
		kl := len(r.key)
		if kl > keyWidth {
			keyWidth = kl
		}
	}
	res := 0
	for _, r := range t.rows {
		n, err := fmt.Fprintf(w, "%-"+strconv.Itoa(keyWidth)+"s%s\n", r.key, r.value)
		if err != nil {
			return 0, err
		}
		res += n
	}
	return int64(res), nil
}
