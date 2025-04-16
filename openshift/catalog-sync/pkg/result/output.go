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
		base := []string{"Catalog"}
		if results[0].TotalPackages > 0 {
			base = append(base, "Total Packages")
		}

		// Determine if failure columns should be included based on test type
		includeFailureColumns := results[0].IsPackageFailureType
		if includeFailureColumns {
			base = append(base, "Failed Packages", "% of Failures")
		}

		// Collect optional columns dynamically
		optHeaders := make(map[string]struct{})
		for _, res := range results {
			for key := range res.OptionalColumns {
				optHeaders[key] = struct{}{}
			}
		}

		// Convert optional headers to a sorted slice
		optHeaderList := make([]string, 0, len(optHeaders))
		for key := range optHeaders {
			optHeaderList = append(optHeaderList, key)
		}

		sort.Strings(optHeaderList)

		// Finalize column list
		allColumns := base
		if len(optHeaderList) > 0 {
			allColumns = append(allColumns, optHeaderList...)
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

		// Print failure details for each table if any failures exist
		hasFailures := false
		for _, res := range results {
			if len(res.FailedPackageNames) > 0 {
				hasFailures = true
				break
			}
		}

		if hasFailures {
			fmt.Println("### Failure Details")
			for _, res := range results {
				if len(res.FailedPackageNames) == 0 {
					continue
				}
				fmt.Printf("#### %s\n", res.CatalogName)
				sort.Strings(res.FailedPackageNames)
				for _, pkg := range res.FailedPackageNames {
					fmt.Printf("- %s\n", pkg)
				}
				fmt.Println()
			}
		}
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
