package main

import (
	"fmt"

	"github.com/balazscsaba2006/specflow/internal/ui"
)

// printTable prints a styled table with headers and rows.
func printTable(headers []string, rows [][]string) {
	fmt.Print(ui.Table(headers, rows))
}
