package utils

import (
	"bytes"
	"golang.org/x/crypto/ssh/terminal"
	"sort"
	"strings"
)

type Row []string

type PrettyTable struct {
	rows []Row
}

func (t *PrettyTable) AddRow(c ...string) {
	t.rows = append(t.rows, c)
}

func (t *PrettyTable) SortRows(col int) {
	sort.SliceStable(t.rows[1:], func(i, j int) bool {
		return t.rows[i+1][col] < t.rows[j+1][col]
	})
}

func (t *PrettyTable) Render(limitWidths []int) string {
	cols := len(t.rows[0])

	maxWidth := func(col int, maxW int) int {
		w := 0
		for _, l := range t.rows {
			if len(l[col]) > w {
				w = len(l[col])
			}
			if maxW != -1 {
				if maxW < w {
					w = maxW
				}
			}
		}
		return w
	}
	subStr := func(str string, s int, e int) string {
		if s > len(str) {
			s = len(str)
		}
		if e > len(str) {
			e = len(str)
		}
		return str[s:e]
	}

	widths := make([]int, cols)
	widthSum := 0
	for i := 0; i < cols; i++ {
		w := -1
		if i < len(limitWidths) {
			w = limitWidths[i]
		}
		widths[i] = maxWidth(i, w)
		widthSum += widths[i]
	}

	if len(limitWidths) < cols {
		tw, _, err := terminal.GetSize(0)
		if err != nil {
			tw = 80
		}
		// last column should use all remaining space
		widths[len(limitWidths)] = tw - widthSum - (cols-1)*3 - 4
	}

	hsep := "+-"
	for i := 0; i < cols; i++ {
		hsep += strings.Repeat("-", widths[i])
		if i != cols-1 {
			hsep += "-+-"
		}
	}
	hsep += "-+\n"

	buf := bytes.NewBuffer(nil)
	buf.WriteString(hsep)
	pos := make([]int, cols)
	for _, l := range t.rows {
		for i := 0; i < cols; i++ {
			pos[i] = 0
		}

		for {
			anyLess := false
			for i := 0; i < cols; i++ {
				if pos[i] < len(l[i]) {
					anyLess = true
				}
			}
			if !anyLess {
				break
			}

			buf.WriteString("| ")
			for i := 0; i < cols; i++ {
				x := subStr(l[i], pos[i], pos[i]+widths[i])
				newLine := strings.IndexRune(x, '\n')
				if newLine != -1 {
					x = x[:newLine]
					pos[i] += 1
				}
				pos[i] += len(x)
				buf.WriteString(x)
				buf.WriteString(strings.Repeat(" ", widths[i]-len(x)))
				if i != cols-1 {
					buf.WriteString(" | ")
				}
			}
			buf.WriteString(" |\n")
		}
		buf.WriteString(hsep)
	}
	return buf.String()
}
