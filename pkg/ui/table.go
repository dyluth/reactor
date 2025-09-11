package ui

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// TableRow represents a single row in a table
type TableRow struct {
	Columns []string
}

// TableRenderer provides consistent table formatting across the CLI
type TableRenderer struct {
	headers []string
	rows    []TableRow
}

// NewTableRenderer creates a new table renderer with the specified headers
func NewTableRenderer(headers []string) *TableRenderer {
	return &TableRenderer{
		headers: headers,
		rows:    make([]TableRow, 0),
	}
}

// AddRow adds a new row to the table
func (t *TableRenderer) AddRow(columns ...string) {
	t.rows = append(t.rows, TableRow{Columns: columns})
}

// Render outputs the table with consistent formatting
func (t *TableRenderer) Render() {
	if len(t.headers) == 0 {
		return
	}

	// Calculate column widths
	colWidths := make([]int, len(t.headers))

	// Start with header widths
	for i, header := range t.headers {
		colWidths[i] = len(header)
	}

	// Check all row columns
	for _, row := range t.rows {
		for i, col := range row.Columns {
			if i < len(colWidths) && len(col) > colWidths[i] {
				colWidths[i] = len(col)
			}
		}
	}

	// Render header
	headerColor := color.New(color.FgBlue, color.Bold)
	for i, header := range t.headers {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Printf("%-*s", colWidths[i], headerColor.Sprint(header))
	}
	fmt.Print("\n")

	// Render separator line
	for i, width := range colWidths {
		if i > 0 {
			fmt.Print(" ")
		}
		fmt.Print(strings.Repeat("-", width))
	}
	fmt.Print("\n")

	// Render rows
	for _, row := range t.rows {
		for i, col := range row.Columns {
			if i > 0 {
				fmt.Print(" ")
			}
			if i < len(colWidths) {
				fmt.Printf("%-*s", colWidths[i], col)
			} else {
				fmt.Print(col)
			}
		}
		fmt.Print("\n")
	}
}

// Colors for consistent CLI output
var (
	// SuccessColor for successful operations
	SuccessColor = color.New(color.FgGreen)
	// WarningColor for warnings
	WarningColor = color.New(color.FgYellow)
	// ErrorColor for errors
	ErrorColor = color.New(color.FgRed)
	// InfoColor for informational messages
	InfoColor = color.New(color.FgCyan)
)

// ServiceColors provides distinct colors for service log prefixes
var ServiceColors = []*color.Color{
	color.New(color.FgCyan),
	color.New(color.FgMagenta),
	color.New(color.FgYellow),
	color.New(color.FgGreen),
	color.New(color.FgBlue),
	color.New(color.FgRed),
	color.New(color.FgWhite),
	color.New(color.FgHiCyan),
}

// GetServiceColor returns a color for a service based on its index
func GetServiceColor(index int) *color.Color {
	return ServiceColors[index%len(ServiceColors)]
}
