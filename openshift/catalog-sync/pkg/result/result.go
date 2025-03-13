package result

// TestResult represents the outcome of a validation check for a catalog.
type TestResult struct {
	// TestID is a unique identifier for the test, allowing aggregation of results in summary tables.
	TestID string

	// TestContextTitle provides a descriptive title for the summary table of the test results.
	TestContextTitle string

	// CatalogName represents the alias of the catalog image being validated.
	CatalogName string

	// TotalPackages indicates the total number of packages in the catalog.
	TotalPackages int

	// FailedPackages represents the total number of packages in the catalog that failed the validation check.
	FailedPackages int

	// FailurePercentage calculates the percentage of packages that failed the check.
	FailurePercentage float64

	// OptionalColumns stores extra metadata or custom columns required for specific checks.
	// For example, if a test needs to indicate whether failures occur only in head channels,
	// this map can be used to store that information.
	OptionalColumns map[string]interface{}

	// IsPackageFailureType is true when we should output failed and percentage per package
	// Info reports should not have those columns
	IsPackageFailureType bool

	// FailedPackageNames contains the names of packages that failed the validation check.
	FailedPackageNames []string
}
