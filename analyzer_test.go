package analyzer_test

import (
	"testing"

	"github.com/bruno-669/loglinter/analyzer"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), analyzer.Analyzer, "./src")
}

func TestFix(t *testing.T) {
	analysistest.RunWithSuggestedFixes(t, analysistest.TestData(), analyzer.Analyzer, "./fix")
}
