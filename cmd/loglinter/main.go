package main

import (
	"github.com/brunp-669/loglinter/analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	// singlechecker запускает один анализатор как отдельную команду.
	singlechecker.Main(analyzer.Analyzer)
}
