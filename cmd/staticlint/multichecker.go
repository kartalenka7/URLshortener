package multichecker

import (
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"honnef.co/go/tools/staticcheck"
)

func main() {
	checks := map[string]bool{
		"ST1019": true,
		"S1005":  true,
	}

	var mychecks []*analysis.Analyzer
	for _, v := range staticcheck.Analyzers {
		// добавляем все анализаторы SA пакета staticcheck
		// и по одному анализатору из пакетов ST1 и S1
		if strings.HasPrefix(v.Analyzer.Name, "SA") || checks[v.Analyzer.Name] {
			mychecks = append(mychecks, v.Analyzer)
		}
	}

	mychecks = append(mychecks, printf.Analyzer)
	mychecks = append(mychecks, shadow.Analyzer)
	mychecks = append(mychecks, structtag.Analyzer)

	multichecker.Main(
		mychecks...,
	)
}
