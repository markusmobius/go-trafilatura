package main

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

type Table struct {
	Headers []string
	Rows    [][]string
}

func NewTable() Table {
	return Table{}
}

func (t *Table) AddHeaders(columns ...string) {
	t.Headers = append(t.Headers, columns...)
}

func (t *Table) AddRow(columns ...string) {
	cells := slices.Clone(columns)
	t.Rows = append(t.Rows, cells)
}

func (t *Table) Print() {
	// Calculate columns width
	var widths []int
	widths = t.getRowWidth(widths, t.Headers)
	for _, row := range t.Rows {
		widths = t.getRowWidth(widths, row)
	}

	// Print headers
	t.printHorizontalLine(widths)
	t.printRow(widths, t.Headers)
	t.printHorizontalLine(widths)
	for _, row := range t.Rows {
		t.printRow(widths, row)
	}
	t.printHorizontalLine(widths)
}

func (t *Table) getRowWidth(widths []int, columns []string) []int {
	// Get slice size
	nColumn := len(columns)
	nWidth := len(widths)

	// If width size is smaller than column, increase its size to columns count
	if nWidth < nColumn {
		for range nColumn - nWidth {
			widths = append(widths, 0)
		}
	}

	// Calculate each column width
	for i, col := range columns {
		colWidth := utf8.RuneCountInString(col) + 2 // +2 for padding around cell
		if colWidth > widths[i] {
			widths[i] = colWidth
		}
	}

	return widths
}

func (t *Table) printHorizontalLine(widths []int) {
	var lines []string
	for _, width := range widths {
		lines = append(lines, strings.Repeat("-", width))
	}

	fmt.Println("+" + strings.Join(lines, "+") + "+")
}

func (t *Table) printRow(widths []int, columns []string) {
	var cells []string
	for i, column := range columns {
		columnSize := utf8.RuneCountInString(column)
		availableSpace := widths[i] - columnSize
		rightPadding := availableSpace / 2
		leftPadding := availableSpace - rightPadding

		cells = append(cells, fmt.Sprintf("%s%s%s",
			strings.Repeat(" ", leftPadding),
			column,
			strings.Repeat(" ", rightPadding),
		))
	}

	fmt.Println("|" + strings.Join(cells, "|") + "|")
}
