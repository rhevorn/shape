package main

import (
	"github.com/rhevorn/shape/tools/shapevet/analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() { singlechecker.Main(analyzer.Analyzer) }
