package multichecker

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestMyAnalyzer(t *testing.T) {
	// функция analysistest.Run применяет тестируемый анализатор ExitCheckAnalyzer
	// к пакетам из папки testdata и проверяет ожидания
	analysistest.Run(t, analysistest.TestData(), ExitCheckAnalyzer, "./...")
}
