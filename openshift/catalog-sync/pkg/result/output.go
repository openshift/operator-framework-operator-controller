package result

import (
	"fmt"
	"sort"
	"strings"
)

// OutputSummaryWith dynamically generates a table with only relevant columns.
func OutputSummaryWith(testResults []TestResult) {
	if len(testResults) == 0 {
		fmt.Println("\n# No test results to display.")
		return
	}

	fmt.Println("\n# Test Results Summary")

	// Group results by TestID
	contextResults := make(map[string][]TestResult)
	for _, result := range testResults {
		contextResults[result.TestID] = append(contextResults[result.TestID], result)
	}

	// Process each test case separately
	for _, results := range contextResults {
		fmt.Printf("\n## %s\n\n", results[0].TestContextTitle)

		// Define base columns
		baseColumns := []string{"Catalog", "Total Packages"}

		// Determine if failure columns should be included based on test type
		includeFailureColumns := results[0].IsPackageFailureType
		if includeFailureColumns {
			baseColumns = append(baseColumns, "Failed Packages", "% of Failures")
		}

		// Collect optional columns dynamically
		optionalHeaders := make(map[string]struct{})
		for _, res := range results {
			for key := range res.OptionalColumns {
				optionalHeaders[key] = struct{}{}
			}
		}

		// Convert optional headers to a sorted slice
		optionalHeaderList := make([]string, 0, len(optionalHeaders))
		for key := range optionalHeaders {
			optionalHeaderList = append(optionalHeaderList, key)
		}

		sort.Strings(optionalHeaderList)

		// Finalize column list
		allColumns := baseColumns
		if len(optionalHeaderList) > 0 {
			allColumns = append(allColumns, optionalHeaderList...)
		}

		// Calculate column widths
		columnWidths := make(map[string]int)
		for _, col := range allColumns {
			columnWidths[col] = len(col)
		}

		// Adjust column width based on data length
		for _, res := range results {
			for _, col := range allColumns {
				val := fmt.Sprintf("%v", getColumnValue(col, res))
				for _, line := range strings.Split(val, "\n") {
					if len(line) > columnWidths[col] {
						columnWidths[col] = len(line)
					}
				}
			}
		}

		// Print table headers
		fmt.Print("| ")
		for _, col := range allColumns {
			fmt.Printf("%-*s | ", columnWidths[col], col)
		}
		fmt.Println()

		// Print separator line
		fmt.Print("|")
		for _, col := range allColumns {
			fmt.Print(strings.Repeat("-", columnWidths[col]+2) + "|")
		}
		fmt.Println()

		// Print each row
		for _, res := range results {
			colLines := make([][]string, len(allColumns))
			maxLines := 1

			for i, col := range allColumns {
				val := fmt.Sprintf("%v", getColumnValue(col, res))
				lines := strings.Split(val, "\n")
				colLines[i] = lines
				if len(lines) > maxLines {
					maxLines = len(lines)
				}
			}

			for lineIdx := 0; lineIdx < maxLines; lineIdx++ {
				fmt.Print("| ")
				for i, col := range allColumns {
					lines := colLines[i]
					text := ""
					if lineIdx < len(lines) {
						text = lines[lineIdx]
					}
					fmt.Printf("%-*s | ", columnWidths[col], text)
				}
				fmt.Println()
			}
		}
		fmt.Println()
	}
}

// getColumnValue ensures proper handling of missing values and test types.
func getColumnValue(column string, res TestResult) interface{} {
	switch column {
	case "Catalog":
		return res.CatalogName
	case "Total Packages":
		return res.TotalPackages
	case "Failed Packages":
		if res.IsPackageFailureType {
			return res.FailedPackages
		}
		return "-"
	case "% of Failures":
		if res.IsPackageFailureType {
			return fmt.Sprintf("%.2f%%", res.FailurePercentage)
		}
		return "-"
	default:
		if val, ok := res.OptionalColumns[column]; ok {
			return val
		}
		return "-"
	}
}
