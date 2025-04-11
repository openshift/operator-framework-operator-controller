package pkg

import "fmt"

// FormatPercentage will format the percentage
func FormatPercentage(part, total int) string {
	if total == 0 {
		return "0.00%"
	}
	return fmt.Sprintf("%.2f%%", (float64(part)/float64(total))*100)
}
