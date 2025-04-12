package analyzer

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	excludeTestFiles = false
	analysistest.Run(t, analysistest.TestData(), Analyzer,
		"a",
		"positive",
		"negative",
		"falsepositive",
		"dotimport",
	)
}

func TestAnalyzerExcludeTests(t *testing.T) {
	previous := excludeTestFiles
	excludeTestFiles = true
	t.Cleanup(func() { excludeTestFiles = previous })
	analysistest.Run(t, analysistest.TestData(), Analyzer, "excluded")
}
